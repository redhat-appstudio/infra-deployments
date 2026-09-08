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
	"io"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/konflux-ci/e2e-tests/pkg/framework"
	releaseapi "github.com/konflux-ci/release-service/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck
	. "github.com/onsi/gomega"    //nolint:staticcheck
	pipeline "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// PodLogTailLines is the number of log lines to capture from each failing step container.
const PodLogTailLines int64 = 80

// ---------------------------------------------------------------------------
// Core PipelineRun counting and waiting — all other helpers build on these two
// ---------------------------------------------------------------------------

// countPRs returns both the succeeded count and the total count (regardless
// of status) of PipelineRuns matching the given filters, from a single List
// call. Both numbers must come from the same observation: reading them via
// two separate List calls lets a PipelineRun transition (or a new one
// appear) between the calls, producing an inconsistent succeeded/total pair
// that misleads waitForSucceededPRCount's baseline — see its doc comment.
// Filters are additive:
//   - pipelineType non-empty: filter by "pipelines.appstudio.openshift.io/type" label
//   - componentName non-empty: filter by "appstudio.openshift.io/component" label
//
// Pass empty strings to skip either filter (e.g., empty pipelineType counts
// all PRs, used for the managed namespace where every PR is a release pipeline).
func countPRs(ctx context.Context, fw *framework.Framework, namespace, pipelineType, componentName string) (succeeded, total int, err error) {
	listOpts := buildListOpts(namespace, pipelineType, componentName)

	prList := &pipeline.PipelineRunList{}
	if err := fw.AsKubeAdmin.CommonController.KubeRest().List(
		ctx, prList, listOpts...); err != nil {
		return 0, 0, fmt.Errorf("listing PipelineRuns in %s: %w", namespace, err)
	}

	total = len(prList.Items)
	for i := range prList.Items {
		for _, c := range prList.Items[i].Status.Conditions {
			if c.Type == "Succeeded" && c.Status == "True" {
				succeeded++
				break
			}
		}
	}
	return succeeded, total, nil
}

// logFailedTaskRuns lists TaskRuns belonging to a failed PipelineRun and logs
// each failed TaskRun's pipeline task name, failure message, and the actual
// container logs from the failing step. The container logs are critical for
// diagnosing OCI-TA and other step-level failures where the condition message
// only says "exited with code 1: Error".
func logFailedTaskRuns(ctx context.Context, fw *framework.Framework, namespace, prName string) {

	trList := &pipeline.TaskRunList{}
	if err := fw.AsKubeAdmin.CommonController.KubeRest().List(
		ctx, trList,
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
					logFailedStepContainers(ctx, fw, namespace, tr)
				}
				break
			}
		}
	}
}

// logFailedStepContainers reads the container logs from the pod backing a
// failed TaskRun. It identifies which step(s) failed from the TaskRun status
// and fetches the last 80 lines of each failing container's logs.
func logFailedStepContainers(ctx context.Context, fw *framework.Framework, namespace string, tr *pipeline.TaskRun) {
	podName := tr.Status.PodName
	if podName == "" {
		GinkgoWriter.Printf("    no pod name in TaskRun %s status — cannot read container logs\n", tr.Name)
		return
	}

	failedContainers := findFailedStepContainers(tr)
	if len(failedContainers) == 0 {
		GinkgoWriter.Printf("    no failed step containers identified in TaskRun %s — dumping all step statuses\n", tr.Name)
		for _, step := range tr.Status.Steps {
			state := "unknown"
			if step.Terminated != nil {
				state = fmt.Sprintf("terminated(exit=%d, reason=%s)", step.Terminated.ExitCode, step.Terminated.Reason)
			} else if step.Running != nil {
				state = "running"
			} else if step.Waiting != nil {
				state = fmt.Sprintf("waiting(reason=%s)", step.Waiting.Reason)
			}
			GinkgoWriter.Printf("    step %s: %s\n", step.Name, state)
		}
		return
	}

	kubeClient := fw.AsKubeAdmin.CommonController.KubeInterface()
	tailLines := PodLogTailLines

	for _, containerName := range failedContainers {
		GinkgoWriter.Printf("    --- container logs: %s/%s (container: %s) ---\n", namespace, podName, containerName)

		logReq := kubeClient.CoreV1().Pods(namespace).GetLogs(podName, &corev1.PodLogOptions{
			Container: containerName,
			TailLines: &tailLines,
		})
		logCtx, logCancel := context.WithTimeout(ctx, 30*time.Second)
		stream, err := logReq.Stream(logCtx)
		if err != nil {
			logCancel()
			GinkgoWriter.Printf("    ERROR reading logs for %s/%s container %s: %v\n",
				namespace, podName, containerName, err)
			continue
		}

		logBytes, err := io.ReadAll(stream)
		stream.Close()
		logCancel()
		if err != nil {
			GinkgoWriter.Printf("    ERROR reading log stream for %s/%s container %s: %v\n",
				namespace, podName, containerName, err)
			continue
		}

		logStr := string(logBytes)
		if logStr == "" {
			GinkgoWriter.Printf("    (empty log output)\n")
		} else {
			GinkgoWriter.Printf("%s\n", logStr)
		}
		GinkgoWriter.Printf("    --- end container logs: %s ---\n", containerName)
	}
}

