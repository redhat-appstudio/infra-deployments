
## Installing and configuring Kanary on a data-plane cluster

Note: The steps below should be handled by Argo CD

- Create the `appstudio-kanary-exporter` namespace on the clusters, if it does not exist yet:

    ```
    $ oc create namespace appstudio-kanary-exporter
    ```

- Create the `base` resources by running the following command:

    ```
    $ kustomize build components/monitoring/grafana/base | oc apply -f -   
    ```