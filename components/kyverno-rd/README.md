# kyverno-rd

Ring-deployment layout for Kyverno, following the components standard
(directory-structure-migration SOP). Kyverno is a Helm-based component, so
per-cluster differences that cannot be expressed by patching a shared Helm
generator are reproduced via Tier-3 post-render patches.

## Layout

- `base/` — Tier 1: resources shared by every ring (the `konflux-kyverno`
  Namespace).
- `rings/empty-base/empty-base/` — placeholder overlay the ApplicationSet
  references before k-components populate the real cluster list.
- `rings/ring-0/base/` — Ring 0 (development): Helm generator, values, image
  overrides and the migrate-resources Job patch. Contains a `base-snapshot/`
  copy of Tier 1 so `kustomize build` stays self-contained per ring.

Staging (ring-1) and production (rings 2-4) rings are added in later SOP parts.

## Notes

- Ring 0 = development.
- `chainsaw/` holds validation tests and is **not** deployed via ArgoCD.
- Rendered output is kept byte-identical to the pre-migration
  `components/kyverno/` overlays (aside from the additively-managed Namespace,
  which `CreateNamespace=true` already guarantees).
