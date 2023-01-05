---
title: GitOps Service
---

Once the cluster is successfully bootstrapped, create a Namespace with the `argocd.argoproj.io/managed-by: gitops-service-argocd` label, for example:

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: (your-user-name)
  labels:
    argocd.argoproj.io/managed-by: gitops-service-argocd
```

The `argocd.argoproj.io/managed-by: gitops-service-argocd` label gives 'permission' to Argo CD (specifically, the instance in `gitops-service-argocd`) to deploy to your namespace.

You may now create `GitOpsDeployment` resources, which the GitOps Service will respond to, deploying resources to your namespace:

```yaml
apiVersion: managed-gitops.redhat.com/v1alpha1
kind: GitOpsDeployment

metadata:
  name: gitops-depl
  namespace: (your-user-name)

spec:

  # Application/component to deploy
  source:
    repoURL: https://github.com/redhat-appstudio/gitops-repository-template
    path: environments/overlays/dev

  # destination: {}  # destination is user namespace if empty

  # Only 'automated' type is currently supported: changes to the GitOps repo immediately take effect (as soon as Argo CD detects them).
  type: automated
```

## Viewing the ArgoCD instance that is used to deploy user workloads

* Determine the route

```
kubectl get  route/gitops-service-argocd-server  -n gitops-service-argocd -o template --template={{.spec.host}}
```

* Determine the password for the 'admin' user

```
kubectl get secret gitops-service-argocd-cluster -n gitops-service-argocd -o=jsonpath='{.data.admin\.password}' | base64 -d
```

Navigate to the URL found above and use *admin* as the user and the *password* from above.

See the [GitOps Service M2 Demo script for more details](https://github.com/redhat-appstudio/managed-gitops/tree/main/examples/m2-demo#run-the-demo).
