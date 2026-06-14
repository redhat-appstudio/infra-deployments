// dr_backwards_compat.go implements the backwards-compatibility DR test scenario.
//
// This scenario validates that backups taken on an OLDER Konflux version can be
// successfully restored on a NEWER Konflux version. It runs as an Ordered
// Ginkgo context (registered by dr_suite.go) with seven phases:
//
//  1. Create tenants on the OLD Konflux version (pre-upgrade).
//  2. Back up tenant data before the upgrade.
//  3. Simulate disaster by deleting tenant namespaces.
//  4. Upgrade Konflux to the new version mid-test.
//  5. Restore tenants from the pre-upgrade backups on the NEW version.
//  6. Post-restore recovery (SA token rotation).
//  7. Verify structural integrity and functional pipeline execution.
//
// The upgrade happens inside the test (Phase 4) rather than between two
// separate ginkgo processes. This is possible because UpgradeCluster and
// CheckOperatorsReady are pure Go functions. See performKonfluxUpgrade
// below for why this helper exists alongside the magefiles upgrade
// infrastructure.
package disaster_recovery

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/go-git/go-git/v5"
	gitconfig "github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	plumbingHttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/konflux-ci/e2e-tests/magefiles/installation"
	"github.com/konflux-ci/e2e-tests/pkg/framework"
	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck
	. "github.com/onsi/gomega"    //nolint:staticcheck
)

func defineBackwardsCompatSpecs() {
	Context("DR Backwards-Compat", Ordered, func() {
		defer GinkgoRecover()

		var fw *framework.Framework
		AfterEach(framework.ReportFailure(&fw))

		bcTenants := []Tenant{BCTenant1, BCTenant2}

		BeforeAll(func() {
			var err error
			fw, err = framework.NewFramework("dr-bc")
			Expect(err).ShouldNot(HaveOccurred(), "failed to create framework for backwards-compat")
			validateDREnvironment(fw)

			for i := range bcTenants {
				forkRepoForTenant(fw, &bcTenants[i])
			}
		})

		// Phase 1: Create tenants on the old Konflux version and run initial pipelines.
		When("creating tenants on the old Konflux version", func() {
			It("should create both tenants concurrently", func() {
				var wg sync.WaitGroup
				for _, t := range bcTenants {
					wg.Add(1)
					go func() {
						defer GinkgoRecover()
						defer wg.Done()
						createTenant(fw, t)
					}()
				}
				wg.Wait()
			})

			It("should wait for all build PipelineRuns to succeed", func() {
				waitForPipelineChains(fw, bcTenants, nil, nil)
			})

			It("should merge PaC configuration PRs on forked repos", func() {
				for _, t := range bcTenants {
					mergePaCConfigPRs(fw, t)
				}
			})
		})

		// Phase 2: Back up tenant data before Konflux upgrade.
		When("backing up tenant data before upgrade", func() {
			It("should create backup CRs for both tenants concurrently", func() {
				var wg sync.WaitGroup
				for _, t := range bcTenants {
					wg.Add(1)
					go func() {
						defer GinkgoRecover()
						defer wg.Done()
						createBackup(fw, t)
					}()
				}
				wg.Wait()
			})
		})

		// Phase 3: Simulate disaster by deleting tenant namespaces.
		When("simulating disaster by deleting namespaces", func() {
			It("should delete both tenant namespaces", func() {
				for _, t := range bcTenants {
					deleteNamespace(fw, t.Namespace)
				}
			})
		})

		// Phase 4: Upgrade Konflux to the new version mid-test.
		// This merges the PR branch into infra-deployments, pushes to the qe
		// remote, and waits for ArgoCD to sync. If the upgrade fails,
		// klog.Fatal kills the process — intentional, see performKonfluxUpgrade.
		When("upgrading Konflux to the new version", func() {
			It("should upgrade the cluster and verify Velero survived", func() {
				performKonfluxUpgrade(fw)
			})
		})

		// Phase 5: Restore tenants from pre-upgrade backups on the new version.
		When("restoring tenants from backup on the new Konflux version", func() {
			It("should restore tenant-1 (KokoHazamar) via velero CLI method", func() {
				restoreFromBackup(fw, BCTenant1, RestoreMethodVeleroCLI)
			})

			It("should restore tenant-2 (MosheKipod) via oc command method", func() {
				restoreFromBackup(fw, BCTenant2, RestoreMethodOCCommand)
			})
		})

		// Phase 6: Post-restore recovery — rotate stale SA tokens.
		When("performing post-restore recovery", func() {
			It("should rotate SA tokens on both tenants", func() {
				for _, t := range bcTenants {
					rotateSATokens(fw, t.Namespace)
				}
			})
		})

		// Phase 7: Verify restored tenants are structurally and functionally intact.
		When("verifying restored tenants", func() {
			It("should confirm structural integrity of both tenants", func() {
				for _, t := range bcTenants {
					verifyResources(fw, t)
				}
			})

			It("should confirm functional pipeline execution after restore", func() {
				triggerBuildsAndVerify(fw, bcTenants)
			})
		})

		AfterAll(func() {
			cleanupForks(fw, bcTenants)
			if CurrentSpecReport().Failed() {
				collectFailureArtifacts(fw, bcTenants)
			} else {
				cleanupTestResources(fw, bcTenants)
			}
		})
	})
}

