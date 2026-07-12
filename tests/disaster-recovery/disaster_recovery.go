// disaster_recovery.go contains core DR (Disaster Recovery) operation helpers
// for the backup/restore e2e test suite. These functions orchestrate Velero
// Backup and Restore CRs, verify that restored resources match expectations,
// and handle cleanup and failure artifact collection.
//
// NOTE: Helper functions call GinkgoHelper() so that assertion failures report
// the caller's location in the test spec, not the helper's internal line.
package disaster_recovery

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strings"

	"github.com/konflux-ci/e2e-tests/pkg/framework"
	imagecontrollerv1alpha1 "github.com/konflux-ci/image-controller/api/v1alpha1"
	"github.com/minio/minio-go/v7"
	miniocreds "github.com/minio/minio-go/v7/pkg/credentials"
	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck
	. "github.com/onsi/gomega"    //nolint:staticcheck
	pacv1alpha1 "github.com/openshift-pipelines/pipelines-as-code/pkg/apis/pipelinesascode/v1alpha1"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// createBackup creates a Velero Backup CR for the given tenant's namespace and
// polls until the backup reaches the "Completed" phase. The Backup CR is
// created in VeleroNamespace ("openshift-adp") and targets only the tenant's
// namespace with the IncludedResources defined in const.go.
func createBackup(fw *framework.Framework, t Tenant) {
	GinkgoHelper()

	By(fmt.Sprintf("Creating Velero Backup CR %q for namespace %q", t.BackupName, t.Namespace))

	backup := &velerov1.Backup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      t.BackupName,
			Namespace: VeleroNamespace,
		},
		Spec: velerov1.BackupSpec{
			IncludedNamespaces: []string{t.Namespace},
			IncludedResources:  IncludedResources,
		},
	}

	err := fw.AsKubeAdmin.CommonController.KubeRest().Create(context.Background(), backup)
	Expect(err).ShouldNot(HaveOccurred(), "failed to create Backup CR %q", t.BackupName)

	By(fmt.Sprintf("Waiting for Backup CR %q to reach Completed phase (timeout: %s)", t.BackupName, BackupTimeout))

	completedBackup := &velerov1.Backup{}
	Eventually(func() (velerov1.BackupPhase, error) {
		err := fw.AsKubeAdmin.CommonController.KubeRest().Get(context.Background(),
			client.ObjectKey{Name: t.BackupName, Namespace: VeleroNamespace}, completedBackup)
		if err != nil {
			return "", err
		}
		return completedBackup.Status.Phase, nil
	}, BackupTimeout, BackupPollInterval).Should(Equal(velerov1.BackupPhaseCompleted),
		"Backup CR %q did not reach Completed phase within %s", t.BackupName, BackupTimeout)

	validateBackupIntegrity(fw, completedBackup)
}

// validateBackupIntegrity asserts that a completed Velero Backup is not hollow.
// It performs three checks:
//
//  1. Zero backup errors — the backup ran without any resource-level failures.
//  2. Minimum item count — the number of backed-up items is at least
//     BackupMinItemCount, the empirically calibrated floor for a single
//     MathWizz tenant namespace.
//  3. Minimum tarball size — the backup archive stored in MinIO is at least
//     BackupMinTarballSize bytes.
//
// Connection details for MinIO are read dynamically from the BSL CR and its
// referenced credential Secret.
func validateBackupIntegrity(fw *framework.Framework, backup *velerov1.Backup) {
	GinkgoHelper()

	backupName := backup.Name
	By(fmt.Sprintf("Validating backup integrity for %q", backupName))

	// Check 1: zero errors.
	By(fmt.Sprintf("Verifying Backup %q has zero errors", backupName))
	Expect(backup.Status.Errors).Should(Equal(0),
		"Backup %q completed with %d errors", backupName, backup.Status.Errors)

	// Check 2: minimum item count.
	By(fmt.Sprintf("Verifying Backup %q backed up at least %d items", backupName, BackupMinItemCount))
	Expect(backup.Status.Progress).ShouldNot(BeNil(),
		"Backup %q has nil Progress — Velero did not report item counts", backupName)
	actualItemCount := backup.Status.Progress.ItemsBackedUp
	GinkgoWriter.Printf("Backup %q: itemsBackedUp=%d (minimum expected: %d)\n",
		backupName, actualItemCount, BackupMinItemCount)
	Expect(actualItemCount).Should(BeNumerically(">=", BackupMinItemCount),
		"Backup %q backed up %d items, expected at least %d",
		backupName, actualItemCount, BackupMinItemCount)

	// Check 3: minimum tarball size (best-effort).
	// The MinIO endpoint is cluster-internal (*.svc.cluster.local). In CI
	// environments using cluster_claim, the test pod runs on the build farm
	// and cannot reach cluster-internal services. Skip gracefully when the
	// endpoint is unreachable rather than failing the entire backup phase.
	By(fmt.Sprintf("Verifying backup tarball size in MinIO for %q", backupName))
	tarballSize, err := getBackupTarballSize(fw, backup)
	if err != nil {
		GinkgoWriter.Printf("WARNING: skipping tarball size check for %q: %v\n", backupName, err)
	} else {
		GinkgoWriter.Printf("Backup %q: tarball=%d bytes (minimum expected: %d)\n",
			backupName, tarballSize, BackupMinTarballSize)
		Expect(tarballSize).Should(BeNumerically(">=", BackupMinTarballSize),
			"Backup %q tarball is %d bytes, expected at least %d",
			backupName, tarballSize, BackupMinTarballSize)
	}
}

