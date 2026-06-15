// Package disaster_recovery implements the DR (Disaster Recovery) backup/restore e2e test suite
// for Konflux tenant namespaces.
//
// The suite validates the single-tenant backup and restore process documented in the
// tenants-restore-from-backup SOP by running two complementary test scenarios on a
// single ROSA cluster:
//
//  1. Backwards-compatibility test: Creates two tenants on an older Konflux version,
//     backs them up, upgrades Konflux to the new version, then restores and verifies.
//     Proves that backups taken on older Konflux versions restore correctly on newer ones.
//
//  2. Same-version test: Creates two tenants on the current Konflux version, backs them
//     up, deletes them, restores them, and verifies. Proves that backup/restore works
//     correctly within a single Konflux version.
//
// Each scenario uses a two-tenant architecture: Tenant 1 (KokoHazamar) restores via
// the Velero CLI method, and Tenant 2 (MosheKipod) restores via the oc command method.
// This tests both documented SOP restore procedures in parallel without doubling runtime.
// All tenants build the same MathWizz application (3 Components from a monorepo).
package disaster_recovery

import (
	"time"

	"github.com/konflux-ci/e2e-tests/pkg/constants"
	"github.com/konflux-ci/e2e-tests/pkg/utils"
	corev1 "k8s.io/api/core/v1"
)

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

// Tenant holds the deterministic identifiers for a single tenant namespace.
// Names are convention-based (no timestamps) with scenario-specific suffixes
// (e.g., "-backwards-compat-dr" vs "-same-version-dr") to avoid collision,
// since all DR tests run on the same ROSA cluster within a single pipeline run.
type Tenant struct {
	Namespace        string
	ManagedNamespace string
	AppName          string
	BackupName       string
	ForkRepoName     string // set at runtime by forkRepoForTenant
	ForkRepoURL      string // set at runtime by forkRepoForTenant
}

// ComponentDef describes a MathWizz application component (microservice) and
// the metadata needed to create a Component CR for it in each tenant namespace.
// All tenants build the same three application components from the MathWizz
// monorepo.
type ComponentDef struct {
	Name          string
	ContextDir    string
	DockerfileURL string
}

// RestoreMethod selects which SOP procedure to use when restoring a tenant.
// Tenant 1 always uses RestoreMethodVeleroCLI; Tenant 2 uses RestoreMethodOCCommand.
type RestoreMethod string

const (
	// RestoreMethodVeleroCLI creates a Restore by invoking the velero binary
	// directly, mirroring the SOP's `velero restore create` procedure.
	RestoreMethodVeleroCLI RestoreMethod = "velero-cli"

	// RestoreMethodOCCommand creates a Restore CR by generating a JSON
	// manifest and applying it via `oc apply -f`, mirroring the SOP's
	// declarative restore procedure.
	RestoreMethodOCCommand RestoreMethod = "oc-command"
)

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

// MathWizzRepo is the monorepo containing all three MathWizz microservices.
// Each Component points to a different context directory within this repo.
var MathWizzRepo = "https://github.com/" +
	utils.GetEnv(constants.GITHUB_E2E_ORGANIZATION_ENV, "redhat-appstudio-qe") +
	"/DR_test_MathWizz"

const (
	// VeleroNamespace is the namespace where OADP/Velero components run.
	// Backup and Restore CRs are created here, not in the tenant namespace.
	VeleroNamespace = "openshift-adp"

	// MathWizzRepoName is the repository name without the GitHub organization.
	// The framework's GitHub client resolves the full path using the
	// GITHUB_E2E_ORGANIZATION_ENV environment variable. The repo must exist
	// as a fork in that organization for triggerBuildsAndVerify to work.
	MathWizzRepoName = "DR_test_MathWizz"

	// MathWizzDefaultBranch is the default branch of the MathWizz repo.
	MathWizzDefaultBranch = "main"

	// Component names shared across all tenants. These map 1:1 to the
	// microservices in the MathWizz monorepo that have real source code builds.
	CompWebServer     = "mathwizz-web-server" // #nosec
	CompHistoryWorker = "mathwizz-history-worker"
	CompFrontend      = "mathwizz-frontend"

	// ComponentsPerTenant is the number of MathWizz Components each tenant
	// creates. Derived from the Components slice but kept as a constant
	// because it's used across multiple files for PipelineRun count assertions.
	ComponentsPerTenant = 3
)

