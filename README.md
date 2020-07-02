# workload-web-app
A test app for simulating the workload on the openshift cluster based on end-user use cases in order to monitor the downtime of component products in integreatly during an upgrade.

## Deploying the Application on the cluster

The app works for both RHMI 1.x and RHMI 2.x clusters.
To deploy the webapp on your cluster:

### RHMI 1.x Clusters

To deploy the app to a RHMI 1.x cluster, you will need to:

1. Login to the RHMI 1.x cluster using ` oc login ` command
2. Set the following environment variables:
   ```
   # These env vars are required
   export RHMI_V1=true
   export USERSSO_NAMESPACE=<user sso namespace>
   export THREESCALE_NAMESPACE=<3scale namespace>
   export AMQONLINE_NAMESPACE=<amqonline namespace>
   # This env var is optional. Only set it if you want to view the metrics data using the Grafana dashboard
   export GRAFANA_DASHBOARD=true
   ```
3. Then run this command to deploy the app:
   ```make local/deploy```

### RHMI 2.x Clusters

To deploy the app to a RHMI 2.x cluster, you will need to:

1. Login to the RHMI 2.x cluster using ` oc login ` command
2. Set the following environment variables:
   ```
   # This env var is optional. Only set it if you want to view the metrics data using the Grafana dashboard
   export GRAFANA_DASHBOARD=true
   ```
3. Then run this command to deploy the app:
   ```make local/deploy```

## Delete the app

To delete the app, run

> ```make local/undeploy```

