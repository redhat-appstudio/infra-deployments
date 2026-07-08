# Unified CI image for appstudio-operator-overlay-e2e-tests (install + e2e steps).
# Built on the fly by OpenShift CI; see openshift/release
# ci-operator/config/redhat-appstudio/infra-deployments/.
#
# - task-runner: oc, kubectl, jq, yq, git, skopeo, oras, buildah, … (bootstrap/preview)
# - ubi10/go-toolset @ digest: Go 1.26.4 for konflux-ci conformance (go test)

FROM registry.access.redhat.com/ubi10/go-toolset@sha256:120c56d15943f96f84c7d415432fc82a9212ea58f6cbebf516587fb854d8631a AS go-toolset

FROM quay.io/konflux-ci/task-runner@sha256:1abfe4e50d4e961d0fd9790202565f93ee650fe8dfc50932c94989acba10485f

USER 0

# Go toolchain (UBI10); task-runner runtime does not ship `go` on PATH.
COPY --from=go-toolset /usr/lib/golang /usr/lib/golang
COPY --from=go-toolset /usr/bin/go /usr/bin/go
COPY --from=go-toolset /usr/bin/gofmt /usr/bin/gofmt

ENV GOROOT=/usr/lib/golang
ENV PATH=/usr/lib/golang/bin:/usr/local/bin:/usr/bin:/sbin:/bin

USER 1000