// performKonfluxUpgrade merges the PR branch into infra-deployments and waits
// for ArgoCD to sync all applications to the new version, then verifies that
// Velero and OADP survived the upgrade.
//
// Why this exists alongside e2e-tests' upgrade infrastructure:
// The existing upgrade functions (UpgradeCluster, MergePRInRemote) live in
// magefiles/magefile.go which is package main — not importable by test code.
// The backwards-compat test needs the upgrade to happen mid-test (between
// the backup and restore phases) within a single Ginkgo describe block,
// rather than between two separate ginkgo process invocations. This function
// mirrors MergePRInRemote using the same go-git library calls and only shells
// out for the merge step (which the original also does via exec.Command).
//
// If the upgrade fails, klog.Fatal in CheckOperatorsReady kills the process.
// This is intentional — a failed upgrade means DR results are meaningless,
// and the fatal crash leaves unmistakable diagnostic output in the logs.
func performKonfluxUpgrade(fw *framework.Framework) {
	GinkgoHelper()

	repoPath := "./tmp/infra-deployments"
	branch := os.Getenv("UPGRADE_BRANCH")
	Expect(branch).ShouldNot(BeEmpty(), "UPGRADE_BRANCH env var must be set")

	forkOrg := os.Getenv("UPGRADE_FORK_ORGANIZATION")
	if forkOrg == "" {
		forkOrg = "redhat-appstudio"
	}

	auth := &plumbingHttp.BasicAuth{
		Username: "123",
		Password: os.Getenv("GITHUB_TOKEN"),
	}

	// Open repo and find the preview branch — mirrors MergePRInRemote's
	// branch discovery in magefiles/utils.go.
	By("Opening infra-deployments repo and finding preview branch")
	repo, err := git.PlainOpen(repoPath)
	Expect(err).ShouldNot(HaveOccurred(), "failed to open infra-deployments repo at %s", repoPath)

	branches, err := repo.Branches()
	Expect(err).ShouldNot(HaveOccurred(), "failed to list branches")

	var previewBranchRef *plumbing.Reference
	err = branches.ForEach(func(ref *plumbing.Reference) error {
		if !strings.Contains(ref.Name().String(), "main") {
			previewBranchRef = ref
		}
		return nil
	})
	Expect(err).ShouldNot(HaveOccurred(), "failed to iterate branches")
	Expect(previewBranchRef).ShouldNot(BeNil(), "no preview branch found in %s", repoPath)

	wt, err := repo.Worktree()
	Expect(err).ShouldNot(HaveOccurred(), "failed to get worktree")

	By(fmt.Sprintf("Checking out preview branch %s", previewBranchRef.Name()))
	Expect(wt.Checkout(&git.CheckoutOptions{
		Branch: previewBranchRef.Name(),
	})).Should(Succeed(), "failed to checkout preview branch")

	// Merge the upgrade branch — mirrors the conditional logic in
	// MergePRInRemote. The merge itself shells out to git (same as original).
	By(fmt.Sprintf("Merging upgrade branch from %s/%s", forkOrg, branch))
	if forkOrg == "redhat-appstudio" {
		mergeBranchOrFail(repoPath, "remotes/upstream/"+branch)
	} else {
		repoURL := fmt.Sprintf("https://github.com/%s/infra-deployments.git", forkOrg)
		_, err = repo.CreateRemote(&gitconfig.RemoteConfig{
			Name: "forked_repo",
			URLs: []string{repoURL},
		})
		Expect(err).ShouldNot(HaveOccurred(), "failed to create forked_repo remote")

		Expect(repo.Fetch(&git.FetchOptions{
			RemoteName: "forked_repo",
		})).Should(Succeed(), "failed to fetch from forked_repo")

		mergeBranchOrFail(repoPath, "remotes/forked_repo/"+branch)
	}

	// Push to the qe remote — mirrors MergePRInRemote's push. The qe remote
	// was configured during BootstrapClusterForUpgrade with the correct URL.
	By("Pushing merged changes to qe remote")
	Expect(repo.Push(&git.PushOptions{
		RefSpecs:   []gitconfig.RefSpec{gitconfig.RefSpec(fmt.Sprintf("%s:%s", previewBranchRef.Name(), previewBranchRef.Name()))},
		RemoteName: "qe",
		Auth:       auth,
	})).Should(Succeed(), "failed to push to qe remote")

	// Wait for ArgoCD to sync — uses the importable installation package.
	By("Waiting for ArgoCD to sync all applications after upgrade")
	ic, err := installation.NewAppStudioInstallController()
	Expect(err).ShouldNot(HaveOccurred(), "failed to initialize installation controller")
	Expect(ic.CheckOperatorsReady()).Should(Succeed(), "operators not ready after upgrade")

	// Verify backup infrastructure survived the upgrade. Only Velero and BSL,
	// not the full validateDREnvironment — the upgrade is tested elsewhere.
	By("Verifying Velero and OADP survived the upgrade")
	validateVeleroReady(fw)
	validateBSLAvailable(fw)
}

// mergeBranchOrFail shells out to git merge, mirroring mergeBranch in
// magefiles/utils.go. go-git doesn't support merge, so exec.Command is
// used (same approach as the original).
func mergeBranchOrFail(repoPath, branchToMerge string) {
	GinkgoHelper()

	cmd := exec.Command("git", "-C", repoPath, "merge", branchToMerge, "-Xtheirs", "-q")
	out, err := cmd.CombinedOutput()
	Expect(err).ShouldNot(HaveOccurred(),
		"failed to merge branch %s in %s: %s", branchToMerge, repoPath, string(out))
}
