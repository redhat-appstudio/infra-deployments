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

#### 4. Grafana dashboards - manual export

  - Create a new folder in [grafana](https://grafana-appstudio-workload-monitoring.apps.appstudio-stage.x99m.p1.openshiftapps.com) for your service. (In the left nav, + Create Folder)

  - [Create a dashboard](https://grafana.com/docs/grafana/v9.0/dashboards/) for your team's view of your service's Service Level Indicators. (After navigating to your folder, + Create Dashboard)

  - Add tiles on the dashboard to track your initial set of service level indicators. If you correctly added your servicemonitor to the stage Prometheus datasource, it will show up in the Query list when you edit a tile.

  - Export the dashboard definition in JSON format. (At the top of the screen, the icon with 3 dots lets you "Share dashboard or panel". Select Export... Save to file.)

  -  Store the dashboard definition in git, in infra-deployments. 

  - Add your dashboard to the [kustomization file](https://github.com/redhat-appstudio/infra-deployments/blob/main/components/monitoring/grafana/base/kustomization.yaml#L15) to automatically add it to future deployments.

  - [Here](https://github.com/redhat-appstudio/infra-deployments/blob/main/components/monitoring/grafana/base/dashboards/example.json) is an example for the default dashboard.
  
    

