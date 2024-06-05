--- 

OpenTelemetry Collector & Konflux Configuration

# *tracing* Component

The otel-collector is installed and configured in the tracing component.  Within base, the helm chart is defined, and the values file is recursively called back into the infra-deployments repository based on the environment.  

**Flow of tracing deployment**
***app-of-app-sets***
ArgoCD deploys an initial `alll-application-sets` application, which spawns the rest of the applications. The initial entry point for tracing is here - 

* ./argo-cd-apps/base/all-clusters/infra-deployments/tracing-workload-otel-collector/tracing-workload-otel-collector.yaml  

Kustomize templates and overrides settings based on the environment.

***tracing-workload-otel-collector***
An application starting with the name `tracing-workload-otel-collector` is created based on the content in `components/tracing/otel-collector`.  ArgoCD and Kustomize will select the appropriate environment to start from (ie, `development` and `staging`) and will apply settings on top of the content in `base`. 

Two application sets are currently created, `otel-collector` and `enable-tekton-tracing`.  

* `otel-collector`
  The `otel-collector` ApplicationSet is the actual OpenTelemetry collector deployed by the Helm chart from the OpenTelemetry project.   The basic definition 

       The configuration for the OpenTelemetry Collector is passed in a valuesFiles item            to the Helm chart.  It refers back to the `infra-deployment` git repo, and selects              the proper configuartion based on the environment. 

* `enable-tekton-tracing`
  
  This ApplicationSet contains only a change to the Tekton Pipelines tracing ConfigMap to enable tracing and configure the endpoint to be the local service provided by the OpenTelemetry collector deployment. 

***
