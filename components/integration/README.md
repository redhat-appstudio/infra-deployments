---
title: Integration Service
---

Integration service is a set of different controllers which are responsible to facilitate
the automated custom testing of the components being built by the build pipeline.

It is composed of Pipeline, Snapshot, Scenario and SnapshotEnvironmentBinding controllers
to watch for Component builds, trigger and manage integration testing Tekton Pipelines,
and create releases based on the testing outcome(s).

The Integration tests are defined in the form of IntegrationTestScenarios which are referenced 
as Tekton pipeline definitions as a Tekton bundle or by directly referencing the git 
repository path where the tests are hosted.

Its purpose is to ensure the successful integration of an application and upon success
promote the application content to the user's defined lowest environments.

For additional information about Integration Service, see the following
resources:

* [Integration Service Architecture](https://github.com/redhat-appstudio/book/blob/main/diagrams/integration-service/integration-service-data-flow.jpg)
* [Book of RHTAP](https://redhat-appstudio.github.io/book/book/integration-service.html)
* [API](https://redhat-appstudio.github.io/book/ref/integration-service.html)
* [Integration Service GitHub Organization](https://github.com/redhat-appstudio/integration-service/)
