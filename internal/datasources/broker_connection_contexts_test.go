// Copyright (c) Snyk Ltd.

package datasources

import (
	"context"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/snyk-labs/snyk-broker-provider/internal/client"
	"github.com/snyk-labs/snyk-broker-provider/internal/common"
	"github.com/snyk-labs/snyk-broker-provider/internal/testutil"
)

var _ = Describe("BrokerConnectionContextsDataSource", func() {
	var (
		d        *BrokerConnectionContextsDataSource
		mockHTTP *testutil.MockHTTPClient
		ctx      context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
		mockHTTP = &testutil.MockHTTPClient{}
		mockAuth := &testutil.MockAuthenticator{}
		c := client.NewClient("https://api.snyk.io", mockAuth, client.WithHTTPClient(mockHTTP))

		d = &BrokerConnectionContextsDataSource{
			client: c,
		}
	})

	Describe("Metadata", func() {
		It("returns the correct type name", func() {
			req := datasource.MetadataRequest{
				ProviderTypeName: "snyk",
			}
			resp := &datasource.MetadataResponse{}

			d.Metadata(ctx, req, resp)

			Expect(resp.TypeName).To(Equal("snyk_broker_connection_contexts"))
		})
	})

	Describe("Schema", func() {
		It("returns a valid schema with required attributes", func() {
			req := datasource.SchemaRequest{}
			resp := &datasource.SchemaResponse{}

			d.Schema(ctx, req, resp)

			Expect(resp.Schema.Attributes).To(HaveKey("id"))
			Expect(resp.Schema.Attributes).To(HaveKey("tenant_id"))
			Expect(resp.Schema.Attributes).To(HaveKey("install_id"))
			Expect(resp.Schema.Attributes).To(HaveKey("connection_id"))
			Expect(resp.Schema.Attributes).To(HaveKey("contexts"))
		})
	})

	Describe("Configure", func() {
		It("sets the client from provider data", func() {
			mockAuth := &testutil.MockAuthenticator{}
			c := client.NewClient("https://api.snyk.io", mockAuth)
			providerData := &common.ProviderData{Client: c}

			newDS := &BrokerConnectionContextsDataSource{}
			req := datasource.ConfigureRequest{
				ProviderData: providerData,
			}
			resp := &datasource.ConfigureResponse{}

			newDS.Configure(ctx, req, resp)

			Expect(resp.Diagnostics.HasError()).To(BeFalse())
			Expect(newDS.client).To(Equal(c))
		})

		It("handles nil provider data gracefully", func() {
			newDS := &BrokerConnectionContextsDataSource{}
			req := datasource.ConfigureRequest{
				ProviderData: nil,
			}
			resp := &datasource.ConfigureResponse{}

			newDS.Configure(ctx, req, resp)

			Expect(resp.Diagnostics.HasError()).To(BeFalse())
		})

		It("returns error for invalid provider data type", func() {
			newDS := &BrokerConnectionContextsDataSource{}
			req := datasource.ConfigureRequest{
				ProviderData: "invalid",
			}
			resp := &datasource.ConfigureResponse{}

			newDS.Configure(ctx, req, resp)

			Expect(resp.Diagnostics.HasError()).To(BeTrue())
		})
	})

	Describe("Client API calls", func() {
		Context("ListConnectionContexts", func() {
			It("lists all contexts for a connection", func() {
				mockHTTP.DoFunc = func(req *http.Request) (*http.Response, error) {
					Expect(req.Method).To(Equal(http.MethodGet))
					Expect(req.URL.Path).To(ContainSubstring("/connections/"))
					Expect(req.URL.Path).To(ContainSubstring("/contexts"))
					return testutil.NewMockJSONResponse(http.StatusOK, `{
						"data": [
							{
								"id": "ctx-1",
								"type": "broker_context",
								"attributes": {
									"context": {
										"GITHUB_TOKEN": "${GITHUB_TOKEN}"
									}
								},
								"relationships": {
									"broker_connections": [
										{"data": {"id": "conn-123", "type": "broker_connection"}}
									],
									"applied_integrations": [
										{"data": {"id": "int-456", "org_id": "org-789", "type": "integration"}}
									]
								}
							},
							{
								"id": "ctx-2",
								"type": "broker_context",
								"attributes": {
									"context": {
										"GITLAB_TOKEN": "${GITLAB_TOKEN}"
									}
								}
							}
						]
					}`), nil
				}

				result, err := d.client.ListConnectionContexts(ctx, "tenant-123", "install-456", "conn-789")

				Expect(err).NotTo(HaveOccurred())
				Expect(result).NotTo(BeNil())
				Expect(result.Data).To(HaveLen(2))
				Expect(result.Data[0].Attributes.Context["GITHUB_TOKEN"]).To(Equal("${GITHUB_TOKEN}"))
				Expect(result.Data[0].Relationships).NotTo(BeNil())
				Expect(result.Data[0].Relationships.BrokerConnections).To(HaveLen(1))
			})

			It("returns an empty list for connection with no contexts", func() {
				mockHTTP.DoFunc = func(req *http.Request) (*http.Response, error) {
					return testutil.NewMockJSONResponse(http.StatusOK, `{"data": []}`), nil
				}

				result, err := d.client.ListConnectionContexts(ctx, "tenant-123", "install-456", "empty-conn")

				Expect(err).NotTo(HaveOccurred())
				Expect(result).NotTo(BeNil())
				Expect(result.Data).To(BeEmpty())
			})

			It("returns an error on failure", func() {
				mockHTTP.DoFunc = func(req *http.Request) (*http.Response, error) {
					return testutil.NewMockJSONResponse(http.StatusInternalServerError, `{"error": "internal error"}`), nil
				}

				result, err := d.client.ListConnectionContexts(ctx, "tenant-123", "install-456", "conn-789")

				Expect(err).To(HaveOccurred())
				Expect(result).To(BeNil())
			})

			It("handles pagination and returns all contexts", func() {
				callCount := 0
				mockHTTP.DoFunc = func(req *http.Request) (*http.Response, error) {
					Expect(req.Method).To(Equal(http.MethodGet))
					callCount++

					if callCount == 1 {
						return testutil.NewMockJSONResponse(http.StatusOK, `{
							"data": [
								{"id": "ctx-1", "type": "broker_context", "attributes": {"context": {"KEY1": "value1"}}}
							],
							"links": {
								"next": "https://api.snyk.io/rest/tenants/tenant-123/brokers/installs/install-456/connections/conn-789/contexts?starting_after=ctx-1"
							}
						}`), nil
					}

					return testutil.NewMockJSONResponse(http.StatusOK, `{
						"data": [
							{"id": "ctx-2", "type": "broker_context", "attributes": {"context": {"KEY2": "value2"}}}
						],
						"links": {}
					}`), nil
				}

				result, err := d.client.ListConnectionContexts(ctx, "tenant-123", "install-456", "conn-789")

				Expect(err).NotTo(HaveOccurred())
				Expect(result.Data).To(HaveLen(2))
				Expect(result.Data[0].ID).To(Equal("ctx-1"))
				Expect(result.Data[1].ID).To(Equal("ctx-2"))
				Expect(callCount).To(Equal(2))
			})
		})
	})
})