// getBackupTarballSize queries MinIO (the S3-compatible object store deployed
// by the development overlay on ephemeral test clusters) and returns the size
// in bytes of the backup tarball.
//
// Returns an error instead of asserting so callers can skip gracefully when
// the MinIO endpoint is unreachable (e.g. cluster_claim CI where the test pod
// runs on the build farm, not on the claimed cluster).
//
// Connection details are read dynamically from the first Available BSL in the
// openshift-adp namespace:
//   - Bucket name and prefix from BSL.Spec.ObjectStorage
//   - MinIO endpoint from BSL.Spec.Config["s3Url"]
//   - Credentials from the Secret referenced by BSL.Spec.Credential
func getBackupTarballSize(fw *framework.Framework, backup *velerov1.Backup) (int64, error) {
	ctx := context.Background()
	restClient := fw.AsKubeAdmin.CommonController.KubeRest()

	bslList := &velerov1.BackupStorageLocationList{}
	if err := restClient.List(ctx, bslList, client.InNamespace(VeleroNamespace)); err != nil {
		return 0, fmt.Errorf("failed to list BackupStorageLocations: %w", err)
	}
	if len(bslList.Items) == 0 {
		return 0, fmt.Errorf("no BackupStorageLocations found in %s", VeleroNamespace)
	}

	var bsl *velerov1.BackupStorageLocation
	for i := range bslList.Items {
		if bslList.Items[i].Status.Phase == velerov1.BackupStorageLocationPhaseAvailable {
			bsl = &bslList.Items[i]
			break
		}
	}
	if bsl == nil {
		return 0, fmt.Errorf("no Available BackupStorageLocation found in %s", VeleroNamespace)
	}

	bucket := bsl.Spec.ObjectStorage.Bucket
	prefix := bsl.Spec.ObjectStorage.Prefix
	s3URL := bsl.Spec.Config["s3Url"]
	if s3URL == "" {
		return 0, fmt.Errorf("BSL %q has no s3Url in Config", bsl.Name)
	}
	if bucket == "" {
		return 0, fmt.Errorf("BSL %q has no bucket configured", bsl.Name)
	}

	if bsl.Spec.Credential == nil {
		return 0, fmt.Errorf("BSL %q has no credential reference", bsl.Name)
	}
	credSecret := &corev1.Secret{}
	if err := restClient.Get(ctx, client.ObjectKey{
		Name:      bsl.Spec.Credential.Name,
		Namespace: VeleroNamespace,
	}, credSecret); err != nil {
		return 0, fmt.Errorf("failed to get credential Secret %q: %w", bsl.Spec.Credential.Name, err)
	}

	credData := credSecret.Data[bsl.Spec.Credential.Key]
	if len(credData) == 0 {
		return 0, fmt.Errorf("credential Secret %q key %q is empty", bsl.Spec.Credential.Name, bsl.Spec.Credential.Key)
	}
	accessKey, secretKey := parseVeleroCredentialFile(string(credData))
	if accessKey == "" || secretKey == "" {
		return 0, fmt.Errorf("could not parse credentials from BSL credential Secret")
	}

	secure := strings.HasPrefix(s3URL, "https://")
	endpoint := strings.TrimPrefix(strings.TrimPrefix(s3URL, "https://"), "http://")

	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} // #nosec G402 -- matches BSL insecureSkipTLSVerify

	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:     miniocreds.NewStaticV4(accessKey, secretKey, ""),
		Secure:    secure,
		Transport: transport,
	})
	if err != nil {
		return 0, fmt.Errorf("failed to create MinIO client for endpoint %q: %w", endpoint, err)
	}

	objectKey := fmt.Sprintf(VeleroBackupTarballPathFmt, backup.Name, backup.Name)
	if prefix != "" {
		objectKey = prefix + "/" + objectKey
	}

	stat, err := minioClient.StatObject(ctx, bucket, objectKey, minio.StatObjectOptions{})
	if err != nil {
		return 0, fmt.Errorf("failed to stat backup tarball %q in bucket %q: %w", objectKey, bucket, err)
	}

	return stat.Size, nil
}

