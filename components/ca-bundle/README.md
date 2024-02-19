# CA Bundle

The purpose of this component if to deploy a custom CA bundle to the cluster.
More information about how OCP handles it can be found [here](https://docs.openshift.com/container-platform/4.14/security/certificates/updating-ca-bundle.html)

To use the injected CA bundle, please reffer to [Certificate injection using Operators](https://docs.openshift.com/container-platform/4.14/networking/configuring-a-custom-pki.html#certificate-injection-using-operators_configuring-a-custom-pki)


**Note**: there is a process in OCP that copies the `user-ca-bundle
` ConfiMap from the `openshift-config` namespace to the 
`openshift-controller-manager` namespace and renames it to `openshift-user-ca`. The labels are and annotation are also copied. Since Argo is using a label to track the resources it created, it thinks it owns the copy and warns that it't not part of the source code.
The issue and solution are described [here](https://github.com/argoproj/argo-cd/issues/5792#issuecomment-800940513).
