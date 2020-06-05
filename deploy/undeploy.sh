#!/usr/bin/env bash

NS=${NAMESPACE:-"workload-web-app"}
DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Clean the Workload App
oc delete all -l app=workload-web-app -n $NS

# Clean AMQ Resources
oc delete address/workload-app.queue-requests -n $NS
oc delete addressspace/workload-app -n $NS
oc delete authenticationservice/none-authservice -n $NS
oc delete project $NS

# Clean 3scale Resources
${DIR}/3scale.sh undeploy
