# App Studio Build System

The App Studio Build System is composed of the following 
components:

1. OpenShift Pipelines. 
2. Pipeline Definitions for building images present in `build-templates`.
3. AppStudio-specific `ClusterTasks`.


## Usage

As a non-admin user, one would have access to the Pipeline definitions in `build-templates`. To be able to use them in my personal namespace, the UI or the client would need to run the equivalent of the following command:

```
oc get pipelines -n build-templates -o yaml | sed 's/namespace: .*/namespace: YOUR-NAMESPACE/' | oc apply -f -
```