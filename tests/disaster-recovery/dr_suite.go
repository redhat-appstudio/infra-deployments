package disaster_recovery

import (
	"github.com/konflux-ci/e2e-tests/pkg/framework"
	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck
)

// The outer Ordered container guarantees backwards-compat runs before
// same-version: backwards-compat upgrades Konflux mid-test, and
// same-version then exercises DR on the upgraded cluster.
var _ = framework.DisasterRecoverySuiteDescribe("DR Suite",
	Label("disaster-recovery"), Serial, Ordered, func() {
		defineBackwardsCompatSpecs()
		defineSameVersionSpecs()
	})
