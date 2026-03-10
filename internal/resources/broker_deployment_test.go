// Copyright (c) Snyk Ltd.

package resources

import (
	"context"
	"errors"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/snyk-labs/snyk-broker-provider/internal/client"
	"github.com/snyk-labs/snyk-broker-provider/internal/common"
	"github.com/snyk-labs/snyk-broker-provider/internal/testutil"
)

var _ = Describe("BrokerDeploymentResource", func() {
	var (
		r        *BrokerDeploymentResource
		mockHTTP *testutil.MockHTTPClient
		ctx      context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
		mockHTTP = &testutil.MockHTTPClient{}
		mockAuth := &testutil.MockAuthenticator{}
		c := client.NewClient("https://api.snyk.io", mockAuth, client.WithHTTPClient(mockHTTP))

		r = &BrokerDeploymentResource{
			client: c,
		}
	})

	Describe("Metadata", func() {
		It("returns the correct type name", func() {
			req := resource.MetadataRequest{
				ProviderTypeName: "snyk",
			}
			resp := &resource.MetadataResponse{}

			r.Metadata(ctx, req, resp)

			Expect(resp.TypeName).To(Equal("snyk_broker_deployment"))
		})
	})

	Describe("Schema", func() {
		It("returns a valid schema with required attributes", func() {
			req := resource.SchemaRequest{}
			resp := &resource.SchemaResponse{}

			r.Schema(ctx, req, resp)

			Expect(resp.Schema.Attributes).To(HaveKey("id"))
			Expect(resp.Schema.Attributes).To(HaveKey("tenant_id"))
			Expect(resp.Schema.Attributes).To(HaveKey("install_id"))
			Expect(resp.Schema.Attributes).To(HaveKey("org_id"))
			Expect(resp.Schema.Attributes).To(HaveKey("name"))
			Expect(resp.Schema.Attributes).To(HaveKey("metadata"))
			Expect(resp.Schema.Attributes).To(HaveKey("client_id"))
			Expect(resp.Schema.Attributes).To(HaveKey("client_secret"))
		})
	})

	Describe("Configure", func() {
		It("sets the client from provider data", func() {
			mockAuth := &testutil.MockAuthenticator{}
			c := client.NewClient("https://api.snyk.io", mockAuth)
			providerData := &common.ProviderData{Client: c}

			newResource := &BrokerDeploymentResource{}
			req := resource.ConfigureRequest{
				ProviderData: providerData,
			}
			resp := &resource.ConfigureResponse{}

			newResource.Configure(ctx, req, resp)

			Expect(resp.Diagnostics.HasError()).To(BeFalse())
			Expect(newResource.client).To(Equal(c))
		})

		It("handles nil provider data gracefully", func() {
			newResource := &BrokerDeploymentResource{}
			req := resource.ConfigureRequest{
				ProviderData: nil,
			}
			resp := &resource.ConfigureResponse{}

			newResource.Configure(ctx, req, resp)

			Expect(resp.Diagnostics.HasError()).To(BeFalse())
		})

		It("returns error for invalid provider data type", func() {
			newResource := &BrokerDeploymentResource{}
			req := resource.ConfigureRequest{
				ProviderData: "invalid",
			}
			resp := &resource.ConfigureResponse{}

			newResource.Configure(ctx, req, resp)

			Expect(resp.Diagnostics.HasError()).To(BeTrue())
		})
	})

	Describe("Client API calls", func() {
		Context("CreateBrokerDeployment", func() {
			It("creates the deployment successfully", func() {
				mockHTTP.DoFunc = func(req *http.Request) (*http.Response, error) {
					Expect(req.Method).To(Equal(http.MethodPost))
					Expect(req.URL.Path).To(ContainSubstring("/deployments"))
					return testutil.NewMockJSONResponse(http.StatusCreated, `{
						"data": {
							"id": "deploy-123",
							"type": "broker_deployment",
							"attributes": {
								"install_id": "install-456",
								"client_id": "oauth-client-id",
								"client_secret": "oauth-client-secret",
								"metadata": {
									"deployment_name": "Test Deployment"
								}
							}
						}
					}`), nil
				}

				metadata := map[string]interface{}{
					"deployment_name": "Test Deployment",
				}
				result, err := r.client.CreateBrokerDeployment(ctx, "tenant-123", "install-456", "org-789", metadata)

				Expect(err).NotTo(HaveOccurred())
				Expect(result).NotTo(BeNil())
				Expect(result.Data.ID).To(Equal("deploy-123"))
				Expect(result.Data.Attributes.ClientID).To(Equal("oauth-client-id"))
				Expect(result.Data.Attributes.ClientSecret).To(Equal("oauth-client-secret"))
			})
		})

		Context("GetBrokerDeployment", func() {
			It("retrieves the deployment", func() {
				mockHTTP.DoFunc = func(req *http.Request) (*http.Response, error) {
					Expect(req.Method).To(Equal(http.MethodGet))
					// GetBrokerDeployment now uses ListBrokerDeployments internally,
					// so we need to return a list response format
					return testutil.NewMockJSONResponse(http.StatusOK, `{
						"data": [
							{
								"id": "deploy-123",
								"type": "broker_deployment",
								"attributes": {
									"install_id": "install-456",
									"metadata": {
										"deployment_name": "Test Deployment"
									}
								}
							}
						]
					}`), nil
				}

				result, err := r.client.GetBrokerDeployment(ctx, "tenant-123", "install-456", "deploy-123")

				Expect(err).NotTo(HaveOccurred())
				Expect(result).NotTo(BeNil())
				Expect(result.Data.ID).To(Equal("deploy-123"))
			})

			It("returns a 404 error for non-existent deployment", func() {
				mockHTTP.DoFunc = func(req *http.Request) (*http.Response, error) {
					// Return an empty list - the deployment is not found
					return testutil.NewMockJSONResponse(http.StatusOK, `{"data": []}`), nil
				}

				result, err := r.client.GetBrokerDeployment(ctx, "tenant-123", "install-456", "nonexistent")

				Expect(err).To(HaveOccurred())
				Expect(result).To(BeNil())

				var apiErr *client.APIError
				Expect(errors.As(err, &apiErr)).To(BeTrue())
				Expect(apiErr.StatusCode).To(Equal(http.StatusNotFound))
			})

			It("finds deployment across paginated results", func() {
				callCount := 0
				mockHTTP.DoFunc = func(req *http.Request) (*http.Response, error) {
					Expect(req.Method).To(Equal(http.MethodGet))
					callCount++

					if callCount == 1 {
						// First page - deployment not here, has next page
						return testutil.NewMockJSONResponse(http.StatusOK, `{
							"data": [
								{
									"id": "deploy-other",
									"type": "broker_deployment",
									"attributes": {
										"install_id": "install-456",
										"metadata": {"deployment_name": "Other Deployment"}
									}
								}
							],
							"links": {
								"next": "https://api.snyk.io/rest/tenants/tenant-123/brokers/installs/install-456/deployments?version=2024-02-08~experimental&starting_after=abc123"
							}
						}`), nil
					}

					// Second page - deployment is here
					return testutil.NewMockJSONResponse(http.StatusOK, `{
						"data": [
							{
								"id": "deploy-target",
								"type": "broker_deployment",
								"attributes": {
									"install_id": "install-456",
									"metadata": {"deployment_name": "Target Deployment"}
								}
							}
						],
						"links": {}
					}`), nil
				}

				result, err := r.client.GetBrokerDeployment(ctx, "tenant-123", "install-456", "deploy-target")

				Expect(err).NotTo(HaveOccurred())
				Expect(result).NotTo(BeNil())
				Expect(result.Data.ID).To(Equal("deploy-target"))
				Expect(callCount).To(Equal(2))
			})
		})

		Context("ListBrokerDeployments", func() {
			It("handles pagination and returns all deployments", func() {
				callCount := 0
				mockHTTP.DoFunc = func(req *http.Request) (*http.Response, error) {
					Expect(req.Method).To(Equal(http.MethodGet))
					callCount++

					if callCount == 1 {
						// First page
						return testutil.NewMockJSONResponse(http.StatusOK, `{
							"data": [
								{"id": "deploy-1", "type": "broker_deployment", "attributes": {"install_id": "install-456"}},
								{"id": "deploy-2", "type": "broker_deployment", "attributes": {"install_id": "install-456"}}
							],
							"links": {
								"next": "https://api.snyk.io/rest/tenants/tenant-123/brokers/installs/install-456/deployments?starting_after=deploy-2"
							}
						}`), nil
					}

					// Second page (final)
					return testutil.NewMockJSONResponse(http.StatusOK, `{
						"data": [
							{"id": "deploy-3", "type": "broker_deployment", "attributes": {"install_id": "install-456"}}
						],
						"links": {}
					}`), nil
				}

				result, err := r.client.ListBrokerDeployments(ctx, "tenant-123", "install-456")

				Expect(err).NotTo(HaveOccurred())
				Expect(result).NotTo(BeNil())
				Expect(result.Data).To(HaveLen(3))
				Expect(result.Data[0].ID).To(Equal("deploy-1"))
				Expect(result.Data[1].ID).To(Equal("deploy-2"))
				Expect(result.Data[2].ID).To(Equal("deploy-3"))
				Expect(callCount).To(Equal(2))
			})

			It("returns single page when no pagination links", func() {
				mockHTTP.DoFunc = func(req *http.Request) (*http.Response, error) {
					return testutil.NewMockJSONResponse(http.StatusOK, `{
						"data": [
							{"id": "deploy-1", "type": "broker_deployment", "attributes": {"install_id": "install-456"}}
						],
						"links": {}
					}`), nil
				}

				result, err := r.client.ListBrokerDeployments(ctx, "tenant-123", "install-456")

				Expect(err).NotTo(HaveOccurred())
				Expect(result.Data).To(HaveLen(1))
			})
		})

		Context("UpdateBrokerDeployment", func() {
			It("updates the deployment metadata", func() {
				mockHTTP.DoFunc = func(req *http.Request) (*http.Response, error) {
					Expect(req.Method).To(Equal(http.MethodPatch))
					return testutil.NewMockJSONResponse(http.StatusOK, `{
						"data": {
							"id": "deploy-123",
							"type": "broker_deployment",
							"attributes": {
								"metadata": {
									"deployment_name": "Updated Deployment"
								}
							}
						}
					}`), nil
				}

				metadata := map[string]interface{}{
					"deployment_name": "Updated Deployment",
				}
				result, err := r.client.UpdateBrokerDeployment(ctx, "tenant-123", "install-456", "deploy-123", metadata)

				Expect(err).NotTo(HaveOccurred())
				Expect(result).NotTo(BeNil())
			})
		})

		Context("DeleteBrokerDeployment", func() {
			It("deletes the deployment", func() {
				mockHTTP.DoFunc = func(req *http.Request) (*http.Response, error) {
					Expect(req.Method).To(Equal(http.MethodDelete))
					return testutil.NewMockJSONResponse(http.StatusNoContent, ""), nil
				}

				err := r.client.DeleteBrokerDeployment(ctx, "tenant-123", "install-456", "deploy-123")
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})

	Describe("ImportState", func() {
		It("returns an error since import is not fully supported", func() {
			req := resource.ImportStateRequest{}
			resp := &resource.ImportStateResponse{}

			r.ImportState(ctx, req, resp)

			Expect(resp.Diagnostics.HasError()).To(BeTrue())
			Expect(resp.Diagnostics.Errors()[0].Summary()).To(Equal("Import Not Fully Supported"))
		})
	})
})
