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
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"time"

	appservice "github.com/konflux-ci/application-api/api/v1alpha1"
	"github.com/konflux-ci/e2e-tests/pkg/constants"
	"github.com/konflux-ci/e2e-tests/pkg/framework"
	imagecontrollerv1alpha1 "github.com/konflux-ci/image-controller/api/v1alpha1"
	releaseapi "github.com/konflux-ci/release-service/api/v1alpha1"
	"github.com/minio/minio-go/v7"
	miniocreds "github.com/minio/minio-go/v7/pkg/credentials"
	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck
	. "github.com/onsi/gomega"    //nolint:staticcheck
	pacv1alpha1 "github.com/openshift-pipelines/pipelines-as-code/pkg/apis/pipelinesascode/v1alpha1"
	pipeline "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	corev1 "k8s.io/api/core/v1"
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

// verifyPaCRepositories checks that PaC Repository CRs exist post-restore
// and reference the correct git URL. Build-service creates Repository CRs
// using URL-derived names and may share a single Repository across all
// Components pointing at the same monorepo. OwnerReference patching is not
// needed — build-service uses URL-based lookup.
func verifyPaCRepositories(fw *framework.Framework, t Tenant) {
	GinkgoHelper()

	ctx := context.Background()
	restClient := fw.AsKubeAdmin.CommonController.KubeRest()

	By(fmt.Sprintf("Listing PaC Repository CRs in namespace %q", t.Namespace))
	repoList := &pacv1alpha1.RepositoryList{}
	err := restClient.List(ctx, repoList, &client.ListOptions{Namespace: t.Namespace})
	Expect(err).ShouldNot(HaveOccurred(),
		"failed to list PaC repositories in namespace %q", t.Namespace)
	Expect(repoList.Items).ShouldNot(BeEmpty(),
		"no PaC Repository CRs found in namespace %q after restore", t.Namespace)

	By(fmt.Sprintf("Verifying %d PaC Repository CRs reference tenant fork URL", len(repoList.Items)))
	Expect(t.ForkRepoURL).ShouldNot(BeEmpty(), "tenant %s ForkRepoURL must be set", t.Namespace)
	for i := range repoList.Items {
		repo := &repoList.Items[i]
		Expect(repo.Spec.URL).Should(Equal(t.ForkRepoURL),
			"PaC Repository %s/%s spec.url should match tenant fork URL", t.Namespace, repo.Name)
		GinkgoWriter.Printf("PaC Repository %s/%s: url=%s, pipelinerun_status entries=%d\n",
			t.Namespace, repo.Name, repo.Spec.URL, len(repo.Status))
	}
}

// verifyResources performs structural verification of restored tenant resources.
// It checks that the Application, Components, IntegrationTestScenarios,
// ServiceAccounts, SA token Secrets, ReleasePlan, and ImageRepository CRs
// all exist and have the expected field values. PaC Repository verification
// is handled upstream by verifyPaCRepositories.
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
			HaveField("Spec.TargetPort", Equal(MathWizzDefaultTargetPort)),
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
	_, rpErr := fw.AsKubeAdmin.ReleaseController.GetReleasePlan(DRReleasePlanName, t.Namespace)
	Expect(rpErr).ShouldNot(HaveOccurred(),
		"ReleasePlan %q should exist in namespace %q (proves release config survived backup/restore)", DRReleasePlanName, t.Namespace)

	By(fmt.Sprintf("Verifying ImageRepository CRs exist in namespace %q (one per component)", t.Namespace))
	imageRepoList := &imagecontrollerv1alpha1.ImageRepositoryList{}
	err = fw.AsKubeAdmin.CommonController.KubeRest().List(context.Background(), imageRepoList, client.InNamespace(t.Namespace))
	Expect(err).ShouldNot(HaveOccurred(), "should be able to list ImageRepositories in namespace %q", t.Namespace)
	Expect(imageRepoList.Items).Should(HaveLen(len(Components)),
		"expected %d ImageRepository CRs in namespace %q (one per component)", len(Components), t.Namespace)
}

