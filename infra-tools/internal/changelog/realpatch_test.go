package changelog_test

import (
	. "github.com/onsi/gomega"
	"github.com/redhat-appstudio/infra-deployments/infra-tools/internal/changelog"
	"testing"
)

func TestExtractImageDigestChanges_RealKonfluxPatches(t *testing.T) {
	g := NewWithT(t)
	files := []changelog.FileChange{
		{Filename: "operator/upstream-kustomizations/namespace-lister/kustomization.yaml", Patch: "@@ -10,7 +10,7 @@ resources:\n - network_policy_allow_from_konfluxui.yaml\n namespace: namespace-lister\n images:\n-- digest: sha256:8e4e3ec5b292ad4c50c6ef54bcd0fea3bd2c10db0756f922bc0bd07f68f22bac\n+- digest: sha256:845c9c58fafccc4d8ba3550687abc16921400b554fad0d401f0b4cba260f5c0c\n   name: quay.io/konflux-ci/namespace-lister\n   newName: quay.io/konflux-ci/namespace-lister\n patches:"},
		{Filename: "operator/upstream-kustomizations/registry/kustomization.yml", Patch: "@@ -10,4 +10,4 @@ resources:\n images:\n   - name: quay.io/konflux-ci/zot\n     newName: quay.io/konflux-ci/zot\n-    digest: sha256:6d8bd88c660378c80e91c675274781550c672ad59f32f4c73f8620868d78dcfa\n+    digest: sha256:742befd7c9b9faa87bac3b11f31eaf4f5f5096bc7d5619005e7cbcace93330f4"},
		{Filename: "operator/upstream-kustomizations/ui/core/proxy/kustomization.yaml", Patch: "@@ -5,18 +5,21 @@ configMapGenerator:\n   name: proxy-caddyfile\n - files:\n   - tekton-results.caddy\n+  - kite.caddy\n+  - kubearchive.caddy\n+  - watson.caddy\n   name: proxy-caddy-templates\n - files:\n   - generate-proxy-config.sh\n   name: proxy-generate-config\n images:\n-- digest: sha256:7c781d6356c69b96de0b6ed0484a3e753d96fa06e1d8b2417a91543afed783f4\n+- digest: sha256:5730bfed253a4e7b33226c05195166f0adde81020df7bfbb7edd42376bf818ad\n   name: quay.io/konflux-ci/konflux-ui\n   newName: quay.io/konflux-ci/konflux-ui\n-- digest: sha256:9141a22f047f059fc75c2d13f642c0d8a07b4d64f65650878aa886357716b4fa\n+- digest: sha256:246ddfbb8554a5109496cfd3938bb4248cd4e48c82380241f2776de40ae7de5a\n   name: quay.io/konflux-ci/oauth2-proxy\n   newName: quay.io/konflux-ci/oauth2-proxy\n-- digest: sha256:eaa34f8d14c56b74e7988d00b57cb009416f92d1fa5708fbef9596ee61a27063\n+- digest: sha256:d0b9faf5b3cd15c818d6035ba8fc981ff73197b2375597861904c5af21d71a38\n   name: quay.io/konflux-ci/reverse-proxy\n   newName: quay.io/konflux-ci/reverse-proxy\n kind: Kustomization"},
		{Filename: "operator/upstream-kustomizations/ui/dex/kustomization.yml", Patch: "@@ -6,7 +6,7 @@ resources:\n images:\n   - name: quay.io/konflux-ci/dex\n     newName: quay.io/konflux-ci/dex\n-    digest: sha256:68f87765e0f947a81791c476bdb7706d3632f483e8594aa5ce1ffc6796f9ab92\n+    digest: sha256:314fb9eccaab47cd92615d763d794e9bcb080dcbda89a1a96162d49742fd10f9\n \n configMapGenerator:\n - files:"},
	}
	changes, skipped := changelog.ExtractImageDigestChanges(files)
	g.Expect(skipped).To(BeFalse())
	g.Expect(changes).To(HaveLen(6))

	names := make([]string, len(changes))
	for i, c := range changes {
		names[i] = c.ImageName
		g.Expect(c.OldDigest).To(MatchRegexp(`^sha256:[0-9a-f]{64}$`))
		g.Expect(c.NewDigest).To(MatchRegexp(`^sha256:[0-9a-f]{64}$`))
		g.Expect(c.OldDigest).NotTo(Equal(c.NewDigest))
	}
	g.Expect(names).To(ConsistOf(
		"quay.io/konflux-ci/namespace-lister",
		"quay.io/konflux-ci/zot",
		"quay.io/konflux-ci/konflux-ui",
		"quay.io/konflux-ci/oauth2-proxy",
		"quay.io/konflux-ci/reverse-proxy",
		"quay.io/konflux-ci/dex",
	))
}
