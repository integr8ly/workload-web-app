#!/usr/bin/env bash

OBSERVABILITY_NS="redhat-rhoam-observability"
if [[ -n "${SANDBOX}" ]]; then
    OBSERVABILITY_NS="sandbox-rhoam-observability"
fi

NS=${NAMESPACE:-"workload-web-app"}
if [[ -z "${RHOAM}" ]]; then
  AMQONLINE_NS=${AMQONLINE_NAMESPACE:-"redhat-rhmi-amq-online"}
fi
DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Clean the Workload App
oc delete all -l app=workload-web-app -n $NS
oc delete grafanadashboards.integreatly.org workload-web-app -n $OBSERVABILITY_NS

# if RHOAM flag passed ignore amq
if [[ -z "${RHOAM}" ]]; then
  # Clean AMQ Resources
  oc delete address/workload-app.queue-requests -n $NS
  oc delete addressspace/workload-app -n $NS
  if [[ ! -z "${RHMI_V1}" ]]; then
    oc delete authenticationservice/none-authservice -n $AMQONLINE_NS
  else
    oc delete authenticationservice/none-authservice -n $NS
  fi
fi
oc delete project $NS

# Clean 3scale Resources
${DIR}/3scale.sh undeploy