// findFailedStepContainers returns container names for steps that terminated
// with a non-zero exit code. Tekton names step containers "step-<stepName>".
func findFailedStepContainers(tr *pipeline.TaskRun) []string {
	var failed []string
	for _, step := range tr.Status.Steps {
		if step.Terminated != nil && step.Terminated.ExitCode != 0 {
			containerName := step.Container
			if containerName == "" {
				containerName = "step-" + strings.ReplaceAll(step.Name, " ", "-")
			}
			failed = append(failed, containerName)
		}
	}
	return failed
}

// waitForSucceededPRCount polls until exactly expectedCount PipelineRuns with
// Succeeded=True exist in the namespace. Both overshoot (count > expected)
// and a permanently-failed shortfall fail fast via StopTrying rather than
// waiting out the full timeout — see below.
//
// baselineSucceeded and baselineTotal must be captured by the caller BEFORE
// triggering whatever action is expected to create new PipelineRuns (e.g.
// before mergePaCConfigPRs or the Dockerfile-push trigger). They must not be
// derived from this function's own first poll: PaC webhook delivery is async
// and tenant-wide, not per-component, so by the time this wait's first poll
// runs, the newly-triggered PipelineRun for THIS component may already exist
// and may already have reached a terminal failure — deriving the baseline
// internally would wrongly absorb that fast failure into "pre-existing",
// masking it until the full timeout instead of failing fast on it.
//
// Filters follow the same rules as countPRs: empty pipelineType or
// componentName skips that filter.
func waitForSucceededPRCount(ctx context.Context, fw *framework.Framework, namespace, pipelineType, componentName string, baselineSucceeded, baselineTotal, expectedCount int, timeout, poll time.Duration) {
	GinkgoHelper()

	componentLabel := "appstudio.openshift.io/component"
	displayType := pipelineType
	if displayType == "" {
		displayType = "release"
	}

	listOpts := buildListOpts(namespace, pipelineType, componentName)
	loggedFailures := map[string]bool{}
	newExpected := expectedCount - baselineSucceeded

	Eventually(func() int {
		prList := &pipeline.PipelineRunList{}
		if err := fw.AsKubeAdmin.CommonController.KubeRest().List(
			ctx, prList, listOpts...); err != nil {
			GinkgoWriter.Printf("error listing %s PipelineRuns in %s: %v\n",
				displayType, namespace, err)
			return 0
		}

		succeededCount := 0
		allTerminal := len(prList.Items) > 0
		for i := range prList.Items {
			pr := &prList.Items[i]
			terminal := false
			for _, c := range pr.Status.Conditions {
				if c.Type == "Succeeded" {
					switch c.Status {
					case "True":
						succeededCount++
						terminal = true
					case "False":
						terminal = true
						GinkgoWriter.Printf(
							"FAILED %s PipelineRun %s (component: %s) in %s: %s\n",
							displayType, pr.Name, pr.Labels[componentLabel],
							namespace, c.Message)
						if !loggedFailures[pr.Name] {
							loggedFailures[pr.Name] = true
							logFailedTaskRuns(ctx, fw, namespace, pr.Name)
						}
					}
					break
				}
			}
			if !terminal {
				allTerminal = false
			}
		}

		newSeen := len(prList.Items) - baselineTotal

		GinkgoWriter.Printf("namespace %s: %d/%d %s PipelineRuns succeeded (total: %d)\n",
			namespace, succeededCount, expectedCount, displayType, len(prList.Items))

		// This diagnostic block is kept as a regression alarm for STONEINTG-1732 — it should never fire again.
		// Succeeded count is monotonically non-decreasing, so overshoot can
		// never resolve back to exactly expectedCount — fail fast instead of
		// spinning out the full timeout.
		if succeededCount > expectedCount {
			GinkgoWriter.Printf("OVERSHOOT DETECTED: %d/%d %s PipelineRuns in %s — dumping diagnostics:\n",
				succeededCount, expectedCount, displayType, namespace)
			for i := range prList.Items {
				pr := &prList.Items[i]
				GinkgoWriter.Printf(
					"  PipelineRun: %s | created: %s | component: %s | type: %s | snapshot: %s | event: %s\n",
					pr.Name,
					pr.CreationTimestamp.Format("15:04:05"),
					pr.Labels["appstudio.openshift.io/component"],
					pr.Labels["pipelines.appstudio.openshift.io/type"],
					pr.Labels["appstudio.openshift.io/snapshot"],
					pr.Labels["pipelinesascode.tekton.dev/event-type"],
				)
			}
			StopTrying(fmt.Sprintf(
				"%s PipelineRun overshoot in %s: %d succeeded, expected exactly %d — this can never converge back to expected, see dumped diagnostics",
				displayType, namespace, succeededCount, expectedCount),
			).Now()
		}

		if succeededCount < expectedCount && allTerminal && newSeen >= newExpected {
			StopTrying(fmt.Sprintf(
				"%s PipelineRun(s) for component %q in %s permanently failed: %d/%d succeeded, all %d PipelineRun(s) reached a terminal state (including %d new since this wait started, as expected), none still running",
				displayType, componentName, namespace, succeededCount, expectedCount, len(prList.Items), newSeen),
			).Now()
		}

		return succeededCount
	}, timeout, poll).Should(Equal(expectedCount),
		"expected exactly %d successful %s PipelineRuns in namespace %s",
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
// Release CR counting — release-service deletes completed PipelineRuns from
// the managed namespace, so release success must be verified via Release CRs
// (which persist in the tenant namespace) rather than PipelineRuns.
// ---------------------------------------------------------------------------

// countReleases returns both the released count and the total count
// (regardless of status) of Release CRs in the namespace, from a single
// List call — see countPRs for why both numbers must come from the same
// observation.
func countReleases(ctx context.Context, fw *framework.Framework, namespace string) (released, total int, err error) {
	releases := &releaseapi.ReleaseList{}
	if err := fw.AsKubeAdmin.CommonController.KubeRest().List(
		ctx, releases, client.InNamespace(namespace)); err != nil {
		return 0, 0, fmt.Errorf("listing Releases in %s: %w", namespace, err)
	}
	total = len(releases.Items)
	for i := range releases.Items {
		if releases.Items[i].IsReleased() {
			released++
		}
	}
	return released, total, nil
}

// waitForReleasedCount polls until exactly expectedCount Release CRs with
// Released=True exist in the namespace. baselineReleased and baselineTotal
// must be captured by the caller before triggering whatever action creates
// new Releases — see waitForSucceededPRCount's doc comment for why deriving
// them from this function's own first poll would be wrong.
func waitForReleasedCount(ctx context.Context, fw *framework.Framework, namespace string, baselineReleased, baselineTotal, expectedCount int, timeout, poll time.Duration) {
	GinkgoHelper()

	newExpected := expectedCount - baselineReleased

	Eventually(func() int {
		releases := &releaseapi.ReleaseList{}
		if err := fw.AsKubeAdmin.CommonController.KubeRest().List(
			ctx, releases, client.InNamespace(namespace)); err != nil {
			GinkgoWriter.Printf("error listing Releases in %s: %v\n", namespace, err)
			return 0
		}

		releasedCount := 0
		allTerminal := len(releases.Items) > 0
		for i := range releases.Items {
			r := &releases.Items[i]
			terminal := false
			if r.IsReleased() {
				releasedCount++
				terminal = true
			} else {
				for _, c := range r.Status.Conditions {
					if c.Type == "Released" {
						GinkgoWriter.Printf("Release %s in %s: Released=%s Reason=%s\n",
							r.Name, namespace, c.Status, c.Reason)
						if c.Reason == "Failed" {
							terminal = true
						}
						break
					}
				}
			}
			if !terminal {
				allTerminal = false
			}
		}

		newSeen := len(releases.Items) - baselineTotal

		GinkgoWriter.Printf("namespace %s: %d/%d Releases released (total: %d)\n",
			namespace, releasedCount, expectedCount, len(releases.Items))

		// This diagnostic block is kept as a regression alarm for STONEINTG-1732 — it should never fire again.
		// Released count is monotonically non-decreasing, so overshoot can
		// never resolve back to exactly expectedCount — fail fast instead of
		// spinning out the full timeout.
		if releasedCount > expectedCount {
			GinkgoWriter.Printf("OVERSHOOT DETECTED: %d/%d released Releases in %s — dumping diagnostics:\n",
				releasedCount, expectedCount, namespace)
			for i := range releases.Items {
				r := &releases.Items[i]
				GinkgoWriter.Printf("  Release: %s | created: %s | released: %v\n",
					r.Name, r.CreationTimestamp.Format("15:04:05"), r.IsReleased())
			}
			StopTrying(fmt.Sprintf(
				"Release overshoot in %s: %d released, expected exactly %d — this can never converge back to expected, see dumped diagnostics",
				namespace, releasedCount, expectedCount),
			).Now()
		}

		if releasedCount < expectedCount && allTerminal && newSeen >= newExpected {
			StopTrying(fmt.Sprintf(
				"Release(s) in %s permanently failed: %d/%d released, all %d Release(s) reached a terminal state (including %d new since this wait started, as expected), none still progressing",
				namespace, releasedCount, expectedCount, len(releases.Items), newSeen),
			).Now()
		}

		return releasedCount
	}, timeout, poll).Should(Equal(expectedCount),
		"expected exactly %d released Releases in namespace %s",
		expectedCount, namespace)
}

// ---------------------------------------------------------------------------
// High-level lifecycle helpers
// ---------------------------------------------------------------------------

// pipelineRunBaseCounts holds pre-trigger baseline counts for one component,
// captured by the caller before whatever action is expected to create new
// PipelineRuns. succeeded counts feed expectedCount; total counts (all
// PipelineRuns regardless of status) let waitForSucceededPRCount tell a
// genuinely new attempt apart from pre-existing ones — see its doc comment.
type pipelineRunBaseCounts struct {
	build      int
	test       int
	buildTotal int
	testTotal  int
}

// releaseBaseCounts holds pre-trigger baseline counts for one tenant
// namespace's Release CRs, analogous to pipelineRunBaseCounts.
type releaseBaseCounts struct {
	released int
	total    int
}

// waitForPipelineChains waits for the full pipeline chain (build → test →
// release) to complete for every component across all tenants. Each
// component's chain runs in its own goroutine so that a slow component
// doesn't block faster ones from progressing through subsequent stages.
// Release CRs are waited for after all build/test chains complete,
// since releases may not be per-component.
//
// baseBuildTest provides per-component starting counts keyed by
// "namespace/componentName". baseRelease provides aggregate starting counts
// keyed by tenant namespace. Both must be captured by the caller before
// triggering new PipelineRuns/Releases — see waitForSucceededPRCount's and
// waitForReleasedCount's doc comments for why. Pass nil for both only when
// the namespaces are freshly created and guaranteed to contain nothing yet.
func waitForPipelineChains(ctx context.Context, fw *framework.Framework, tenants []Tenant,
	baseBuildTest map[string]pipelineRunBaseCounts, baseRelease map[string]releaseBaseCounts) {
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
				waitForSucceededPRCount(ctx, fw, tenant.Namespace, "build", component.Name,
					base.build, base.buildTotal, base.build+1, PipelineTimeout, PipelinePoll)

				By(fmt.Sprintf("Waiting for test PipelineRun for %s in %s (base: %d)",
					component.Name, tenant.Namespace, base.test))
				waitForSucceededPRCount(ctx, fw, tenant.Namespace, "test", component.Name,
					base.test, base.testTotal, base.test+1, PipelineTimeout, PipelinePoll)
			}(t, comp)
		}
	}
	wg.Wait()

	logReleaseChainDiagnostics(tenants)

	// Release CRs live in the tenant namespace and persist after release-service
	// cleans up completed PipelineRuns from the managed namespace.
	for _, t := range tenants {
		releaseBase := baseRelease[t.Namespace] // zero-value if nil map or missing key
		expected := releaseBase.released + ComponentsPerTenant
		By(fmt.Sprintf("Waiting for %d released Releases in %s (base: %d)",
			expected, t.Namespace, releaseBase.released))
		waitForReleasedCount(ctx, fw, t.Namespace, releaseBase.released, releaseBase.total,
			expected, ReleaseChainTimeout, ReleaseChainPoll)
	}
}

