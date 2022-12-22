---
title: Internal Services
---

As part of HACBS-1443, The Release team has developed an operator that is designed to run on a private, internal cluster
provided by AppSRE. This is called the Internal Services Controller.

Please see https://github.com/redhat-appstudio/book/blob/main/ADR/0003-interacting-with-internal-services.md

This operator will reach out to a public *StoneSoup* cluster to watch for InternalRequest CRs and execute internal pipelines

This folder permits the objects needed for the operator to watch the CRs (ServiceAccount, Role, RoleBindings)

Once deployed, an admin can retrieve the token and securely provide it to the admins of the Internal Services Controller.
