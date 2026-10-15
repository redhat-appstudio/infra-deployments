# MintMaker ring deployment

This component contains MintMaker's `rd-dev` migration. Ring 0 is the only ring
migrated here; staging and production continue to use the legacy
`components/mintmaker` overlays until their migrations are completed.

## Layout

- `base/` contains the source manifests for the ring deployment.
- `rings/ring-0/base/base-snapshot/` is the deployed snapshot of those source
  manifests.
- `rings/ring-0/base/kustomization.yaml` pins the MintMaker upstream config and
  images for the development ring.
- `argo-cd-apps/overlays/rd-dev/mintmaker/` defines the development
  ApplicationSet.

## Keeping development aligned

During the migration, `components/mintmaker/development/` is the reference for
the ring-0 upstream refs, image pins, and rendered manifests. Release update PRs
currently update the legacy development and staging paths, so mirror development
pin changes into `rings/ring-0/base/kustomization.yaml` until that automation is
updated. Keep `base/` aligned with `components/mintmaker/base/`; promote manifest
changes through `base-snapshot/` so they do not bypass the ring.

## Validate

From the repository root:

```sh
kustomize build --enable-helm components/mintmaker-rd/rings/ring-0/base
hack/test-ring-render.sh mintmaker-rd dev
```