// validateDockerConfigSecret fetches the named Secret in the given namespace,
// parses .dockerconfigjson, and returns true only when every registry entry
// contains a valid base64-encoded "user:password" auth token. The label
// parameter (e.g. "push" or "pull") is used in diagnostic log messages.
func validateDockerConfigSecret(fw *framework.Framework, namespace, secretName, label, robotAccountName string) bool {
	secret := &corev1.Secret{}
	if err := fw.AsKubeAdmin.CommonController.KubeRest().Get(
		context.Background(),
		client.ObjectKey{Name: secretName, Namespace: namespace},
		secret,
	); err != nil {
		GinkgoWriter.Printf("  %s secret %s/%s not found: %v\n", label, namespace, secretName, err)
		return false
	}

	dockerCfgJSON, ok := secret.Data[".dockerconfigjson"]
	if !ok || len(dockerCfgJSON) == 0 {
		GinkgoWriter.Printf("  %s secret %s/%s has no .dockerconfigjson\n", label, namespace, secretName)
		return false
	}

	var cfg struct {
		Auths map[string]struct {
			Auth string `json:"auth"`
		} `json:"auths"`
	}
	if err := json.Unmarshal(dockerCfgJSON, &cfg); err != nil {
		GinkgoWriter.Printf("  %s secret %s/%s: malformed .dockerconfigjson: %v\n",
			label, namespace, secretName, err)
		return false
	}

	if len(cfg.Auths) == 0 {
		GinkgoWriter.Printf("  %s secret %s/%s: auths map is empty\n", label, namespace, secretName)
		return false
	}

	for registry, cred := range cfg.Auths {
		if cred.Auth == "" {
			GinkgoWriter.Printf("  %s secret %s/%s: empty auth for registry %s\n",
				label, namespace, secretName, registry)
			return false
		}
		decoded, err := base64.StdEncoding.DecodeString(cred.Auth)
		if err != nil {
			GinkgoWriter.Printf("  %s secret %s/%s: auth for %s is not valid base64: %v\n",
				label, namespace, secretName, registry, err)
			return false
		}
		username, password, ok := strings.Cut(string(decoded), ":")
		if !ok {
			GinkgoWriter.Printf("  %s secret %s/%s: auth for %s decoded but missing colon separator\n",
				label, namespace, secretName, registry)
			return false
		}
		if username == "" || password == "" {
			GinkgoWriter.Printf("  %s secret %s/%s: auth for %s has empty username or password\n",
				label, namespace, secretName, registry)
			return false
		}
		if robotAccountName != "" && username != robotAccountName {
			GinkgoWriter.Printf("  %s secret %s/%s: auth username %q does not match expected robot account %q\n",
				label, namespace, secretName, username, robotAccountName)
			return false
		}
	}

	GinkgoWriter.Printf("  %s secret %s/%s: valid credentials for %d registries (robot: %s)\n",
		label, namespace, secretName, len(cfg.Auths), robotAccountName)
	return true
}

