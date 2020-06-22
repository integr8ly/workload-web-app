# workload-web-app
A test app for simulating the workload on the openshift cluster based on end-user use cases in order to monitor the downtime of component products in integreatly during an upgrade.


## Deploying the Application on the cluster

To deploy the webapp on your cluster:

### Steps

 Login to your cluster using ` oc login ` command and run 

> ```make local/deploy```

#### Grafana dashboard for the app

By default, the Grafana dashboard will not be created (to not break this [test](https://github.com/integr8ly/integreatly-operator/blob/master/test/common/dashboards_exist.go#L81)).

If you want to deploy the Grafana dashboard as part of the deploy, set the following environment variable (value doesn't matter, as long as it's not empty):

```
export GRAFANA_DASHBOARD=true
```  

## Delete the app

To delete the app, run

> ```make local/undeploy```

