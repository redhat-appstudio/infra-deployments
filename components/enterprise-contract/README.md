---
title: Enterprise Contract
---

Enterprise Contract, sometimes abbreviated as "EC", is a set of tools for
applying and maintaining policies about container builds created by Stonesoup.

Its purpose is to ensure Stonesoup releases meet a set of clearly defined
requirements before being considered releaseable. Additionally it confirms the
container images are securely signed and have signed attestations verifying
how they were built.

The Stonesoup release pipeline is expected to include a task that runs the
Enterprise Contract against the set of container images being released. If
there are any policy failures then the release will not proceed. The
Enterprise Contract task will also be able to run automatically when a build
pipeline completes to show pass/fail results and detailed reasons for any
failures.

For additional information about Enterprise Contract, see the following
resources:

* [Enterprise Contract Documentation](https://enterprise-contract.github.io/)
* [Book of Stonesoup](https://redhat-appstudio.github.io/book/book/enterprise-contract.html)
* [Enterprise Contract GitHub Organization](https://github.com/enterprise-contract)
