# Infra Deployments CI documentation

Currently in infra-deployments all tests are running in Openshift CI [Openshift CI](https://prow.ci.openshift.org/?job=*infra*deployments*) for each PR, before merging it.

## Openshift CI

Openshift CI is a Kubernetes based CI/CD system. Jobs can be triggered by various types of events and report their status to many different services. In addition to job execution, Openshift CI provides GitHub automation in a form of policy enforcement, chat-ops via /foo style commands and automatic PR merging.

All documentation about how to onboard components in Openshift CI can be found in the Openshift CI jobs [repository](https://github.com/openshift/release). All infra-deployments jobs configurations are defined in https://github.com/openshift/release/tree/master/ci-operator/config/redhat-appstudio/infra-deployments.

- `appstudio-e2e-deployment` Run all e2e test suites of Red Hat App Studio defined in the [e2e-tests](https://github.com/redhat-appstudio/e2e-tests) repository.

The test container to run the e2e tests in Openshift CI is built from: https://github.com/redhat-appstudio/infra-deployments/blob/main/.ci/openshift-ci/Dockerfile