// waitForPushSecretReadiness polls each tenant's ImageRepository push and pull
// secrets until all contain valid .dockerconfigjson auth entries. After a Velero
// restore, image-controller must reconcile each ImageRepository — creating
// new Quay robot accounts and writing fresh credentials into both secrets.
// Builds that start before push reconciliation completes fail with
// "unauthorized: Could not find robot with specified username".
// EC verify tasks that run before pull reconciliation completes fail the same way.
func waitForPushSecretReadiness(fw *framework.Framework, tenants []Tenant) {
	GinkgoHelper()

	By("Waiting for ImageRepository push and pull secrets to contain valid credentials")

	for _, t := range tenants {
		imageRepoList := &imagecontrollerv1alpha1.ImageRepositoryList{}
		Expect(fw.AsKubeAdmin.CommonController.KubeRest().List(
			context.Background(), imageRepoList, client.InNamespace(t.Namespace),
		)).Should(Succeed(), "failed to list ImageRepositories in %s", t.Namespace)

		for i := range imageRepoList.Items {
			ir := &imageRepoList.Items[i]
			secretName := ir.Status.Credentials.PushSecretName
			if secretName == "" {
				GinkgoWriter.Printf("ImageRepository %s/%s has no push secret name in status — waiting for reconciliation\n",
					t.Namespace, ir.Name)
			}

			recoveryAttempts := 0
			const maxRecoveryAttempts = 3

			Eventually(func() bool {
				// Re-read ImageRepository to pick up status changes.
				freshIR := &imagecontrollerv1alpha1.ImageRepository{}
				if err := fw.AsKubeAdmin.CommonController.KubeRest().Get(
					context.Background(),
					client.ObjectKey{Name: ir.Name, Namespace: t.Namespace},
					freshIR,
				); err != nil {
					GinkgoWriter.Printf("  error reading ImageRepository %s/%s: %v\n", t.Namespace, ir.Name, err)
					return false
				}

				if freshIR.Status.State != imagecontrollerv1alpha1.ImageRepositoryStateReady {
					if freshIR.Status.State == "damaged" && recoveryAttempts < maxRecoveryAttempts {
						const irFinalizer = "appstudio.openshift.io/image-repository"
						finalizers := freshIR.GetFinalizers()
						filtered := make([]string, 0, len(finalizers))
						removed := false
						for _, f := range finalizers {
							if f == irFinalizer {
								removed = true
							} else {
								filtered = append(filtered, f)
							}
						}
						if removed {
							GinkgoWriter.Printf("  ImageRepository %s/%s is damaged — removing finalizer (attempt %d/%d)\n",
								t.Namespace, ir.Name, recoveryAttempts+1, maxRecoveryAttempts)
							freshIR.SetFinalizers(filtered)
							if err := fw.AsKubeAdmin.CommonController.KubeRest().Update(
								context.Background(), freshIR,
							); err != nil {
								GinkgoWriter.Printf("  recovery attempt %d failed: %v — will retry\n",
									recoveryAttempts+1, err)
							} else {
								recoveryAttempts++
								GinkgoWriter.Printf("  recovery attempt %d succeeded\n", recoveryAttempts)
							}
						} else {
							GinkgoWriter.Printf("  ImageRepository %s/%s is damaged but has no finalizer — waiting for controller\n",
								t.Namespace, ir.Name)
						}
					} else if freshIR.Status.State == "damaged" {
						GinkgoWriter.Printf("  ImageRepository %s/%s still damaged after %d recovery attempts — no further recovery will be tried\n",
							t.Namespace, ir.Name, recoveryAttempts)
					}
					GinkgoWriter.Printf("  ImageRepository %s/%s state: %s (waiting for ready)\n",
						t.Namespace, ir.Name, freshIR.Status.State)
					return false
				}

				sName := freshIR.Status.Credentials.PushSecretName
				if sName == "" {
					GinkgoWriter.Printf("  ImageRepository %s/%s ready but no push secret name yet\n",
						t.Namespace, ir.Name)
					return false
				}

				if !validateDockerConfigSecret(fw, t.Namespace, sName, "push",
					freshIR.Status.Credentials.PushRobotAccountName) {
					return false
				}

				pullName := freshIR.Status.Credentials.PullSecretName
				if pullName == "" {
					GinkgoWriter.Printf("  ImageRepository %s/%s ready but no pull secret name yet\n",
						t.Namespace, ir.Name)
					return false
				}

				if !validateDockerConfigSecret(fw, t.Namespace, pullName, "pull",
					freshIR.Status.Credentials.PullRobotAccountName) {
					return false
				}

				return true
			}, 5*time.Minute, 5*time.Second).Should(BeTrue(),
				"ImageRepository %s/%s push/pull secrets did not become ready within 5 minutes",
				t.Namespace, ir.Name)
		}
	}

	GinkgoWriter.Println("All ImageRepository push and pull secrets contain valid credentials")
}

