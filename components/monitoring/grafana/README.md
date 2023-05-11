
## Installing and configuring Grafana on the control-plane cluster

We use Grafana Operator to create all needed services and routes 

Note: The steps below should be handled by Argo CD

- Create the `appstudio-grafana` namespace on each Prometheus or Grafana cluster, if it does not exist yet:

    ```
    $ oc create namespace appstudio-grafana
    ```

- Create the `base` resources by running the following command:

    ```
    $ kustomize build components/monitoring/grafana/base | oc apply -f -   
    ```