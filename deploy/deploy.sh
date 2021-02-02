#!/usr/bin/env bash

NS=${NAMESPACE:-"workload-web-app"}
if [[ -z "${RHOAM}" ]]; then
  AMQONLINE_NS=${AMQONLINE_NAMESPACE:-"redhat-rhmi-amq-online"}
  USERSSO_NS=${USERSSO_NAMESPACE:-"redhat-rhmi-user-sso"}
else
  USERSSO_NS=${USERSSO_NAMESPACE:-"redhat-rhoam-user-sso"}
fi
IMAGE="quay.io/integreatly/workload-web-app:master"
DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CUSTOMER_ADMIN="customer-admin01"
CUSTOMER_ADMIN_PASSWORD="Password1"

if [[ ! -z "${WORKLOAD_WEB_APP_IMAGE}" ]]; then
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

if [[ ! -z "${RHMI_V1}" ]]; then
  echo "Delete all network policies"
  sleep 5
  oc delete networkpolicy --all -n $NS
fi

if [[ -z "${RHOAM}" ]]; then
  # Deploy AMQ Resorces
  echo "Creating required AMQ resources"
  if [[ ! -z "${RHMI_V1}" ]]; then
    echo "Creating none-authservice in $AMQONLINE_NS"
    oc apply -f $DIR/amq/auth.yaml -n $AMQONLINE_NS
  else
    oc apply -f $DIR/amq/auth.yaml -n $NS
  fi

  oc apply -f $DIR/amq/addressspace.yaml -n $NS
  oc apply -f $DIR/amq/address.yaml -n $NS

  echo "Waiting for AMQ AddressSpace to be ready"
  # unfortunately oc wait doesn't work for addressspace and address types (problem with AMQ itself)
  wait_for "oc get addressspace/workload-app -n $NS -o 'jsonpath={.status.isReady}' | grep -q 'true'" "Addressspace is ready" "5m" "10"

  echo "Waiting for AMQ AddressSpace serviceHost"
  wait_for "oc get addressspace/workload-app -n $NS -o 'jsonpath={.status.endpointStatuses[?(@.name==\"messaging\")].serviceHost}' | grep -q '.svc'" "Addressspace serviceHost is ready" "5m" "10"

  echo "Waiting for AMQ Address to be ready"
  wait_for "oc get address/workload-app.queue-requests -n $NS -o 'jsonpath={.status.isReady}' | grep -q 'true'" "Address is ready" "5m" "10"

  AMQ_ADDRESS="amqps://$(oc get addressspace/workload-app -n $NS -o 'jsonpath={.status.endpointStatuses[?(@.name=="messaging")].serviceHost}')"
  AMQ_QUEUE="/$(oc get address/workload-app.queue-requests -n $NS -o 'jsonpath={.spec.address}')"

  AMQ_CONSOLE_URL="https://$(oc get routes -l name=console -n $AMQONLINE_NS -o 'jsonpath={.items[].spec.host}')"
  oc create secret generic amq-console-secret \
    --from-literal=AMQ_CONSOLE_USER=${CUSTOMER_ADMIN} \
    --from-literal=AMQ_CONSOLE_PWD=${CUSTOMER_ADMIN_PASSWORD} \
    -n $NS
fi

#SSO credentials
if [[ ! -z "${RHMI_V1}" ]]; then
  RHSSO_SERVER_URL="https://$(oc get routes -n $USERSSO_NS sso -o 'jsonpath={.spec.host}')"
  RHSSO_USER="$(oc get secret -n $USERSSO_NS credential-rhsso -o 'jsonpath={.data.SSO_ADMIN_USERNAME}' | base64 --decode)"
  RHSSO_PWD="$(oc get secret -n $USERSSO_NS credential-rhsso -o 'jsonpath={.data.SSO_ADMIN_PASSWORD}'| base64 --decode)"
