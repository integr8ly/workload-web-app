#!/usr/bin/env bash

NS=${NAMESPACE:-"workload-web-app"}
DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

function retry {
  local retries=$1; shift
  local wait=$1; shift

  local count=0
  until "$@"; do
    exit=$?
    count=$(($count + 1))
    if [ $count -lt $retries ]; then
      echo "Retry $count/$retries exited $exit, retrying in $wait seconds..."
      sleep $wait
    else
      echo "Retry $count/$retries exited $exit, no more retries left."
      return $exit
    fi
  done
  return 0
}

oc new-project $NS

echo "Creating required AMQ resources"
oc apply -f $DIR/amq/auth.yaml -n $NS
oc apply -f $DIR/amq/addressspace.yaml -n $NS
oc apply -f $DIR/amq/address.yaml -n $NS

echo "Waiting for AMQ AddressSpace to be ready"
# unfortunately oc wait doesn't work for addressspace and address types (problem with AMQ itself)
retry 20 5 oc get addressspace/workload-app -n $NS -o 'jsonpath={.status.isReady}' | grep 'true'

echo "Waiting for AMQ Address to be ready"
retry 20 5 oc get address/workload-app.queue-requests -n $NS -o 'jsonpath={.status.isReady}' | grep 'true'

AMQ_ADDRESS="amqps://$(oc get addressspace/workload-app -n $NS -o 'jsonpath={.status.endpointStatuses[?(@.name=="messaging")].serviceHost}')"
AMQ_QUEUE="/$(oc get address/workload-app.queue-requests -n $NS -o 'jsonpath={.spec.address}')"

echo "Deploying the webapp with the following parameters:"
echo "AMQ_ADDRESS=$AMQ_ADDRESS"
echo "AMQ_QUEUE=$AMQ_QUEUE"
oc process -n $NS -f $DIR/template.yaml \
   -p AMQ_ADDRESS=$AMQ_ADDRESS \
   -p AMQ_QUEUE_NAME=$AMQ_QUEUE \
   | oc apply -n $NS -f -

echo "Waiting for pod to be ready"
sleep 5 #give it a bit time to create the pods
oc wait -n $NS --for="condition=Ready" pod -l app=workload-web-app --timeout=120s