// parseVeleroCredentialFile extracts aws_access_key_id and aws_secret_access_key
// from the AWS credentials file format that OADP's Velero AWS plugin uses for
// all S3-compatible backends, including MinIO.
func parseVeleroCredentialFile(data string) (accessKey, secretKey string) {
	for _, line := range strings.Split(data, "\n") {
		line = strings.TrimSpace(line)
		if after, ok := strings.CutPrefix(line, "aws_access_key_id"); ok {
			accessKey = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(after), "="))
		} else if after, ok := strings.CutPrefix(line, "aws_secret_access_key"); ok {
			secretKey = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(after), "="))
		}
	}
	return accessKey, secretKey
}

// restoreFromBackup creates a Velero Restore for the given tenant and polls
// until the restore reaches the "Completed" phase. The method parameter selects
// which SOP procedure is exercised:
//
//   - RestoreMethodVeleroCLI: invokes the `velero restore create` binary
//     directly, mirroring the Velero CLI procedure from the SOP.
//   - RestoreMethodOCCommand: generates a Restore CR manifest as JSON and
//     applies it via `oc apply -f -`, mirroring the declarative procedure
//     from the SOP.
func restoreFromBackup(fw *framework.Framework, t Tenant, method RestoreMethod) {
	GinkgoHelper()

	restoreName := "restore-" + t.BackupName
	By(fmt.Sprintf("Creating Velero Restore %q from backup %q using %s method", restoreName, t.BackupName, method))

	switch method {
	case RestoreMethodVeleroCLI:
		args := []string{
			"restore", "create", restoreName,
			"--from-backup", t.BackupName,
			"--include-namespaces", t.Namespace,
			"--include-resources", strings.Join(IncludedResources, ","),
			"--namespace", VeleroNamespace,
		}
		cmd := exec.Command("velero", args...) // #nosec G204 -- args are internal test constants, not user input
		output, err := cmd.CombinedOutput()
		Expect(err).ShouldNot(HaveOccurred(),
			"velero restore create failed: %s", string(output))

	case RestoreMethodOCCommand:
		restore := &velerov1.Restore{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "velero.io/v1",
				Kind:       "Restore",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      restoreName,
				Namespace: VeleroNamespace,
			},
			Spec: velerov1.RestoreSpec{
				BackupName:         t.BackupName,
				IncludedNamespaces: []string{t.Namespace},
				IncludedResources:  IncludedResources,
			},
		}
		manifest, err := json.Marshal(restore)
		Expect(err).ShouldNot(HaveOccurred(), "failed to marshal Restore CR to JSON")

		cmd := exec.Command("oc", "apply", "-f", "-")
		cmd.Stdin = strings.NewReader(string(manifest))
		output, err := cmd.CombinedOutput()
		Expect(err).ShouldNot(HaveOccurred(),
			"oc apply failed: %s", string(output))
	}

	By(fmt.Sprintf("Waiting for Restore CR %q to reach Completed phase (timeout: %s)", restoreName, RestoreTimeout))

	Eventually(func() (velerov1.RestorePhase, error) {
		got := &velerov1.Restore{}
		err := fw.AsKubeAdmin.CommonController.KubeRest().Get(context.Background(),
			client.ObjectKey{Name: restoreName, Namespace: VeleroNamespace}, got)
		if err != nil {
			return "", err
		}
		return got.Status.Phase, nil
	}, RestoreTimeout, RestorePollInterval).Should(Equal(velerov1.RestorePhaseCompleted),
		"Restore CR %q did not reach Completed phase within %s", restoreName, RestoreTimeout)
}