// ---------------------------------------------------------------------------
// Tenant pairs — backwards-compatibility scenario
// ---------------------------------------------------------------------------

// BCTenant1 is the first backwards-compat tenant (KokoHazamar).
// Restores via the Velero CLI method.
var BCTenant1 = Tenant{
	Namespace:        "dr-test-kokohazamar-backwards-compat-dr",
	ManagedNamespace: "dr-test-kokohazamar-backwards-compat-dr-managed",
	AppName:          "kokohazamar-backwards-compat-dr",
	BackupName:       "backup-kokohazamar-backwards-compat-dr",
}

// BCTenant2 is the second backwards-compat tenant (MosheKipod).
// Restores via the oc command method.
var BCTenant2 = Tenant{
	Namespace:        "dr-test-moshekipod-backwards-compat-dr",
	ManagedNamespace: "dr-test-moshekipod-backwards-compat-dr-managed",
	AppName:          "moshekipod-backwards-compat-dr",
	BackupName:       "backup-moshekipod-backwards-compat-dr",
}

// ---------------------------------------------------------------------------
// Tenant pairs — same-version scenario
// ---------------------------------------------------------------------------

// SVTenant1 is the first same-version tenant (KokoHazamar).
// Restores via the Velero CLI method.
var SVTenant1 = Tenant{
	Namespace:        "dr-test-kokohazamar-same-version-dr",
	ManagedNamespace: "dr-test-kokohazamar-same-version-dr-managed",
	AppName:          "kokohazamar-same-version-dr",
	BackupName:       "backup-kokohazamar-same-version-dr",
}

// SVTenant2 is the second same-version tenant (MosheKipod).
// Restores via the oc command method.
var SVTenant2 = Tenant{
	Namespace:        "dr-test-moshekipod-same-version-dr",
	ManagedNamespace: "dr-test-moshekipod-same-version-dr-managed",
	AppName:          "moshekipod-same-version-dr",
	BackupName:       "backup-moshekipod-same-version-dr",
}

// ---------------------------------------------------------------------------
// Components
// ---------------------------------------------------------------------------

// Components defines the three MathWizz application components that get
// registered as Component CRs in each tenant namespace. Each points to a
// different context directory in the MathWizz monorepo. The two infrastructure
// services (database, message-queue) are omitted because they use stock images
// and don't have source code builds.
var Components = []ComponentDef{
	{Name: CompWebServer, ContextDir: "web-server", DockerfileURL: "Dockerfile"},
	{Name: CompHistoryWorker, ContextDir: "history-worker", DockerfileURL: "Dockerfile"},
	{Name: CompFrontend, ContextDir: "frontend", DockerfileURL: "Dockerfile"},
}

// ---------------------------------------------------------------------------
// Release infrastructure — constants for setting up the release pipeline chain
// ---------------------------------------------------------------------------
//
// Each tenant gets a "managed namespace" (Tenant.ManagedNamespace) where the
// release pipeline runs. This namespace holds the ServiceAccount, RBAC,
// registry credentials, Enterprise Contract policy, PVC, ReleasePlanAdmission,
// and release PipelineRuns. The ReleasePlan itself lives in the tenant
// namespace and is included in the Velero backup.
//
// The managed namespace is NOT deleted during disaster simulation (only the
// tenant namespace is lost). After restore, the ReleasePlan is restored in the
// tenant namespace and the ReleasePlanAdmission still exists in the managed
// namespace, so the release chain resumes automatically when new integration
// tests pass.

