// tenant_application_lifecycle.go provides helpers for verifying the MathWizz
// application build lifecycle on tenant namespaces: waiting for the full
// pipeline chain (build → integration test → release) and triggering new
// builds via git push to verify the pipeline chain survives backup/restore.
//
// NOTE: Helper functions call GinkgoHelper() so that assertion failures report
// the caller's location in the test spec, not the helper's internal line.
package disaster_recovery

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/konflux-ci/e2e-tests/pkg/framework"
	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck
	. "github.com/onsi/gomega"    //nolint:staticcheck
	pipeline "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ---------------------------------------------------------------------------
// Core PipelineRun counting and waiting — all other helpers build on these two
// ---------------------------------------------------------------------------

// countSucceededPRs returns the number of PipelineRuns with Succeeded=True in
// the given namespace. Filters are additive:
//   - pipelineType non-empty: filter by "pipelines.appstudio.openshift.io/type" label
//   - componentName non-empty: filter by "appstudio.openshift.io/component" label
//
// Pass empty strings to skip either filter (e.g., empty pipelineType counts
// all PRs, used for the managed namespace where every PR is a release pipeline).
func countSucceededPRs(fw *framework.Framework, namespace, pipelineType, componentName string) int {
	listOpts := buildListOpts(namespace, pipelineType, componentName)

	prList := &pipeline.PipelineRunList{}
	if err := fw.AsKubeAdmin.CommonController.KubeRest().List(
		context.Background(), prList, listOpts...); err != nil {
		return 0
	}

	count := 0
	for i := range prList.Items {
		for _, c := range prList.Items[i].Status.Conditions {
			if c.Type == "Succeeded" && c.Status == "True" {
				count++
				break
			}
		}
	}
	return count
}

// logFailedTaskRuns lists TaskRuns belonging to a failed PipelineRun and logs
// each failed TaskRun's pipeline task name and failure message.
func logFailedTaskRuns(fw *framework.Framework, namespace, prName string) {
	trList := &pipeline.TaskRunList{}
	if err := fw.AsKubeAdmin.CommonController.KubeRest().List(
		context.Background(), trList,
		client.InNamespace(namespace),
		client.MatchingLabels{"tekton.dev/pipelineRun": prName},
	); err != nil {
		GinkgoWriter.Printf("  could not list TaskRuns for PipelineRun %s: %v\n", prName, err)
		return
	}

	for i := range trList.Items {
		tr := &trList.Items[i]
		for _, c := range tr.Status.Conditions {
			if c.Type == "Succeeded" {
				if c.Status == "False" {
					taskName := tr.Labels["tekton.dev/pipelineTask"]
					GinkgoWriter.Printf("  FAILED TaskRun %s (task: %s) in PipelineRun %s: %s\n",
						tr.Name, taskName, prName, c.Message)
				}
				break
			}
		}
	}
}

// waitForSucceededPRCount polls until exactly expectedCount PipelineRuns with
// Succeeded=True exist in the namespace. Any deviation from the expected count
// (including exceeding it) is treated as a finding. Failed PipelineRuns are
// logged with their component name and failure reason for debugging.
//
// Filters follow the same rules as countSucceededPRs: empty pipelineType or
// componentName skips that filter.
func waitForSucceededPRCount(fw *framework.Framework, namespace, pipelineType, componentName string, expectedCount int, timeout, poll time.Duration) {
	GinkgoHelper()

	componentLabel := "appstudio.openshift.io/component"
	displayType := pipelineType
	if displayType == "" {
		displayType = "release"
	}

	listOpts := buildListOpts(namespace, pipelineType, componentName)
	loggedFailures := map[string]bool{}

	Eventually(func() int {
		prList := &pipeline.PipelineRunList{}
		if err := fw.AsKubeAdmin.CommonController.KubeRest().List(
			context.Background(), prList, listOpts...); err != nil {
			GinkgoWriter.Printf("error listing %s PipelineRuns in %s: %v\n",
				displayType, namespace, err)
			return 0
		}

		succeededCount := 0
		for i := range prList.Items {
			pr := &prList.Items[i]
			for _, c := range pr.Status.Conditions {
				if c.Type == "Succeeded" {
					switch c.Status {
					case "True":
						succeededCount++
					case "False":
						GinkgoWriter.Printf(
							"FAILED %s PipelineRun %s (component: %s) in %s: %s\n",
							displayType, pr.Name, pr.Labels[componentLabel],
							namespace, c.Message)
						if !loggedFailures[pr.Name] {
							loggedFailures[pr.Name] = true
							logFailedTaskRuns(fw, namespace, pr.Name)
						}
					}
					break
				}
			}
		}

		GinkgoWriter.Printf("namespace %s: %d/%d %s PipelineRuns succeeded\n",
			namespace, succeededCount, expectedCount, displayType)
		return succeededCount
	}, timeout, poll).Should(Equal(expectedCount),
		"expected %d successful %s PipelineRuns in namespace %s",
		expectedCount, displayType, namespace)
}

