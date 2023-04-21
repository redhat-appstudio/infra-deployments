---
title: Installing and configuring Prometheus on the data plane clusters
---

We use the Openshift provided Prometheus deployments, platform and user-workload-monitoring (UWM).
Custom metrics provided by the service deployed by the different teams should be scraped by the
UWM Prom, while generic metrics (produced for example by cAdvisor, kube-state-metrics, etc...)
will be scraped by the Platform Prom.


In Production and Staging, UWM is enabled using OCM (since the Prom config is controlled by it).
In Development it's enabled by deploying a configmap using ArgoCD.
