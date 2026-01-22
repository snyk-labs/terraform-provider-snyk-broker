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

var _ = Describe("BrokerMigrationOrgsDataSource", func() {
	var (
		d        *BrokerMigrationOrgsDataSource
		mockHTTP *testutil.MockHTTPClient
		ctx      context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
		mockHTTP = &testutil.MockHTTPClient{}
		mockAuth := &testutil.MockAuthenticator{}
		c := client.NewClient("https://api.snyk.io", mockAuth, client.WithHTTPClient(mockHTTP))

		d = &BrokerMigrationOrgsDataSource{
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

			Expect(resp.TypeName).To(Equal("snyk_broker_migration_orgs"))
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
			Expect(resp.Schema.Attributes).To(HaveKey("deployment_id"))
			Expect(resp.Schema.Attributes).To(HaveKey("connection_id"))
			Expect(resp.Schema.Attributes).To(HaveKey("organizations"))
		})
	})

	Describe("Configure", func() {
		It("sets the client from provider data", func() {
			mockAuth := &testutil.MockAuthenticator{}
			c := client.NewClient("https://api.snyk.io", mockAuth)
			providerData := &common.ProviderData{Client: c}

			newDS := &BrokerMigrationOrgsDataSource{}
			req := datasource.ConfigureRequest{
				ProviderData: providerData,
			}
			resp := &datasource.ConfigureResponse{}

			newDS.Configure(ctx, req, resp)

			Expect(resp.Diagnostics.HasError()).To(BeFalse())
			Expect(newDS.client).To(Equal(c))
		})

		It("handles nil provider data gracefully", func() {
			newDS := &BrokerMigrationOrgsDataSource{}
			req := datasource.ConfigureRequest{
				ProviderData: nil,
			}
			resp := &datasource.ConfigureResponse{}

			newDS.Configure(ctx, req, resp)

			Expect(resp.Diagnostics.HasError()).To(BeFalse())
		})

		It("returns error for invalid provider data type", func() {
			newDS := &BrokerMigrationOrgsDataSource{}
			req := datasource.ConfigureRequest{
				ProviderData: "invalid",
			}
			resp := &datasource.ConfigureResponse{}

			newDS.Configure(ctx, req, resp)

			Expect(resp.Diagnostics.HasError()).To(BeTrue())
		})
	})

	Describe("Client API calls", func() {
		Context("ListBrokerMigrationOrgs", func() {
			It("lists organizations available for migration", func() {
				mockHTTP.DoFunc = func(req *http.Request) (*http.Response, error) {
					Expect(req.Method).To(Equal(http.MethodGet))
					Expect(req.URL.Path).To(ContainSubstring("/installs/"))
					Expect(req.URL.Path).To(ContainSubstring("/deployments/"))
					Expect(req.URL.Path).To(ContainSubstring("/connections/"))
					Expect(req.URL.Path).To(ContainSubstring("/bulk_migration"))
					return testutil.NewMockJSONResponse(http.StatusOK, `{
						"data": [
							{
								"id": "record-1",
								"type": "broker_migration_org",
								"attributes": {
									"org_id": "org-abc",
									"org_name": "ACME Corp"
								}
							},
							{
								"id": "record-2",
								"type": "broker_migration_org",
								"attributes": {
									"org_id": "org-def",
									"org_name": "Beta Inc"
								}
							},
							{
								"id": "record-3",
								"type": "broker_migration_org",
								"attributes": {
									"org_id": "org-ghi",
									"org_name": "Gamma LLC"
								}
							}
						]
					}`), nil
				}

				result, err := d.client.ListBrokerMigrationOrgs(ctx, "tenant-123", "install-789", "deploy-abc", "conn-456")

				Expect(err).NotTo(HaveOccurred())
				Expect(result).NotTo(BeNil())
				Expect(result.Data).To(HaveLen(3))
				Expect(result.Data[0].Attributes.OrgID).To(Equal("org-abc"))
				Expect(result.Data[0].Attributes.OrgName).To(Equal("ACME Corp"))
				Expect(result.Data[1].Attributes.OrgName).To(Equal("Beta Inc"))
				Expect(result.Data[2].Attributes.OrgName).To(Equal("Gamma LLC"))
			})

			It("returns an empty list with no organizations", func() {
				mockHTTP.DoFunc = func(req *http.Request) (*http.Response, error) {
					return testutil.NewMockJSONResponse(http.StatusOK, `{"data": []}`), nil
				}

				result, err := d.client.ListBrokerMigrationOrgs(ctx, "tenant-123", "install-789", "deploy-abc", "conn-no-orgs")

				Expect(err).NotTo(HaveOccurred())
				Expect(result).NotTo(BeNil())
				Expect(result.Data).To(BeEmpty())
			})

			It("returns an error on failure", func() {
				mockHTTP.DoFunc = func(req *http.Request) (*http.Response, error) {
					return testutil.NewMockJSONResponse(http.StatusForbidden, `{"error": "access denied"}`), nil
				}

				result, err := d.client.ListBrokerMigrationOrgs(ctx, "tenant-123", "install-789", "deploy-abc", "conn-456")

				Expect(err).To(HaveOccurred())
				Expect(result).To(BeNil())

				var apiErr *client.APIError
				Expect(errors.As(err, &apiErr)).To(BeTrue())
				Expect(apiErr.StatusCode).To(Equal(http.StatusForbidden))
			})

			It("handles pagination and returns all organizations", func() {
				callCount := 0
				mockHTTP.DoFunc = func(req *http.Request) (*http.Response, error) {
					Expect(req.Method).To(Equal(http.MethodGet))
					callCount++

					if callCount == 1 {
						return testutil.NewMockJSONResponse(http.StatusOK, `{
							"data": [
								{"id": "record-1", "type": "broker_migration_org", "attributes": {"org_id": "org-abc", "org_name": "ACME Corp"}}
							],
							"links": {
								"next": "https://api.snyk.io/rest/tenants/tenant-123/brokers/installs/install-789/deployments/deploy-abc/connections/conn-456/bulk_migration?starting_after=record-1"
							}
						}`), nil
					}

					return testutil.NewMockJSONResponse(http.StatusOK, `{
						"data": [
							{"id": "record-2", "type": "broker_migration_org", "attributes": {"org_id": "org-def", "org_name": "Beta Inc"}}
						],
						"links": {}
					}`), nil
				}

				result, err := d.client.ListBrokerMigrationOrgs(ctx, "tenant-123", "install-789", "deploy-abc", "conn-456")

				Expect(err).NotTo(HaveOccurred())
				Expect(result.Data).To(HaveLen(2))
				Expect(result.Data[0].Attributes.OrgID).To(Equal("org-abc"))
				Expect(result.Data[1].Attributes.OrgID).To(Equal("org-def"))
				Expect(callCount).To(Equal(2))
			})
		})
	})
})