const (
	// DRReleasePipelineSA is the ServiceAccount in the managed namespace
	// that the release pipeline runs as.
	DRReleasePipelineSA = "release-service-account"

	// DRReleasePlanName is the ReleasePlan CR created in each tenant namespace.
	// Auto-release is enabled so that releases trigger automatically when
	// integration tests pass.
	DRReleasePlanName = "dr-releaseplan"

	// DRReleasePlanAdmissionName is the ReleasePlanAdmission CR created in the
	// managed namespace. It authorizes releases from the tenant namespace.
	DRReleasePlanAdmissionName = "dr-releaseplanadmission"

	// DRECPolicyName is the Enterprise Contract policy created in the managed
	// namespace. It validates the build pipeline output before allowing release.
	DRECPolicyName = "dr-ec-policy"

	// DRReleasePVCName is the PersistentVolumeClaim for the release pipeline
	// workspace in the managed namespace.
	DRReleasePVCName = "dr-release-pvc"

	// DRCosignSecretName holds the cosign public key used to verify image
	// signatures in the Enterprise Contract policy.
	DRCosignSecretName = "cosign-public-key" // #nosec

	// DRQuayAuthSecret holds Quay registry authentication credentials for the
	// release pipeline to push released images.
	DRQuayAuthSecret = "hacbs-release-tests-token" // #nosec

	// DRReleaseCatalogTAQuaySecret holds Quay credentials for the release
	// service catalog trusted artifacts.
	DRReleaseCatalogTAQuaySecret = "release-catalog-trusted-artifacts-quay-secret"
)

// DRCosignPublicKey is the test cosign public key used for Enterprise Contract
// policy verification. This is a well-known test key shared across the e2e
// test suites (same key as used in tests/release/).
var DRCosignPublicKey = []byte("-----BEGIN PUBLIC KEY-----\n" +
	"MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEocSG/SnE0vQ20wRfPltlXrY4Ib9B\n" +
	"CRnFUCg/fndZsXdz0IX5sfzIyspizaTbu4rapV85KirmSBU6XUaLY347xg==\n" +
	"-----END PUBLIC KEY-----")

// DRManagedNamespaceSecrets lists the Secrets that must exist in the managed
// namespace before the release ServiceAccount is created. These are referenced
// by the ServiceAccount's secrets field so the release pipeline can pull images
// and access trusted artifacts.
var DRManagedNamespaceSecrets = []corev1.ObjectReference{
	{Name: DRQuayAuthSecret},
	{Name: DRReleaseCatalogTAQuaySecret},
}

// Release pipeline configuration — env-var driven with defaults that match CI.
var (
	// RelSvcCatalogURL is the Git URL for the release-service-catalog repo
	// containing the release pipeline definitions.
	RelSvcCatalogURL = utils.GetEnv("RELEASE_SERVICE_CATALOG_URL",
		"https://github.com/konflux-ci/release-service-catalog")

	// RelSvcCatalogRevision is the Git revision (branch/tag/commit) of the
	// release-service-catalog to use for release pipeline definitions.
	RelSvcCatalogRevision = utils.GetEnv("RELEASE_SERVICE_CATALOG_REVISION",
		"development")

	// DRReleasedImagePushRepo is the Quay repository where the release pipeline
	// pushes released images. Uses the e2e organization from the environment.
	DRReleasedImagePushRepo = "quay.io/" +
		utils.GetEnv(constants.QUAY_E2E_ORGANIZATION_ENV, "redhat-appstudio-qe") +
		"/dr_test_mathwizz"
)

// ---------------------------------------------------------------------------
// IncludedResources — the Konflux tenant resource types to back up and restore
// ---------------------------------------------------------------------------

// IncludedResources defines the Konflux tenant resource types included in
// backup and restore operations. This list must match the current SOP at:
//
//	sop/infra/disaster-recovery/cluster/tenants-restore-from-backup.md
//
// Update this list as new resource types are added to the backup scope.
var IncludedResources = []string{
	"applications.appstudio.redhat.com",
	"components.appstudio.redhat.com",
	"environments.appstudio.redhat.com",
	"integrationtestscenarios.appstudio.redhat.com",
	"secrets",
	"snapshots.appstudio.redhat.com",
	"serviceaccounts",
	"rolebindings",
	"namespaces",
	"imagerepositories.appstudio.redhat.com",
	"repositories.pipelinesascode.tekton.dev",
	"releases.appstudio.redhat.com",
	"releaseplans.appstudio.redhat.com",
	"releaseplanadmissions.appstudio.redhat.com",
}

