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

var _ = Describe("BrokerBulkMigrationResource", func() {
	var (
		r        *BrokerBulkMigrationResource
		mockHTTP *testutil.MockHTTPClient
		ctx      context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
		mockHTTP = &testutil.MockHTTPClient{}
		mockAuth := &testutil.MockAuthenticator{}
		c := client.NewClient("https://api.snyk.io", mockAuth, client.WithHTTPClient(mockHTTP))

		r = &BrokerBulkMigrationResource{
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

			Expect(resp.TypeName).To(Equal("snyk_broker_bulk_migration"))
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
			Expect(resp.Schema.Attributes).To(HaveKey("org_ids"))
			Expect(resp.Schema.Attributes).To(HaveKey("status"))
		})
	})

	Describe("Configure", func() {
		It("sets the client from provider data", func() {
			mockAuth := &testutil.MockAuthenticator{}
			c := client.NewClient("https://api.snyk.io", mockAuth)
			providerData := &common.ProviderData{Client: c}

			newResource := &BrokerBulkMigrationResource{}
			req := resource.ConfigureRequest{
				ProviderData: providerData,
			}
			resp := &resource.ConfigureResponse{}

			newResource.Configure(ctx, req, resp)

			Expect(resp.Diagnostics.HasError()).To(BeFalse())
			Expect(newResource.client).To(Equal(c))
		})

		It("handles nil provider data gracefully", func() {
			newResource := &BrokerBulkMigrationResource{}
			req := resource.ConfigureRequest{
				ProviderData: nil,
			}
			resp := &resource.ConfigureResponse{}

			newResource.Configure(ctx, req, resp)

			Expect(resp.Diagnostics.HasError()).To(BeFalse())
		})

		It("returns error for invalid provider data type", func() {
			newResource := &BrokerBulkMigrationResource{}
			req := resource.ConfigureRequest{
				ProviderData: "invalid",
			}
			resp := &resource.ConfigureResponse{}

			newResource.Configure(ctx, req, resp)

			Expect(resp.Diagnostics.HasError()).To(BeTrue())
		})
	})

	Describe("Client API calls", func() {
		Context("CreateBrokerBulkMigration", func() {
			It("creates the bulk migration successfully", func() {
				mockHTTP.DoFunc = func(req *http.Request) (*http.Response, error) {
					Expect(req.Method).To(Equal(http.MethodPost))
					Expect(req.URL.Path).To(ContainSubstring("/installs/"))
					Expect(req.URL.Path).To(ContainSubstring("/deployments/"))
					Expect(req.URL.Path).To(ContainSubstring("/connections/"))
					Expect(req.URL.Path).To(ContainSubstring("/bulk_migration"))
					return testutil.NewMockJSONResponse(http.StatusCreated, `{
						"data": {
							"id": "migration-123",
							"type": "broker_bulk_migration",
							"attributes": {
								"status": "completed"
							}
						}
					}`), nil
				}

				orgIDs := []string{"org-1", "org-2", "org-3"}
				result, err := r.client.CreateBrokerBulkMigration(ctx, "tenant-123", "install-789", "deploy-abc", "conn-456", orgIDs)

				Expect(err).NotTo(HaveOccurred())
				Expect(result).NotTo(BeNil())
				Expect(result.Data.ID).To(Equal("migration-123"))
				Expect(result.Data.Attributes.Status).To(Equal("completed"))
			})

			It("handles empty org list", func() {
				mockHTTP.DoFunc = func(req *http.Request) (*http.Response, error) {
					return testutil.NewMockJSONResponse(http.StatusCreated, `{
						"data": {
							"id": "migration-empty",
							"type": "broker_bulk_migration",
							"attributes": {
								"status": "completed"
							}
						}
					}`), nil
				}

				orgIDs := []string{}
				result, err := r.client.CreateBrokerBulkMigration(ctx, "tenant-123", "install-789", "deploy-abc", "conn-456", orgIDs)

				Expect(err).NotTo(HaveOccurred())
				Expect(result).NotTo(BeNil())
			})

			It("returns error on failure", func() {
				mockHTTP.DoFunc = func(req *http.Request) (*http.Response, error) {
					return testutil.NewMockJSONResponse(http.StatusBadRequest, `{"error": "invalid org IDs"}`), nil
				}

				orgIDs := []string{"invalid-org"}
				result, err := r.client.CreateBrokerBulkMigration(ctx, "tenant-123", "install-789", "deploy-abc", "conn-456", orgIDs)

				Expect(err).To(HaveOccurred())
				Expect(result).To(BeNil())
			})
		})

		Context("ListBrokerMigrationOrgs", func() {
			It("lists organizations available for migration", func() {
				mockHTTP.DoFunc = func(req *http.Request) (*http.Response, error) {
					Expect(req.Method).To(Equal(http.MethodGet))
					Expect(req.URL.Path).To(ContainSubstring("/bulk_migration"))
					return testutil.NewMockJSONResponse(http.StatusOK, `{
						"data": [
							{
								"id": "record-1",
								"type": "broker_migration_org",
								"attributes": {
									"org_id": "org-1",
									"org_name": "Organization One"
								}
							},
							{
								"id": "record-2",
								"type": "broker_migration_org",
								"attributes": {
									"org_id": "org-2",
									"org_name": "Organization Two"
								}
							}
						]
					}`), nil
				}

				result, err := r.client.ListBrokerMigrationOrgs(ctx, "tenant-123", "install-789", "deploy-abc", "conn-456")

				Expect(err).NotTo(HaveOccurred())
				Expect(result).NotTo(BeNil())
				Expect(result.Data).To(HaveLen(2))
				Expect(result.Data[0].Attributes.OrgID).To(Equal("org-1"))
				Expect(result.Data[0].Attributes.OrgName).To(Equal("Organization One"))
			})
		})
	})
})
