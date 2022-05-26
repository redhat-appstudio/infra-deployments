FROM quay.io/redhat-appstudio/e2e-tests:latest as e2e
FROM registry.ci.openshift.org/openshift/release:golang-1.17

SHELL ["/bin/bash", "-c"]

# Get the appstudio e2e binary from https://github.com/redhat-appstudio/e2e-tests/blob/main/Dockerfile#L11
COPY --from=e2e /root/e2e-appstudio /usr/local/bin

# Install yq, kubectl, kustomize
RUN wget https://github.com/mikefarah/yq/releases/download/v4.20.2/yq_linux_amd64 -O /usr/local/bin/yq && \
    chmod +x /usr/local/bin/yq && yq --version && \
    curl -LO https://storage.googleapis.com/kubernetes-release/release/$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/linux/amd64/kubectl && \
    chmod +x ./kubectl && \
    mv ./kubectl /usr/local/bin && \
    curl -s "https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh"  | bash && \
    cp kustomize /usr/bin/ && \
    chmod ug+x /usr/bin/kustomize && \
    chmod +x /usr/local/bin/e2e-appstudio
