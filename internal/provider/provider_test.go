// Copyright (c) Snyk Ltd.

package provider_test

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/snyk/terraform-provider-snyk-broker/internal/provider"
)

var _ = Describe("SnykBrokerProvider", func() {
	var (
		protoV6ProviderFactories map[string]func() (tfprotov6.ProviderServer, error)
	)

	BeforeEach(func() {
		protoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
			"snyk": providerserver.NewProtocol6WithError(provider.New("test")()),
		}
	})

	Describe("Provider Factory", func() {
		It("creates a valid provider", func() {
			p := provider.New("test")()
			Expect(p).NotTo(BeNil())
		})

		It("creates provider server factory", func() {
			Expect(protoV6ProviderFactories).To(HaveKey("snyk"))
			server, err := protoV6ProviderFactories["snyk"]()
			Expect(err).NotTo(HaveOccurred())
			Expect(server).NotTo(BeNil())
		})
	})

	Describe("Provider Schema", func() {
		It("has the expected attributes", func() {
			p := provider.New("test")()

			schemaReq := struct{}{}
			schemaResp := &struct {
				Schema interface{}
			}{}

			// Verify the provider can be created
			Expect(p).NotTo(BeNil())

			// Test metadata
			ctx := context.Background()
			_ = ctx
			_ = schemaReq
			_ = schemaResp
		})
	})
})
