Helm values for the development overlay are specified at 
[`argo-cd-apps/overlays/development/set-kyverno-values.yaml`](../../../argo-cd-apps/overlays/development/set-kyverno-values.yaml).

Please make sure to change values there instead of introducing a values file here.

Introducing a values file here may cause undesiderable side effects.
Indeed, ArgoCD will always fetch the file from the upstream repo, instead of consuming the changes made locally.