// buildListOpts constructs the label-based list options shared by
// countSucceededPRs and waitForSucceededPRCount.
func buildListOpts(namespace, pipelineType, componentName string) []client.ListOption {
	opts := []client.ListOption{client.InNamespace(namespace)}
	if pipelineType != "" {
		opts = append(opts,
			client.MatchingLabels{"pipelines.appstudio.openshift.io/type": pipelineType})
	}
	if componentName != "" {
		opts = append(opts,
			client.MatchingLabels{"appstudio.openshift.io/component": componentName})
	}
	return opts
}

// ---------------------------------------------------------------------------
// High-level lifecycle helpers
// ---------------------------------------------------------------------------

// pipelineRunBaseCounts holds per-component build and test PipelineRun counts.
// Used as a baseline for waitForPipelineChains so it can wait for counts
// relative to an initial snapshot (e.g., after triggering a new build).
type pipelineRunBaseCounts struct {
	build int
	test  int
}

// waitForPipelineChains waits for the full pipeline chain (build → test →
// release) to complete for every component across all tenants. Each
// component's chain runs in its own goroutine so that a slow component
// doesn't block faster ones from progressing through subsequent stages.
// Release PipelineRuns are waited for after all build/test chains complete,
// since release PRs may not be per-component.
//
// baseBuildTest provides per-component starting counts keyed by
// "namespace/componentName". baseRelease provides aggregate starting counts
// keyed by managed namespace. Pass nil for both on the first run (base of 0).
func waitForPipelineChains(fw *framework.Framework, tenants []Tenant,
	baseBuildTest map[string]pipelineRunBaseCounts, baseRelease map[string]int) {
	GinkgoHelper()

	By("Waiting for per-component build → test chains across all tenants")

	var wg sync.WaitGroup
	for _, t := range tenants {
		for _, comp := range Components {
			wg.Add(1)
			go func(tenant Tenant, component ComponentDef) {
				defer GinkgoRecover()
				defer wg.Done()

				key := tenant.Namespace + "/" + component.Name
				base := baseBuildTest[key] // zero-value if nil map or missing key

				By(fmt.Sprintf("Waiting for build PipelineRun for %s in %s (base: %d)",
					component.Name, tenant.Namespace, base.build))
				waitForSucceededPRCount(fw, tenant.Namespace, "build", component.Name,
					base.build+1, PipelineTimeout, PipelinePoll)

				By(fmt.Sprintf("Waiting for test PipelineRun for %s in %s (base: %d)",
					component.Name, tenant.Namespace, base.test))
				waitForSucceededPRCount(fw, tenant.Namespace, "test", component.Name,
					base.test+1, PipelineTimeout, PipelinePoll)
			}(t, comp)
		}
	}
	wg.Wait()

	// Release PipelineRuns run in the managed namespace and may not map 1:1
	// to components, so wait for them in aggregate after all builds/tests pass.
	for _, t := range tenants {
		releaseBase := baseRelease[t.ManagedNamespace] // zero if nil map or missing key
		expected := releaseBase + ComponentsPerTenant
		By(fmt.Sprintf("Waiting for %d release PipelineRuns in %s (base: %d)",
			expected, t.ManagedNamespace, releaseBase))
		waitForSucceededPRCount(fw, t.ManagedNamespace, "", "", expected,
			ReleaseChainTimeout, ReleaseChainPoll)
	}
}

