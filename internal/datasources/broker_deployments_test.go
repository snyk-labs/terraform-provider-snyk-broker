// Copyright (c) Snyk Ltd.

package datasources

import (
	"context"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/snyk/terraform-provider-snyk-broker/internal/client"
	"github.com/snyk/terraform-provider-snyk-broker/internal/common"
	"github.com/snyk/terraform-provider-snyk-broker/internal/testutil"
)

var _ = Describe("BrokerDeploymentsDataSource", func() {
	var (
		d        *BrokerDeploymentsDataSource
		mockHTTP *testutil.MockHTTPClient
		ctx      context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
		mockHTTP = &testutil.MockHTTPClient{}
		mockAuth := &testutil.MockAuthenticator{}
		c := client.NewClient("https://api.snyk.io", mockAuth, client.WithHTTPClient(mockHTTP))

		d = &BrokerDeploymentsDataSource{
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

			Expect(resp.TypeName).To(Equal("snyk_broker_deployments"))
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
			Expect(resp.Schema.Attributes).To(HaveKey("deployments"))
		})
	})

	Describe("Configure", func() {
		It("sets the client from provider data", func() {
			mockAuth := &testutil.MockAuthenticator{}
			c := client.NewClient("https://api.snyk.io", mockAuth)
			providerData := &common.ProviderData{Client: c}

			newDS := &BrokerDeploymentsDataSource{}
			req := datasource.ConfigureRequest{
				ProviderData: providerData,
			}
			resp := &datasource.ConfigureResponse{}

			newDS.Configure(ctx, req, resp)

			Expect(resp.Diagnostics.HasError()).To(BeFalse())
			Expect(newDS.client).To(Equal(c))
		})

		It("handles nil provider data gracefully", func() {
			newDS := &BrokerDeploymentsDataSource{}
			req := datasource.ConfigureRequest{
				ProviderData: nil,
			}
			resp := &datasource.ConfigureResponse{}

			newDS.Configure(ctx, req, resp)

			Expect(resp.Diagnostics.HasError()).To(BeFalse())
		})

		It("returns error for invalid provider data type", func() {
			newDS := &BrokerDeploymentsDataSource{}
			req := datasource.ConfigureRequest{
				ProviderData: "invalid",
			}
			resp := &datasource.ConfigureResponse{}

			newDS.Configure(ctx, req, resp)

			Expect(resp.Diagnostics.HasError()).To(BeTrue())
		})
	})

	Describe("Client API calls", func() {
		Context("ListBrokerDeployments with install ID", func() {
			It("lists deployments for a specific install", func() {
				mockHTTP.DoFunc = func(req *http.Request) (*http.Response, error) {
					Expect(req.Method).To(Equal(http.MethodGet))
					Expect(req.URL.Path).To(ContainSubstring("/installs/"))
					Expect(req.URL.Path).To(ContainSubstring("/deployments"))
					return testutil.NewMockJSONResponse(http.StatusOK, `{
						"data": [
							{
								"id": "deploy-1",
								"type": "broker_deployment",
								"attributes": {
									"install_id": "install-123",
									"metadata": {
										"deployment_name": "Production"
									}
								}
							},
							{
								"id": "deploy-2",
								"type": "broker_deployment",
								"attributes": {
									"install_id": "install-123",
									"metadata": {
										"deployment_name": "Staging"
									}
								}
							}
						]
					}`), nil
				}

				result, err := d.client.ListBrokerDeployments(ctx, "tenant-123", "install-123")

				Expect(err).NotTo(HaveOccurred())
				Expect(result).NotTo(BeNil())
				Expect(result.Data).To(HaveLen(2))
				Expect(result.Data[0].ID).To(Equal("deploy-1"))
				Expect(result.Data[1].ID).To(Equal("deploy-2"))
			})
		})

		Context("ListBrokerDeploymentsForTenant", func() {
			It("lists all deployments for a tenant", func() {
				mockHTTP.DoFunc = func(req *http.Request) (*http.Response, error) {
					Expect(req.Method).To(Equal(http.MethodGet))
					Expect(req.URL.Path).To(ContainSubstring("/tenants/"))
					Expect(req.URL.Path).To(ContainSubstring("/brokers/deployments"))
					return testutil.NewMockJSONResponse(http.StatusOK, `{
						"data": [
							{
								"id": "deploy-1",
								"type": "broker_deployment",
								"attributes": {
									"metadata": {"deployment_name": "Deployment 1"}
								}
							}
						]
					}`), nil
				}

				result, err := d.client.ListBrokerDeploymentsForTenant(ctx, "tenant-123")

				Expect(err).NotTo(HaveOccurred())
				Expect(result).NotTo(BeNil())
				Expect(result.Data).To(HaveLen(1))
			})

			It("returns an error on failure", func() {
				mockHTTP.DoFunc = func(req *http.Request) (*http.Response, error) {
					return testutil.NewMockJSONResponse(http.StatusForbidden, `{"error": "access denied"}`), nil
				}

				result, err := d.client.ListBrokerDeploymentsForTenant(ctx, "tenant-123")

				Expect(err).To(HaveOccurred())
				Expect(result).To(BeNil())
			})
		})
	})
})
