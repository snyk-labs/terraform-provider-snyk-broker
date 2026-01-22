# Copyright (c) Snyk Ltd.
# SPDX-License-Identifier: MPL-2.0

default: testacc

# Build the provider
build:
	go build -o terraform-provider-snyk-broker

# Install the provider locally
install: build
	mkdir -p ~/.terraform.d/plugins/registry.terraform.io/snyk/snyk-broker/0.1.0/$$(go env GOOS)_$$(go env GOARCH)
	mv terraform-provider-snyk-broker ~/.terraform.d/plugins/registry.terraform.io/snyk/snyk-broker/0.1.0/$$(go env GOOS)_$$(go env GOARCH)/

# Run unit tests with Ginkgo
test:
	ginkgo -r -v --randomize-all --randomize-suites

# Run tests with coverage (cross-package coverage tracking)
test-coverage:
	go test ./internal/... -coverprofile=coverage.out -covermode=atomic -coverpkg=./internal/...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"
	@go tool cover -func=coverage.out | tail -1

# Run specific package tests
test-client:
	ginkgo -v ./internal/client/...

test-provider:
	ginkgo -v ./internal/provider/...

test-resources:
	ginkgo -v ./internal/resources/...

test-datasources:
	ginkgo -v ./internal/datasources/...

# Watch mode for development
test-watch:
	ginkgo watch -r -v

# Run acceptance tests (requires real API credentials)
testacc:
	TF_ACC=1 go test ./... -v $(TESTARGS) -timeout 120m

# Generate documentation
generate:
	go generate ./...

# Format code
fmt:
	gofmt -s -w .

# Lint code
lint:
	golangci-lint run

# Clean build artifacts
clean:
	rm -f terraform-provider-snyk-broker
	rm -f coverage.out coverage.html

# Tidy dependencies
tidy:
	go mod tidy

.PHONY: default build install test test-coverage test-client test-provider test-resources test-datasources test-watch testacc generate fmt lint clean tidy

