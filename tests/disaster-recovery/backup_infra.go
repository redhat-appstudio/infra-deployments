// backup_infra.go provides tenant infrastructure helpers for the DR test suite:
// creating tenant namespaces with Applications and Component CRs, deleting
// namespaces, and rotating ServiceAccount tokens after a Velero restore.
//
// NOTE: Helper functions in this file call GinkgoHelper() so that assertion
// failures report the caller's location in the test spec, not the helper's
// internal line number. This is a Ginkgo feature analogous to t.Helper() in
// the standard testing package.
package disaster_recovery

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/devfile/library/v2/pkg/util"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	ecp "github.com/conforma/crds/api/v1alpha1"
	appservice "github.com/konflux-ci/application-api/api/v1alpha1"
	"github.com/konflux-ci/e2e-tests/pkg/constants"
	"github.com/konflux-ci/e2e-tests/pkg/framework"
	"github.com/konflux-ci/e2e-tests/pkg/utils"
	tektonutils "github.com/konflux-ci/release-service/tekton/utils"
)

// forkRepoForTenant creates a unique GitHub fork of the MathWizz source repo
// for this tenant. Each tenant gets its own fork so that PaC can configure
// webhooks independently (avoids PaC error 53 when multiple namespaces target
// the same repo). Sets the ForkRepoName and ForkRepoURL fields on the Tenant.
func forkRepoForTenant(fw *framework.Framework, t *Tenant) {
	GinkgoHelper()

	forkName := fmt.Sprintf("DR-MathWizz-%s", util.GenerateRandomString(6))
	By(fmt.Sprintf("Forking %s → %s for tenant %s", MathWizzRepoName, forkName, t.Namespace))

	ghClient := fw.AsKubeAdmin.HasController.Github
	_, err := ghClient.ForkRepository(MathWizzRepoName, forkName)
	Expect(err).ShouldNot(HaveOccurred(),
		"failed to fork %s to %s for tenant %s", MathWizzRepoName, forkName, t.Namespace)

	org := utils.GetEnv(constants.GITHUB_E2E_ORGANIZATION_ENV, "redhat-appstudio-qe")
	t.ForkRepoName = forkName
	t.ForkRepoURL = fmt.Sprintf("https://github.com/%s/%s", org, forkName)
}

// cleanupForks deletes the forked GitHub repos for all tenants. Safe to call
// even if a fork was never created (empty ForkRepoName is a no-op).
func cleanupForks(fw *framework.Framework, tenants []Tenant) {
	ghClient := fw.AsKubeAdmin.HasController.Github
	for _, t := range tenants {
		if t.ForkRepoName == "" {
			continue
		}
		GinkgoWriter.Printf("Deleting fork repo %s for tenant %s\n", t.ForkRepoName, t.Namespace)
		if err := ghClient.DeleteRepositoryIfExists(t.ForkRepoName); err != nil {
			GinkgoWriter.Printf("WARNING: failed to delete fork %s: %v\n", t.ForkRepoName, err)
		}
	}
}

