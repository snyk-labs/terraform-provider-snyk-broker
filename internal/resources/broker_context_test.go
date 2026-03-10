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

var _ = Describe("BrokerContextResource", func() {
	var (
		r        *BrokerContextResource
		mockHTTP *testutil.MockHTTPClient
		ctx      context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
		mockHTTP = &testutil.MockHTTPClient{}
		mockAuth := &testutil.MockAuthenticator{}
		c := client.NewClient("https://api.snyk.io", mockAuth, client.WithHTTPClient(mockHTTP))

		r = &BrokerContextResource{
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

			Expect(resp.TypeName).To(Equal("snyk_broker_context"))
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
			Expect(resp.Schema.Attributes).To(HaveKey("connection_id"))
			Expect(resp.Schema.Attributes).To(HaveKey("context"))
		})
	})

	Describe("Configure", func() {
		It("sets the client from provider data", func() {
			mockAuth := &testutil.MockAuthenticator{}
			c := client.NewClient("https://api.snyk.io", mockAuth)
			providerData := &common.ProviderData{Client: c}

			newResource := &BrokerContextResource{}
			req := resource.ConfigureRequest{
				ProviderData: providerData,
			}
			resp := &resource.ConfigureResponse{}

			newResource.Configure(ctx, req, resp)

			Expect(resp.Diagnostics.HasError()).To(BeFalse())
			Expect(newResource.client).To(Equal(c))
		})

		It("handles nil provider data gracefully", func() {
			newResource := &BrokerContextResource{}
			req := resource.ConfigureRequest{
				ProviderData: nil,
			}
			resp := &resource.ConfigureResponse{}

			newResource.Configure(ctx, req, resp)

			Expect(resp.Diagnostics.HasError()).To(BeFalse())
		})

		It("returns error for invalid provider data type", func() {
			newResource := &BrokerContextResource{}
			req := resource.ConfigureRequest{
				ProviderData: "invalid",
			}
			resp := &resource.ConfigureResponse{}

			newResource.Configure(ctx, req, resp)

			Expect(resp.Diagnostics.HasError()).To(BeTrue())
		})
	})

	Describe("Client API calls", func() {
		Context("CreateBrokerContext", func() {
			It("creates the context successfully", func() {
				mockHTTP.DoFunc = func(req *http.Request) (*http.Response, error) {
					Expect(req.Method).To(Equal(http.MethodPost))
					Expect(req.URL.Path).To(ContainSubstring("/contexts"))
					return testutil.NewMockJSONResponse(http.StatusCreated, `{
						"data": {
							"id": "ctx-123",
							"type": "broker_context",
							"attributes": {
								"context": {
									"GITHUB_TOKEN": "${GITHUB_TOKEN}",
									"BROKER_URL": "https://broker.example.com"
								}
							}
						}
					}`), nil
				}

				contextData := map[string]string{
					"GITHUB_TOKEN": "${GITHUB_TOKEN}",
					"BROKER_URL":   "https://broker.example.com",
				}
				result, err := r.client.CreateBrokerContext(ctx, "tenant-123", "install-456", "deploy-789", "conn-123", contextData)

				Expect(err).NotTo(HaveOccurred())
				Expect(result).NotTo(BeNil())
				Expect(result.Data.ID).To(Equal("ctx-123"))
				Expect(result.Data.Attributes.Context["GITHUB_TOKEN"]).To(Equal("${GITHUB_TOKEN}"))
			})
		})

		Context("GetBrokerContext", func() {
			It("retrieves the context", func() {
				mockHTTP.DoFunc = func(req *http.Request) (*http.Response, error) {
					Expect(req.Method).To(Equal(http.MethodGet))
					return testutil.NewMockJSONResponse(http.StatusOK, `{
						"data": {
							"id": "ctx-123",
							"type": "broker_context",
							"attributes": {
								"context": {
									"GITHUB_TOKEN": "${GITHUB_TOKEN}"
								}
							}
						}
					}`), nil
				}

				result, err := r.client.GetBrokerContext(ctx, "tenant-123", "install-456", "ctx-123")

				Expect(err).NotTo(HaveOccurred())
				Expect(result).NotTo(BeNil())
				Expect(result.Data.ID).To(Equal("ctx-123"))
			})

			It("returns a 404 error for non-existent context", func() {
				mockHTTP.DoFunc = func(req *http.Request) (*http.Response, error) {
					return testutil.NewMockJSONResponse(http.StatusNotFound, `{"error": "not found"}`), nil
				}

				result, err := r.client.GetBrokerContext(ctx, "tenant-123", "install-456", "nonexistent")

				Expect(err).To(HaveOccurred())
				Expect(result).To(BeNil())

				var apiErr *client.APIError
				Expect(errors.As(err, &apiErr)).To(BeTrue())
				Expect(apiErr.StatusCode).To(Equal(http.StatusNotFound))
			})
		})

		Context("UpdateBrokerContext", func() {
			It("updates the context", func() {
				mockHTTP.DoFunc = func(req *http.Request) (*http.Response, error) {
					Expect(req.Method).To(Equal(http.MethodPatch))
					return testutil.NewMockJSONResponse(http.StatusOK, `{
						"data": {
							"id": "ctx-123",
							"type": "broker_context",
							"attributes": {
								"context": {
									"GITHUB_TOKEN": "${NEW_TOKEN}"
								}
							}
						}
					}`), nil
				}

				contextData := map[string]string{
					"GITHUB_TOKEN": "${NEW_TOKEN}",
				}
				result, err := r.client.UpdateBrokerContext(ctx, "tenant-123", "install-456", "ctx-123", contextData)

				Expect(err).NotTo(HaveOccurred())
				Expect(result).NotTo(BeNil())
				Expect(result.Data.Attributes.Context["GITHUB_TOKEN"]).To(Equal("${NEW_TOKEN}"))
			})
		})

		Context("DeleteBrokerContext", func() {
			It("deletes the context", func() {
				mockHTTP.DoFunc = func(req *http.Request) (*http.Response, error) {
					Expect(req.Method).To(Equal(http.MethodDelete))
					return testutil.NewMockJSONResponse(http.StatusNoContent, ""), nil
				}

				err := r.client.DeleteBrokerContext(ctx, "tenant-123", "install-456", "ctx-123")
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("ListConnectionContexts", func() {
			It("lists all contexts for a connection", func() {
				mockHTTP.DoFunc = func(req *http.Request) (*http.Response, error) {
					Expect(req.Method).To(Equal(http.MethodGet))
					return testutil.NewMockJSONResponse(http.StatusOK, `{
						"data": [
							{
								"id": "ctx-1",
								"type": "broker_context",
								"attributes": {
									"context": {"KEY1": "value1"}
								}
							},
							{
								"id": "ctx-2",
								"type": "broker_context",
								"attributes": {
									"context": {"KEY2": "value2"}
								}
							}
						]
					}`), nil
				}

				result, err := r.client.ListConnectionContexts(ctx, "tenant-123", "install-456", "conn-789")

				Expect(err).NotTo(HaveOccurred())
				Expect(result).NotTo(BeNil())
				Expect(result.Data).To(HaveLen(2))
			})
		})

		Context("ListDeploymentContexts", func() {
			It("lists all contexts for a deployment", func() {
				mockHTTP.DoFunc = func(req *http.Request) (*http.Response, error) {
					Expect(req.Method).To(Equal(http.MethodGet))
					return testutil.NewMockJSONResponse(http.StatusOK, `{
						"data": [
							{
								"id": "ctx-1",
								"type": "broker_context",
								"attributes": {
									"context": {"KEY1": "value1"}
								}
							}
						]
					}`), nil
				}

				result, err := r.client.ListDeploymentContexts(ctx, "tenant-123", "install-456", "deploy-789")

				Expect(err).NotTo(HaveOccurred())
				Expect(result).NotTo(BeNil())
				Expect(result.Data).To(HaveLen(1))
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
