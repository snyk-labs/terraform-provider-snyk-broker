// Copyright (c) Snyk Ltd.

package datasources_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestDataSources(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "DataSources Suite")
}
