###On boarding to services to Monitoring

#1. Metrics exporter services
a. The intended service should export the metrics from the application so that prometheus is able to understand it. https://prometheus.io/docs/instrumenting/writing_exporters/
b. Exported port, service, route should be accessible to prometheus service.

For example https://github.com/redhat-appstudio/service-provider-integration-operator/blob/main/config/rbac/auth_proxy_service.yaml
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

#2. Service monitors
a.
i. If service monitor is for prometheus it self
* Add service monitor declaration for scrapping
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
      caFile: /etc/prometheus/tls/tls.crt # volume mounted in the `prometheus` container of the prometheus pods
      serverName: "prometheus-oauth2.appstudio-workload-monitoring.svc"
  namespaceSelector:
    matchNames:
    - appstudio-workload-monitoring
  selector:
    matchLabels:
      app.kubernetes.io/instance: monitoring-workload-in-cluster
```
* [Here](https://github.com/redhat-appstudio/infra-deployments/blob/main/components/monitoring/prometheus/base/prometheus-servicemonitors.yaml) is an example servicemonitor for prometheus itself
ii. If the service is for getting other components added to prometheus monitoring
* Add a Sevice monitoring decalration for scrapping
```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  labels:
    prometheus: appstudio-workload <label getting discovered by prometheus-operator>
  name: release-service
  namespace: release-service <name of the namespace the service is in>
spec:
  endpoints:
  - path: /metrics
    port: https
    scheme: https
    bearerTokenSecret:
      name: <secret for accessing the endpoint path on the service>
      key: token
    tlsConfig:
      insecureSkipVerify: true
  namespaceSelector:
    matchNames:
    - release-service
  selector:
    matchLabels:
      control-plane: controller-manager
```
* Note the namespace to be the namespace for the intended service itself.
b. It should have the accessible port and route to the service (or service url)
c. Access token or service accounts as required.
b. Use label selectors to select the desired service uniquely in the cluster 

#3. View access to the exporter service for Prometheus
a. Prometheus should have view access to the metrics exporter service namespace
b. Add the Rolebinding to give prometheus view access. [Here](https://github.com/redhat-appstudio/infra-deployments/blob/main/components/monitoring/prometheus/base/prometheus-view.yaml) is an example providing prometheus view access to the cluster
```yaml
kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: prometheus-view
subjects:
- kind: ServiceAccount
  name: prometheus-k8s
  namespace: appstudio-workload-monitoring
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: view
```

4. Grafana dashboards - manual export
a. Create a new folder in grafana for your service. (In the left nav, + Create Folder)
b. Create a dashboard for your team's view on your service's service level indicators. (After navigating to your folder, + Create Dashboard)
c. Add tiles on the dashboard to track your initial set of service level indicators. If you correctly added your servicemonitor to the stage Prometheus datasource, it will show up in the Query list when you edit a tile.
d. Export the dashboard definition in JSON format. (At the top of the screen, the icon with 3 dots lets you "Share dashboard or panel". Select Export... Save to file.)
e.  Store the dashboard definition in git, in infra-deployments. [Here](https://github.com/redhat-appstudio/infra-deployments/blob/main/components/monitoring/grafana/base/dashboards/example.json) is an example for the default dashboard.
