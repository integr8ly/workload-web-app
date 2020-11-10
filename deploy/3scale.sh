#!/usr/bin/env bash

set -euo pipefail

SCRIPT="3scale.sh"
TOKEN=
API_URL=
if [[ -z "${RHOAM}" ]]; then
  NAMESPACE=${THREESCALE_NAMESPACE:-"redhat-rhmi-3scale"}
else
  NAMESPACE=${THREESCALE_NAMESPACE:-"redhat-rhoam-3scale"}
fi

function log() {
    # always log to stderr to avoid interfiring with the stdout
    echo "$SCRIPT: $1" >&2
}

function xget() {
    local xml=$(cat) # read stdin
    local query=$1

    local r=$(xpath "${query}" 2>/dev/null <<<"${xml}")

    if [[ -z "${r}" ]]; then
        log "error: xget: failed to find '${query}' in: '${xml}'"
        return 1
    fi

    echo "${r}"
}

function jget() {
    local json=$(cat) # read stdin
    local query=$1

    local r=$(jq -r "${query}" <<<"${json}")

    if [[ -z "${r}" ]]; then
        log "error: jget: failed to find '${query}' in: '${json}'"
        return 1
    fi

    echo "${r}"
}

# Retrieve the 3scale API endpoint ant admin Token form OpenShift
function setup() {
    TOKEN="$(oc -n ${NAMESPACE} get secret system-seed -o jsonpath={.data.ADMIN_ACCESS_TOKEN} | base64 --decode)"

    local host="$(oc -n "${NAMESPACE}" get route -l zync.3scale.net/route-to=system-provider -o=json | jq -r '.items[].spec | select(.host|match("3scale-admin")) | .host')"
    API_URL="https://${host}/admin/api"

    log "API_URL=${API_URL}"
}

function create_account() {
    local name=$1

    curl -sS -X POST \
        -d access_token=${TOKEN} \
        -d org_name=${name} \
        -d username="${name}-user" \
        ${API_URL}/signup.xml |
        xget "//account/id/text()"
}

function create_backend() {
    local name=$1
    local endpoint=$2

    curl -sS -X POST \
        -d access_token=${TOKEN} \
        -d name=${name} \
        -d private_endpoint=${endpoint} \
        ${API_URL}/backend_apis.json |
        jget ".backend_api.id"
}

function create_metric() {
    local backend_id=$1
    local name=$2

    curl -sS -X POST \
        -d access_token=${TOKEN} \
        -d friendly_name=${name} \
        -d unit=hit \
        ${API_URL}/backend_apis/${backend_id}/metrics.json |
        jget ".metric.id"
}

function create_mapping_rule() {
    local backend_id=$1
    local metric_id=$2

    curl -sS -X POST \
        -d access_token=${TOKEN} \
        -d http_method=GET \
        -d pattern="/" \
        -d delta=1 \
        -d metric_id=${metric_id} \
        ${API_URL}/backend_apis/${backend_id}/mapping_rules.json
}

function create_service() {
    local name=$1

    curl -sS -X POST \
        -d access_token=${TOKEN} \
        -d name=${name} \
        -d system_name=${name} \
        ${API_URL}/services.xml |
        xget "//service/id/text()"
}

function create_backend_usage() {
    local service_id=$1
    local backend_id=$2

    curl -sS -X POST \
        -d access_token=${TOKEN} \
        -d backend_api_id=${backend_id} \
        -d path="/" \
        ${API_URL}/services/${service_id}/backend_usages.json
}

function create_application_plan() {
    local service_id=$1
    local name=$2

    curl -sS -X POST \
        -d access_token=${TOKEN} \
        -d name=${name} \
        ${API_URL}/services/${service_id}/application_plans.xml |
        xget "//plan/id/text()"
}

function create_application() {
    local account_id=$1
    local plan_id=$2
    local name=$3

    curl -sS -X POST \
        -d access_token=${TOKEN} \
        -d plan_id=${plan_id} \
        -d name=${name} \
        -d description=${name} \
        ${API_URL}/accounts/${account_id}/applications.xml |
        xget "//application/user_key/text()"
}

function deploy_proxy() {
    local service_id=$1

    curl -fsS -X POST \
        -d access_token=${TOKEN} \
        ${API_URL}/services/${service_id}/proxy/deploy.xml
}