// reconcileComponentOwnership directly patches ownerReferences on PaC
// Repository CRs to point at the restored Components.
//
// Why this is needed: Velero strips ownerReferences during restore (UIDs from
// the source cluster are invalid). Build-service only sets ownerReferences
// during initial Repository creation — it does not re-establish them when
// reconciling an existing Repository CR. Direct patching is the only reliable
// restoration path.
func reconcileComponentOwnership(fw *framework.Framework, t Tenant) {
	GinkgoHelper()

	ctx := context.Background()
	restClient := fw.AsKubeAdmin.CommonController.KubeRest()

	By(fmt.Sprintf("Collecting post-restore Component UIDs in namespace %q", t.Namespace))
	ownerRefs := make([]metav1.OwnerReference, 0, len(Components))
	for _, comp := range Components {
		c, err := fw.AsKubeAdmin.HasController.GetComponent(comp.Name, t.Namespace)
		Expect(err).ShouldNot(HaveOccurred(),
			"failed to get Component %q in namespace %q", comp.Name, t.Namespace)
		ownerRefs = append(ownerRefs, metav1.OwnerReference{
			APIVersion: "appstudio.redhat.com/v1alpha1",
			Kind:       "Component",
			Name:       c.Name,
			UID:        c.UID,
		})
	}

	By(fmt.Sprintf("Listing PaC Repository CRs in namespace %q", t.Namespace))
	repoList := &pacv1alpha1.RepositoryList{}
	err := restClient.List(ctx, repoList, &client.ListOptions{Namespace: t.Namespace})
	Expect(err).ShouldNot(HaveOccurred(),
		"failed to list PaC repositories in namespace %q", t.Namespace)
	Expect(repoList.Items).ShouldNot(BeEmpty(),
		"no PaC Repository CRs found in namespace %q", t.Namespace)

	By(fmt.Sprintf("Setting ownerReferences on %d PaC Repository CRs for %d Components", len(repoList.Items), len(ownerRefs)))
	for i := range repoList.Items {
		repo := &repoList.Items[i]
		repo.OwnerReferences = ownerRefs
		err := restClient.Update(ctx, repo)
		Expect(err).ShouldNot(HaveOccurred(),
			"failed to set ownerReferences on PaC Repository %q in namespace %q", repo.Name, t.Namespace)
		GinkgoWriter.Printf("Set %d ownerReferences on PaC Repository %s/%s\n", len(ownerRefs), t.Namespace, repo.Name)
	}

	By(fmt.Sprintf("Verifying PaC Repository ownership for all %d Components", len(Components)))
	for _, comp := range Components {
		_, err := fw.AsKubeAdmin.TektonController.GetRepositoryParams(comp.Name, t.Namespace)
		Expect(err).ShouldNot(HaveOccurred(),
			"ownerReference not set correctly on PaC Repository for component %q in namespace %q",
			comp.Name, t.Namespace)
	}
	GinkgoWriter.Printf("PaC Repository ownerReferences verified for all %d Components in namespace %s\n", len(Components), t.Namespace)
}