// mergePaCConfigPRs finds and merges all PaC configuration PRs on a tenant's
// forked repo. Build-service opens one PR per Component (branch prefix
// "konflux-"), so we expect ComponentsPerTenant PRs. Merging is required so
// that subsequent PRs (e.g., from triggerBuildsAndVerify) trigger PipelineRuns
// via the PaC pipeline definitions on the default branch.
func mergePaCConfigPRs(fw *framework.Framework, t Tenant) {
	GinkgoHelper()

	Expect(t.ForkRepoName).ShouldNot(BeEmpty(), "ForkRepoName not set for tenant %s", t.Namespace)
	ghClient := fw.AsKubeAdmin.HasController.Github

	By(fmt.Sprintf("Waiting for %d PaC config PRs on %s", ComponentsPerTenant, t.ForkRepoName))

	var pacPRNumbers []int
	Eventually(func() int {
		prs, err := ghClient.ListPullRequests(t.ForkRepoName)
		if err != nil {
			GinkgoWriter.Printf("error listing PRs on %s: %v\n", t.ForkRepoName, err)
			return 0
		}

		pacPRNumbers = nil
		for _, pr := range prs {
			head := pr.GetHead().GetRef()
			if strings.HasPrefix(head, constants.PaCPullRequestBranchPrefix) {
				pacPRNumbers = append(pacPRNumbers, pr.GetNumber())
			}
		}
		GinkgoWriter.Printf("found %d PaC config PRs on %s (need %d)\n",
			len(pacPRNumbers), t.ForkRepoName, ComponentsPerTenant)
		return len(pacPRNumbers)
	}, 10*time.Minute, 15*time.Second).Should(Equal(ComponentsPerTenant),
		"expected %d PaC config PRs on %s", ComponentsPerTenant, t.ForkRepoName)

	By(fmt.Sprintf("Merging %d PaC config PRs on %s", len(pacPRNumbers), t.ForkRepoName))
	for _, prNum := range pacPRNumbers {
		Eventually(func() error {
			_, err := ghClient.MergePullRequest(t.ForkRepoName, prNum)
			return err
		}, 2*time.Minute, 10*time.Second).Should(Succeed(),
			"failed to merge PaC config PR #%d on %s", prNum, t.ForkRepoName)
		GinkgoWriter.Printf("Merged PaC config PR #%d on %s\n", prNum, t.ForkRepoName)
	}
}

// createTenant provisions a full tenant namespace with an Application and all
// Components defined in the Components slice. After this function returns the
// tenant is ready for the full build → integration test → release pipeline chain.
func createTenant(fw *framework.Framework, t Tenant) {
	GinkgoHelper()

	By(fmt.Sprintf("Creating tenant namespace %s", t.Namespace))
	_, err := fw.AsKubeAdmin.CommonController.CreateTestNamespace(t.Namespace)
	Expect(err).ShouldNot(HaveOccurred(), "failed to create tenant namespace %s", t.Namespace)

	By(fmt.Sprintf("Creating Application %s in namespace %s", t.AppName, t.Namespace))
	_, err = fw.AsKubeAdmin.HasController.CreateApplication(t.AppName, t.Namespace)
	Expect(err).ShouldNot(HaveOccurred(), "failed to create Application %s in namespace %s", t.AppName, t.Namespace)

	for _, comp := range Components {
		By(fmt.Sprintf("Creating Component %s in namespace %s", comp.Name, t.Namespace))

		repoURL := t.ForkRepoURL
		Expect(repoURL).ShouldNot(BeEmpty(),
			"ForkRepoURL not set for tenant %s — call forkRepoForTenant first", t.Namespace)

		spec := appservice.ComponentSpec{
			ComponentName: comp.Name,
			Source: appservice.ComponentSource{
				ComponentSourceUnion: appservice.ComponentSourceUnion{
					GitSource: &appservice.GitSource{
						URL:           repoURL,
						Context:       comp.ContextDir,
						DockerfileURL: comp.DockerfileURL,
					},
				},
			},
		}

		_, err = fw.AsKubeAdmin.HasController.CreateComponent(spec, t.Namespace, "", "", t.AppName, false, nil)
		Expect(err).ShouldNot(HaveOccurred(), "failed to create Component %s in namespace %s", comp.Name, t.Namespace)
	}

	setupReleaseInfra(fw, t)
}

