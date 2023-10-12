#!/usr/bin/env bash

OBSERVABILITY_NS="redhat-rhoam-customer-monitoring-operator"
if [[ -n "${SANDBOX}" ]]; then
    OBSERVABILITY_NS="sandbox-rhoam-customer-monitoring-operator"
fi

NS=${NAMESPACE:-"workload-web-app"}
DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Delete the grafana dashboard
GRAFANA_ADMIN="$(oc get secret -n $OBSERVABILITY_NS grafana-admin-credentials -o 'jsonpath={.data.GF_SECURITY_ADMIN_USER}'| base64 --decode)"
GRAFANA_ADMIN_PASSWORD="$(oc get secret -n $OBSERVABILITY_NS grafana-admin-credentials -o 'jsonpath={.data.GF_SECURITY_ADMIN_PASSWORD}'| base64 --decode)"
WORKLOAD_WEB_APP_POD_NAME="$(oc get po --namespace=$NS | grep Running | head -1 | awk '{print $1}')"
GRAFANA_SVC="$(oc get svc --namespace=$OBSERVABILITY_NS | grep grafana-service | awk '{print $3}')"
TITLE="Workload"
# search for the dashboard to get the UID
SEARCH_RESULT=$(oc exec -it $WORKLOAD_WEB_APP_POD_NAME -- /bin/sh -c 'wget -q -O - --header="Accept: application/json" --header="Content-Type: application/json" http://'$GRAFANA_ADMIN':'$GRAFANA_ADMIN_PASSWORD'@'$GRAFANA_SVC':3000/api/search?query='$TITLE'')
DASHBOARD_UID=$(echo "$SEARCH_RESULT" | jq -r '.[0].uid')
oc exec -it $WORKLOAD_WEB_APP_POD_NAME -- /bin/sh -c 'curl -X DELETE -H "Content-Type: application/json" http://'$GRAFANA_ADMIN':'$GRAFANA_ADMIN_PASSWORD'@'$GRAFANA_SVC':3000/api/dashboards/uid/'$DASHBOARD_UID''
echo " "
echo Grafana dashboard completed
echo " "

# Clean the Workload App
oc delete all -l app=workload-web-app -n $NS
oc delete project $NS

# Clean 3scale Resources
${DIR}/3scale.sh undeploy
