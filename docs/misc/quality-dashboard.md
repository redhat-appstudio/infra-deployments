---
title: Quality Dashboard
---

The purpose of the Quality Dashboard is to provide information that indicates the quality
of the different StoneSoup services. More details can be found here https://github.com/redhat-appstudio/quality-dashboard

The manifests can be found [here](../../components/quality-dashboard/)

By default the frontend is using StoneSoup Staging cluster for backend. If you want to use backend on your cluster you need to set secrets for `rds-endpoint`, `POSTGRES_PASSWORD` and `github-token` in `quality-dashboard` namespace and also push `components/quality-dashboard/frontend/kustomization.yaml`(the route to backend is changed by script `hack/util-set-quality-dashboard-backend-route.sh` in development and preview cluster modes).
