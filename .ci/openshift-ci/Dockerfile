FROM registry.ci.openshift.org/openshift/release:golang-1.18

ENV YQ_VERSION=v4.25.2

SHELL ["/bin/bash", "-c"]

# Install yq, kubectl, kustomize
RUN curl -LO https://storage.googleapis.com/kubernetes-release/release/$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/linux/amd64/kubectl && \
    chmod +x ./kubectl && \
    mv ./kubectl /usr/local/bin
RUN curl -Lso /usr/local/bin/yq https://github.com/mikefarah/yq/releases/download/${YQ_VERSION}/yq_linux_amd64 && chmod +x /usr/local/bin/yq
