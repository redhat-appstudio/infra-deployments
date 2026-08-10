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
	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck
	. "github.com/onsi/gomega"    //nolint:staticcheck
	pipeline "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const podLogTailLines int64 = 80

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
func countSucceededPRs(ctx context.Context, fw *framework.Framework, namespace, pipelineType, componentName string) int {
	listOpts := buildListOpts(namespace, pipelineType, componentName)

	prList := &pipeline.PipelineRunList{}
	if err := fw.AsKubeAdmin.CommonController.KubeRest().List(
		ctx, prList, listOpts...); err != nil {
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
	tailLines := podLogTailLines

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

// waitForSucceededPRCount polls until at least expectedCount PipelineRuns with
// Succeeded=True exist in the namespace. Overshoot (count > expected) is
// tolerated with diagnostic logging — see TODO below.
//
// Filters follow the same rules as countSucceededPRs: empty pipelineType or
// componentName skips that filter.
func waitForSucceededPRCount(ctx context.Context, fw *framework.Framework, namespace, pipelineType, componentName string, expectedCount int, timeout, poll time.Duration) {
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
			ctx, prList, listOpts...); err != nil {
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
							logFailedTaskRuns(ctx, fw, namespace, pr.Name)
						}
					}
					break
				}
			}
		}

		GinkgoWriter.Printf("namespace %s: %d/%d %s PipelineRuns succeeded (total: %d)\n",
			namespace, succeededCount, expectedCount, displayType, len(prList.Items))

		// TODO: integration-service has a crash-recovery bug where its PipelineRun
		// dedup check relies on annotation state, not cluster state. A controller
		// restart between PipelineRun creation and annotation write produces
		// duplicates. DR amplifies this because ArgoCD resyncs restart pods.
		// File bug against konflux-ci/integration-service; revert to Equal once fixed.
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
		}

		return succeededCount
	}, timeout, poll).Should(BeNumerically(">=", expectedCount),
		"expected at least %d successful %s PipelineRuns in namespace %s",
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
func waitForPipelineChains(ctx context.Context, fw *framework.Framework, tenants []Tenant,
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
				waitForSucceededPRCount(ctx, fw, tenant.Namespace, "build", component.Name,
					base.build+1, PipelineTimeout, PipelinePoll)

				By(fmt.Sprintf("Waiting for test PipelineRun for %s in %s (base: %d)",
					component.Name, tenant.Namespace, base.test))
				waitForSucceededPRCount(ctx, fw, tenant.Namespace, "test", component.Name,
					base.test+1, PipelineTimeout, PipelinePoll)
			}(t, comp)
		}
	}
	wg.Wait()

	logReleaseChainDiagnostics(tenants)

	// Release PipelineRuns run in the managed namespace and may not map 1:1
	// to components, so wait for them in aggregate after all builds/tests pass.
	for _, t := range tenants {
		releaseBase := baseRelease[t.ManagedNamespace] // zero if nil map or missing key
		expected := releaseBase + ComponentsPerTenant
		By(fmt.Sprintf("Waiting for %d release PipelineRuns in %s (base: %d)",
			expected, t.ManagedNamespace, releaseBase))
		waitForSucceededPRCount(ctx, fw, t.ManagedNamespace, "", "", expected,
			ReleaseChainTimeout, ReleaseChainPoll)
	}
}

// triggerBuildsAndVerify pushes commits to each tenant's forked MathWizz
// repo's default branch to trigger new builds via PaC push webhooks, then
// waits for the full pipeline chain (build → integration test → release) to
// complete. Push events (not PRs) are required because integration-service
// only auto-releases Snapshots with push event type.
//
// The method:
//  1. Snapshots current per-component PipelineRun counts.
//  2. For each tenant: pushes a Dockerfile change per component directly to
//     the default branch (matching PaC .pathChanged() filters).
//  3. Waits for new build and test PipelineRuns per component (parallel).
//  4. Waits for new release PipelineRuns (aggregate).
func triggerBuildsAndVerify(ctx context.Context, fw *framework.Framework, tenants []Tenant) {
	GinkgoHelper()

	By("Snapshotting current per-component PipelineRun counts before triggering")

	initialPerComp := make(map[string]pipelineRunBaseCounts)
	initialRelease := make(map[string]int)

	for _, t := range tenants {
		for _, comp := range Components {
			key := t.Namespace + "/" + comp.Name
			initialPerComp[key] = pipelineRunBaseCounts{
				build: countSucceededPRs(ctx, fw, t.Namespace, "build", comp.Name),
				test:  countSucceededPRs(ctx, fw, t.Namespace, "test", comp.Name),
			}
			GinkgoWriter.Printf("initial counts for %s: build=%d, test=%d\n",
				key, initialPerComp[key].build, initialPerComp[key].test)
		}
		initialRelease[t.ManagedNamespace] = countSucceededPRs(ctx, fw, t.ManagedNamespace, "", "")
		GinkgoWriter.Printf("initial release count for %s: %d\n",
			t.ManagedNamespace, initialRelease[t.ManagedNamespace])
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
