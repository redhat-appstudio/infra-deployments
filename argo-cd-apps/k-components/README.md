# When to use a Specific k-component

This document discusses the various available k-components, what they do, and when to use them.

## Development

### inject-infra-deployments-repo-details

Injects the infra-deployments repo and target revision into a component's `ApplicationSet` and generated `Application`(s). When the `hack/preview.sh` script is run in e2e tests, this k-component is overwritten to use the PR author's fork repo and the temporary preview branch the script has generated. This should be used in a component's `rd-dev` overlay Kustomize file when the component is defined in the infra-deployments repository so that e2e tests use the changes to the component in a PR's branch.

### inject-internal-infra-deployments-repo-details

Injects the internal-infra-deployments repo and target revision (main branch) into a component's `ApplicationSet` and generated `Application`(s). This should be used in a component's `rd-dev` overlay Kustomize file when the component is defined in the internal-infra-deployments repository.

## Staging

### deploy-to-all-staging-clusters
Populates the appropriate cluster and list generators to a component's `ApplicationSet` that should be deployed to *all* staging clusters: tenant, EaaS, hub , and spoke. This is automatically applied in the rd-staging overlay to any `ApplicationSet` that has the `appstudio.redhat.com/deployment-clusters=all` label.

### deploy-to-staging-eaas-clusters

Populates the appropriate cluster and list generators to a component's `ApplicationSet` that should be deployed to EaaS staging cluster(s). This is automatically applied in the rd-staging overlay to any `ApplicationSet` that has the `appstudio.redhat.com/deployment-clusters=eaas` label.

### deploy-to-staging-all-tenant-clusters

Populates the appropriate cluster and list generators to a component's `ApplicationSet` that should be deployed to *all* tenant staging clusters (including hub & spoke clusters). This is automatically applied in the rd-staging overlay to any `ApplicationSet` that has the `appstudio.redhat.com/deployment-clusters=tenant` and does not have the `appstudio.redhat.com/internal-only=true` nor the `appstudio.redhat.com/hub-only=true` labels.

### deploy-to-staging-internal-tenant-clusters

Populates the appropriate cluster and list generators to a component's `ApplicationSet` that should be deployed to internal tenant staging clusters (including hub & spoke clusters). This is automatically applied in the rd-staging overlay to any `ApplicationSet` that has the `appstudio.redhat.com/deployment-clusters=tenant` and `appstudio.redhat.com/internal-only=true` labels and does not have the `appstudio.redhat.com/hub-only=true` label.

### deploy-to-staging-internal-tenant-and-hub-clusters

Populates the appropriate cluster and list generators to a component's `ApplicationSet` that should be deployed to internal tenant staging clusters (including hub clusters, *not* spoke clusters). This is automatically applied in the rd-staging overlay to any `ApplicationSet` that has the `appstudio.redhat.com/deployment-clusters=tenant`, `appstudio.redhat.com/hub-only=true`, and `appstudio.redhat.com/internal-only=true` labels.

### deploy-to-staging-tenant-and-hub-clusters

Populates the appropriate cluster and list generators to a component's `ApplicationSet` that should be deployed to tenant staging clusters (including hub clusters, *not* spoke clusters). This is automatically applied in the rd-staging overlay to any `ApplicationSet` that has the `appstudio.redhat.com/deployment-clusters=tenant` and `appstudio.redhat.com/hub-only=true` labels and does not have the `appstudio.redhat.com/internal-only=true` label.

## Production

### deploy-to-all-production-clusters

Populates the appropriate cluster and list generators to a component's `ApplicationSet` that should be deployed to *all* production clusters: tenant and EaaS. This is automatically applied in the rd-production overlay to any `ApplicationSet` that has the `appstudio.redhat.com/deployment-clusters=all` label.

### deploy-to-production-eaas-clusters

Populates the appropriate cluster and list generators to a component's `ApplicationSet` that should be deployed to EaaS production cluster(s). This is automatically applied in the rd-production overlay to any `ApplicationSet` that has the `appstudio.redhat.com/deployment-clusters=eaas` label.

### deploy-to-production-internal-tenant-clusters

Populates the appropriate cluster and list generators to a component's `ApplicationSet` that should be deployed to internal tenant production clusters. This is automatically applied in the rd-production overlay to any `ApplicationSet` that has the `appstudio.redhat.com/deployment-clusters=tenant` and `appstudio.redhat.com/internal-only=true` labels.

### deploy-to-production-tenant-clusters

Populates the appropriate cluster and list generators to a component's `ApplicationSet` that should be deployed to tenant production clusters. This is automatically applied in the rd-production overlay to any `ApplicationSet` that has the `appstudio.redhat.com/deployment-clusters=tenant` label.
