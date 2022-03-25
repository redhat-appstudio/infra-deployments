# Configuring NFS storage provisioner on Quicklab clusters 

###### Requiremements 

This script can only be executed once:
 
- your cluster has been provisioned.
- openshift has been installed.

###### Input

This script takes 1 parameter - the UPI hostname in Quicklab.
It can be found in the Cluster Information section.

```
Cluster Information
Username: quicklab
Hosts:
 - upi-0.shebertquick.lab.upshift.rdu2.redhat.com ( 10.0.91.104 )
``` 

The script uses this info to obtain credentials in order to configure the storage on your cluster

###### Usage

`# setup-nfs-quicklab.sh upi-0.shebertquick.lab.upshift.rdu2.redhat.com`

###### Testing

Once the scripts executes, run the following command to test the setup:

```
% export KUBECONFIG=/tmp/kubeconfig
% oc new-project test-pvc
% oc create -f templates/test-pvc.yaml
% oc get pvc
```

The status should be *Bound*

```
NAME         STATUS   VOLUME                                     CAPACITY   ACCESS MODES   STORAGECLASS          AGE
test-claim   Bound    pvc-1392e83f-60ec-4caf-b71b-8d040b8becd5   100Mi      RWX            managed-nfs-storage   6s
```

You may now run the bootstrap scripts.