// triggerBuildsAndVerify creates a pull request on each tenant's forked
// MathWizz repo to trigger new builds via PaC webhooks, then waits for the
// full pipeline chain (build → integration test → release) to complete across
// all tenants. This proves that PaC webhooks, Secrets, ServiceAccounts,
// IntegrationTestScenarios, ReleasePlans, and the full build/test/release
// chain survived the backup/restore cycle.
//
// The method:
//  1. Snapshots current per-component PipelineRun counts.
//  2. For each tenant: creates a branch, appends a timestamp to README.md,
//     opens a PR on the tenant's fork repo.
//  3. Waits for new build and test PipelineRuns per component (parallel).
//  4. Waits for new release PipelineRuns (aggregate).
//  5. Cleans up the branches (which closes the PRs).
func triggerBuildsAndVerify(fw *framework.Framework, tenants []Tenant) {
	GinkgoHelper()

	By("Snapshotting current per-component PipelineRun counts before triggering")

	initialPerComp := make(map[string]pipelineRunBaseCounts)
	initialRelease := make(map[string]int)

	for _, t := range tenants {
		for _, comp := range Components {
			key := t.Namespace + "/" + comp.Name
			initialPerComp[key] = pipelineRunBaseCounts{
				build: countSucceededPRs(fw, t.Namespace, "build", comp.Name),
				test:  countSucceededPRs(fw, t.Namespace, "test", comp.Name),
			}
			GinkgoWriter.Printf("initial counts for %s: build=%d, test=%d\n",
				key, initialPerComp[key].build, initialPerComp[key].test)
		}
		initialRelease[t.ManagedNamespace] = countSucceededPRs(fw, t.ManagedNamespace, "", "")
		GinkgoWriter.Printf("initial release count for %s: %d\n",
			t.ManagedNamespace, initialRelease[t.ManagedNamespace])
	}

	ghClient := fw.AsKubeAdmin.HasController.Github

	for _, t := range tenants {
		Expect(t.ForkRepoName).ShouldNot(BeEmpty(),
			"ForkRepoName not set for tenant %s", t.Namespace)

		branchName := fmt.Sprintf("dr-test-trigger-%s-%d", t.AppName, time.Now().Unix())

		By(fmt.Sprintf("Creating trigger PR on fork %s for tenant %s", t.ForkRepoName, t.Namespace))

		err := ghClient.CreateRef(t.ForkRepoName, MathWizzDefaultBranch, "", branchName)
		Expect(err).ShouldNot(HaveOccurred(),
			"failed to create branch %s in %s", branchName, t.ForkRepoName)

		defer func(repo, branch string) {
			By(fmt.Sprintf("Cleaning up trigger branch %s on %s", branch, repo))
			if deleteErr := ghClient.DeleteRef(repo, branch); deleteErr != nil {
				GinkgoWriter.Printf("WARNING: failed to delete trigger branch %s on %s: %v\n",
					branch, repo, deleteErr)
			}
		}(t.ForkRepoName, branchName)

		readmeFile, err := ghClient.GetFile(t.ForkRepoName, "README.md", branchName)
		Expect(err).ShouldNot(HaveOccurred(),
			"failed to get README.md from branch %s in %s", branchName, t.ForkRepoName)

		existingContent, err := readmeFile.GetContent()
		Expect(err).ShouldNot(HaveOccurred(), "failed to decode README.md content")

		updatedContent := existingContent + fmt.Sprintf("\n<!-- DR test trigger %s: %d -->\n",
			t.AppName, time.Now().Unix())
		_, err = ghClient.UpdateFile(t.ForkRepoName, "README.md",
			updatedContent, branchName, readmeFile.GetSHA())
		Expect(err).ShouldNot(HaveOccurred(),
			"failed to update README.md on branch %s in %s", branchName, t.ForkRepoName)

		pr, err := ghClient.CreatePullRequest(t.ForkRepoName,
			fmt.Sprintf("DR test: trigger builds for %s", t.AppName),
			"Automated PR to verify the full build/test/release pipeline chain "+
				"survives backup/restore. Created by the DR e2e test suite.",
			branchName, MathWizzDefaultBranch)
		Expect(err).ShouldNot(HaveOccurred(),
			"failed to create pull request on %s", t.ForkRepoName)
		GinkgoWriter.Printf("Created PR #%d on %s to trigger builds for tenant %s\n",
			pr.GetNumber(), t.ForkRepoName, t.Namespace)
	}

	waitForPipelineChains(fw, tenants, initialPerComp, initialRelease)
}
