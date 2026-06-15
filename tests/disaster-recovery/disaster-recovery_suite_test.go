// disaster-recovery_suite_test.go provides a test runner for running the disaster-recovery suite
// directly via `go test ./tests/disaster-recovery/` or `ginkgo ./tests/disaster-recovery/`.
//
// When running from cmd/ (the standard CI path via `ginkgo ./cmd/`), this
// file is NOT compiled — Ginkgo discovers the specs via the blank import in
// cmd/e2e_test.go instead, and each Describe block handles its own setup
// in BeforeAll.
package disaster_recovery

import (
	"testing"

	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck
	. "github.com/onsi/gomega"    //nolint:staticcheck
)

func TestDisasterRecovery(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "DR backup/restore e2e suite")
}
