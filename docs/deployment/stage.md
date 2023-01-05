---
title: Staging Environment
---

Currently all the components are deployed to a single cluster using ArgoCD.
Components are declared using [ArgoCD Applications](../../argo-cd-apps/base/), and synced periodically by it to
the Staging cluster.  

Having access to the staging cluster and ArgoCD can help a component maintainer
ensuring its component were properly deployed and debug it in cases they are not functioning properly,
for example, by accessing the logs of of the component's pods.  

If you are not a component maintainer, you probably won't need
access to the staging cluster.

## Accessing the Staging Cluster

### Getting Permissions

In order to connect to the staging cluster you should be part of the 
[stage](https://github.com/orgs/redhat-appstudio-sre/teams/stage) team in [redhat-appstudio-sre](https://github.com/redhat-appstudio-sre) organization.

For getting an invite for the team, please send a request on [Slack](https://coreos.slack.com/archives/C04F4NE15U1)
and explain why you need the permissions.

After you join the team, you should be able to connect to the [cluster's UI](https://console-openshift-console.apps.appstudio-stage.x99m.p1.openshiftapps.com/).

## Accessing the ArgoCD instance

Once you have access to the stage cluster, you can access ArgoCD
by clicking [here](https://openshift-gitops-server-openshift-gitops.apps.appstudio-stage.x99m.p1.openshiftapps.com/)