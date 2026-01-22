// Copyright (c) Snyk Ltd.

package client_test

import (
	"context"
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/snyk/terraform-provider-snyk-broker/internal/client"
	"github.com/snyk/terraform-provider-snyk-broker/internal/testutil"
)

var _ = Describe("Client", func() {
	var (
		c        *client.Client
		mockHTTP *testutil.MockHTTPClient
		mockAuth *testutil.MockAuthenticator
	)

	BeforeEach(func() {
		mockHTTP = &testutil.MockHTTPClient{}
		mockAuth = &testutil.MockAuthenticator{}
	})

	Describe("NewClient", func() {
		It("creates a client with default settings", func() {
			c = client.NewClient("https://api.snyk.io", mockAuth)
			Expect(c).NotTo(BeNil())
			Expect(c.GetAPIVersion()).To(Equal("2024-10-15"))
		})

		It("accepts custom HTTP client option", func() {
			c = client.NewClient("https://api.snyk.io", mockAuth, client.WithHTTPClient(mockHTTP))
			Expect(c).NotTo(BeNil())
		})

		It("accepts custom API version option", func() {
			c = client.NewClient("https://api.snyk.io", mockAuth, client.WithAPIVersion("2025-01-01"))
			Expect(c.GetAPIVersion()).To(Equal("2025-01-01"))
		})
	})

	Describe("RegionToBaseURL", func() {
		DescribeTable("returns correct URL for region",
			func(region, expectedURL string) {
				url := client.RegionToBaseURL(region)
				Expect(url).To(Equal(expectedURL))
			},
			Entry("US region", "us", "https://api.snyk.io"),
			Entry("EU region", "eu", "https://api.eu.snyk.io"),
			Entry("AU region", "au", "https://api.au.snyk.io"),
			Entry("empty defaults to US", "", "https://api.snyk.io"),
			Entry("unknown defaults to US", "unknown", "https://api.snyk.io"),
		)
	})

	Describe("Get", func() {
		BeforeEach(func() {
			c = client.NewClient("https://api.snyk.io", mockAuth, client.WithHTTPClient(mockHTTP))
		})

		Context("with a successful response", func() {
			It("returns the parsed response", func() {
				mockHTTP.DoFunc = func(req *http.Request) (*http.Response, error) {
					Expect(req.Method).To(Equal(http.MethodGet))
					Expect(req.URL.String()).To(ContainSubstring("/test"))
					return testutil.NewMockJSONResponse(http.StatusOK, `{"data": {"id": "123"}}`), nil
				}

				var result map[string]interface{}
				err := c.Get(context.Background(), "/test", &result)
				Expect(err).NotTo(HaveOccurred())
				Expect(result["data"]).NotTo(BeNil())
			})
		})

		Context("with an error response", func() {
			It("returns an API error", func() {
				mockHTTP.DoFunc = func(req *http.Request) (*http.Response, error) {
					return testutil.NewMockJSONResponse(http.StatusNotFound, `{"error": "not found"}`), nil
				}

				var result map[string]interface{}
				err := c.Get(context.Background(), "/test", &result)
				Expect(err).To(HaveOccurred())

				apiErr, ok := err.(*client.APIError)
				Expect(ok).To(BeTrue())
				Expect(apiErr.StatusCode).To(Equal(http.StatusNotFound))
			})

			It("includes snyk-request-id in error when present", func() {
				mockHTTP.DoFunc = func(req *http.Request) (*http.Response, error) {
					return testutil.NewMockJSONResponseWithRequestID(
						http.StatusGatewayTimeout,
						`{"error": "Gateway Timeout"}`,
						"abc-123-request-id",
					), nil
				}

				var result map[string]interface{}
				err := c.Get(context.Background(), "/test", &result)
				Expect(err).To(HaveOccurred())

				apiErr, ok := err.(*client.APIError)
				Expect(ok).To(BeTrue())
				Expect(apiErr.StatusCode).To(Equal(http.StatusGatewayTimeout))
				Expect(apiErr.RequestID).To(Equal("abc-123-request-id"))
				Expect(err.Error()).To(ContainSubstring("snyk-request-id: abc-123-request-id"))
			})

			It("does not include snyk-request-id in error when not present", func() {
				mockHTTP.DoFunc = func(req *http.Request) (*http.Response, error) {
					return testutil.NewMockJSONResponse(http.StatusInternalServerError, `{"error": "server error"}`), nil
				}

				var result map[string]interface{}
				err := c.Get(context.Background(), "/test", &result)
				Expect(err).To(HaveOccurred())

				apiErr, ok := err.(*client.APIError)
				Expect(ok).To(BeTrue())
				Expect(apiErr.RequestID).To(BeEmpty())
				Expect(err.Error()).NotTo(ContainSubstring("snyk-request-id"))
			})
		})
	})

	Describe("Post", func() {
		BeforeEach(func() {
			c = client.NewClient("https://api.snyk.io", mockAuth, client.WithHTTPClient(mockHTTP))
		})

		It("sends a POST request with body", func() {
			mockHTTP.DoFunc = func(req *http.Request) (*http.Response, error) {
				Expect(req.Method).To(Equal(http.MethodPost))
				Expect(req.Header.Get("Content-Type")).To(Equal("application/vnd.api+json"))
				return testutil.NewMockJSONResponse(http.StatusCreated, `{"data": {"id": "456"}}`), nil
			}

			body := map[string]string{"name": "test"}
			var result map[string]interface{}
			err := c.Post(context.Background(), "/test", body, &result)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("Delete", func() {
		BeforeEach(func() {
			c = client.NewClient("https://api.snyk.io", mockAuth, client.WithHTTPClient(mockHTTP))
		})

		It("sends a DELETE request", func() {
			mockHTTP.DoFunc = func(req *http.Request) (*http.Response, error) {
				Expect(req.Method).To(Equal(http.MethodDelete))
				return testutil.NewMockJSONResponse(http.StatusNoContent, ""), nil
			}

			err := c.Delete(context.Background(), "/test/123")
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
