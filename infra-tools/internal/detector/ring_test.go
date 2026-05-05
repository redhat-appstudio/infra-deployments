package detector_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/redhat-appstudio/infra-deployments/infra-tools/internal/detector"
)

// ---------------------------------------------------------------------------
// ClassifyFileEnv
// ---------------------------------------------------------------------------

func TestClassifyFileEnv_StagingComponent(t *testing.T) {
	g := NewWithT(t)
	g.Expect(detector.ClassifyFileEnv("components/build-service/staging/base/deploy.yaml")).To(Equal(detector.Staging))
}

func TestClassifyFileEnv_ProductionComponent(t *testing.T) {
	g := NewWithT(t)
	g.Expect(detector.ClassifyFileEnv("components/build-service/production/base/deploy.yaml")).To(Equal(detector.Production))
}

func TestClassifyFileEnv_StagingDownstream(t *testing.T) {
	g := NewWithT(t)
	g.Expect(detector.ClassifyFileEnv("components/multi-platform-controller/staging-downstream/kustomization.yaml")).To(Equal(detector.Staging))
}

func TestClassifyFileEnv_ProductionDownstream(t *testing.T) {
	g := NewWithT(t)
	g.Expect(detector.ClassifyFileEnv("components/multi-platform-controller/production-downstream/kustomization.yaml")).To(Equal(detector.Production))
}

func TestClassifyFileEnv_NestedStagingComponent(t *testing.T) {
	g := NewWithT(t)
	g.Expect(detector.ClassifyFileEnv("components/monitoring/grafana/staging/base/datasources.yaml")).To(Equal(detector.Staging))
}

func TestClassifyFileEnv_NestedProductionComponent(t *testing.T) {
	g := NewWithT(t)
	g.Expect(detector.ClassifyFileEnv("components/monitoring/prometheus/production/base/monitoringstack/endpoints-params.yaml")).To(Equal(detector.Production))
}

func TestClassifyFileEnv_ArgoCDOverlayStagingDownstream(t *testing.T) {
	g := NewWithT(t)
	g.Expect(detector.ClassifyFileEnv("argo-cd-apps/overlays/staging-downstream/kustomization.yaml")).To(Equal(detector.Staging))
}

func TestClassifyFileEnv_ArgoCDOverlayKonfluxPublicProduction(t *testing.T) {
	g := NewWithT(t)
	g.Expect(detector.ClassifyFileEnv("argo-cd-apps/overlays/konflux-public-production/production-overlay-patch.yaml")).To(Equal(detector.Production))
}

func TestClassifyFileEnv_ArgoCDOverlayKonfluxPublicStaging(t *testing.T) {
	g := NewWithT(t)
	g.Expect(detector.ClassifyFileEnv("argo-cd-apps/overlays/konflux-public-staging/kustomization.yaml")).To(Equal(detector.Staging))
}

func TestClassifyFileEnv_BaseFile(t *testing.T) {
	g := NewWithT(t)
	g.Expect(detector.ClassifyFileEnv("components/build-service/base/deploy.yaml")).To(Equal(detector.Environment("")))
}

func TestClassifyFileEnv_RootFile(t *testing.T) {
	g := NewWithT(t)
	g.Expect(detector.ClassifyFileEnv("README.md")).To(Equal(detector.Environment("")))
}

func TestClassifyFileEnv_DevelopmentFile(t *testing.T) {
	g := NewWithT(t)
	g.Expect(detector.ClassifyFileEnv("components/build-service/development/kustomization.yaml")).To(Equal(detector.Environment("")))
}

func TestClassifyFileEnv_ConfigsDir(t *testing.T) {
	g := NewWithT(t)
	g.Expect(detector.ClassifyFileEnv("configs/etcd-defrag/staging/config.yaml")).To(Equal(detector.Staging))
}

// ---------------------------------------------------------------------------
// CheckRingDeployment — direct conflict
// ---------------------------------------------------------------------------

