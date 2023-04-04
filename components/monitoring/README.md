## Onboarding Services to Monitoring

### 1. Metrics-exporting services

- The intended service should export the metrics from the application so that prometheus is able to understand it.

- For reference, see
  [Writing Exporters](https://prometheus.io/docs/instrumenting/writing_exporters)

- The exported port, service and route should be accessible to the prometheus service.

- Check out [this example](./prometheus/development/dummy-service.yaml) of a
  metrics-exporting service.

#### 2. Service monitors

Creating a service monitor instructs Prometheus to create a new target to collect
metrics from.

Copy and modify
[this example](./prometheus/development/dummy-service-service-monitor.yaml)
for adding the service monitor declaration:

- The service monitor should be defined under the component which it monitors. Copy the
  example under your [component](../../components/) and reference it in your
  kustomization.yaml file.

- Namespace: the service monitor should be defined under the same namespace as the
  service it monitors. Same goes for the namespaces for the rest of the resources
  defined for the service monitor. Namely, service, servicemonitor and the
  servicemonitor's serviceaccount and secret.

- ClusterRole and ClusterRoleBinding: make sure you edit the cluster role and cluster
  role binding definitions so their names are unique.

- ServiceMonitor: Verify the validity of the service monitor's selector. For example,
  it can be matching a label - make sure you specify your app's label appropriately
  (e.g. `app: my-app`, `control-plane: controller-manager`).

> **_IMPORTANT:_** make sure your service's namespace does NOT contain label
                   `openshift.io/cluster-monitoring: 'true'`. Otherwise, it will not be
                   monitored by the user workload Prometheus instance.

#### 3. Grafana dashboards

A dashboard is a set of one or more panels organized and arranged into one or more rows. It visualizes results from multiple data sources simultaneously.
New dashboards can be added through the user interface, preconfigured in infra-deployment project, or imported from other projects.

##### Manual export

  - Create a new folder in [grafana](https://grafana-appstudio-workload-monitoring.apps.appstudio-stage.x99m.p1.openshiftapps.com) for your service. (In the left nav, + Create Folder)

  - [Create a dashboard](https://grafana.com/docs/grafana/v9.0/dashboards/) for your team's view of your service's Service Level Indicators. (After navigating to your folder, + Create Dashboard)

  - Add tiles on the dashboard to track your initial set of service level indicators. If you correctly added your servicemonitor to the stage Prometheus datasource, it will show up in the Query list when you edit a tile.

  - Export the dashboard definition in JSON format. (At the top of the screen, the icon with 3 dots lets you "Share dashboard or panel". Select Export... Save to file.)

  -  Store the dashboard definition in git, in infra-deployments. 

  - Add your dashboard to the [kustomization file](https://github.com/redhat-appstudio/infra-deployments/blob/main/components/monitoring/grafana/base/kustomization.yaml#L15) to automatically add it to future deployments.

  - [Here](https://github.com/redhat-appstudio/infra-deployments/blob/main/components/monitoring/grafana/base/dashboards/example.json) is an example for the default dashboard.

##### In place configuration

  - Add dashboard json to the [dashboards](https://github.com/redhat-appstudio/infra-deployments/tree/main/components/monitoring/grafana/base/dashboards) folder

  - Add dashboard's file name to [kustomization.yaml](https://github.com/redhat-appstudio/infra-deployments/blob/main/components/monitoring/grafana/base/kustomization.yaml#L15)
    ```yaml
    configMapGenerator:
    - name: grafana-dashboard-definitions
      files:
      - example.json=dashboards/example.json
    ```

##### External configuration
  - Create in your project `kustomization.yaml` which will provide a configmaps with Grafana dashboards.
    ```yaml
    kind: Kustomization
    apiVersion: kustomize.config.k8s.io/v1beta1
    
    generatorOptions:
      disableNameSuffixHash: true
    
    
    configMapGenerator:
      - name: grafana-dashboard-spi-health
        files:
          - grafana-dashboards/spi-health.json
      - name: grafana-dashboard-spi-outbound-traffic
        files:
          - grafana-dashboards/spi-outbound-traffic.json
      - name: grafana-dashboard-spi-slo
        files:
          - grafana-dashboards/spi-slo.json
    ```

  - Include a reference to this dashboard in [kustomization.yaml](https://github.com/redhat-appstudio/infra-deployments/blob/f0d3956f4a11d25291e91773d74b5942ce943f39/components/monitoring/grafana/base/spi/kustomization.yaml#L4)
    ```yaml
    apiVersion: kustomize.config.k8s.io/v1beta1
    kind: Kustomization
    resources:
    - https://github.com/redhat-appstudio/service-provider-integration-operator/config/monitoring/base?ref=8456502ae3a4dca0688bc70abfac2db58ee8acb4
    ```
  - Ensure that project's kustomization.yaml is included in [grafana/base/kustomization.yaml](https://github.com/redhat-appstudio/infra-deployments/blob/main/components/monitoring/grafana/base/kustomization.yaml)
  ```yaml
     apiVersion: kustomize.config.k8s.io/v1beta1
     kind: Kustomization
     resources:
     ...
       - spi/
     ...
  ```
  - Note: to keep the `ref={id}` up to date such a configuration of PipelineRun can be used
    ```yaml
    apiVersion: tekton.dev/v1beta1
    kind: PipelineRun
    metadata:
      name: spi-controller-on-push
    annotations:
      pipelinesascode.tekton.dev/on-event: "[push]"
      pipelinesascode.tekton.dev/on-target-branch: "[main]"
      pipelinesascode.tekton.dev/max-keep-runs: "5"
    spec:
      params:
      ...
      - name: infra-deployment-update-script
        value: |
         sed -i -e 's|\(https://github.com/redhat-appstudio/service-provider-integration-operator/config/monitoring/base?ref=\)\(.*\)|\1{{ revision }}|' components/monitoring/grafana/base/spi/kustomization.yaml
        pipelineRef:
         name: docker-build
         bundle: quay.io/redhat-appstudio/hacbs-core-service-templates-bundle:latest
      ...
    ```

  - Add a [volume](https://github.com/redhat-appstudio/infra-deployments/blob/2a6e4dcb272fa01d330b393d87b8f5ea5c434687/components/monitoring/grafana/base/grafana-app.yaml#L175) to `grafana-app.yaml` 
    ```yaml
    - name: grafana-dashboard-dora-metrics-volume
      projected:
       sources:
         - configMap:
             name: grafana-dashboard-dora-metrics
     ```

  - Add a [volumeMounts](https://github.com/redhat-appstudio/infra-deployments/blob/2a6e4dcb272fa01d330b393d87b8f5ea5c434687/components/monitoring/grafana/base/grafana-app.yaml#L87) to `grafana-app.yaml`
    ```yaml
    volumeMounts:
    - mountPath: /var/lib/grafana/dashboards-dora-metrics
      name: grafana-dashboard-dora-metrics-volume
    ```

  - Add a link to the new maps folder in `appstudio-workload-monitoring` [ConfigMap](https://github.com/redhat-appstudio/infra-deployments/blob/2a6e4dcb272fa01d330b393d87b8f5ea5c434687/components/monitoring/grafana/base/grafana-app.yaml#L234) in  `grafana-app.yaml`
    ```yaml
    apiVersion: v1
    kind: ConfigMap
    metadata:
      namespace: appstudio-workload-monitoring
      name: grafana-dashboards
    data:
      default.yaml: |
        apiVersion: 1
        providers:
          - name: Dora Metrics
            folder: QE
            type: file
            disableDeletion: true
            options:
              path: /var/lib/grafana/dashboards-dora-metrics
    ```
  
  - Note: Grafana dashboards has to have a predefined datasource name. It is recommended to use templating to select them. For example:
    ```json
      "templating": {
        "list": [
          {
            "current": {
              "selected": true,
              "text": "prometheus-appstudio-ds",
              "value": "prometheus-appstudio-ds"
            },
            "hide": 0,
            "includeAll": false,
            "multi": false,
            "name": "datasource",
            "options": [],
            "query": "prometheus",
            "queryValue": "",
            "refresh": 1,
            "regex": ".*-(appstudio)-.*",
            "skipUrlSync": false,
            "type": "datasource"
          }
        ]
      },
    ```
    In this example dashboard will use `prometheus-appstudio-ds` datasource, with ability to use other datasource that contains `appstudio` in the name.
    

