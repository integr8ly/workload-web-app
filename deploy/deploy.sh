#!/usr/bin/env bash

OBSERVABILITY_NS="redhat-rhoam-customer-monitoring"
NS=${NAMESPACE:-"workload-web-app"}

if [[ -z "${SANDBOX}" ]]; then
  SSO_NS=${USERSSO_NAMESPACE:-"redhat-rhoam-user-sso"}
else
  SSO_NS=${USERSSO_NAMESPACE:-"sandbox-rhoam-rhsso"}
  OBSERVABILITY_NS="sandbox-rhoam-customer-monitoring"
fi

IMAGE="quay.io/integreatly/workload-web-app:master"
DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CUSTOMER_ADMIN="customer-admin01"
CUSTOMER_ADMIN_PASSWORD="Password1"

if [[ -n "${WORKLOAD_WEB_APP_IMAGE}" ]]; then
  echo "Attention: using alternative image: ${WORKLOAD_WEB_APP_IMAGE}"
  IMAGE=${WORKLOAD_WEB_APP_IMAGE}
fi

wait_for() {
    local command="${1}"
    local description="${2}"
    local timeout="${3}"
    local interval="${4}"

    printf "Waiting for %s for %s...\n" "${description}" "${timeout}"
    timeout --foreground "${timeout}" bash -c "
    until ${command}
    do
        printf \"Waiting for %s... Trying again in ${interval}s\n\" \"${description}\"
        sleep ${interval}
    done
    "
    printf "%s finished!\n" "${description}"
}

oc new-project $NS
oc label namespace $NS monitoring-key=middleware integreatly-middleware-service=true

#SSO credentials
RHSSO_SERVER_URL=$(oc get routes -n $SSO_NS keycloak-edge -o 'jsonpath={.spec.host}')
# Following condition was added due to deprecation of keycloak-edge route
# see https://issues.redhat.com/browse/MGDAPI-1079 for more details
if [ -z "$RHSSO_SERVER_URL" ]; then
  echo 'Ignoring missing "keycloak-edge" route and using route "keycloak" instead'
  RHSSO_SERVER_URL=$(oc get routes -n $SSO_NS keycloak -o 'jsonpath={.spec.host}')
fi
RHSSO_SERVER_URL="https://$RHSSO_SERVER_URL"
if [[ -z "${SANDBOX}" ]]; then
  RHSSO_USER="$(oc get secret -n $SSO_NS credential-rhssouser -o 'jsonpath={.data.ADMIN_USERNAME}' | base64 --decode)"
  RHSSO_PWD="$(oc get secret -n $SSO_NS credential-rhssouser -o 'jsonpath={.data.ADMIN_PASSWORD}'| base64 --decode)"
else
  RHSSO_USER="$(oc get secret -n $SSO_NS credential-rhsso -o 'jsonpath={.data.ADMIN_USERNAME}' | base64 --decode)"
  HSSO_PWD="$(oc get secret -n $SSO_NS credential-rhsso -o 'jsonpath={.data.ADMIN_PASSWORD}'| base64 --decode)"
fi

#Create rhsso secret
oc create secret generic rhsso-secret --from-literal=RHSSO_PWD=$RHSSO_PWD --from-literal=RHSSO_USER=$RHSSO_USER -n $NS

# Deploy 3scale Resources
THREE_SCALE_URL=$(${DIR}/3scale.sh deploy)
echo "Waiting for the ${THREE_SCALE_URL} to be reachable"
wait_for "curl -s -o /dev/null -w '%{http_code}' ${THREE_SCALE_URL} | grep 200" "3SCALE API to be reachable" "10m" "10"

# Deploy the Workload App
echo "Deploying the webapp with the following parameters:"
echo "RHSSO_SERVER_URL=$RHSSO_SERVER_URL"
echo "THREE_SCALE_URL=$THREE_SCALE_URL"
oc process -n $NS -f $DIR/template-rhoam.yaml \
  -p RHSSO_SERVER_URL=$RHSSO_SERVER_URL \
  -p THREE_SCALE_URL=$THREE_SCALE_URL \
  -p WORKLOAD_WEB_APP_IMAGE=$IMAGE |
  oc apply -n $NS -f -
echo "Waiting for pods to be ready"
sleep 5 #give it a bit time to create the pods
oc wait -n $NS --for="condition=Ready" pod -l app=workload-web-app --timeout=120s
status=$?

# Ugly hack to start rollout again if pods did not get ready
if [[ $status -ne 0 ]]; then
  oc rollout cancel dc/workload-web-app -n workload-web-app
  sleep 5 #give it a bit time to cancel
  oc rollout latest dc/workload-web-app -n workload-web-app
  sleep 5 #give it a bit time to create the pods
  oc wait -n $NS --for="condition=Ready" pod -l app=workload-web-app --timeout=120s
  status=$?
fi

if [[ $status -ne 0 ]]; then
  exit $status
fi

if [[ -n "${GRAFANA_DASHBOARD}" ]]; then
  echo "Creating Grafana Dashboard for the app"
  GRAFANA_ADMIN="$(oc get secret -n $OBSERVABILITY_NS grafana-admin-credentials -o 'jsonpath={.data.GF_SECURITY_ADMIN_USER}'| base64 --decode)"
  GRAFANA_ADMIN_PASSWORD="$(oc get secret -n $OBSERVABILITY_NS grafana-admin-credentials -o 'jsonpath={.data.GF_SECURITY_ADMIN_PASSWORD}'| base64 --decode)"
  WORKLOAD_WEB_APP_POD_NAME="$(oc get po --namespace=$NS | grep Running | head -1 | awk '{print $1}')"
  GRAFANA_SVC="$(oc get svc --namespace=$OBSERVABILITY_NS | grep grafana-service | awk '{print $3}')"
  oc exec -it $WORKLOAD_WEB_APP_POD_NAME -- /bin/sh -c 'wget -O - --header="Accept: application/json" --header="Content-Type: application/json" --post-file=- http://'$GRAFANA_ADMIN':'$GRAFANA_ADMIN_PASSWORD'@'$GRAFANA_SVC':3000/api/dashboards/db' < $DIR/dashboard-rhoam.json
fi
