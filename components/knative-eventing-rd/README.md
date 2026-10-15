# Knative Eventing

This installation of Knative Eventing is running with an `eventing-controller` image
and `ApiServerSource` adapter image customized by KubeArchive called `remove-sar`. These
images contain the following changes:

* `eventing-controller`: this image does not submit `SubjectAccessReview` requests
to check that the permissions required by the `ApiServerSource` adapter pod will work.
This prevents a huge load on the K8S API on Konflux, as this process submits around 15k
`SubjectAccessReview` requests. This image is used both in staging and production.

* `ApiServerSource` adapter: this image adds verbosity to the `ApiServerSource` adapter
related to the Kubernetes official tooling using `klog`. The verbosity is set to `4`
which logs the relevant information we needed to debug why we were missing activity
(watchers were disconnected). This image is used only on staging to prevent flooding
the production logging systems.
