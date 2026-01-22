// Copyright (c) Snyk Ltd.

package resources

import (
	"context"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/snyk/terraform-provider-snyk-broker/internal/client"
	"github.com/snyk/terraform-provider-snyk-broker/internal/common"
	"github.com/snyk/terraform-provider-snyk-broker/internal/testutil"
)

var _ = Describe("BrokerConnectionIntegrationResource", func() {
	var (
		r        *BrokerConnectionIntegrationResource
		mockHTTP *testutil.MockHTTPClient
		ctx      context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
		mockHTTP = &testutil.MockHTTPClient{}
		mockAuth := &testutil.MockAuthenticator{}
		c := client.NewClient("https://api.snyk.io", mockAuth, client.WithHTTPClient(mockHTTP))

		r = &BrokerConnectionIntegrationResource{
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

			Expect(resp.TypeName).To(Equal("snyk_broker_connection_integration"))
		})
	})

	Describe("Schema", func() {
		It("returns a valid schema with required attributes", func() {
			req := resource.SchemaRequest{}
			resp := &resource.SchemaResponse{}

			r.Schema(ctx, req, resp)

			Expect(resp.Schema.Attributes).To(HaveKey("id"))
			Expect(resp.Schema.Attributes).To(HaveKey("tenant_id"))
			Expect(resp.Schema.Attributes).To(HaveKey("connection_id"))
			Expect(resp.Schema.Attributes).To(HaveKey("org_id"))
			Expect(resp.Schema.Attributes).To(HaveKey("integration_id"))
			Expect(resp.Schema.Attributes).To(HaveKey("type"))
		})
	})

	Describe("Configure", func() {
		It("sets the client from provider data", func() {
			mockAuth := &testutil.MockAuthenticator{}
			c := client.NewClient("https://api.snyk.io", mockAuth)
			providerData := &common.ProviderData{Client: c}

			newResource := &BrokerConnectionIntegrationResource{}
			req := resource.ConfigureRequest{
				ProviderData: providerData,
			}
			resp := &resource.ConfigureResponse{}

			newResource.Configure(ctx, req, resp)

			Expect(resp.Diagnostics.HasError()).To(BeFalse())
			Expect(newResource.client).To(Equal(c))
		})

		It("handles nil provider data gracefully", func() {
			newResource := &BrokerConnectionIntegrationResource{}
			req := resource.ConfigureRequest{
				ProviderData: nil,
			}
			resp := &resource.ConfigureResponse{}

			newResource.Configure(ctx, req, resp)

			Expect(resp.Diagnostics.HasError()).To(BeFalse())
		})

		It("returns error for invalid provider data type", func() {
			newResource := &BrokerConnectionIntegrationResource{}
			req := resource.ConfigureRequest{
				ProviderData: "invalid",
			}
			resp := &resource.ConfigureResponse{}

			newResource.Configure(ctx, req, resp)

			Expect(resp.Diagnostics.HasError()).To(BeTrue())
		})
	})

	Describe("Client API calls", func() {
		Context("CreateBrokerConnectionIntegration", func() {
			It("creates the connection integration successfully", func() {
				mockHTTP.DoFunc = func(req *http.Request) (*http.Response, error) {
					Expect(req.Method).To(Equal(http.MethodPost))
					Expect(req.URL.Path).To(ContainSubstring("/integration"))
					return testutil.NewMockJSONResponse(http.StatusCreated, `{
						"data": {
							"id": "integ-123",
							"type": "broker_connection_integration",
							"attributes": {
								"org_id": "org-456",
								"integration_id": "int-789",
								"type": "github"
							}
						}
					}`), nil
				}

				result, err := r.client.CreateBrokerConnectionIntegration(ctx, "tenant-123", "conn-456", "org-789", "int-abc", "github")

				Expect(err).NotTo(HaveOccurred())
				Expect(result).NotTo(BeNil())
				Expect(result.Data.ID).To(Equal("integ-123"))
				Expect(result.Data.Attributes.Type).To(Equal("github"))
			})
		})

		Context("GetBrokerConnectionIntegrations", func() {
			It("lists all integrations for a connection", func() {
				mockHTTP.DoFunc = func(req *http.Request) (*http.Response, error) {
					Expect(req.Method).To(Equal(http.MethodGet))
					return testutil.NewMockJSONResponse(http.StatusOK, `{
						"data": [
							{
								"id": "integ-1",
								"type": "broker_connection_integration",
								"attributes": {
									"org_id": "org-1",
									"integration_id": "int-1",
									"type": "github"
								}
							},
							{
								"id": "integ-2",
								"type": "broker_connection_integration",
								"attributes": {
									"org_id": "org-2",
									"integration_id": "int-2",
									"type": "github"
								}
							}
						]
					}`), nil
				}

				result, err := r.client.GetBrokerConnectionIntegrations(ctx, "tenant-123", "conn-456")

				Expect(err).NotTo(HaveOccurred())
				Expect(result).NotTo(BeNil())
				Expect(result.Data).To(HaveLen(2))
				Expect(result.Data[0].Attributes.OrgID).To(Equal("org-1"))
				Expect(result.Data[1].Attributes.OrgID).To(Equal("org-2"))
			})
		})

		Context("DeleteBrokerConnectionIntegration", func() {
			It("deletes the connection integration", func() {
				mockHTTP.DoFunc = func(req *http.Request) (*http.Response, error) {
					Expect(req.Method).To(Equal(http.MethodDelete))
					return testutil.NewMockJSONResponse(http.StatusNoContent, ""), nil
				}

				err := r.client.DeleteBrokerConnectionIntegration(ctx, "tenant-123", "conn-456", "org-789", "int-abc")
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})

	Describe("Update", func() {
		It("returns an error since updates are not supported", func() {
			req := resource.UpdateRequest{}
			resp := &resource.UpdateResponse{}

			r.Update(ctx, req, resp)

			Expect(resp.Diagnostics.HasError()).To(BeTrue())
			Expect(resp.Diagnostics.Errors()[0].Summary()).To(Equal("Update Not Supported"))
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
