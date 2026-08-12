# Disaster Recovery

This component contains automations to test that Disaster Recovery processes work as expected.

## Prerequisites

- [kind](https://kind.sigs.k8s.io/)
- [kubectl](https://kubernetes.io/docs/tasks/tools/)
- [kustomize](https://kubectl.docs.kubernetes.io/installation/kustomize/)

## Install Locally

To install locally the automation you can use the following steps.

Create a kind cluster:

```bash
kind create cluster
```

Verify you are using the correct k8s context:

```bash
kubectl cluster-info
```

If needed, switch the k8s context:

```bash
kubectl config use-context kind-kind
```

Install Tekton:

```bash
kubectl apply --filename https://storage.googleapis.com/tekton-releases/pipeline/latest/release.yaml
```

Wait for Tekton pipelines to be ready:

```bash
kubectl wait --for=condition=Available deployment --all --timeout 300s -n tekton-pipelines
```

```bash
kubectl wait --for=condition=Available deployment --all --timeout 300s -n tekton-pipelines-resolvers
```

Install Tekton Triggers:

```bash
kubectl apply --filename https://storage.googleapis.com/tekton-releases/triggers/latest/release.yaml
```

```bash
kubectl apply --filename https://storage.googleapis.com/tekton-releases/triggers/latest/interceptors.yaml
```

Wait for Tekton Triggers to be ready:

```bash
kubectl wait --for=condition=Available deployment --all --timeout 300s -n tekton-pipelines
```

Apply manifests:

```bash
kustomize build components/disaster-recovery/development/ | kubectl apply -f -
```

## Test Locally

The CronJob is suspended in the development overlay. To trigger a pipeline run manually, forward the event listener and send a request:

```bash
kubectl port-forward -n konflux-disaster-recovery services/el-cron-listener 8080:8080
```

```bash
curl -X POST --data '{}' localhost:8080
```

### Tekton Dashboard (optional)

Install the [Tekton Dashboard](https://tekton.dev/docs/dashboard/) to monitor pipelines and tasks from a web UI:

```bash
kubectl apply --filename https://infra.tekton.dev/tekton-releases/dashboard/latest/release.yaml
```

```bash
kubectl wait --for=condition=Available deployment tekton-dashboard --timeout 300s -n tekton-pipelines
```

Forward the dashboard to your local machine:

```bash
kubectl port-forward -n tekton-pipelines service/tekton-dashboard 9097:9097
```

Then open http://localhost:9097 in your browser.

## Cleanup

Delete the kind cluster when you are done:

```bash
kind delete cluster
```
