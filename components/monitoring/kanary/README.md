
## Installing and configuring Kanary on a data-plane cluster

Note: The steps below should be handled by Argo CD

- Create the `appstudio-kanary-exporter` namespace on the clusters, if it does not exist yet:

    ```
    $ oc create namespace appstudio-kanary-exporter
    ```

- Create the Grafana resources by running the following command:

    ```
    $ kustomize build components/monitoring/grafana/development | oc apply -f -   
    ```

## S3 artifacts

Archive filenames start with `passed-`, `failed-`, or `unknown-`,
using the probe's `test-outcome` result. Invalid outcome values become `unknown`.
The S3 directories remain `run-{probe|simple}/<cluster>/<test-type>/`.
Filter filenames from an S3 listing before downloading failure artifacts;
include `unknown-` archives for incomplete results. Older archives have no
outcome prefix and require inspecting their contents.
