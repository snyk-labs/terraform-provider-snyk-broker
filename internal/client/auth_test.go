// Copyright (c) Snyk Ltd.

package client_test

import (
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/snyk/terraform-provider-snyk-broker/internal/client"
	"github.com/snyk/terraform-provider-snyk-broker/internal/testutil"
)

var _ = Describe("TokenAuthenticator", func() {
	var auth *client.TokenAuthenticator

	BeforeEach(func() {
		auth = client.NewTokenAuthenticator("test-token")
	})

	Describe("Authenticate", func() {
		Context("with a valid request", func() {
			It("adds the Authorization header", func() {
				req, _ := http.NewRequest("GET", "https://api.snyk.io", nil)
				err := auth.Authenticate(req)

				Expect(err).NotTo(HaveOccurred())
				Expect(req.Header.Get("Authorization")).To(Equal("token test-token"))
			})
		})

		Context("with a nil request", func() {
			It("returns an error", func() {
				err := auth.Authenticate(nil)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("request cannot be nil"))
			})
		})
	})
})

var _ = Describe("OAuthAuthenticator", func() {
	var (
		auth     *client.OAuthAuthenticator
		mockHTTP *testutil.MockHTTPClient
	)

	BeforeEach(func() {
		mockHTTP = &testutil.MockHTTPClient{}
		auth = client.NewOAuthAuthenticator("client-id", "client-secret", "https://api.snyk.io/oauth2/token")
		auth.WithHTTPClient(mockHTTP)
	})

	Describe("Authenticate", func() {
		Context("with a nil request", func() {
			It("returns an error", func() {
				err := auth.Authenticate(nil)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("request cannot be nil"))
			})
		})

		Context("with a valid token response", func() {
			It("adds the Bearer token header", func() {
				mockHTTP.DoFunc = func(req *http.Request) (*http.Response, error) {
					return testutil.NewMockJSONResponse(http.StatusOK, `{
						"access_token": "oauth-access-token",
						"token_type": "Bearer",
						"expires_in": 3600
					}`), nil
				}

				req, _ := http.NewRequest("GET", "https://api.snyk.io", nil)
				err := auth.Authenticate(req)

				Expect(err).NotTo(HaveOccurred())
				Expect(req.Header.Get("Authorization")).To(Equal("Bearer oauth-access-token"))
			})
		})

		Context("with a failed token request", func() {
			It("returns an error", func() {
				mockHTTP.DoFunc = func(req *http.Request) (*http.Response, error) {
					return testutil.NewMockJSONResponse(http.StatusUnauthorized, `{"error": "invalid_client"}`), nil
				}

				req, _ := http.NewRequest("GET", "https://api.snyk.io", nil)
				err := auth.Authenticate(req)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("failed to get access token"))
			})
		})
	})
})

var _ = Describe("OAuthTokenURL", func() {
	DescribeTable("returns correct URL for region",
		func(region, expectedURL string) {
			url := client.OAuthTokenURL(region)
			Expect(url).To(Equal(expectedURL))
		},
		Entry("US region", "us", "https://api.snyk.io/oauth2/token"),
		Entry("EU region", "eu", "https://api.eu.snyk.io/oauth2/token"),
		Entry("AU region", "au", "https://api.au.snyk.io/oauth2/token"),
		Entry("empty defaults to US", "", "https://api.snyk.io/oauth2/token"),
		Entry("unknown defaults to US", "unknown", "https://api.snyk.io/oauth2/token"),
	)
})
