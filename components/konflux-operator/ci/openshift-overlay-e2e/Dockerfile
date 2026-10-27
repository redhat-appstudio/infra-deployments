# Unified CI image for appstudio-operator-overlay-e2e-tests (install + e2e steps).
# Built on the fly by OpenShift CI; see openshift/release
# ci-operator/config/redhat-appstudio/infra-deployments/.
#
# - task-runner: oc, kubectl, jq, yq, git, skopeo, oras, buildah, … (bootstrap/preview)
# - ubi10/go-toolset @ digest: Go 1.26.10 for konflux-ci conformance (go test)

FROM registry.access.redhat.com/ubi10/go-toolset@sha256:557f358c83892a8b94fb72649a99e1c303a625d240547fb97a9cbc1620e808c0 AS go-toolset

FROM quay.io/konflux-ci/task-runner@sha256:1abfe4e50d4e961d0fd9790202565f93ee650fe8dfc50932c94989acba10485f

USER 0

# Go toolchain (UBI10); task-runner runtime does not ship `go` on PATH.
COPY --from=go-toolset /usr/lib/golang /usr/lib/golang
COPY --from=go-toolset /usr/bin/go /usr/bin/go
COPY --from=go-toolset /usr/bin/gofmt /usr/bin/gofmt

ENV GOROOT=/usr/lib/golang
ENV PATH=/usr/lib/golang/bin:/usr/local/bin:/usr/bin:/sbin:/bin

USER 1000