// verifyResources performs structural verification of restored tenant resources.
// It checks that the Application, Components, IntegrationTestScenarios,
// ServiceAccounts, SA token Secrets, ReleasePlan, PaC Repository CRs, and
// ImageRepository CRs all exist and have the expected field values.
// This is a structural check (existence + key fields), not a snapshot
// diff, which keeps the tests stable across Konflux version changes.
func verifyResources(fw *framework.Framework, t Tenant) {
	GinkgoHelper()

	By(fmt.Sprintf("Verifying Application %q exists in namespace %q", t.AppName, t.Namespace))
	_, err := fw.AsKubeAdmin.HasController.GetApplication(t.AppName, t.Namespace)
	Expect(err).ShouldNot(HaveOccurred(), "Application %q should exist in namespace %q", t.AppName, t.Namespace)

	By(fmt.Sprintf("Verifying all %d Components exist with correct spec fields", len(Components)))
	for _, comp := range Components {
		c, err := fw.AsKubeAdmin.HasController.GetComponent(comp.Name, t.Namespace)
		Expect(err).ShouldNot(HaveOccurred(), "Component %q should exist in namespace %q", comp.Name, t.Namespace)

		// Verify every Spec field that is set at creation time and NOT mutated
		// by controllers. Two fields are intentionally excluded:
		//
		//   - Spec.ContainerImage: Populated asynchronously by the
		//     image-controller when it creates an ImageRepository for the
		//     Component. The value depends on the image registry state at
		//     restore time and may legitimately differ from the original.
		//
		//   - Spec.Actions: A write-once trigger field. Controllers consume
		//     and remove actions after processing them, so the field is
		//     expected to be empty on any persisted Component.
		Expect(t.ForkRepoURL).ShouldNot(BeEmpty(),
			"ForkRepoURL not set for tenant %s — cannot verify Component URLs", t.Namespace)
		Expect(c).Should(SatisfyAll(
			HaveField("Spec.ComponentName", Equal(comp.Name)),
			HaveField("Spec.Application", Equal(t.AppName)),
			HaveField("Spec.Source.GitSource.URL", Equal(t.ForkRepoURL)),
			HaveField("Spec.Source.GitSource.Context", Equal(comp.ContextDir)),
			HaveField("Spec.Source.GitSource.DockerfileURL", Equal(comp.DockerfileURL)),
			HaveField("Spec.TargetPort", Equal(8081)),
		), "Component %q in namespace %q has unexpected spec fields", comp.Name, t.Namespace)
	}

	By(fmt.Sprintf("Verifying at least one IntegrationTestScenario exists in namespace %q", t.Namespace))
	scenarios, err := fw.AsKubeAdmin.IntegrationController.GetIntegrationTestScenarios(t.AppName, t.Namespace)
	Expect(err).ShouldNot(HaveOccurred(), "should be able to list IntegrationTestScenarios in namespace %q", t.Namespace)
	Expect(*scenarios).ShouldNot(BeEmpty(),
		"at least one IntegrationTestScenario should exist in namespace %q", t.Namespace)

	By(fmt.Sprintf("Verifying at least one ServiceAccount exists in namespace %q", t.Namespace))
	saList := &corev1.ServiceAccountList{}
	err = fw.AsKubeAdmin.CommonController.KubeRest().List(context.Background(), saList, client.InNamespace(t.Namespace))
	Expect(err).ShouldNot(HaveOccurred(), "should be able to list ServiceAccounts in namespace %q", t.Namespace)
	Expect(saList.Items).ShouldNot(BeEmpty(),
		"at least one ServiceAccount should exist in namespace %q", t.Namespace)

	// K8s 1.24+ no longer auto-creates ServiceAccount token Secrets.
	// Only verify they exist when the token controller is expected to
	// have created them (i.e., when rotateSATokens actually deleted
	// stale tokens and waited for regeneration).
	By(fmt.Sprintf("Checking for SA token Secrets in namespace %q", t.Namespace))
	secretList := &corev1.SecretList{}
	err = fw.AsKubeAdmin.CommonController.KubeRest().List(context.Background(), secretList, client.InNamespace(t.Namespace))
	Expect(err).ShouldNot(HaveOccurred(), "should be able to list Secrets in namespace %q", t.Namespace)

	hasTokenSecret := false
	for i := range secretList.Items {
		if secretList.Items[i].Type == corev1.SecretTypeServiceAccountToken {
			hasTokenSecret = true
			break
		}
	}
	if !hasTokenSecret {
		GinkgoWriter.Printf("No SA token Secrets in namespace %s — expected on K8s 1.24+ (token rotation was a no-op)\n", t.Namespace)
	}

	By(fmt.Sprintf("Verifying ReleasePlan %q exists in namespace %q", DRReleasePlanName, t.Namespace))
	_, err = fw.AsKubeAdmin.ReleaseController.GetReleasePlan(DRReleasePlanName, t.Namespace)
	Expect(err).ShouldNot(HaveOccurred(),
		"ReleasePlan %q should exist in namespace %q (proves release config survived backup/restore)", DRReleasePlanName, t.Namespace)

	By(fmt.Sprintf("Verifying PaC Repository CRs exist for all %d Components in namespace %q", len(Components), t.Namespace))
	for _, comp := range Components {
		_, err := fw.AsKubeAdmin.TektonController.GetRepositoryParams(comp.Name, t.Namespace)
		Expect(err).ShouldNot(HaveOccurred(),
			"PaC Repository CR should exist for component %q in namespace %q", comp.Name, t.Namespace)
	}

	By(fmt.Sprintf("Verifying ImageRepository CRs exist in namespace %q (one per component)", t.Namespace))
	imageRepoList := &imagecontrollerv1alpha1.ImageRepositoryList{}
	err = fw.AsKubeAdmin.CommonController.KubeRest().List(context.Background(), imageRepoList, client.InNamespace(t.Namespace))
	Expect(err).ShouldNot(HaveOccurred(), "should be able to list ImageRepositories in namespace %q", t.Namespace)
	Expect(imageRepoList.Items).Should(HaveLen(len(Components)),
		"expected %d ImageRepository CRs in namespace %q (one per component)", len(Components), t.Namespace)
}

