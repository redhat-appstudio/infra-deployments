## Onboarding Services to Monitoring


### 1. Metrics-exporting services

- The intended service should export the metrics from the application so that prometheus is able to understand it.

- For reference, see
  [Writing Exporters](https://prometheus.io/docs/instrumenting/writing_exporters)

- The exported port, service and route should be accessible to the prometheus service.

- Check out [this example](./prometheus/development/dummy-service.yaml) of a
  metrics-exporting service.


### 2. Service monitors

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
                  

### 3. Grafana Datasource

Grafana datasources contain the connection settings to the Prometheus instances.

A single [datasource (Thanos querier)](https://github.com/redhat-appstudio/infra-deployments/blob/main/components/monitoring/grafana/base/grafana-app.yaml#L206), `appstudio-datasource` is defined and it lets us query metrics from the Platform and User Workload Monitoring Prometheus.

To use this default datasource any definition of a datasource in the dashboard json file should be removed or a `templating` should be used.


### 4. Grafana dashboards

A dashboard is a set of one or more panels organized and arranged into one or more rows. 
It visualizes results from multiple data sources simultaneously.
New dashboards can be added through the user interface, preconfigured in the infra-deployments repository, 
or imported from other projects.


#### Create a dashboard

  1. [Create a dashboard](https://grafana.com/docs/grafana/v9.0/dashboards/)
  for your team's view of your service's Service Level Indicators.
  (After navigating to your folder, + Create Dashboard)  
  ***Note:***  
  Creating a new dashboard manually is available only for development environment.  
  You may copy and edit the `example` dashboard json instead, and test the new dashboard on the staging and production environments. 

      The `example` dashboard json definition can be found 
      [here](https://github.com/redhat-appstudio/infra-deployments/blob/main/components/monitoring/grafana/base/dashboards/example.json)


  2. Add tiles to the dashboard to track your initial set of service level indicators.  
  If the servicemonitor was added correctly to the stage Prometheus datasource, 
  it will show up in the query list when you edit a tile.

  3. Export the dashboard definition in JSON format. 
  (At the top of the screen, the icon with 3 dots lets you "Share dashboard or panel". Select Export... Save to file.)
   

#### Team's repository

Follow the next steps to define a dashboard in your team's repository

  1. The dashboard should be located in the team’s repository, no change in `infra-deployments` is required,
  the recommended structure is:
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

  2. Create a folder in your team's repository to maintain the dashboard configuration (e.g. grafana)
  
  3. Edit the dashboard json file:  
      - to pick the default predefined datasource Edit the dashboard json file and **remove** `datasource` from it, for example:
          ```yaml 
            "datasource": {
              "type": "prometheus",
              "uid": "PF224BEF3374A25F8"
            }
          ```
        
      - Alternatively it is possible to use templating to select a datasource, for example: 
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
        
        In this example, the dashboard will use `Prometheus` datasource, with ability to use other datasource that contains `appstudio` in the name.
      
  4. Add the dashboard json file to the folder you created.

  5. Create a `kustomization.yaml` to generate a config map from the dashboard json, 
  and to add the `GrafanaDashboard` resource file (`dashboard.yaml`), for example: 
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
  
  6. Create the `GrafanaDashboard` resource file that uses the config map to create the dashboard
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
 7. Push the code to the team's repository

 - Check out [this example o11y PR](https://github.com/redhat-appstudio/o11y/pull/39) 
 for creating a dashboard in the team's repository
  

#### infra-deployments repository
Follow these steps to add a dashboard to the `infra-deployments` repository

  1. Create a team folder under `components/monitoring/grafana/base/<team_name>`
  
  2. Create a `kustomization.yaml` file and add the dashboard you created as a resource by using the commit sha as ref, 
  for example:
      ```yaml
      apiVersion: kustomize.config.k8s.io/v1beta1
      kind: Kustomization
      resources:
        - https://github.com/redhat-appstudio/o11y/grafana/?ref=82bec1c488250ae32d458c77755e432329be1b45
      ```
    
  3. Add your team's folder to the base [kustomization file](https://github.com/redhat-appstudio/infra-deployments/blob/main/components/monitoring/grafana/base/kustomization.yaml#L14) to automatically add it to future deployments.
   
  4. Push the code to the `infra-deployments` repository


#### Verification

Follow the next steps to to verify and check your dashboard after merging the configurations

1. Open the Grafana UI

2. Click the `Manage` option in the `Dashboards` menu

3. Verify that your team’s dashboard is located under `appstudio-grafana` folder 