// setupReleaseInfra creates the release pipeline infrastructure in the managed
// namespace and the ReleasePlan in the tenant namespace. This follows the same
// sequence as tests/release/service/happy_path.go BeforeAll.
func setupReleaseInfra(fw *framework.Framework, t Tenant) {
	GinkgoHelper()

	By(fmt.Sprintf("Creating managed namespace %s for release pipelines", t.ManagedNamespace))
	_, err := fw.AsKubeAdmin.CommonController.CreateTestNamespace(t.ManagedNamespace)
	Expect(err).ShouldNot(HaveOccurred(), "failed to create managed namespace %s", t.ManagedNamespace)

	By(fmt.Sprintf("Creating release pipeline ServiceAccount %s in managed namespace %s", DRReleasePipelineSA, t.ManagedNamespace))
	managedServiceAccount, err := fw.AsKubeAdmin.CommonController.CreateServiceAccount(
		DRReleasePipelineSA, t.ManagedNamespace, DRManagedNamespaceSecrets, nil)
	Expect(err).ShouldNot(HaveOccurred(), "failed to create release pipeline SA in %s", t.ManagedNamespace)

	By(fmt.Sprintf("Creating release pipeline RoleBinding in managed namespace %s", t.ManagedNamespace))
	_, err = fw.AsKubeAdmin.ReleaseController.CreateReleasePipelineRoleBindingForServiceAccount(
		t.ManagedNamespace, managedServiceAccount)
	Expect(err).ShouldNot(HaveOccurred(), "failed to create release pipeline RoleBinding in %s", t.ManagedNamespace)

	By(fmt.Sprintf("Creating registry auth secrets in managed namespace %s", t.ManagedNamespace))
	sourceAuthJson := utils.GetEnv("QUAY_TOKEN", "")
	Expect(sourceAuthJson).ShouldNot(BeEmpty(), "QUAY_TOKEN env var must be set for release pipeline")

	releaseCatalogTAQuayAuthJson := utils.GetEnv("RELEASE_CATALOG_TA_QUAY_TOKEN", "")
	Expect(releaseCatalogTAQuayAuthJson).ShouldNot(BeEmpty(), "RELEASE_CATALOG_TA_QUAY_TOKEN env var must be set for release pipeline")

	_, err = fw.AsKubeAdmin.CommonController.CreateRegistryAuthSecret(
		DRQuayAuthSecret, t.ManagedNamespace, sourceAuthJson)
	Expect(err).ShouldNot(HaveOccurred(), "failed to create Quay auth secret in %s", t.ManagedNamespace)

	_, err = fw.AsKubeAdmin.CommonController.CreateRegistryAuthSecret(
		DRReleaseCatalogTAQuaySecret, t.ManagedNamespace, releaseCatalogTAQuayAuthJson)
	Expect(err).ShouldNot(HaveOccurred(), "failed to create release catalog TA Quay secret in %s", t.ManagedNamespace)

	By(fmt.Sprintf("Linking Quay auth secret to release SA in managed namespace %s", t.ManagedNamespace))
	err = fw.AsKubeAdmin.CommonController.LinkSecretToServiceAccount(
		t.ManagedNamespace, DRQuayAuthSecret, DRReleasePipelineSA, true)
	Expect(err).ShouldNot(HaveOccurred(), "failed to link Quay auth secret to release SA in %s", t.ManagedNamespace)

	By(fmt.Sprintf("Creating cosign signing secret in managed namespace %s", t.ManagedNamespace))
	Expect(fw.AsKubeAdmin.TektonController.CreateOrUpdateSigningSecret(
		DRCosignPublicKey, DRCosignSecretName, t.ManagedNamespace)).
		Should(Succeed(), "failed to create cosign signing secret in %s", t.ManagedNamespace)

	By("Getting default Enterprise Contract policy from enterprise-contract-service namespace")
	defaultEcPolicy, err := fw.AsKubeAdmin.TektonController.GetEnterpriseContractPolicy(
		"default", "enterprise-contract-service")
	Expect(err).ShouldNot(HaveOccurred(), "failed to get default EC policy")

	By(fmt.Sprintf("Creating Enterprise Contract policy %s in managed namespace %s", DRECPolicyName, t.ManagedNamespace))
	ecPolicySpec := ecp.EnterpriseContractPolicySpec{
		Description: "DR test Enterprise Contract policy",
		PublicKey:   fmt.Sprintf("k8s://%s/%s", t.ManagedNamespace, DRCosignSecretName),
		Sources:     defaultEcPolicy.Spec.Sources,
		Configuration: &ecp.EnterpriseContractPolicyConfiguration{
			Collections: []string{"@slsa3"},
			Exclude:     []string{"step_image_registries", "tasks.required_tasks_found:prefetch-dependencies"},
		},
	}
	_, err = fw.AsKubeAdmin.TektonController.CreateEnterpriseContractPolicy(
		DRECPolicyName, t.ManagedNamespace, ecPolicySpec)
	Expect(err).ShouldNot(HaveOccurred(), "failed to create EC policy %s in %s", DRECPolicyName, t.ManagedNamespace)

	By(fmt.Sprintf("Creating ReleasePlan %s in tenant namespace %s", DRReleasePlanName, t.Namespace))
	_, err = fw.AsKubeAdmin.ReleaseController.CreateReleasePlan(
		DRReleasePlanName, t.Namespace, t.AppName, t.ManagedNamespace,
		"true", nil, nil, nil)
	Expect(err).ShouldNot(HaveOccurred(), "failed to create ReleasePlan %s in %s", DRReleasePlanName, t.Namespace)

	By(fmt.Sprintf("Creating ReleasePlanAdmission %s in managed namespace %s", DRReleasePlanAdmissionName, t.ManagedNamespace))
	mappingData := buildReleaseMappingData()
	_, err = fw.AsKubeAdmin.ReleaseController.CreateReleasePlanAdmission(
		DRReleasePlanAdmissionName, t.ManagedNamespace, "", t.Namespace,
		DRECPolicyName, DRReleasePipelineSA,
		[]string{t.AppName}, false,
		&tektonutils.PipelineRef{
			Resolver: "git",
			Params: []tektonutils.Param{
				{Name: "url", Value: RelSvcCatalogURL},
				{Name: "revision", Value: RelSvcCatalogRevision},
				{Name: "pathInRepo", Value: "pipelines/managed/e2e/e2e.yaml"},
			},
		},
		&runtime.RawExtension{Raw: mappingData})
	Expect(err).ShouldNot(HaveOccurred(), "failed to create ReleasePlanAdmission %s in %s", DRReleasePlanAdmissionName, t.ManagedNamespace)

	By(fmt.Sprintf("Creating release PVC %s in managed namespace %s", DRReleasePVCName, t.ManagedNamespace))
	_, err = fw.AsKubeAdmin.TektonController.CreatePVCInAccessMode(
		DRReleasePVCName, t.ManagedNamespace, corev1.ReadWriteOnce)
	Expect(err).ShouldNot(HaveOccurred(), "failed to create release PVC in %s", t.ManagedNamespace)

	By(fmt.Sprintf("Creating secrets access Role in managed namespace %s", t.ManagedNamespace))
	_, err = fw.AsKubeAdmin.CommonController.CreateRole(
		"role-release-service-account", t.ManagedNamespace,
		map[string][]string{
			"apiGroupsList": {""},
			"roleResources": {"secrets"},
			"roleVerbs":     {"get", "list", "watch"},
		})
	Expect(err).ShouldNot(HaveOccurred(), "failed to create secrets access Role in %s", t.ManagedNamespace)

	By(fmt.Sprintf("Creating secrets access RoleBinding in managed namespace %s", t.ManagedNamespace))
	_, err = fw.AsKubeAdmin.CommonController.CreateRoleBinding(
		"role-release-service-account-binding", t.ManagedNamespace,
		"ServiceAccount", DRReleasePipelineSA, t.ManagedNamespace,
		"Role", "role-release-service-account", "rbac.authorization.k8s.io")
	Expect(err).ShouldNot(HaveOccurred(), "failed to create secrets access RoleBinding in %s", t.ManagedNamespace)
}