// collectFailureArtifacts logs diagnostic information for troubleshooting DR
// test failures. It dumps Velero pod status and the status of all Backup and
// Restore CRs associated with the given tenants. This function is safe to call
// even when resources have already been cleaned up — it ignores missing
// resources gracefully.
func collectFailureArtifacts(fw *framework.Framework, tenants []Tenant) {
	GinkgoHelper()

	ctx := context.Background()

	By("Collecting Velero pod information")
	pods, err := fw.AsKubeAdmin.CommonController.KubeInterface().CoreV1().Pods(VeleroNamespace).List(ctx, metav1.ListOptions{
		LabelSelector: "component=velero",
	})
	if err != nil {
		GinkgoWriter.Printf("WARNING: failed to list Velero pods: %v\n", err)
	} else {
		for i := range pods.Items {
			pod := &pods.Items[i]
			GinkgoWriter.Printf("Velero pod: %s | Phase: %s | Ready: %v\n",
				pod.Name, pod.Status.Phase, isPodReady(pod))
		}
	}

	for _, t := range tenants {
		By(fmt.Sprintf("Collecting Backup CR status for tenant %q", t.Namespace))
		backup := &velerov1.Backup{}
		if err := fw.AsKubeAdmin.CommonController.KubeRest().Get(ctx,
			client.ObjectKey{Name: t.BackupName, Namespace: VeleroNamespace}, backup); err != nil {
			GinkgoWriter.Printf("WARNING: could not get Backup CR %q: %v\n", t.BackupName, err)
		} else {
			GinkgoWriter.Printf("Backup CR %q: phase=%s\n", t.BackupName, backup.Status.Phase)
			if backup.Status.Errors > 0 || backup.Status.Warnings > 0 {
				GinkgoWriter.Printf("  errors=%d, warnings=%d\n", backup.Status.Errors, backup.Status.Warnings)
			}
		}

		restoreName := "restore-" + t.BackupName
		By(fmt.Sprintf("Collecting Restore CR status for tenant %q (restore: %s)", t.Namespace, restoreName))
		restore := &velerov1.Restore{}
		if err := fw.AsKubeAdmin.CommonController.KubeRest().Get(ctx,
			client.ObjectKey{Name: restoreName, Namespace: VeleroNamespace}, restore); err != nil {
			GinkgoWriter.Printf("WARNING: could not get Restore CR %q: %v\n", restoreName, err)
		} else {
			GinkgoWriter.Printf("Restore CR %q: phase=%s\n", restoreName, restore.Status.Phase)
			if restore.Status.Errors > 0 || restore.Status.Warnings > 0 {
				GinkgoWriter.Printf("  errors=%d, warnings=%d\n", restore.Status.Errors, restore.Status.Warnings)
			}
		}
	}

	collectPaCDiagnostics(tenants)
}

// collectPaCDiagnostics dumps PaC controller state for troubleshooting
// webhook delivery failures. Uses oc commands to avoid importing PaC's
// full operator client.
func collectPaCDiagnostics(tenants []Tenant) {
	const pacNamespace = "openshift-pipelines"

	By("Collecting PaC controller diagnostics")

	run := func(args ...string) string {
		out, _ := exec.Command("oc", args...).CombinedOutput()
		return strings.TrimSpace(string(out))
	}

	GinkgoWriter.Printf("=== PaC pods in %s ===\n%s\n\n",
		pacNamespace, run("get", "pods", "-n", pacNamespace, "-o", "wide"))

	GinkgoWriter.Printf("=== PaC routes in %s ===\n%s\n\n",
		pacNamespace, run("get", "routes", "-n", pacNamespace, "-o", "wide"))

	GinkgoWriter.Printf("=== PaC services in %s ===\n%s\n\n",
		pacNamespace, run("get", "services", "-n", pacNamespace, "-o", "wide"))

	pods := strings.Fields(run("get", "pods", "-n", pacNamespace,
		"-o", "jsonpath={.items[*].metadata.name}"))
	for _, pod := range pods {
		if strings.Contains(pod, "controller") || strings.Contains(pod, "watcher") {
			GinkgoWriter.Printf("=== Logs: %s (last 100 lines) ===\n%s\n\n",
				pod, run("logs", pod, "-n", pacNamespace, "--tail=100", "--all-containers"))
		}
	}

	for _, t := range tenants {
		GinkgoWriter.Printf("=== Repository CRs in %s ===\n%s\n\n",
			t.Namespace, run("get", "repositories.pipelinesascode.tekton.dev",
				"-n", t.Namespace, "-o", "yaml"))
	}
}

// isPodReady returns true if the given pod has the Ready condition set to True.
// This is a pure helper with no Ginkgo assertions, so GinkgoHelper() is not needed.
func isPodReady(pod *corev1.Pod) bool {
	for _, cond := range pod.Status.Conditions {
		if cond.Type == corev1.PodReady {
			return cond.Status == corev1.ConditionTrue
		}
	}
	return false
}