func TestCheckRingDeployment_DirectConflict(t *testing.T) {
	g := NewWithT(t)

	changed := []string{
		"components/build-service/staging/base/deploy.yaml",
		"components/build-service/production/base/deploy.yaml",
	}
	affected := map[detector.Environment]bool{detector.Staging: true, detector.Production: true}

	result := detector.CheckRingDeployment(changed, affected)

	g.Expect(result.DirectConflict).To(BeTrue())
	g.Expect(result.IndirectConflict).To(BeFalse())
	g.Expect(result.StagingFiles).To(ConsistOf("components/build-service/staging/base/deploy.yaml"))
	g.Expect(result.ProductionFiles).To(ConsistOf("components/build-service/production/base/deploy.yaml"))
}

func TestCheckRingDeployment_DirectConflictMixedComponents(t *testing.T) {
	g := NewWithT(t)

	changed := []string{
		"components/integration/staging/base/kustomization.yaml",
		"components/pipeline-service/production/base/kustomization.yaml",
	}
	affected := map[detector.Environment]bool{detector.Staging: true, detector.Production: true}

	result := detector.CheckRingDeployment(changed, affected)

	g.Expect(result.DirectConflict).To(BeTrue())
	g.Expect(result.StagingFiles).To(HaveLen(1))
	g.Expect(result.ProductionFiles).To(HaveLen(1))
}

func TestCheckRingDeployment_DirectConflictDownstream(t *testing.T) {
	g := NewWithT(t)

	changed := []string{
		"components/multi-platform-controller/staging-downstream/kustomization.yaml",
		"components/multi-platform-controller/production-downstream/kustomization.yaml",
	}
	affected := map[detector.Environment]bool{detector.Staging: true, detector.Production: true}

	result := detector.CheckRingDeployment(changed, affected)

	g.Expect(result.DirectConflict).To(BeTrue())
}

func TestCheckRingDeployment_DirectConflictArgoCDOverlays(t *testing.T) {
	g := NewWithT(t)

	changed := []string{
		"argo-cd-apps/overlays/konflux-public-staging/kustomization.yaml",
		"argo-cd-apps/overlays/konflux-public-production/production-overlay-patch.yaml",
	}
	affected := map[detector.Environment]bool{detector.Staging: true, detector.Production: true}

	result := detector.CheckRingDeployment(changed, affected)

	g.Expect(result.DirectConflict).To(BeTrue())
}

func TestCheckRingDeployment_DirectConflictNestedComponent(t *testing.T) {
	g := NewWithT(t)

	changed := []string{
		"components/monitoring/grafana/staging/base/datasources.yaml",
		"components/monitoring/prometheus/production/base/endpoints-params.yaml",
	}
	affected := map[detector.Environment]bool{detector.Staging: true, detector.Production: true}

	result := detector.CheckRingDeployment(changed, affected)

	g.Expect(result.DirectConflict).To(BeTrue())
}

// ---------------------------------------------------------------------------
// CheckRingDeployment — indirect conflict (base-only changes)
// ---------------------------------------------------------------------------

func TestCheckRingDeployment_IndirectConflict(t *testing.T) {
	g := NewWithT(t)

	changed := []string{
		"components/build-service/base/deploy.yaml",
	}
	affected := map[detector.Environment]bool{detector.Staging: true, detector.Production: true}

	result := detector.CheckRingDeployment(changed, affected)

	g.Expect(result.DirectConflict).To(BeFalse())
	g.Expect(result.IndirectConflict).To(BeTrue())
	g.Expect(result.StagingFiles).To(BeEmpty())
	g.Expect(result.ProductionFiles).To(BeEmpty())
}

func TestCheckRingDeployment_IndirectConflictMultipleBaseFiles(t *testing.T) {
	g := NewWithT(t)

	changed := []string{
		"components/build-service/base/deploy.yaml",
		"components/build-service/base/rbac.yaml",
		"components/integration/base/kustomization.yaml",
	}
	affected := map[detector.Environment]bool{detector.Staging: true, detector.Production: true}

	result := detector.CheckRingDeployment(changed, affected)

	g.Expect(result.DirectConflict).To(BeFalse())
	g.Expect(result.IndirectConflict).To(BeTrue())
}

