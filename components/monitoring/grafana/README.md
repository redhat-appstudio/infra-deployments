
## Installing and configuring Grafana on the control-plane cluster

We use Grafana Operator to create all needed services and routes 

Note: The steps below should be handled by Argo CD

- Create the `appstudio-grafana` namespace on each Prometheus or Grafana cluster, if it does not exist yet:

    ```
    $ oc create namespace appstudio-grafana
    ```

- Build and apply from the per environment overlay:

    ```
    $ kustomize build components/monitoring/grafana/development | oc apply -f -
    ```

    Replace `development` with `staging` or `production`. 

## Dashboard migration from base to per environment overlays

The `base/dashboards/` directory is being deprecated. Team dashboards should be moved from `base/dashboards/` into per environment overlays `development/dashboards/`, `staging/dashboards/` and `production/dashboards/`. This allows each environment to reference its own dashboard source independently.

The `release/` dashboard has already been migrated and can be used as a reference for the new structure. The remaining dashboards in `base/dashboards/` will continue to work via `../base` but each team should plan to migrate.