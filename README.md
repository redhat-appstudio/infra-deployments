# StoneSoup Infrastructure Deployments

For the full documentation click [here](https://redhat-appstudio.github.io/infra-deployments/docs/introduction/about.html)

When working on documentation updates, you may want to enable GitHub pages for your forked repository,
see the [GitHub docs: Publishing from a branch](https://docs.github.com/en/pages/getting-started-with-github-pages/configuring-a-publishing-source-for-your-github-pages-site#publishing-from-a-branch).
The site will be available at `https://<username>.github.io/<repository>` (`https://<username>.github.io/infra-deployments` if you didn't rename yor fork).

This repository is an initial set of Argo-CD-based deployments of StoneSoup components to a cluster, plus a script to bootstrap Argo CD onto that cluster (to drive these Argo-CD-based deployments, via OpenShift GitOps).

This repository is structured as a GitOps monorepo (e.g. the repository contains the K8s resources for *multiple* applications), using [Kustomize](https://kustomize.io/).

The contents of this repository are not owned by any single individual, and should instead be collectively managed and maintained through PRs by individual teams. More information about that can be found in the documentation section about how to [Extend The Service](https://redhat-appstudio.github.io/infra-deployments/docs/deployment/extending-the-service.html).
