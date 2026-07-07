package disaster_recovery

import (
	"github.com/konflux-ci/e2e-tests/pkg/framework"
	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck
)

// The outer Ordered container guarantees backwards-compat runs before
// same-version: backwards-compat upgrades Konflux mid-test, and
// same-version then exercises DR on the upgraded cluster.
//
// TEMPORARY: backwards-compat disabled to isolate PaC webhook delivery
// failure. The upgrade step in backwards-compat may be breaking the
// SprayProxy→PaC event listener chain. Running same-version alone
// determines whether the failure is upgrade-related or a PaC/Velero bug.
// See: plans_n_docs/KFLUXINFRA-2236_dr-ci-debug/pac-webhook-investigation/
var _ = framework.DisasterRecoverySuiteDescribe("DR Suite",
	Label("disaster-recovery"), Serial, Ordered, func() {
		// defineBackwardsCompatSpecs()
		defineSameVersionSpecs()
	})