// ensurePullSecretsOnSA links each ImageRepository's pull secret to the
// pipeline ServiceAccount. After backup/restore + re-provisioning, the SA
// may not have pull secrets in its .secrets field. Without this, Tekton
// credential initialization won't provide registry auth to integration test
// tasks (EC verify), causing UNAUTHORIZED errors even though the secrets
// contain valid credentials.
func ensurePullSecretsOnSA(fw *framework.Framework, tenants []Tenant) {
	GinkgoHelper()

	By("Ensuring pull secrets are linked to pipeline ServiceAccount")

	for _, t := range tenants {
		var pipelineSA *corev1.ServiceAccount
		var saName string
		for _, candidate := range []string{constants.DefaultPipelineServiceAccount, "appstudio-pipeline"} {
			sa := &corev1.ServiceAccount{}
			if err := fw.AsKubeAdmin.CommonController.KubeRest().Get(
				context.Background(),
				client.ObjectKey{Name: candidate, Namespace: t.Namespace},
				sa,
			); err == nil {
				pipelineSA = sa
				saName = candidate
				break
			}
		}
		Expect(pipelineSA).ShouldNot(BeNil(),
			"no pipeline SA found in %s (tried %s, appstudio-pipeline)",
			t.Namespace, constants.DefaultPipelineServiceAccount)

		secretRefs := make([]string, len(pipelineSA.Secrets))
		for i, s := range pipelineSA.Secrets {
			secretRefs[i] = s.Name
		}
		GinkgoWriter.Printf("SA %s/%s .secrets: %v\n", t.Namespace, saName, secretRefs)

		pullRefs := make([]string, len(pipelineSA.ImagePullSecrets))
		for i, s := range pipelineSA.ImagePullSecrets {
			pullRefs[i] = s.Name
		}
		GinkgoWriter.Printf("SA %s/%s .imagePullSecrets: %v\n", t.Namespace, saName, pullRefs)

		irList := &imagecontrollerv1alpha1.ImageRepositoryList{}
		Expect(fw.AsKubeAdmin.CommonController.KubeRest().List(
			context.Background(), irList, client.InNamespace(t.Namespace),
		)).Should(Succeed(), "failed to list ImageRepositories in %s", t.Namespace)

		for i := range irList.Items {
			pullSecretName := irList.Items[i].Status.Credentials.PullSecretName
			if pullSecretName == "" {
				continue
			}

			alreadyLinked := false
			for _, s := range pipelineSA.Secrets {
				if s.Name == pullSecretName {
					alreadyLinked = true
					break
				}
			}

			if alreadyLinked {
				GinkgoWriter.Printf("  pull secret %s already in SA %s .secrets\n", pullSecretName, saName)
			} else {
				err := fw.AsKubeAdmin.CommonController.LinkSecretToServiceAccount(
					t.Namespace, pullSecretName, saName, true)
				Expect(err).ShouldNot(HaveOccurred(),
					"failed to link pull secret %s to SA %s in %s",
					pullSecretName, saName, t.Namespace)
				GinkgoWriter.Printf("  linked pull secret %s to SA %s\n", pullSecretName, saName)
			}
		}
	}
}

