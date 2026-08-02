package disaster_recovery

import (
	"github.com/konflux-ci/e2e-tests/pkg/framework"
	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck
)

var _ = framework.DisasterRecoverySuiteDescribe("DR Suite",
	Label("disaster-recovery"), Serial, Ordered, func() {
		defineSameVersionSpecs()
	})