// buildReleaseMappingData constructs the JSON mapping data for the
// ReleasePlanAdmission. Each MathWizz component maps to the DR release
// image push repository.
func buildReleaseMappingData() []byte {
	GinkgoHelper()
	components := make([]map[string]interface{}, 0, len(Components))
	for _, comp := range Components {
		components = append(components, map[string]interface{}{
			"component":  comp.Name,
			"repository": DRReleasedImagePushRepo,
		})
	}

	data, err := json.Marshal(map[string]interface{}{
		"mapping": map[string]interface{}{
			"components": components,
		},
	})
	Expect(err).ShouldNot(HaveOccurred(), "failed to marshal release mapping data")
	return data
}

// deleteNamespace removes a tenant namespace. The framework's DeleteNamespace
// method (pkg/clients/common/namespace.go) internally waits up to 10 minutes
// for the namespace to disappear and reports any stuck finalizers on failure.
func deleteNamespace(fw *framework.Framework, namespace string) {
	GinkgoHelper()

	By(fmt.Sprintf("Deleting namespace %s and waiting for removal", namespace))
	err := fw.AsKubeAdmin.CommonController.DeleteNamespace(namespace)
	Expect(err).ShouldNot(HaveOccurred(), "failed to delete namespace %s", namespace)
}

// listSATokenSecrets returns all ServiceAccount token Secrets in a namespace.
func listSATokenSecrets(ctx context.Context, fw *framework.Framework, namespace string) ([]corev1.Secret, error) {
	GinkgoHelper()

	secretList := &corev1.SecretList{}
	err := fw.AsKubeAdmin.CommonController.KubeRest().List(
		ctx,
		secretList,
		client.InNamespace(namespace),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list Secrets in namespace %s: %w", namespace, err)
	}

	var tokens []corev1.Secret
	for i := range secretList.Items {
		if secretList.Items[i].Type == corev1.SecretTypeServiceAccountToken {
			tokens = append(tokens, secretList.Items[i])
		}
	}
	return tokens, nil
}