function promote_proxy() {
    local service_id=$1

    local version=$(curl -sS -X GET \
        -d access_token=${TOKEN} \
        ${API_URL}/services/${service_id}/proxy/configs/sandbox/latest.json |
        jget ".proxy_config.version")

    curl -sS -X POST \
        -d access_token=${TOKEN} \
        -d to=production \
        ${API_URL}/services/${service_id}/proxy/configs/sandbox/${version}/promote.json |
        jget ".proxy_config.content.proxy.endpoint"
}

function delete_service() {
    local name=$1

    local id=$(curl -sS -X GET \
        -d access_token=${TOKEN} \
        ${API_URL}/services.xml |
        xget "//services/service/system_name[text()='${name}']/../id/text()")

    if [[ ! -z "${id}" ]]; then
        curl -fsS -X DELETE -d access_token=${TOKEN} ${API_URL}/services/${id}.xml
    fi
}

function delete_backend() {
    local name=$1

    # we assume that there are no more than 500 backends
    local id=$(curl -sS -X GET \
        -d access_token=${TOKEN} \
        ${API_URL}/backend_apis.json |
        jget ".backend_apis[] | select(.backend_api.name == \"${name}\") | .backend_api.id")

    if [[ ! -z "${id}" ]]; then
        curl -fsS -X DELETE -d access_token=${TOKEN} ${API_URL}/backend_apis/${id}.json
    fi
}

function delete_account() {
    local name=$1

    local id=$(curl -sS -X GET \
        -d access_token=${TOKEN} \
        ${API_URL}/accounts.xml |
        xget "//accounts/account/org_name[text()='${name}']/../id/text()")

    if [[ ! -z "${id}" ]]; then
        curl -fsS -X DELETE -d access_token=${TOKEN} ${API_URL}/accounts/${id}.xml
    fi
}

function deploy() {

    setup

    # create an account and user (needed to create an application)
    log "create workload-app-api-account"
    local account_id=$(create_account workload-app-api-account)

    # create the backend
    log "create workload-app-api-backend"
    local backend_id=$(create_backend workload-app-api-backend "https://echo-api.3scale.net:443")

    # create a metric for the backend (needed from the mapping rule)
    log "create workload-app-api-metric in backend=${backend_id}"
    local metric_id=$(create_metric "${backend_id}" workload-app-api-metric)

    # create a default mapping rule so that all request will reach the endpoint
    log "create mapping_rule with backend_id=${backend_id} and metric_id=${metric_id}"
    create_mapping_rule "${backend_id}" "${metric_id}" >/dev/null

    # create the service (also know as product or api)
    log "create workload-app-api service"
    local service_id=$(create_service workload-app-api)

    # bind the backend to the service
    log "create backend_usage with service_id=${service_id} and backend_id=${backend_id}"
    create_backend_usage "${service_id}" ${backend_id} >/dev/null

    # create the application plan (needed to crate the application)
    log "create workload-app-api-plan in service=${service_id}"
    local plan_id=$(create_application_plan "${service_id}" workload-app-api-plan)

    # create application to garant the user (account) access to the service (through the application plan)
    log "create workload-app-api-app in account=${account_id} with plan_id=${plan_id}"
    user_key=$(create_application "${account_id}" "${plan_id}" workload-app-api-app)

    # need to wait for a bit otherwise deploy_proxy might fail
    sleep 5

    # promote the API to the staging (sandbox) environment
    log "promote api to staging service_id=${service_id}"
    deploy_proxy "${service_id}" >/dev/null

    sleep 5

    # promote the API to the production environment
    log "promote api to production service_id=${service_id}"
    local endpoint=$(promote_proxy "${service_id}")

    # print the 3scale API url to stdout
    echo "${endpoint}?user_key=${user_key}"
}

function undeploy() {
    setup

    log "delete workload-app-api-account"
    delete_account workload-app-api-account

    log "delete workload-app-api service"
    delete_service workload-app-api

    log "delete workload-app-api-backend service"
    delete_backend workload-app-api-backend
}

function help() {
    echo "Usage: $SCRIPT COMMAND"
    echo
    echo "Commands:"
    echo "  deploy     create a 3scale api"
    echo "  undeploy   remove the 3scale api"
}

# Run
case "$1" in
-h | --help)
    help
    ;;
deploy)
    deploy
    ;;
undeploy)
    undeploy
    ;;
*)
    log "error: unknow command $1"
    exit 1
    ;;
esac
