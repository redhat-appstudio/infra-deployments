# Disaster Recovery

This component contains automations to test that Disaster Recovery processes work as expected.

## Install Locally

To install locally the automation you can use the following script.

```bash
# Create a kind cluster
kind create cluster

# Install Tekton
kubectl apply --filename https://storage.googleapis.com/tekton-releases/pipeline/latest/release.yaml
kubectl wait --for=condition=Available deployment --all --timeout 300s -n tekton-pipelines
kubectl wait --for=condition=Available deployment --all --timeout 300s -n tekton-pipelines-resolvers

# Install Tekton Triggers
kubectl apply --filename \
  https://storage.googleapis.com/tekton-releases/triggers/latest/release.yaml
kubectl apply --filename \
  https://storage.googleapis.com/tekton-releases/triggers/latest/interceptors.yaml
kubectl wait --for=condition=Available deployment --all --timeout 300s -n tekton-pipelines

# Apply manifests
kustomize build components/disaster-recovery/development/ | kubectl apply -f -
```