// triggerBuildsAndVerify pushes commits to each tenant's forked MathWizz
// repo's default branch to trigger new builds via PaC push webhooks, then
// waits for the full pipeline chain (build → integration test → release) to
// complete. Push events (not PRs) are required because integration-service
// only auto-releases Snapshots with push event type.
//
// The method:
//  1. Snapshots current per-component PipelineRun counts and Release CR counts.
//  2. For each tenant: pushes a Dockerfile change per component directly to
//     the default branch (matching PaC .pathChanged() filters).
//  3. Waits for new build and test PipelineRuns per component (parallel).
//  4. Waits for new released Release CRs (aggregate, in tenant namespace).
func triggerBuildsAndVerify(ctx context.Context, fw *framework.Framework, tenants []Tenant) {
	GinkgoHelper()

	By("Snapshotting current per-component PipelineRun counts before triggering")

	initialPerComp := make(map[string]pipelineRunBaseCounts)
	initialRelease := make(map[string]releaseBaseCounts)

	for _, t := range tenants {
		for _, comp := range Components {
			key := t.Namespace + "/" + comp.Name
			buildCount, buildTotal, err := countPRs(ctx, fw, t.Namespace, "build", comp.Name)
			Expect(err).ShouldNot(HaveOccurred(), "baseline build counts for %s", key)
			testCount, testTotal, err := countPRs(ctx, fw, t.Namespace, "test", comp.Name)
			Expect(err).ShouldNot(HaveOccurred(), "baseline test counts for %s", key)
			initialPerComp[key] = pipelineRunBaseCounts{
				build:      buildCount,
				test:       testCount,
				buildTotal: buildTotal,
				testTotal:  testTotal,
			}
			GinkgoWriter.Printf("initial counts for %s: build=%d/%d, test=%d/%d (succeeded/total)\n",
				key, initialPerComp[key].build, initialPerComp[key].buildTotal,
				initialPerComp[key].test, initialPerComp[key].testTotal)
		}
		releaseCount, releaseTotal, err := countReleases(ctx, fw, t.Namespace)
		Expect(err).ShouldNot(HaveOccurred(), "baseline release counts for %s", t.Namespace)
		initialRelease[t.Namespace] = releaseBaseCounts{
			released: releaseCount,
			total:    releaseTotal,
		}
		GinkgoWriter.Printf("initial released count for %s: %d/%d (released/total)\n",
			t.Namespace, initialRelease[t.Namespace].released, initialRelease[t.Namespace].total)
	}

	initialTotalPRs := make(map[string]int)
	for _, t := range tenants {
		allPRs := &pipeline.PipelineRunList{}
		Expect(fw.AsKubeAdmin.CommonController.KubeRest().List(
			ctx, allPRs,
			client.InNamespace(t.Namespace),
		)).Should(Succeed(), "failed to list all PipelineRuns in %s", t.Namespace)
		initialTotalPRs[t.Namespace] = len(allPRs.Items)
		GinkgoWriter.Printf("initial total PipelineRun count in %s: %d\n",
			t.Namespace, len(allPRs.Items))
	}

	ghClient := fw.AsKubeAdmin.HasController.Github

	for _, t := range tenants {
		Expect(t.ForkRepoName).ShouldNot(BeEmpty(),
			"ForkRepoName not set for tenant %s", t.Namespace)

		By(fmt.Sprintf("Pushing Dockerfile changes to %s/%s for tenant %s",
			t.ForkRepoName, MathWizzDefaultBranch, t.Namespace))

		for _, comp := range Components {
			dfPath := path.Join(comp.ContextDir, "Dockerfile")
			dfFile, err := ghClient.GetFile(t.ForkRepoName, dfPath, MathWizzDefaultBranch)
			Expect(err).ShouldNot(HaveOccurred(),
				"failed to get %s from %s in %s", dfPath, MathWizzDefaultBranch, t.ForkRepoName)

			dfContent, err := dfFile.GetContent()
			Expect(err).ShouldNot(HaveOccurred(), "failed to decode %s content", dfPath)

			dfContent += fmt.Sprintf("\n# DR trigger %s %d\n", t.AppName, time.Now().Unix())
			_, err = ghClient.UpdateFile(t.ForkRepoName, dfPath,
				dfContent, MathWizzDefaultBranch, dfFile.GetSHA())
			Expect(err).ShouldNot(HaveOccurred(),
				"failed to update %s on %s in %s", dfPath, MathWizzDefaultBranch, t.ForkRepoName)
		}

		GinkgoWriter.Printf("Pushed Dockerfile changes to %s/%s for tenant %s\n",
			t.ForkRepoName, MathWizzDefaultBranch, t.Namespace)
	}

	By("Verifying PaC webhook delivery — expecting new PipelineRuns within 5 minutes")
	Eventually(func(g Gomega) {
		for _, t := range tenants {
			allPRs := &pipeline.PipelineRunList{}
			err := fw.AsKubeAdmin.CommonController.KubeRest().List(
				ctx, allPRs, client.InNamespace(t.Namespace))
			if err != nil {
				GinkgoWriter.Printf("DIAGNOSTIC: error listing PipelineRuns in %s: %v\n",
					t.Namespace, err)
			}
			g.Expect(err).ShouldNot(HaveOccurred(),
				"failed to list PipelineRuns in %s", t.Namespace)
			newCount := len(allPRs.Items) - initialTotalPRs[t.Namespace]
			GinkgoWriter.Printf("DIAGNOSTIC: PipelineRuns in %s — total: %d, baseline: %d, new: %d\n",
				t.Namespace, len(allPRs.Items), initialTotalPRs[t.Namespace], newCount)
			g.Expect(newCount).Should(BeNumerically(">", 0),
				"no new PipelineRuns in %s after push trigger", t.Namespace)
		}
	}, WebhookDeliveryTimeout, WebhookDeliveryPoll).Should(Succeed(),
		"not all tenants received new PipelineRuns within %v of push triggers — "+
			"PaC webhook delivery is broken post-restore; check PaC controller pods "+
			"in openshift-pipelines namespace and SprayProxy route registration",
		WebhookDeliveryTimeout)

	waitForPipelineChains(ctx, fw, tenants, initialPerComp, initialRelease)
}
