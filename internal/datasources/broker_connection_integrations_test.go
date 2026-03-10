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

var _ = Describe("BrokerConnectionIntegrationsDataSource", func() {
	var (
		d        *BrokerConnectionIntegrationsDataSource
		mockHTTP *testutil.MockHTTPClient
		ctx      context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
		mockHTTP = &testutil.MockHTTPClient{}
		mockAuth := &testutil.MockAuthenticator{}
		c := client.NewClient("https://api.snyk.io", mockAuth, client.WithHTTPClient(mockHTTP))

		d = &BrokerConnectionIntegrationsDataSource{
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

			Expect(resp.TypeName).To(Equal("snyk_broker_connection_integrations"))
		})
	})

	Describe("Schema", func() {
		It("returns a valid schema with required attributes", func() {
			req := datasource.SchemaRequest{}
			resp := &datasource.SchemaResponse{}

			d.Schema(ctx, req, resp)

			Expect(resp.Schema.Attributes).To(HaveKey("id"))
			Expect(resp.Schema.Attributes).To(HaveKey("tenant_id"))
			Expect(resp.Schema.Attributes).To(HaveKey("connection_id"))
			Expect(resp.Schema.Attributes).To(HaveKey("integrations"))
		})
	})

	Describe("Configure", func() {
		It("sets the client from provider data", func() {
			mockAuth := &testutil.MockAuthenticator{}
			c := client.NewClient("https://api.snyk.io", mockAuth)
			providerData := &common.ProviderData{Client: c}

			newDS := &BrokerConnectionIntegrationsDataSource{}
			req := datasource.ConfigureRequest{
				ProviderData: providerData,
			}
			resp := &datasource.ConfigureResponse{}

			newDS.Configure(ctx, req, resp)

			Expect(resp.Diagnostics.HasError()).To(BeFalse())
			Expect(newDS.client).To(Equal(c))
		})

		It("handles nil provider data gracefully", func() {
			newDS := &BrokerConnectionIntegrationsDataSource{}
			req := datasource.ConfigureRequest{
				ProviderData: nil,
			}
			resp := &datasource.ConfigureResponse{}

			newDS.Configure(ctx, req, resp)

			Expect(resp.Diagnostics.HasError()).To(BeFalse())
		})

		It("returns error for invalid provider data type", func() {
			newDS := &BrokerConnectionIntegrationsDataSource{}
			req := datasource.ConfigureRequest{
				ProviderData: "invalid",
			}
			resp := &datasource.ConfigureResponse{}

			newDS.Configure(ctx, req, resp)

			Expect(resp.Diagnostics.HasError()).To(BeTrue())
		})
	})

	Describe("Client API calls", func() {
		Context("GetBrokerConnectionIntegrations", func() {
			It("lists all integrations for a connection", func() {
				mockHTTP.DoFunc = func(req *http.Request) (*http.Response, error) {
					Expect(req.Method).To(Equal(http.MethodGet))
					Expect(req.URL.Path).To(ContainSubstring("/integrations"))
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
							},
							{
								"id": "integ-3",
								"type": "broker_connection_integration",
								"attributes": {
									"org_id": "org-3",
									"integration_id": "int-3",
									"type": "github"
								}
							}
						]
					}`), nil
				}

				result, err := d.client.GetBrokerConnectionIntegrations(ctx, "tenant-123", "conn-456")

				Expect(err).NotTo(HaveOccurred())
				Expect(result).NotTo(BeNil())
				Expect(result.Data).To(HaveLen(3))
				Expect(result.Data[0].Attributes.OrgID).To(Equal("org-1"))
				Expect(result.Data[1].Attributes.IntegrationID).To(Equal("int-2"))
				Expect(result.Data[2].Attributes.Type).To(Equal("github"))
			})

			It("returns an empty list with no integrations", func() {
				mockHTTP.DoFunc = func(req *http.Request) (*http.Response, error) {
					return testutil.NewMockJSONResponse(http.StatusOK, `{"data": []}`), nil
				}

				result, err := d.client.GetBrokerConnectionIntegrations(ctx, "tenant-123", "conn-no-integrations")

				Expect(err).NotTo(HaveOccurred())
				Expect(result).NotTo(BeNil())
				Expect(result.Data).To(BeEmpty())
			})

			It("returns an error on failure", func() {
				mockHTTP.DoFunc = func(req *http.Request) (*http.Response, error) {
					return testutil.NewMockJSONResponse(http.StatusNotFound, `{"error": "connection not found"}`), nil
				}

				result, err := d.client.GetBrokerConnectionIntegrations(ctx, "tenant-123", "nonexistent-conn")

				Expect(err).To(HaveOccurred())
				Expect(result).To(BeNil())
			})
		})
	})
})
