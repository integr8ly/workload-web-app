# workload-web-app
A test app for simulating the workload on the openshift cluster based on end-user use cases in order to monitor the downtime of component products in integreatly during an upgrade.

## Deploying the Application on the RHOAM cluster

To deploy the app to a RHOAM cluster, you will need to:

1. Login to the RHOAM cluster using ` oc login ` command
2. Set this optional environment variable only if you want to view the metrics data using the Grafana dashboard:
   ```bash
   export GRAFANA_DASHBOARD=true
   ```
3. Sandbox RHOAM only. Export an additional envar to switch web-app into a multitenant-managed-api compliant mode: 
   ```bash
   export SANDBOX=true
   ```
4. Then run this command to deploy the app:
   ```make local/deploy```

## Delete the app

To delete the app, run:
```
make local/undeploy
```

Note: It might take up to 15 minutes for 3scale to fully remove the service (Product) hence you need to wait this long after undeploy if you want to deploy the workload-web-app again. In case the service is not fully removed yet the deployment fails with `System name has already been taken` error.

## Troubleshooting

In case of `make: stat: Makefile: Permission denied` error try to use privileged:

```
ADDITIONAL_CONTAINER_ENGINE_PARAMS="--privileged" CONTAINER_ENGINE=podman make local/deploy
```

