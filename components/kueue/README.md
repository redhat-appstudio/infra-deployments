# Tekton Kueue ownership

The ring-1 Konflux staging Hub and spokes share `base/multikueue`. It uses the
promoted Kueue snapshot, adds the shared build queue, and points monitoring and
RBAC at `openshift-pipelines`, where the Pipelines Operator installs Tekton Kueue.
The Hub patches its admission check and quota locally; the spokes use `base/spoke`
to add resource flavors and secret-syncer RBAC.
Other clusters continue to install Tekton Kueue from the ring's
`base/tekton-kueue/kustomization.yaml`.

These opt-in directories are kept out of the ring base's `kustomization.yaml`.
