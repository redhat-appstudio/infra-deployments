This folder contains any ad-hoc modifications needed to keep metric monitoring in good shape. This include changes to resources not owned and managed through this repository, such as resources deployed by operators.

The resources deployed by other entities require modification [via server-side apply synchronization option in ArgoCD](https://github.com/argoproj/argo-cd/issues/3984#issuecomment-3268258579) to take effect.