// ---------------------------------------------------------------------------
// Timeouts and polling intervals
// ---------------------------------------------------------------------------
//
// Each pair controls an Eventually() or Consistently() call. Timeouts are
// generous to accommodate real-world variance on shared CI infrastructure.
// Poll intervals balance responsiveness against API server / etcd load —
// shorter for fast operations (namespace deletion, token regeneration) and
// longer for slow operations (backup, restore) to avoid log bloat.

const (
	// BackupTimeout is how long to wait for a Velero Backup CR to reach
	// the Completed phase. Backups are read-only snapshots, relatively fast.
	BackupTimeout      = 15 * time.Minute
	BackupPollInterval = 1 * time.Minute

	// RestoreTimeout is how long to wait for a Velero Restore CR to reach
	// the Completed phase. Restores recreate all resources and can be slow,
	// especially when the cluster is under load from other reconciliation.
	RestoreTimeout      = 2 * time.Hour
	RestorePollInterval = 1 * time.Minute

	// NamespaceTimeout is how long to wait for a namespace to be fully deleted.
	// Namespace deletion is usually fast but can stall on finalizers.
	NamespaceTimeout = 5 * time.Minute
	NamespacePoll    = 5 * time.Second

	// SATokenTimeout is how long to wait for new ServiceAccount token Secrets
	// to appear after deleting the old (UID-invalid) ones post-restore.
	SATokenTimeout = 5 * time.Minute
	SATokenPoll    = 30 * time.Second

	// PipelineTimeout is how long to wait for all build and integration test
	// PipelineRuns to complete in a tenant namespace.
	PipelineTimeout = 90 * time.Minute
	PipelinePoll    = 30 * time.Second

	// ReleaseChainTimeout is how long to wait for all release PipelineRuns to
	// complete in the managed namespace after integration tests pass. Release
	// pipelines are slower because they include image signing, EC validation,
	// and pushing to the release registry.
	ReleaseChainTimeout = 60 * time.Minute
	ReleaseChainPoll    = 5 * time.Minute

	// VeleroReadyTimeout is how long to wait for Velero deployment readiness
	// and BSL availability during precondition checks.
	VeleroReadyTimeout = 5 * time.Minute
	VeleroReadyPoll    = 10 * time.Second
)

// ---------------------------------------------------------------------------
// Backup integrity validation
// ---------------------------------------------------------------------------

const (
	// BackupMinItemCount is the minimum number of Kubernetes resources Velero
	// should back up for a single tenant namespace. The assertion uses >=
	// because the exact count varies with controller-managed resources
	// (e.g., Snapshots, token Secrets) that may differ between runs.
	// Calibrate this value by inspecting Backup.Status.Progress.ItemsBackedUp
	// from CI runs (logged by validateBackupIntegrity).
	//
	// Calculated lower-bound based on a MathWizz tenant with 3 Components:
	//   1 Namespace + 1 Application + 3 Components + 1 ITS + ~3 SAs +
	//   ~10 Secrets + ~3 RoleBindings + 3 PaC Repos + 3 ImageRepos +
	//   1 ReleasePlan ≈ 29 (excluding variable Snapshots)
	BackupMinItemCount = 25

	// BackupMinTarballSize is the minimum expected size in bytes of the
	// Velero backup tarball stored in MinIO for a single tenant namespace.
	// The assertion uses >= because tarball size varies with the number and
	// content of backed-up resources. Calibrate this value by inspecting
	// the tarball size logged by validateBackupIntegrity from CI runs.
	BackupMinTarballSize int64 = 10000
)

// VeleroBackupTarballPathFmt is the S3 object key pattern for the backup
// tarball. Velero writes the compressed backup data at this path inside the
// bucket. Both %s placeholders are the backup name.
const VeleroBackupTarballPathFmt = "backups/%s/%s.tar.gz"