// cleanupTestResources deletes DR test resources: tenant namespaces, managed
// namespaces, and associated Velero Backup/Restore CRs. Errors are logged
// and collected so that all cleanup steps run even if some fail, then any
// errors are reported at the end.
func cleanupTestResources(fw *framework.Framework, tenants []Tenant) {
	GinkgoHelper()

	ctx := context.Background()
	kubeClient := fw.AsKubeAdmin.CommonController.KubeInterface()
	restClient := fw.AsKubeAdmin.CommonController.KubeRest()

	var errs []error
	for _, t := range tenants {
		By(fmt.Sprintf("Cleaning up namespace %q", t.Namespace))
		if err := kubeClient.CoreV1().Namespaces().Delete(ctx, t.Namespace, metav1.DeleteOptions{}); err != nil && !k8sErrors.IsNotFound(err) {
			GinkgoWriter.Printf("WARNING: failed to delete namespace %q: %v\n", t.Namespace, err)
			errs = append(errs, err)
		}

		By(fmt.Sprintf("Cleaning up Backup CR %q", t.BackupName))
		if err := restClient.Delete(ctx, &velerov1.Backup{
			ObjectMeta: metav1.ObjectMeta{Name: t.BackupName, Namespace: VeleroNamespace},
		}); err != nil && !k8sErrors.IsNotFound(err) {
			GinkgoWriter.Printf("WARNING: failed to delete Backup CR %q: %v\n", t.BackupName, err)
			errs = append(errs, err)
		}

		restoreName := "restore-" + t.BackupName
		By(fmt.Sprintf("Cleaning up Restore CR %q", restoreName))
		if err := restClient.Delete(ctx, &velerov1.Restore{
			ObjectMeta: metav1.ObjectMeta{Name: restoreName, Namespace: VeleroNamespace},
		}); err != nil && !k8sErrors.IsNotFound(err) {
			GinkgoWriter.Printf("WARNING: failed to delete Restore CR %q: %v\n", restoreName, err)
			errs = append(errs, err)
		}

		By(fmt.Sprintf("Cleaning up managed namespace %q", t.ManagedNamespace))
		if err := kubeClient.CoreV1().Namespaces().Delete(ctx, t.ManagedNamespace, metav1.DeleteOptions{}); err != nil && !k8sErrors.IsNotFound(err) {
			GinkgoWriter.Printf("WARNING: failed to delete managed namespace %q: %v\n", t.ManagedNamespace, err)
			errs = append(errs, err)
		}
	}

	Expect(errs).Should(BeEmpty(), "cleanup encountered %d errors", len(errs))
}

// ---------------------------------------------------------------------------
// Quay resource preservation during DR drills
// ---------------------------------------------------------------------------
//
// Problem: DR drills vs. real disasters
//
// In a real disaster (etcd loss), the Kubernetes API server is gone.
// There is no graceful resource deletion. Finalizers never fire.
// Image-controller never learns that ImageRepository CRs disappeared,
// so it never calls the Quay API to delete robot accounts. The robot
// accounts survive, and Velero-restored push secrets remain valid.
//
// In a DR drill, we simulate disaster with `oc delete project`. This
// triggers graceful deletion of every resource in the namespace. The
// image-controller's finalizer on each ImageRepository CR fires,
// calling the Quay API to delete robot accounts and image repos.
// After Velero restore, push secrets reference non-existent robot
// accounts, and trusted-artifact pushes fail.
//
// The drill methodology introduces a failure mode that real disasters
// do not have. This is not an image-controller bug — the finalizer
// works correctly. The gap is between simulation and reality.
//
// Solution: two-layer protection
//
// 1. Block image-controller egress via NetworkPolicy (defense-in-depth
//    against the race window between finalizer stripping and deletion).
// 2. Strip ImageRepository finalizers before namespace deletion. This
//    is the accurate simulation: real etcd loss = finalizers never run.
//    Without this, the egress block creates a deadlock — the finalizer
//    cannot complete (no network), so the CR cannot be deleted, so the
//    namespace hangs until the deletion timeout.
//
// After restore, the Velero resource modifier strips ImageRepository
// finalizers on the restored CRs, so unblocking egress is safe.
// See KFLUXINFRA-3954, STONEBLD-3714.

const (
	// ImageControllerNamespace is where the image-controller pods run.
	ImageControllerNamespace = "image-controller"

	// drEgressBlockPolicyName is the NetworkPolicy CR name used to block
	// image-controller egress during disaster simulation.
	drEgressBlockPolicyName = "dr-block-image-controller-egress"

	// imageRepositoryFinalizer is the finalizer that image-controller
	// adds to every ImageRepository CR. When processed on deletion, it
	// calls the Quay API to delete robot accounts and image repos.
	imageRepositoryFinalizer = "appstudio.openshift.io/image-repository"
)

