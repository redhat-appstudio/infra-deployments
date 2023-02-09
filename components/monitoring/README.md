## On boarding to services to Monitoring

### 1. Metrics exporter services

  - The intended service should export the metrics from the application so that prometheus is able to understand it. 

  - For reference, see 
    [Writing Exporters](https://prometheus.io/docs/instrumenting/writing_exporters)

  - Exported port, service, route should be accessible to prometheus service.

  - [Here](https://github.com/redhat-appstudio/service-provider-integration-operator/blob/main/config/rbac/auth_proxy_service.yaml) is an example for the spi-system

```yaml
apiVersion: v1
kind: Service
metadata:
  labels:
    control-plane: controller-manager
  name: controller-manager-metrics-service
  namespace: system
spec:
  ports:
  - name: metrics
    port: 8443
    protocol: TCP
    targetPort: https
  selector:
    control-plane: controller-manager
    app.kubernetes.io/name: service-provider-integration-operator
```

#### 2. Service monitors

  - Adding the servicemonitor declaration

    - If servicemonitor is for prometheus itself

      - Add the servicemonitor declaration for scraping the prometheus service

      - [Here](https://github.com/redhat-appstudio/infra-deployments/blob/main/components/monitoring/prometheus/base/servicemonitors/prometheus.yaml) is an example servicemonitor for prometheus itself
      
      ```yaml
      apiVersion: monitoring.coreos.com/v1
      kind: ServiceMonitor
      metadata:
        namespace: appstudio-workload-monitoring
        name: prometheus
        labels:
          prometheus: appstudio-workload
      spec:
        endpoints:
        - bearerTokenSecret:
            key: ""
          interval: 15s
          path: /metrics
          port: oauth2-proxy
          scheme: HTTPS
          tlsConfig:
            caFile: /etc/prometheus/tls/tls.crt 
            serverName: "prometheus-oauth2.appstudio-workload-monitoring.svc"
        namespaceSelector:
          matchNames:
          - appstudio-workload-monitoring
        selector:
          matchLabels:
            app.kubernetes.io/instance: monitoring-workload-in-cluster
      ```
     
      

  - If the servicemonitor is for getting other components added to prometheus monitoring
      - create a new file under [components/monitoring/prometheus/base/servicemonitors](https://github.com/redhat-appstudio/infra-deployments/blob/main/components/monitoring/prometheus/base/servicemonitors)
      - Add an entry into the [components/monitoring/prometheus/base/servicemonitors/kustomization.yaml](https://github.com/redhat-appstudio/infra-deployments/blob/main/components/monitoring/prometheus/base/servicemonitors/kustomization.yaml)
      - Add a ServiceMonitor along with a ClusterRoleBinding to scrape the intended service
      - [Here](https://github.com/redhat-appstudio/infra-deployments/blob/main/components/monitoring/prometheus/base/servicemonitors/spi-operator.yaml) is an example for servicemonitor
      
          ```yaml
          apiVersion: monitoring.coreos.com/v1
          kind: ServiceMonitor
          metadata:
            namespace: appstudio-workload-monitoring
            name: spi-operator <name of the servicemonitor>
            labels:
              prometheus: appstudio-workload
          spec:
            endpoints:
            - bearerTokenSecret:
                name: <secret name for prometheus sa>
                key: token
              scheme: https
              tlsConfig:
                insecureSkipVerify: true
              interval: 15s
              path: /metrics
              port: metrics <port for metrics exporter svc>
            namespaceSelector:
              matchNames:
              - spi-system
            selector:
              matchLabels:
                control-plane: "controller-manager"
          ```

      - Note: The namespace of the ServiceMonitor matches the namespace for the prometheus service, in this case, `appstudio-workload-monitoring`.

  - It should have the accessible port and route to the service (or service url)

  - Access token or service accounts as required.

  - Use label selectors to select the desired service uniquely in the cluster.

### 3. View access to the exporter service for Prometheus

  - Prometheus should have view access to the metrics exporter service namespace

  - Add the Rolebinding to give prometheus view access in the same servicemonitor file. 
  - [Here](https://github.com/redhat-appstudio/infra-deployments/blob/main/components/monitoring/prometheus/base/servicemonitors/spi-operator.yaml) is an example providing prometheus view access to the cluster

  ```yaml
  apiVersion: rbac.authorization.k8s.io/v1
  kind: ClusterRoleBinding
  metadata:
    name: prometheus-spi-metrics-reader
  roleRef:
    apiGroup: rbac.authorization.k8s.io
    kind: ClusterRole
    name: spi-metrics-reader
  subjects:
  - kind: ServiceAccount
    name: prometheus-k8s
    namespace: appstudio-workload-monitoring
  ```

#### 4. Grafana dashboards

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
    

