package main

import (
	"context"
	"errors"
	"testing"

	. "github.com/onsi/gomega"

	ghclient "github.com/redhat-appstudio/infra-deployments/infra-tools/internal/github"
)

type fakeKargoChecker struct {
	exempt bool
	err    error
}

func (f *fakeKargoChecker) KargoSoleCommitter(context.Context, int, string, string) (bool, error) {
	return f.exempt, f.err
}

func TestFormatUnreferencedChangesSanitizesPaths(t *testing.T) {
	g := NewWithT(t)

	msg := formatUnreferencedChanges([]string{
		"components/x/`\n::error::boom.yaml",
		"components/x/##[warning]fake.yaml",
		"components/x/\u202eevil.yaml",
	})

	g.Expect(msg).To(ContainSubstring("- `components/x/' : :error: :boom.yaml`\n"))
	g.Expect(msg).NotTo(ContainSubstring("::"))
	g.Expect(msg).NotTo(ContainSubstring("##["))
	g.Expect(msg).NotTo(ContainSubstring("\u202e"))
}

func TestSanitizeLineNeutralizesWorkflowCommands(t *testing.T) {
	g := NewWithT(t)
	g.Expect(sanitizeLine("##[warning]fake")).To(Equal("# #[warning]fake"))
	g.Expect(sanitizeLine("::error::fake")).To(Equal(": :error: :fake"))
}

func TestProductionHoldRequired(t *testing.T) {
	g := NewWithT(t)

	cases := []struct {
		name   string
		client kargoProvenanceChecker
		want   bool
	}{
		{"nil client fails closed", nil, true},
		{"api error fails closed", &fakeKargoChecker{err: errors.New("boom")}, true},
		{"not exempt", &fakeKargoChecker{exempt: false}, true},
		{"verified kargo-only promotion", &fakeKargoChecker{exempt: true}, false},
	}
	for _, tc := range cases {
		g.Expect(productionHoldRequired(context.Background(), tc.client, 1, "head-sha", "sender")).To(Equal(tc.want), tc.name)
	}
}

func TestAddProductionHold(t *testing.T) {
	g := NewWithT(t)

	labels := addProductionHold([]string{"environment/production"})
	g.Expect(labels).To(ConsistOf(
		"environment/production",
		ghclient.HoldProductionLabel,
		ghclient.NeedsApprovalProductionLabel,
	))

	again := addProductionHold(labels)
	g.Expect(again).To(HaveLen(3))
}

func TestKargoPromotionTouchesOnlyRing(t *testing.T) {
	g := NewWithT(t)

	valid := []string{
		"components/foo/rings/ring-2/base/kustomization.yaml",
		"components/foo/rings/ring-2/base/base-snapshot/deploy.yaml",
	}
	g.Expect(kargoPromotionTouchesOnlyRing(valid, "kargo-konflux-core/foo/ring-2")).To(BeTrue())

	g.Expect(kargoPromotionTouchesOnlyRing(
		[]string{"configs/etcd-defrag/rings/ring-1/base/cronjob.yaml"},
		"kargo-konflux-infrastructure/etcd-defrag/ring-1",
	)).To(BeTrue())

	g.Expect(kargoPromotionTouchesOnlyRing(
		[]string{"components/foo/rings/ring-1/base/kustomization.yaml"},
		"kargo-konflux-core/foo/ring-2",
	)).To(BeFalse())

	g.Expect(kargoPromotionTouchesOnlyRing(
		[]string{"components/bar/rings/ring-2/base/kustomization.yaml"},
		"kargo-konflux-core/foo/ring-2",
	)).To(BeFalse())

	g.Expect(kargoPromotionTouchesOnlyRing(
		[]string{"components/foo/rings/ring-2/base/kustomization.yaml", "components/foo/base/x.yaml"},
		"kargo-konflux-core/foo/ring-2",
	)).To(BeFalse())

	g.Expect(kargoPromotionTouchesOnlyRing(nil, "kargo-konflux-core/foo/ring-2")).To(BeFalse())
	g.Expect(kargoPromotionTouchesOnlyRing(valid, "not-a-promotion-branch")).To(BeFalse())
}
