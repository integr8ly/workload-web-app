#!/usr/bin/env bash

NS=${NAMESPACE:-"workload-web-app"}
oc delete all -l app=workload-web-app -n $NS
oc delete address/workload-app.queue-requests -n $NS
oc delete addressspace/workload-app -n $NS
oc delete authenticationservice/none-authservice -n $NS
oc delete project $NS