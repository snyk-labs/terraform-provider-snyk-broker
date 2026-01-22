// Copyright (c) Snyk Ltd.

package datasources

import (
	"context"
	"errors"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/snyk/terraform-provider-snyk-broker/internal/client"
	"github.com/snyk/terraform-provider-snyk-broker/internal/common"
	"github.com/snyk/terraform-provider-snyk-broker/internal/testutil"
)

var _ = Describe("BrokerConnectionsForOrgDataSource", func() {
	var (
		d        *BrokerConnectionsForOrgDataSource
		mockHTTP *testutil.MockHTTPClient
		ctx      context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
		mockHTTP = &testutil.MockHTTPClient{}
		mockAuth := &testutil.MockAuthenticator{}
		c := client.NewClient("https://api.snyk.io", mockAuth, client.WithHTTPClient(mockHTTP))

		d = &BrokerConnectionsForOrgDataSource{
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

			Expect(resp.TypeName).To(Equal("snyk_broker_connections_for_org"))
		})
	})

	Describe("Schema", func() {
		It("returns a valid schema with required attributes", func() {
			req := datasource.SchemaRequest{}
			resp := &datasource.SchemaResponse{}

			d.Schema(ctx, req, resp)

			Expect(resp.Schema.Attributes).To(HaveKey("id"))
			Expect(resp.Schema.Attributes).To(HaveKey("org_id"))
			Expect(resp.Schema.Attributes).To(HaveKey("connections"))
		})
	})

	Describe("Configure", func() {
		It("sets the client from provider data", func() {
			mockAuth := &testutil.MockAuthenticator{}
			c := client.NewClient("https://api.snyk.io", mockAuth)
			providerData := &common.ProviderData{Client: c}

			newDS := &BrokerConnectionsForOrgDataSource{}
			req := datasource.ConfigureRequest{
				ProviderData: providerData,
			}
			resp := &datasource.ConfigureResponse{}

			newDS.Configure(ctx, req, resp)

			Expect(resp.Diagnostics.HasError()).To(BeFalse())
			Expect(newDS.client).To(Equal(c))
		})

		It("handles nil provider data gracefully", func() {
			newDS := &BrokerConnectionsForOrgDataSource{}
			req := datasource.ConfigureRequest{
				ProviderData: nil,
			}
			resp := &datasource.ConfigureResponse{}

			newDS.Configure(ctx, req, resp)

			Expect(resp.Diagnostics.HasError()).To(BeFalse())
		})

		It("returns error for invalid provider data type", func() {
			newDS := &BrokerConnectionsForOrgDataSource{}
			req := datasource.ConfigureRequest{
				ProviderData: "invalid",
			}
			resp := &datasource.ConfigureResponse{}

			newDS.Configure(ctx, req, resp)

			Expect(resp.Diagnostics.HasError()).To(BeTrue())
		})
	})

	Describe("Client API calls", func() {
		Context("ListBrokerConnectionsForOrg", func() {
			It("lists all connections for an organization", func() {
				mockHTTP.DoFunc = func(req *http.Request) (*http.Response, error) {
					Expect(req.Method).To(Equal(http.MethodGet))
					Expect(req.URL.Path).To(ContainSubstring("/orgs/"))
					Expect(req.URL.Path).To(ContainSubstring("/brokers/connections"))
					return testutil.NewMockJSONResponse(http.StatusOK, `{
						"data": [
							{
								"id": "conn-1",
								"type": "broker_connection",
								"attributes": {
									"deployment_id": "deploy-abc",
									"name": "GitHub Connection",
									"configuration": {
										"type": "github",
										"required": {}
									}
								}
							}
						]
					}`), nil
				}

				result, err := d.client.ListBrokerConnectionsForOrg(ctx, "org-123")

				Expect(err).NotTo(HaveOccurred())
				Expect(result).NotTo(BeNil())
				Expect(result.Data).To(HaveLen(1))
				Expect(result.Data[0].Attributes.Name).To(Equal("GitHub Connection"))
				Expect(result.Data[0].Attributes.DeploymentID).To(Equal("deploy-abc"))
			})

			It("returns an empty list with no connections", func() {
				mockHTTP.DoFunc = func(req *http.Request) (*http.Response, error) {
					return testutil.NewMockJSONResponse(http.StatusOK, `{"data": []}`), nil
				}

				result, err := d.client.ListBrokerConnectionsForOrg(ctx, "org-no-connections")

				Expect(err).NotTo(HaveOccurred())
				Expect(result).NotTo(BeNil())
				Expect(result.Data).To(BeEmpty())
			})

			It("returns an error on unauthorized request", func() {
				mockHTTP.DoFunc = func(req *http.Request) (*http.Response, error) {
					return testutil.NewMockJSONResponse(http.StatusUnauthorized, `{"error": "unauthorized"}`), nil
				}

				result, err := d.client.ListBrokerConnectionsForOrg(ctx, "org-123")

				Expect(err).To(HaveOccurred())
				Expect(result).To(BeNil())

				var apiErr *client.APIError
				Expect(errors.As(err, &apiErr)).To(BeTrue())
				Expect(apiErr.StatusCode).To(Equal(http.StatusUnauthorized))
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
								"next": "https://api.snyk.io/rest/orgs/org-123/brokers/connections?starting_after=conn-1"
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

				result, err := d.client.ListBrokerConnectionsForOrg(ctx, "org-123")

				Expect(err).NotTo(HaveOccurred())
				Expect(result.Data).To(HaveLen(2))
				Expect(result.Data[0].ID).To(Equal("conn-1"))
				Expect(result.Data[1].ID).To(Equal("conn-2"))
				Expect(callCount).To(Equal(2))
			})
		})
	})
})
