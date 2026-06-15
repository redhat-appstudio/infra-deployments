// preconditions.go contains environment validation functions that run before
// any DR test. These are in a regular .go file (not _test.go) so they are
// compiled when the backup package is imported via blank import from cmd/.
//
// Each Describe block calls validateDREnvironment() in its BeforeAll to
// fail fast if the cluster is not ready for DR testing.
//
// NOTE: Helper functions call GinkgoHelper() so that assertion failures report
// the caller's location in the test spec, not the helper's internal line.
package disaster_recovery

import (
	"context"
	"fmt"

	"github.com/konflux-ci/e2e-tests/pkg/framework"
	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck
	. "github.com/onsi/gomega"    //nolint:staticcheck
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// validateDREnvironment runs all precondition checks for the DR test suite.
// Call this from BeforeAll in each Describe block to fail fast if the cluster
// is not ready. Each check is a separate By() step for clear diagnostics.
func validateDREnvironment(fw *framework.Framework) {
	GinkgoHelper()

	validateOADPOperator(fw)
	validateVeleroReady(fw)
	validateBSLAvailable(fw)
	validateGitHubRepo(fw)
}

// validateOADPOperator checks that OADP operator pods are running in the
// openshift-adp namespace.
func validateOADPOperator(fw *framework.Framework) {
	GinkgoHelper()
	By("Validating OADP operator health: pods running in " + VeleroNamespace)

	ctx := context.Background()
	pods, err := fw.AsKubeAdmin.CommonController.KubeInterface().CoreV1().Pods(VeleroNamespace).List(ctx, metav1.ListOptions{
		LabelSelector: "control-plane=controller-manager",
	})
	Expect(err).ShouldNot(HaveOccurred(), "failed to list pods in %s", VeleroNamespace)
	Expect(pods.Items).ShouldNot(BeEmpty(), "no pods found in %s — OADP operator may not be installed", VeleroNamespace)

	runningCount := 0
	for i := range pods.Items {
		if pods.Items[i].Status.Phase == "Running" {
			runningCount++
		}
	}
	Expect(runningCount).Should(BeNumerically(">", 0),
		"no Running pods found in %s — OADP operator is unhealthy", VeleroNamespace)
}

// validateVeleroReady checks that the Velero deployment in openshift-adp has
// at least one ready replica. Uses Eventually to tolerate brief unavailability
// during operator reconciliation.
func validateVeleroReady(fw *framework.Framework) {
	GinkgoHelper()
	By("Validating Velero readiness: deployment has ready replicas")

	ctx := context.Background()
	Eventually(func() (int32, error) {
		deploy, err := fw.AsKubeAdmin.CommonController.KubeInterface().AppsV1().Deployments(VeleroNamespace).Get(ctx, "velero", metav1.GetOptions{})
		if err != nil {
			return 0, err
		}
		return deploy.Status.ReadyReplicas, nil
	}, VeleroReadyTimeout, VeleroReadyPoll).Should(Equal(int32(1)),
		"velero deployment in %s should have exactly 1 ready replica", VeleroNamespace)
}

// validateBSLAvailable checks that at least one BackupStorageLocation CR in
// openshift-adp has status.phase == "Available". Uses Eventually because the
// BSL controller may take time to reconcile after the operator starts.
func validateBSLAvailable(fw *framework.Framework) {
	GinkgoHelper()
	By("Validating BSL availability: at least one BackupStorageLocation is Available")

	Eventually(func() (bool, error) {
		bslList := &velerov1.BackupStorageLocationList{}
		err := fw.AsKubeAdmin.CommonController.KubeRest().List(
			context.Background(),
			bslList,
			client.InNamespace(VeleroNamespace),
		)
		if err != nil {
			return false, fmt.Errorf("failed to list BackupStorageLocations: %w", err)
		}

		for i := range bslList.Items {
			if bslList.Items[i].Status.Phase == velerov1.BackupStorageLocationPhaseAvailable {
				return true, nil
			}
		}
		return false, nil
	}, VeleroReadyTimeout, VeleroReadyPoll).Should(BeTrue(),
		"no BackupStorageLocation in %s reached phase Available", VeleroNamespace)
}

// validateGitHubRepo verifies the MathWizz repo is reachable and contains the
// expected component structure. Each component's Dockerfile must exist at the
// path declared in the Components slice (const.go), so that any drift between
// the repo layout and the test constants is caught before the suite runs.
func validateGitHubRepo(fw *framework.Framework) {
	GinkgoHelper()
	By("Validating GitHub repo is reachable: " + MathWizzRepo)

	By(fmt.Sprintf("Validating repo structure: %d component Dockerfiles exist", len(Components)))
	ghClient := fw.AsKubeAdmin.HasController.Github
	for _, comp := range Components {
		dockerfilePath := comp.ContextDir + "/" + comp.DockerfileURL
		_, err := ghClient.GetFile(MathWizzRepoName, dockerfilePath, MathWizzDefaultBranch)
		Expect(err).ShouldNot(HaveOccurred(),
			"component %q Dockerfile not found at %s in repo %s — repo structure may have changed",
			comp.Name, dockerfilePath, MathWizzRepoName)
	}
}