else
  RHSSO_SERVER_URL=$(oc get routes -n "$USERSSO_NS" keycloak-edge -o 'jsonpath={.spec.host}')
  # Following condition was added due to deprecation of keycloak-edge route
  # see https://issues.redhat.com/browse/MGDAPI-1079 for more details
  if [ -z "$RHSSO_SERVER_URL" ]; then
    echo 'Ignoring missing "keycloak-edge" route and using route "keycloak" instead'
    RHSSO_SERVER_URL=$(oc get routes -n "$USERSSO_NS" keycloak -o 'jsonpath={.spec.host}')
  fi
  RHSSO_SERVER_URL="https://$RHSSO_SERVER_URL"
  RHSSO_USER="$(oc get secret -n $USERSSO_NS credential-rhssouser -o 'jsonpath={.data.ADMIN_USERNAME}' | base64 --decode)"
  RHSSO_PWD="$(oc get secret -n $USERSSO_NS credential-rhssouser -o 'jsonpath={.data.ADMIN_PASSWORD}'| base64 --decode)"
fi

#Create rhsso secret
oc create secret generic rhsso-secret --from-literal=RHSSO_PWD=$RHSSO_PWD --from-literal=RHSSO_USER=$RHSSO_USER -n $NS

# Deploy 3scale Resources
THREE_SCALE_URL=$(${DIR}/3scale.sh deploy)
echo "Waiting for the ${THREE_SCALE_URL} to be reachable"
wait_for "curl -s -o /dev/null -w '%{http_code}' ${THREE_SCALE_URL} | grep 200" "3SCALE API to be reachable" "10m" "10"

# Deploy the Workload App
if [[ -z "${RHOAM}" ]]; then
  echo "Deploying the webapp with the following parameters:"
  echo "AMQ_ADDRESS=$AMQ_ADDRESS"
  echo "AMQ_QUEUE=$AMQ_QUEUE"
  echo "AMQ_CONSOLE_URL=$AMQ_CONSOLE_URL"
  echo "RHSSO_SERVER_URL=$RHSSO_SERVER_URL"
  echo "THREE_SCALE_URL=$THREE_SCALE_URL"
  oc process -n $NS -f $DIR/template.yaml \
    -p AMQ_ADDRESS=$AMQ_ADDRESS \
    -p AMQ_QUEUE_NAME=$AMQ_QUEUE \
    -p AMQ_CONSOLE_URL=$AMQ_CONSOLE_URL \
    -p RHSSO_SERVER_URL=$RHSSO_SERVER_URL \
    -p THREE_SCALE_URL=$THREE_SCALE_URL \
    -p AMQ_CRUD_NAMESPACE=$NS \
    -p WORKLOAD_WEB_APP_IMAGE=$IMAGE |
    oc apply -n $NS -f -
else
  echo "Deploying the webapp with the following parameters:"
  echo "RHSSO_SERVER_URL=$RHSSO_SERVER_URL"
  echo "THREE_SCALE_URL=$THREE_SCALE_URL"
  oc process -n $NS -f $DIR/template-rhoam.yaml \
    -p RHSSO_SERVER_URL=$RHSSO_SERVER_URL \
    -p THREE_SCALE_URL=$THREE_SCALE_URL \
    -p WORKLOAD_WEB_APP_IMAGE=$IMAGE |
    oc apply -n $NS -f -
fi
echo "Waiting for pod to be ready"
sleep 5 #give it a bit time to create the pods
oc wait -n $NS --for="condition=Ready" pod -l app=workload-web-app --timeout=120s

if [[ ! -z "${GRAFANA_DASHBOARD}" ]]; then
  echo "Creating Grafana Dashboard for the app"
  if [[ -z "${RHOAM}" ]]; then
    oc apply -n $NS -f $DIR/dashboard.yaml
  else
    oc apply -n $NS -f $DIR/dashboard-rhoam.yaml
  fi
fi