// logReleaseChainDiagnostics dumps the state of the release trigger chain
// (Snapshots, ReleasePlan, Releases, ReleasePlanAdmission, release PipelineRuns)
// for each tenant. Called before the release PipelineRun wait to capture
// the chain state at the moment builds and tests complete. Uses oc commands
// to avoid importing Snapshot types.
func logReleaseChainDiagnostics(tenants []Tenant) {
	run := func(args ...string) string {
		out, _ := exec.Command("oc", args...).CombinedOutput() // #nosec G204
		return strings.TrimSpace(string(out))
	}

	for _, t := range tenants {
		GinkgoWriter.Printf("=== RELEASE CHAIN DIAGNOSTICS: %s ===\n", t.Namespace)

		GinkgoWriter.Printf("Snapshots in %s:\n%s\n\n",
			t.Namespace, run("get", "snapshots.appstudio.redhat.com",
				"-n", t.Namespace, "-o", "wide"))

		GinkgoWriter.Printf("Snapshot annotations in %s:\n%s\n\n",
			t.Namespace, run("get", "snapshots.appstudio.redhat.com",
				"-n", t.Namespace, "-o",
				`jsonpath={range .items[*]}{.metadata.name}{"\t"}event-type={.metadata.labels.pac\.test\.appstudio\.openshift\.io/event-type}{"\t"}test-status={.metadata.annotations.test\.appstudio\.openshift\.io/status}{"\n"}{end}`))

		GinkgoWriter.Printf("ReleasePlan %s status:\n%s\n\n",
			DRReleasePlanName,
			run("get", "releaseplans.appstudio.redhat.com", DRReleasePlanName,
				"-n", t.Namespace, "-o", "jsonpath={.status}"))

		GinkgoWriter.Printf("Releases in %s:\n%s\n\n",
			t.Namespace, run("get", "releases.appstudio.redhat.com",
				"-n", t.Namespace, "-o", "wide"))

		GinkgoWriter.Printf("ReleasePlanAdmission in %s:\n%s\n\n",
			t.ManagedNamespace,
			run("get", "releaseplanadmissions.appstudio.redhat.com",
				"-n", t.ManagedNamespace, "-o", "jsonpath={.items[0].status}"))

		GinkgoWriter.Printf("PipelineRuns in %s:\n%s\n\n",
			t.ManagedNamespace,
			run("get", "pipelinerun", "-n", t.ManagedNamespace, "-o", "wide"))

		GinkgoWriter.Printf("=== END DIAGNOSTICS: %s ===\n", t.Namespace)
	}
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
		out, _ := exec.Command("oc", args...).CombinedOutput() // #nosec G204
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
// In a real disaster (etcd loss), the API server is gone. There is no
// graceful deletion. Finalizers never fire. Controllers never learn that
// resources disappeared — external state (Quay robot accounts, GitHub
// webhooks) survives intact.
//
// `oc delete project` triggers graceful deletion instead. Every resource
// gets a deletionTimestamp, and every finalizer fires. Multiple Konflux
// controllers set finalizers on tenant namespace resources:
//
//   - application-service  → Application
//   - image-controller     → Application, Component, ImageRepository
//   - integration-service  → Component, PipelineRun
//   - release-service      → Release
//   - pipelines-as-code    → PipelineRun
//
// Each finalizer's controller makes external API calls (Quay, GitHub,
// Tekton) that can stall or deadlock, blocking namespace deletion past
// the 10-minute timeout. This is a simulation gap, not a controller bug.
//
// Solution: strip ALL finalizers from ALL resources in tenant namespaces
// before deletion. One deterministic pre-delete step that matches real
// disaster semantics — no finalizer processing, no external side effects.
//
// After restore, the Velero resource modifier strips ImageRepository
// finalizers on restored CRs (KFLUXINFRA-2177).
// See also KFLUXINFRA-3954, STONEBLD-3714.

// stripAndDeleteNamespaces strips all finalizers from each tenant namespace
// and immediately deletes it before controllers can re-add finalizers.
// The strip-then-delete MUST happen without interruption per namespace —
// splitting them across Ginkgo specs allows controllers to re-add finalizers
// in the gap (observed: 2-43s is enough for image-controller to restore its
// finalizer, which then deletes Quay robot accounts during graceful deletion).
func stripAndDeleteNamespaces(fw *framework.Framework, tenants []Tenant) {
	GinkgoHelper()

	ctx := context.Background()
	restClient := fw.AsKubeAdmin.CommonController.KubeRest()

	for _, t := range tenants {
		By(fmt.Sprintf("Stripping all finalizers in namespace %q to simulate etcd-loss semantics", t.Namespace))

		total := 0

		total += stripFinalizers(ctx, restClient, t.Namespace, &appservice.ApplicationList{}, "Application")
		total += stripFinalizers(ctx, restClient, t.Namespace, &appservice.ComponentList{}, "Component")
		total += stripFinalizers(ctx, restClient, t.Namespace, &imagecontrollerv1alpha1.ImageRepositoryList{}, "ImageRepository")
		total += stripFinalizers(ctx, restClient, t.Namespace, &releaseapi.ReleaseList{}, "Release")
		total += stripFinalizers(ctx, restClient, t.Namespace, &pipeline.PipelineRunList{}, "PipelineRun")

		GinkgoWriter.Printf("Stripped finalizers from %d resources in namespace %s\n", total, t.Namespace)

		deleteNamespace(fw, t.Namespace)
	}
}

// stripFinalizers lists all resources of a given type in a namespace and
// removes every finalizer from each. Returns the number of resources modified.
// When bestEffort is true, Update failures are logged instead of asserting —
// use this in cleanup paths (AfterAll) where an assertion would abort remaining cleanup.
func stripFinalizers(ctx context.Context, restClient client.Client, namespace string, list client.ObjectList, kind string, bestEffort ...bool) int {
	isBestEffort := len(bestEffort) > 0 && bestEffort[0]

	err := restClient.List(ctx, list, client.InNamespace(namespace))
	if err != nil {
		GinkgoWriter.Printf("  %s: list failed (may not exist on cluster): %v\n", kind, err)
		return 0
	}

	stripped := 0
	items := extractItems(list)
	for _, obj := range items {
		if len(obj.GetFinalizers()) == 0 {
			continue
		}
		GinkgoWriter.Printf("  %s/%s: removing finalizers %v\n", kind, obj.GetName(), obj.GetFinalizers())

		const maxConflictRetries = 5
		var updateErr error
		for attempt := 0; attempt < maxConflictRetries; attempt++ {
			if attempt > 0 {
				if err := restClient.Get(ctx, client.ObjectKeyFromObject(obj), obj); err != nil {
					if k8sErrors.IsNotFound(err) {
						updateErr = nil
						break
					}
					updateErr = err
					break
				}
			}
			obj.SetFinalizers(nil)
			updateErr = restClient.Update(ctx, obj)
			if updateErr == nil || !k8sErrors.IsConflict(updateErr) {
				break
			}
			GinkgoWriter.Printf("  %s/%s: conflict on attempt %d, retrying\n", kind, obj.GetName(), attempt+1)
		}
		if updateErr != nil {
			if isBestEffort {
				GinkgoWriter.Printf("  WARNING: failed to strip finalizers from %s %s/%s: %v\n",
					kind, namespace, obj.GetName(), updateErr)
				continue
			}
			Expect(updateErr).ShouldNot(HaveOccurred(),
				"failed to strip finalizers from %s %s/%s", kind, namespace, obj.GetName())
		}
		stripped++
	}

	if stripped > 0 || len(items) > 0 {
		GinkgoWriter.Printf("  %s: stripped %d/%d\n", kind, stripped, len(items))
	}
	return stripped
}

func toObjects[E any, P interface {
	*E
	client.Object
}](items []E) []client.Object {
	out := make([]client.Object, len(items))
	for i := range items {
		out[i] = P(&items[i])
	}
	return out
}

// extractItems converts a typed ObjectList into a slice of client.Object
// for generic finalizer processing.
func extractItems(list client.ObjectList) []client.Object {
	switch l := list.(type) {
	case *appservice.ApplicationList:
		return toObjects[appservice.Application, *appservice.Application](l.Items)
	case *appservice.ComponentList:
		return toObjects[appservice.Component, *appservice.Component](l.Items)
	case *imagecontrollerv1alpha1.ImageRepositoryList:
		return toObjects[imagecontrollerv1alpha1.ImageRepository, *imagecontrollerv1alpha1.ImageRepository](l.Items)
	case *releaseapi.ReleaseList:
		return toObjects[releaseapi.Release, *releaseapi.Release](l.Items)
	case *pipeline.PipelineRunList:
		return toObjects[pipeline.PipelineRun, *pipeline.PipelineRun](l.Items)
	default:
		GinkgoWriter.Printf("WARNING: extractItems: unhandled list type %T\n", list)
		return nil
	}
}
