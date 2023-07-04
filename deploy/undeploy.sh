#!/usr/bin/env bash

OBSERVABILITY_NS="redhat-rhoam-customer-monitoring-operator"
if [[ -n "${SANDBOX}" ]]; then
    OBSERVABILITY_NS="sandbox-rhoam-customer-monitoring-operator"
fi

NS=${NAMESPACE:-"workload-web-app"}
DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Clean the Workload App
oc delete all -l app=workload-web-app -n $NS
oc delete grafanadashboards.integreatly.org workload-web-app -n $OBSERVABILITY_NS
oc delete project $NS

# Clean 3scale Resources
${DIR}/3scale.sh undeploy
