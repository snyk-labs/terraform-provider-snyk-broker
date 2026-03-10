// Copyright (c) Snyk Ltd.

package resources

import (
	"context"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/snyk-labs/snyk-broker-provider/internal/client"
	"github.com/snyk-labs/snyk-broker-provider/internal/common"
	"github.com/snyk-labs/snyk-broker-provider/internal/testutil"
)

var _ = Describe("BrokerCredentialResource", func() {
	var (
		r        *BrokerCredentialResource
		mockHTTP *testutil.MockHTTPClient
		ctx      context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
		mockHTTP = &testutil.MockHTTPClient{}
		mockAuth := &testutil.MockAuthenticator{}
		c := client.NewClient("https://api.snyk.io", mockAuth, client.WithHTTPClient(mockHTTP))

		r = &BrokerCredentialResource{
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

			Expect(resp.TypeName).To(Equal("snyk_broker_credential"))
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
			Expect(resp.Schema.Attributes).To(HaveKey("environment_variable_name"))
			Expect(resp.Schema.Attributes).To(HaveKey("type"))
			Expect(resp.Schema.Attributes).To(HaveKey("comment"))
		})
	})

	Describe("Configure", func() {
		It("sets the client from provider data", func() {
			mockAuth := &testutil.MockAuthenticator{}
			c := client.NewClient("https://api.snyk.io", mockAuth)
			providerData := &common.ProviderData{Client: c}

			newResource := &BrokerCredentialResource{}
			req := resource.ConfigureRequest{
				ProviderData: providerData,
			}
			resp := &resource.ConfigureResponse{}

			newResource.Configure(ctx, req, resp)

			Expect(resp.Diagnostics.HasError()).To(BeFalse())
			Expect(newResource.client).To(Equal(c))
		})

		It("handles nil provider data gracefully", func() {
			newResource := &BrokerCredentialResource{}
			req := resource.ConfigureRequest{
				ProviderData: nil,
			}
			resp := &resource.ConfigureResponse{}

			newResource.Configure(ctx, req, resp)

			Expect(resp.Diagnostics.HasError()).To(BeFalse())
		})

		It("returns error for invalid provider data type", func() {
			newResource := &BrokerCredentialResource{}
			req := resource.ConfigureRequest{
				ProviderData: "invalid",
			}
			resp := &resource.ConfigureResponse{}

			newResource.Configure(ctx, req, resp)

			Expect(resp.Diagnostics.HasError()).To(BeTrue())
		})
	})

	Describe("Client API calls", func() {
		Context("CreateBrokerCredential", func() {
			It("creates the credential successfully", func() {
				mockHTTP.DoFunc = func(req *http.Request) (*http.Response, error) {
					Expect(req.Method).To(Equal(http.MethodPost))
					Expect(req.URL.Path).To(ContainSubstring("/credentials"))
					return testutil.NewMockJSONResponse(http.StatusCreated, `{
						"data": [{
							"id": "cred-123",
							"type": "deployment_credential",
							"attributes": {
								"comment": "GitHub token",
								"deployment_id": "deploy-456",
								"environment_variable_name": "GITHUB_TOKEN",
								"type": "github"
							}
						}]
					}`), nil
				}

				cred := client.BrokerCredentialAttributes{
					Comment:                 "GitHub token",
					EnvironmentVariableName: "GITHUB_TOKEN",
					Type:                    "github",
				}
				result, err := r.client.CreateBrokerCredential(ctx, "tenant-123", "install-456", "deploy-789", cred)

				Expect(err).NotTo(HaveOccurred())
				Expect(result).NotTo(BeNil())
				Expect(len(result.Data)).To(Equal(1))
				Expect(result.Data[0].ID).To(Equal("cred-123"))
				Expect(result.Data[0].Attributes.EnvironmentVariableName).To(Equal("GITHUB_TOKEN"))
				Expect(result.Data[0].Attributes.Type).To(Equal("github"))
			})
		})

		Context("ListBrokerCredentials", func() {
			It("lists all credentials for a deployment", func() {
				mockHTTP.DoFunc = func(req *http.Request) (*http.Response, error) {
					Expect(req.Method).To(Equal(http.MethodGet))
					return testutil.NewMockJSONResponse(http.StatusOK, `{
						"data": [
							{
								"id": "cred-1",
								"type": "deployment_credential",
								"attributes": {
									"environment_variable_name": "GITHUB_TOKEN",
									"type": "github"
								}
							},
							{
								"id": "cred-2",
								"type": "deployment_credential",
								"attributes": {
									"environment_variable_name": "GITLAB_TOKEN",
									"type": "gitlab"
								}
							}
						]
					}`), nil
				}

				result, err := r.client.ListBrokerCredentials(ctx, "tenant-123", "install-456", "deploy-789")

				Expect(err).NotTo(HaveOccurred())
				Expect(result).NotTo(BeNil())
				Expect(result.Data).To(HaveLen(2))
				Expect(result.Data[0].ID).To(Equal("cred-1"))
				Expect(result.Data[1].ID).To(Equal("cred-2"))
			})

			It("handles pagination and returns all credentials", func() {
				callCount := 0
				mockHTTP.DoFunc = func(req *http.Request) (*http.Response, error) {
					Expect(req.Method).To(Equal(http.MethodGet))
					callCount++

					if callCount == 1 {
						return testutil.NewMockJSONResponse(http.StatusOK, `{
							"data": [
								{"id": "cred-1", "type": "deployment_credential", "attributes": {"environment_variable_name": "TOKEN1", "type": "github"}}
							],
							"links": {
								"next": "https://api.snyk.io/rest/tenants/tenant-123/brokers/installs/install-456/deployments/deploy-789/credentials?starting_after=cred-1"
							}
						}`), nil
					}

					return testutil.NewMockJSONResponse(http.StatusOK, `{
						"data": [
							{"id": "cred-2", "type": "deployment_credential", "attributes": {"environment_variable_name": "TOKEN2", "type": "gitlab"}}
						],
						"links": {}
					}`), nil
				}

				result, err := r.client.ListBrokerCredentials(ctx, "tenant-123", "install-456", "deploy-789")

				Expect(err).NotTo(HaveOccurred())
				Expect(result.Data).To(HaveLen(2))
				Expect(result.Data[0].ID).To(Equal("cred-1"))
				Expect(result.Data[1].ID).To(Equal("cred-2"))
				Expect(callCount).To(Equal(2))
			})
		})

		Context("DeleteBrokerCredential", func() {
			It("deletes the credential", func() {
				mockHTTP.DoFunc = func(req *http.Request) (*http.Response, error) {
					Expect(req.Method).To(Equal(http.MethodDelete))
					return testutil.NewMockJSONResponse(http.StatusNoContent, ""), nil
				}

				err := r.client.DeleteBrokerCredential(ctx, "tenant-123", "install-456", "deploy-789", "cred-123")
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