// rotateSATokens deletes all ServiceAccount token Secrets in a namespace and
// waits until the token controller regenerates exactly as many replacements.
//
// Why this is needed: after a Velero restore, ServiceAccount UIDs change but
// the old token Secrets still reference the pre-restore UIDs, making them
// invalid. Deleting the stale tokens forces the token controller to mint new
// ones that match the current SA UIDs. See:
// https://konflux-ci.dev/docs/troubleshooting/service-accounts/
func rotateSATokens(fw *framework.Framework, namespace string) {
	GinkgoHelper()

	ctx := context.Background()

	By(fmt.Sprintf("Rotating ServiceAccount tokens in namespace %s", namespace))

	tokens, err := listSATokenSecrets(ctx, fw, namespace)
	Expect(err).ShouldNot(HaveOccurred())

	if len(tokens) == 0 {
		GinkgoWriter.Printf("WARNING: no SA token Secrets found in %s — token rotation is a no-op\n", namespace)
		return
	}

	// Delete every SA token secret (old tokens are invalid after restore).
	for i := range tokens {
		err = fw.AsKubeAdmin.CommonController.KubeRest().Delete(ctx, &tokens[i])
		Expect(err).ShouldNot(HaveOccurred(), "failed to delete SA token Secret %s in namespace %s", tokens[i].Name, namespace)
	}
	deletedCount := len(tokens)
	GinkgoWriter.Printf("Deleted %d stale SA token Secrets in namespace %s\n", deletedCount, namespace)

	// Wait for the token controller to regenerate at least as many new tokens.
	Eventually(func() int {
		newTokens, err := listSATokenSecrets(ctx, fw, namespace)
		if err != nil {
			GinkgoWriter.Printf("error listing SA token Secrets in %s: %v\n", namespace, err)
			return 0
		}
		return len(newTokens)
	}, SATokenTimeout, SATokenPoll).Should(Equal(deletedCount),
		fmt.Sprintf("expected exactly %d new SA token Secrets in namespace %s", deletedCount, namespace))
}
