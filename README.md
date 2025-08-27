# workload-web-app
A test app for simulating the workload on the openshift cluster based on end-user use cases in order to monitor the downtime of component products in integreatly during an upgrade.

## Container Engine Support

This application supports both **Docker** and **Podman**. The Makefile automatically detects which container engine is available:

- If Podman is available, it uses Podman
- If only Docker is available, it uses Docker
- You can override the detection by setting `CONTAINER_ENGINE=docker` or `CONTAINER_ENGINE=podman`

### Check your container engine

You can check which container engines are available and get setup guidance:

```bash
# Quick environment check with helpful tips
./check-container-engine.sh

# Or use the Makefile target for basic info
make container-engine
```

## Deploying the Application on the RHOAM cluster

To deploy the app to a RHOAM cluster, you will need to:

1. Login to the RHOAM cluster using ` oc login ` command
2. Set this optional environment variable only if you want to view the metrics data using the Grafana dashboard:
   ```bash
   export GRAFANA_DASHBOARD=true
   ```
3. Internal RHOAMI only. Export an additional envar to switch web-app into a Internal/edge compliant mode: 
   ```bash
   export RHOAMI=true
   ```
4. Then run this command to deploy the app:
   ```bash
   make local/deploy
   ```

### Using a specific container engine

If you want to use a specific container engine, set the `CONTAINER_ENGINE` variable:

```bash
# Using Docker
CONTAINER_ENGINE=docker make local/deploy

# Using Podman
CONTAINER_ENGINE=podman make local/deploy
```

## Delete the app

To delete the app, run:
```
make local/undeploy
```

Note: It might take up to 15 minutes for 3scale to fully remove the service (Product) hence you need to wait this long after undeploy if you want to deploy the workload-web-app again. In case the service is not fully removed yet the deployment fails with `System name has already been taken` error.

## Troubleshooting

### Container Engine Issues

1. **Check if your container engine is working:**
   ```bash
   make validate-engine
   ```

2. **Permission denied errors with Podman:**
   ```bash
   ADDITIONAL_CONTAINER_ENGINE_PARAMS="--privileged" make local/deploy
   ```

3. **SELinux issues with volume mounts:**
   The Makefile automatically adds `:z` labels for SELinux compatibility. If you still have issues:
   ```bash
   ADDITIONAL_CONTAINER_ENGINE_PARAMS="--privileged --security-opt label=disable" make local/deploy
   ```

4. **Force a specific container engine:**
   ```bash
   # Force Docker even if Podman is available
   CONTAINER_ENGINE=docker make local/deploy
   
   # Force Podman even if Docker is available
   CONTAINER_ENGINE=podman make local/deploy
   ```

### 3scale Service Issues

Note: It might take up to 15 minutes for 3scale to fully remove the service (Product) hence you need to wait this long after undeploy if you want to deploy the workload-web-app again. In case the service is not fully removed yet the deployment fails with `System name has already been taken` error.