// ---------------------------------------------------------------------------
// CheckRingDeployment — staging-only (no conflict)
// ---------------------------------------------------------------------------

func TestCheckRingDeployment_StagingOnly(t *testing.T) {
	g := NewWithT(t)

	changed := []string{
		"components/build-service/staging/base/deploy.yaml",
		"components/build-service/staging/stone-stage-p01/patch.yaml",
	}
	affected := map[detector.Environment]bool{detector.Staging: true}

	result := detector.CheckRingDeployment(changed, affected)

	g.Expect(result.DirectConflict).To(BeFalse())
	g.Expect(result.IndirectConflict).To(BeFalse())
	g.Expect(result.StagingFiles).To(HaveLen(2))
	g.Expect(result.ProductionFiles).To(BeEmpty())
}

// ---------------------------------------------------------------------------
// CheckRingDeployment — production-only (no conflict)
// ---------------------------------------------------------------------------

func TestCheckRingDeployment_ProductionOnly(t *testing.T) {
	g := NewWithT(t)

	changed := []string{
		"components/build-service/production/base/deploy.yaml",
		"components/build-service/production/kflux-ocp-p01/patch.yaml",
	}
	affected := map[detector.Environment]bool{detector.Production: true}

	result := detector.CheckRingDeployment(changed, affected)

	g.Expect(result.DirectConflict).To(BeFalse())
	g.Expect(result.IndirectConflict).To(BeFalse())
	g.Expect(result.StagingFiles).To(BeEmpty())
	g.Expect(result.ProductionFiles).To(HaveLen(2))
}

// ---------------------------------------------------------------------------
// CheckRingDeployment — no environment affected
// ---------------------------------------------------------------------------

func TestCheckRingDeployment_NoEnvsAffected(t *testing.T) {
	g := NewWithT(t)

	changed := []string{"README.md", "docs/introduction/index.md"}
	affected := map[detector.Environment]bool{}

	result := detector.CheckRingDeployment(changed, affected)

	g.Expect(result.DirectConflict).To(BeFalse())
	g.Expect(result.IndirectConflict).To(BeFalse())
	g.Expect(result.StagingFiles).To(BeEmpty())
	g.Expect(result.ProductionFiles).To(BeEmpty())
}

// ---------------------------------------------------------------------------
// CheckRingDeployment — staging direct + production indirect (no direct conflict)
// ---------------------------------------------------------------------------

func TestCheckRingDeployment_StagingDirectProductionIndirect(t *testing.T) {
	g := NewWithT(t)

	changed := []string{
		"components/build-service/staging/base/deploy.yaml",
		"components/build-service/base/common.yaml",
	}
	affected := map[detector.Environment]bool{detector.Staging: true, detector.Production: true}

	result := detector.CheckRingDeployment(changed, affected)

	g.Expect(result.DirectConflict).To(BeFalse(), "only staging files are directly changed; production is only indirectly affected")
	g.Expect(result.IndirectConflict).To(BeTrue())
	g.Expect(result.StagingFiles).To(HaveLen(1))
	g.Expect(result.ProductionFiles).To(BeEmpty())
}

// ---------------------------------------------------------------------------
// CheckRingDeployment — output is sorted
// ---------------------------------------------------------------------------

func TestCheckRingDeployment_OutputSorted(t *testing.T) {
	g := NewWithT(t)

	changed := []string{
		"components/z-service/staging/b.yaml",
		"components/a-service/staging/a.yaml",
		"components/z-service/production/b.yaml",
		"components/a-service/production/a.yaml",
	}
	affected := map[detector.Environment]bool{detector.Staging: true, detector.Production: true}

	result := detector.CheckRingDeployment(changed, affected)

	g.Expect(result.DirectConflict).To(BeTrue())
	g.Expect(result.StagingFiles).To(Equal([]string{
		"components/a-service/staging/a.yaml",
		"components/z-service/staging/b.yaml",
	}))
	g.Expect(result.ProductionFiles).To(Equal([]string{
		"components/a-service/production/a.yaml",
		"components/z-service/production/b.yaml",
	}))
}