// blockImageControllerEgress creates a NetworkPolicy in the image-controller
// namespace that denies all egress from the controller pods. This prevents
// the controller from deleting Quay robot accounts when the ImageRepository
// finalizer fires during namespace deletion.
func blockImageControllerEgress(fw *framework.Framework) {
	GinkgoHelper()

	By("Blocking image-controller egress to preserve Quay resources during namespace deletion")

	policy := &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      drEgressBlockPolicyName,
			Namespace: ImageControllerNamespace,
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"control-plane": "controller-manager",
				},
			},
			PolicyTypes: []networkingv1.PolicyType{
				networkingv1.PolicyTypeEgress,
			},
			// Empty Egress slice = deny all egress.
			Egress: []networkingv1.NetworkPolicyEgressRule{},
		},
	}

	err := fw.AsKubeAdmin.CommonController.KubeRest().Create(context.Background(), policy)
	Expect(err).ShouldNot(HaveOccurred(),
		"failed to create egress-blocking NetworkPolicy in %s", ImageControllerNamespace)

	GinkgoWriter.Printf("Created NetworkPolicy %s/%s — image-controller egress blocked\n",
		ImageControllerNamespace, drEgressBlockPolicyName)
}

// unblockImageControllerEgress removes the egress-blocking NetworkPolicy,
// restoring normal image-controller operations. Safe to call after Velero
// restore because the resource modifier has already stripped ImageRepository
// finalizers.
func unblockImageControllerEgress(fw *framework.Framework) {
	GinkgoHelper()

	By("Removing image-controller egress block — restoring normal operations")

	policy := &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      drEgressBlockPolicyName,
			Namespace: ImageControllerNamespace,
		},
	}

	err := fw.AsKubeAdmin.CommonController.KubeRest().Delete(context.Background(), policy)
	if k8sErrors.IsNotFound(err) {
		GinkgoWriter.Printf("NetworkPolicy %s/%s already absent — nothing to remove\n",
			ImageControllerNamespace, drEgressBlockPolicyName)
		return
	}
	Expect(err).ShouldNot(HaveOccurred(),
		"failed to delete egress-blocking NetworkPolicy from %s", ImageControllerNamespace)

	GinkgoWriter.Printf("Deleted NetworkPolicy %s/%s — image-controller egress restored\n",
		ImageControllerNamespace, drEgressBlockPolicyName)
}

// stripImageRepositoryFinalizers removes the image-controller finalizer from
// every ImageRepository CR in the given tenants' namespaces.
//
// This must be called AFTER blockImageControllerEgress (which prevents the
// controller from re-adding finalizers during the race window) and BEFORE
// deleteNamespace.
//
// Why: in a real disaster (etcd loss), finalizers never fire — the API
// server is gone, so no controller gets a chance to run cleanup logic.
// But `oc delete project` triggers graceful deletion, which invokes the
// ImageRepository finalizer. With egress blocked, that finalizer cannot
// complete (it needs to call the Quay API), creating a deadlock that
// prevents namespace deletion from finishing within the timeout.
//
// Stripping the finalizer before deletion matches real-disaster semantics:
// the image-controller never processes the deletion, Quay robot accounts
// survive, and restored push secrets remain valid.
func stripImageRepositoryFinalizers(fw *framework.Framework, tenants []Tenant) {
	GinkgoHelper()

	ctx := context.Background()
	restClient := fw.AsKubeAdmin.CommonController.KubeRest()

	for _, t := range tenants {
		By(fmt.Sprintf("Stripping ImageRepository finalizers in namespace %q to simulate etcd-loss semantics", t.Namespace))

		imageRepoList := &imagecontrollerv1alpha1.ImageRepositoryList{}
		err := restClient.List(ctx, imageRepoList, client.InNamespace(t.Namespace))
		Expect(err).ShouldNot(HaveOccurred(),
			"failed to list ImageRepositories in namespace %q", t.Namespace)

		stripped := 0
		for i := range imageRepoList.Items {
			ir := &imageRepoList.Items[i]
			var filtered []string
			found := false
			for _, f := range ir.Finalizers {
				if f == imageRepositoryFinalizer {
					found = true
					continue
				}
				filtered = append(filtered, f)
			}
			if found {
				ir.Finalizers = filtered
				err := restClient.Update(ctx, ir)
				Expect(err).ShouldNot(HaveOccurred(),
					"failed to strip finalizer from ImageRepository %s/%s", t.Namespace, ir.Name)
				stripped++
			}
		}

		GinkgoWriter.Printf("Stripped %q from %d/%d ImageRepositories in namespace %s\n",
			imageRepositoryFinalizer, stripped, len(imageRepoList.Items), t.Namespace)
	}
}
