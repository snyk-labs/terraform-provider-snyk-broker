// Copyright (c) Snyk Ltd.

package resources

import (
	"context"
	"errors"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/snyk/terraform-provider-snyk-broker/internal/client"
	"github.com/snyk/terraform-provider-snyk-broker/internal/common"
	"github.com/snyk/terraform-provider-snyk-broker/internal/testutil"
)

var _ = Describe("BrokerConnectionResource", func() {
	var (
		r        *BrokerConnectionResource
		mockHTTP *testutil.MockHTTPClient
		ctx      context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
		mockHTTP = &testutil.MockHTTPClient{}
		mockAuth := &testutil.MockAuthenticator{}
		c := client.NewClient("https://api.snyk.io", mockAuth, client.WithHTTPClient(mockHTTP))

		r = &BrokerConnectionResource{
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

			Expect(resp.TypeName).To(Equal("snyk_broker_connection"))
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
			Expect(resp.Schema.Attributes).To(HaveKey("deployment_id"))
			Expect(resp.Schema.Attributes).To(HaveKey("name"))
			Expect(resp.Schema.Attributes).To(HaveKey("type"))
			Expect(resp.Schema.Attributes).To(HaveKey("configuration"))
		})
	})

	Describe("Configure", func() {
		It("sets the client from provider data", func() {
			mockAuth := &testutil.MockAuthenticator{}
			c := client.NewClient("https://api.snyk.io", mockAuth)
			providerData := &common.ProviderData{Client: c}

			newResource := &BrokerConnectionResource{}
			req := resource.ConfigureRequest{
				ProviderData: providerData,
			}
			resp := &resource.ConfigureResponse{}

			newResource.Configure(ctx, req, resp)

			Expect(resp.Diagnostics.HasError()).To(BeFalse())
			Expect(newResource.client).To(Equal(c))
		})

		It("handles nil provider data gracefully", func() {
			newResource := &BrokerConnectionResource{}
			req := resource.ConfigureRequest{
				ProviderData: nil,
			}
			resp := &resource.ConfigureResponse{}

			newResource.Configure(ctx, req, resp)

			Expect(resp.Diagnostics.HasError()).To(BeFalse())
		})

		It("returns error for invalid provider data type", func() {
			newResource := &BrokerConnectionResource{}
			req := resource.ConfigureRequest{
				ProviderData: "invalid",
			}
			resp := &resource.ConfigureResponse{}

			newResource.Configure(ctx, req, resp)

			Expect(resp.Diagnostics.HasError()).To(BeTrue())
		})
	})

	Describe("Client API calls", func() {
		Context("CreateBrokerConnection", func() {
			It("creates the connection successfully", func() {
				mockHTTP.DoFunc = func(req *http.Request) (*http.Response, error) {
					Expect(req.Method).To(Equal(http.MethodPost))
					Expect(req.URL.Path).To(ContainSubstring("/connections"))
					return testutil.NewMockJSONResponse(http.StatusCreated, `{
						"data": {
							"id": "conn-123",
							"type": "broker_connection",
							"attributes": {
								"deployment_id": "deploy-456",
								"name": "GitHub Connection",
								"configuration": {
									"type": "github",
									"required": {
										"github_token": "${GITHUB_TOKEN}",
										"broker_client_url": "https://broker.example.com"
									}
								}
							}
						}
					}`), nil
				}

				config := client.BrokerConnectionConfiguration{
					Type: "github",
					Required: map[string]interface{}{
						"github_token":      "cred-ref-123",
						"broker_client_url": "https://broker.example.com",
					},
				}
				result, err := r.client.CreateBrokerConnection(ctx, "tenant-123", "install-456", "deploy-789", "GitHub Connection", config)

				Expect(err).NotTo(HaveOccurred())
				Expect(result).NotTo(BeNil())
				Expect(result.Data.ID).To(Equal("conn-123"))
				Expect(result.Data.Attributes.Name).To(Equal("GitHub Connection"))
				Expect(result.Data.Attributes.Configuration.Type).To(Equal("github"))
			})
		})

		Context("GetBrokerConnection", func() {
			It("retrieves the connection", func() {
				mockHTTP.DoFunc = func(req *http.Request) (*http.Response, error) {
					Expect(req.Method).To(Equal(http.MethodGet))
					return testutil.NewMockJSONResponse(http.StatusOK, `{
						"data": {
							"id": "conn-123",
							"type": "broker_connection",
							"attributes": {
								"name": "GitHub Connection",
								"configuration": {
									"type": "github",
									"required": {}
								}
							}
						}
					}`), nil
				}

				result, err := r.client.GetBrokerConnection(ctx, "tenant-123", "install-456", "deploy-789", "conn-123")

				Expect(err).NotTo(HaveOccurred())
				Expect(result).NotTo(BeNil())
				Expect(result.Data.ID).To(Equal("conn-123"))
			})

			It("returns a 404 error for non-existent connection", func() {
				mockHTTP.DoFunc = func(req *http.Request) (*http.Response, error) {
					return testutil.NewMockJSONResponse(http.StatusNotFound, `{"error": "not found"}`), nil
				}

				result, err := r.client.GetBrokerConnection(ctx, "tenant-123", "install-456", "deploy-789", "nonexistent")

				Expect(err).To(HaveOccurred())
				Expect(result).To(BeNil())

				var apiErr *client.APIError
				Expect(errors.As(err, &apiErr)).To(BeTrue())
				Expect(apiErr.StatusCode).To(Equal(http.StatusNotFound))
			})
		})

		Context("ListBrokerConnections", func() {
			It("lists all connections for a deployment", func() {
				mockHTTP.DoFunc = func(req *http.Request) (*http.Response, error) {
					Expect(req.Method).To(Equal(http.MethodGet))
					return testutil.NewMockJSONResponse(http.StatusOK, `{
						"data": [
							{
								"id": "conn-1",
								"type": "broker_connection",
								"attributes": {
									"name": "GitHub",
									"configuration": {"type": "github", "required": {}}
								}
							},
							{
								"id": "conn-2",
								"type": "broker_connection",
								"attributes": {
									"name": "GitLab",
									"configuration": {"type": "gitlab", "required": {}}
								}
							}
						]
					}`), nil
				}

				result, err := r.client.ListBrokerConnections(ctx, "tenant-123", "install-456", "deploy-789")

				Expect(err).NotTo(HaveOccurred())
				Expect(result).NotTo(BeNil())
				Expect(result.Data).To(HaveLen(2))
			})

			It("handles pagination and returns all connections", func() {
				callCount := 0
				mockHTTP.DoFunc = func(req *http.Request) (*http.Response, error) {
					Expect(req.Method).To(Equal(http.MethodGet))
					callCount++

					if callCount == 1 {
						return testutil.NewMockJSONResponse(http.StatusOK, `{
							"data": [
								{"id": "conn-1", "type": "broker_connection", "attributes": {"name": "GitHub", "configuration": {"type": "github", "required": {}}}}
							],
							"links": {
								"next": "https://api.snyk.io/rest/tenants/tenant-123/brokers/installs/install-456/deployments/deploy-789/connections?starting_after=conn-1"
							}
						}`), nil
					}

					return testutil.NewMockJSONResponse(http.StatusOK, `{
						"data": [
							{"id": "conn-2", "type": "broker_connection", "attributes": {"name": "GitLab", "configuration": {"type": "gitlab", "required": {}}}}
						],
						"links": {}
					}`), nil
				}

				result, err := r.client.ListBrokerConnections(ctx, "tenant-123", "install-456", "deploy-789")

				Expect(err).NotTo(HaveOccurred())
				Expect(result.Data).To(HaveLen(2))
				Expect(result.Data[0].ID).To(Equal("conn-1"))
				Expect(result.Data[1].ID).To(Equal("conn-2"))
				Expect(callCount).To(Equal(2))
			})
		})

		Context("UpdateBrokerConnection", func() {
			It("updates the connection", func() {
				mockHTTP.DoFunc = func(req *http.Request) (*http.Response, error) {
					Expect(req.Method).To(Equal(http.MethodPatch))
					return testutil.NewMockJSONResponse(http.StatusOK, `{
						"data": {
							"id": "conn-123",
							"type": "broker_connection",
							"attributes": {
								"name": "Updated Connection",
								"configuration": {"type": "github", "required": {}}
							}
						}
					}`), nil
				}

				config := client.BrokerConnectionConfiguration{
					Type:     "github",
					Required: map[string]interface{}{},
				}
				result, err := r.client.UpdateBrokerConnection(ctx, "tenant-123", "install-456", "deploy-789", "conn-123", "Updated Connection", config)

				Expect(err).NotTo(HaveOccurred())
				Expect(result).NotTo(BeNil())
				Expect(result.Data.Attributes.Name).To(Equal("Updated Connection"))
			})
		})

		Context("DeleteBrokerConnection", func() {
			It("deletes the connection", func() {
				mockHTTP.DoFunc = func(req *http.Request) (*http.Response, error) {
					Expect(req.Method).To(Equal(http.MethodDelete))
					return testutil.NewMockJSONResponse(http.StatusNoContent, ""), nil
				}

				err := r.client.DeleteBrokerConnection(ctx, "tenant-123", "install-456", "deploy-789", "conn-123")
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})

	Describe("ImportState", func() {
		It("returns an error since import is not supported", func() {
			req := resource.ImportStateRequest{}
			resp := &resource.ImportStateResponse{}

			r.ImportState(ctx, req, resp)

			Expect(resp.Diagnostics.HasError()).To(BeTrue())
			Expect(resp.Diagnostics.Errors()[0].Summary()).To(Equal("Import Not Supported"))
		})
	})
})
