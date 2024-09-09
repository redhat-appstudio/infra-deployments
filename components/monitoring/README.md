# Onboarding Services to Monitoring

This document contains guidelines and references for configuring services in the
Konflux clusters to be monitored and graphed.

## 1. Metrics-exporting Services

For a service to be monitored, it needs to instrument Prometheus metrics, or have an
external service that will poll it and generate metrics on its behalf.

For reference, see
[Writing Exporters](https://prometheus.io/docs/instrumenting/writing_exporters)

Prometheus needs to be able to reach the service in order to scrape the metrics it
generates. Check out [this example](./prometheus/development/dummy-service.yaml) of a
metrics-exporting service.

## 2. Service Monitors

Service Monitors (ServiceMonitors) are Kubernetes custom resources that the Prometheus
operator uses for creating scraping configurations for Prometheus pods. Consequently,
Creating a Service Monitor creates a new target for Prometheus to collect metrics from.

Copy and modify
[this example](./prometheus/development/dummy-service-service-monitor.yaml)
for adding the Service Monitor declaration.

Ideally, the Service Monitor should be defined inside the component repository and then
referenced by the infra-deployment [component's](../../components/) configurations.

**Namespace**: the Service Monitor should be defined under the **same namespace** as the
service it monitors. Same goes for the namespaces for the rest of the resources
defined for the Service Monitor. Namely, `service`, `servicemonitor` and the
servicemonitor's `serviceaccount` and `secret`.

**ClusterRole and ClusterRoleBinding**: make sure you edit the cluster role and cluster
role binding definitions so their names are **unique**.

**ServiceMonitor**: Verify the validity of the Service Monitor's selector. For example,
if it's matching a label, make sure you specify your app's label appropriately (e.g.
`app: my-app`, `control-plane: controller-manager`).

> **_IMPORTANT:_** make sure your service's namespace does NOT contain label
                   `openshift.io/cluster-monitoring: 'true'`. Otherwise, it will not be
                   monitored by the user workload Prometheus instance.

## 3. Grafana Datasource

Grafana datasources contain the connection settings to the Prometheus instances.

A single
[datasource (Thanos querier)](https://github.com/redhat-appstudio/infra-deployments/blob/16e48656370dc65dba6471a9f50d745832535723/components/monitoring/grafana/base/grafana-app.yaml#L216),
`appstudio-datasource` is defined and it lets us query metrics from the Platform and
User Workload Monitoring Prometheus.

To use this default datasource any definition of a datasource in the dashboard json file
should be removed or a `templating` should be used.

## 4. Grafana Dashboards

A dashboard is a set of one or more panels organized and arranged into one or more rows. 
It visualizes results from multiple data sources simultaneously.
New dashboards can be added through the user interface, preconfigured in the
infra-deployments repository, or imported from other projects.

### Create a Dashboard

1. [Create a dashboard](https://grafana.com/docs/grafana/v10.4/dashboards/)
   for your team's view of your service's Service Level Indicators
   (After navigating to your folder + Create Dashboard).

   > **_Note:_**  Creating a new dashboard manually is available only for development
                   environment. You may copy and edit the `example` dashboard json
                   instead, and test the new dashboard on the staging and production
                   environments. The `example` dashboard json definition can be found
                   [here](https://github.com/redhat-appstudio/infra-deployments/blob/main/components/monitoring/grafana/base/dashboards/generic-dashboards/example.json)

2. Add tiles to the dashboard to track your initial set of service level indicators.
   If the servicemonitor was added correctly to the stage Prometheus datasource,
   it will show up in the query list when you edit a tile.

3. Export the dashboard definition in JSON format: At the top of the screen, select the
   icon with 3 dots and click `Share dashboard or panel` --> `Export` -->
   `Save to file`.

### Team's Repository

Follow the next steps to define a dashboard in your team's repository:

1. The dashboard should be located in the team’s repository, so no change in
   `infra-deployments` is required.

   The recommended structure is:

   ```
   ├── grafana
   │   ├── dashboards
   │   │   └── <teams_dashboard>.json
   │   ├── <GrafanaDashboard resource file>
   │   └── kustomization.yaml
   ```

   For example:
   ```
   ├── grafana
   │   ├── dashboards
   │   │   └── example-dashboard.json
   │   ├── dashboard.yaml
   │   └── kustomization.yaml
   ```

2. Create a folder in your team's repository to maintain the dashboard configuration
   (e.g. grafana)
  
3. Edit the dashboard json file:

   To pick the default predefined datasource, edit the dashboard json file and
   **remove** `datasource` from it. For example:

   ```yaml
     "datasource": {
       "type": "prometheus",
       "uid": "PF224BEF3374A25F8"
     }
   ```
        
   Alternatively it is possible to use templating to select a datasource. For example:

   ```json
   "templating": {
     "list": [
       {
         "current": {
           "selected": true,
           "text": "Prometheus",
           "value": "Prometheus"
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
  
   In this example, the dashboard will use `Prometheus` datasource, with ability to use
   other datasource that contains `appstudio` in the name.

4. Add the dashboard json file to the folder you created.

5. Create a `kustomization.yaml` file to generate a config map from the dashboard json,
   and to add the `GrafanaDashboard` resource file (`dashboard.yaml`). For example:

   ```yaml
   kind: Kustomization
   apiVersion: kustomize.config.k8s.io/v1beta1

   namespace: example

   configMapGenerator:
     - name: grafana-dashboard-example
       files:
         - dashboards/example-dashboard.json

   resources:
     - dashboard.yaml
   ```

6. Create the `GrafanaDashboard` resource file that uses the config map to create the
   dashboard:

   ```yaml
   apiVersion: grafana.integreatly.org/v1beta1
   kind: GrafanaDashboard
   metadata:
     name: grafana-dashboard-example
     labels:
       app: appstudio-grafana
   spec:
     instanceSelector:
       matchLabels:
         dashboards: "appstudio-grafana"
     configMapRef:
       name: grafana-dashboard-example
       key: example-dashboard.json
   ```

7. Push the code to the team's repository.

Check out [this example o11y PR](https://github.com/redhat-appstudio/o11y/pull/39)
for creating a dashboard in the team's repository.

### infra-deployments Repository

Follow these steps to add a dashboard to the `infra-deployments` repository:

1. Create a team folder under `components/monitoring/grafana/base/<team_name>`.

2. Create a `kustomization.yaml` file and add the dashboard you created as a resource by
   using the commit sha as ref. For example:

   ```yaml
   apiVersion: kustomize.config.k8s.io/v1beta1
   kind: Kustomization
   resources:
     - https://github.com/redhat-appstudio/o11y/grafana/?ref=82bec1c488250ae32d458c77755e432329be1b45
   ```

3. Add your team's folder to the base
   [kustomization file](https://github.com/redhat-appstudio/infra-deployments/blob/main/components/monitoring/grafana/base/kustomization.yaml#L14)
   to automatically add it to future deployments.
  
4. Push the code to the `infra-deployments` repository.


### Verification

Follow the next steps to verify and check your dashboard after merging the
configurations:

1. Open the Grafana UI.

2. Click the `Manage` option in the `Dashboards` menu.

3. Verify that your team’s dashboard is located under `appstudio-grafana` folder.
