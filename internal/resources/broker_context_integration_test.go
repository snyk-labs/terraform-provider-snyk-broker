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

var _ = Describe("BrokerContextIntegrationResource", func() {
	var (
		r        *BrokerContextIntegrationResource
		mockHTTP *testutil.MockHTTPClient
		ctx      context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
		mockHTTP = &testutil.MockHTTPClient{}
		mockAuth := &testutil.MockAuthenticator{}
		c := client.NewClient("https://api.snyk.io", mockAuth, client.WithHTTPClient(mockHTTP))

		r = &BrokerContextIntegrationResource{
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

			Expect(resp.TypeName).To(Equal("snyk_broker_context_integration"))
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
			Expect(resp.Schema.Attributes).To(HaveKey("context_id"))
			Expect(resp.Schema.Attributes).To(HaveKey("integration_id"))
			Expect(resp.Schema.Attributes).To(HaveKey("org_id"))
		})
	})

	Describe("Configure", func() {
		It("sets the client from provider data", func() {
			mockAuth := &testutil.MockAuthenticator{}
			c := client.NewClient("https://api.snyk.io", mockAuth)
			providerData := &common.ProviderData{Client: c}

			newResource := &BrokerContextIntegrationResource{}
			req := resource.ConfigureRequest{
				ProviderData: providerData,
			}
			resp := &resource.ConfigureResponse{}

			newResource.Configure(ctx, req, resp)

			Expect(resp.Diagnostics.HasError()).To(BeFalse())
			Expect(newResource.client).To(Equal(c))
		})

		It("handles nil provider data gracefully", func() {
			newResource := &BrokerContextIntegrationResource{}
			req := resource.ConfigureRequest{
				ProviderData: nil,
			}
			resp := &resource.ConfigureResponse{}

			newResource.Configure(ctx, req, resp)

			Expect(resp.Diagnostics.HasError()).To(BeFalse())
		})

		It("returns error for invalid provider data type", func() {
			newResource := &BrokerContextIntegrationResource{}
			req := resource.ConfigureRequest{
				ProviderData: "invalid",
			}
			resp := &resource.ConfigureResponse{}

			newResource.Configure(ctx, req, resp)

			Expect(resp.Diagnostics.HasError()).To(BeTrue())
		})
	})

	Describe("Client API calls", func() {
		Context("UpdateBrokerContextIntegration", func() {
			It("creates the context integration successfully", func() {
				mockHTTP.DoFunc = func(req *http.Request) (*http.Response, error) {
					Expect(req.Method).To(Equal(http.MethodPatch))
					Expect(req.URL.Path).To(ContainSubstring("/integration"))
					return testutil.NewMockJSONResponse(http.StatusOK, `{
						"data": {
							"id": "ctx-123",
							"type": "broker_context",
							"relationships": {
								"integrations_relationships": [
									{
										"data": {
											"id": "int-456",
											"org_id": "org-789",
											"integration_type": "github",
											"type": "integration"
										}
									}
								]
							}
						}
					}`), nil
				}

				result, err := r.client.UpdateBrokerContextIntegration(ctx, "tenant-123", "install-456", "ctx-123", "int-456", "org-789")

				Expect(err).NotTo(HaveOccurred())
				Expect(result).NotTo(BeNil())
				Expect(result.Data.ID).To(Equal("ctx-123"))
			})
		})

		Context("DeleteBrokerContextIntegration", func() {
			It("deletes the context integration", func() {
				mockHTTP.DoFunc = func(req *http.Request) (*http.Response, error) {
					Expect(req.Method).To(Equal(http.MethodDelete))
					Expect(req.URL.Path).To(ContainSubstring("/integrations/"))
					return testutil.NewMockJSONResponse(http.StatusNoContent, ""), nil
				}

				err := r.client.DeleteBrokerContextIntegration(ctx, "tenant-123", "install-456", "ctx-123", "int-456")
				Expect(err).NotTo(HaveOccurred())
			})

			It("returns a 404 error for non-existent integration", func() {
				mockHTTP.DoFunc = func(req *http.Request) (*http.Response, error) {
					return testutil.NewMockJSONResponse(http.StatusNotFound, `{"error": "not found"}`), nil
				}

				err := r.client.DeleteBrokerContextIntegration(ctx, "tenant-123", "install-456", "ctx-123", "nonexistent")

				Expect(err).To(HaveOccurred())

				var apiErr *client.APIError
				Expect(errors.As(err, &apiErr)).To(BeTrue())
				Expect(apiErr.StatusCode).To(Equal(http.StatusNotFound))
			})
		})
	})

	Describe("Update", func() {
		It("returns an error since update is not supported", func() {
			req := resource.UpdateRequest{}
			resp := &resource.UpdateResponse{}

			r.Update(ctx, req, resp)

			Expect(resp.Diagnostics.HasError()).To(BeTrue())
			Expect(resp.Diagnostics.Errors()[0].Summary()).To(Equal("Update Not Supported"))
		})
	})
})
