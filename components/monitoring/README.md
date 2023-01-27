## Onboarding to services to Monitoring

See [Prometheus](./prometheus/README.md) and [Grafana](./grafana/README.md) for more information on how to deploy these 2 components.

### 2. Grafana dashboards - manual export

  - Create a new folder in [grafana](https://grafana-appstudio-workload-monitoring.apps.appstudio-stage.x99m.p1.openshiftapps.com) for your service. (In the left nav, + Create Folder)

  - [Create a dashboard](https://grafana.com/docs/grafana/v9.0/dashboards/) for your team's view of your service's Service Level Indicators. (After navigating to your folder, + Create Dashboard)

  - Add tiles on the dashboard to track your initial set of service level indicators. If you correctly added your servicemonitor to the stage Prometheus datasource, it will show up in the Query list when you edit a tile.

  - Export the dashboard definition in JSON format. (At the top of the screen, the icon with 3 dots lets you "Share dashboard or panel". Select Export... Save to file.)

  -  Store the dashboard definition in git, in infra-deployments. 

  - Add your dashboard to the [kustomization file](https://github.com/redhat-appstudio/infra-deployments/blob/main/components/monitoring/grafana/base/kustomization.yaml#L15) to automatically add it to future deployments.

  - [Here](https://github.com/redhat-appstudio/infra-deployments/blob/main/components/monitoring/grafana/base/dashboards/example.json) is an example for the default dashboard.
  
    

