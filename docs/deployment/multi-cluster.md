---
title: Multi-Cluster Deployment
---

For handling the scale required for production usage, a multi-cluster deployment of StoneSoup
should be used. 

In this document, the term *fleet* will be used to refer to all the clusters.

Each cluster which is part of the fleet can have the following roles:

**ArgoCD**: Used for hosting an ArgoCD instance that will deploy applications to the entire fleet (including itself). In a fleet only one cluster will have this role.

**Member**: Used for running the StoneSoup controllers, host
the user's data, and pipelines. Usually, a fleet will have
more than one member.

**Host**: Used an entrypoint for the service. Proxies API request and Webhooks to the member clusters.

**Note**: The same cluster can be used as a `Host` and for running ArgoCD.


## Secret Management

TBA


## Deploying ArgoCD

TBA

## Deploying the Host cluster

TBA

## Deploying a Member cluster

TBA