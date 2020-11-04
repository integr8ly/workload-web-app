BUILD_TARGET=workload-app
NAMESPACE=workload-web-app
CONTAINER_ENGINE=docker
TOOLS_IMAGE=quay.io/integreatly/workload-web-app-tools
WORKLOAD_WEB_APP_IMAGE?= # Alternative image 
KUBECONFIG?=${HOME}/.kube/config

in_container = ${CONTAINER_ENGINE} run --rm -it \
	-e KUBECONFIG=/kube.config \
	-e RHMI_V1=${RHMI_V1} \
	-e RHOAM=${RHOAM} \
	-e GRAFANA_DASHBOARD=${GRAFANA_DASHBOARD} \
	-e USERSSO_NAMESPACE=${USERSSO_NAMESPACE} \
	-e THREESCALE_NAMESPACE=${THREESCALE_NAMESPACE} \
	-e AMQONLINE_NAMESPACE=${AMQONLINE_NAMESPACE} \
	-e WORKLOAD_WEB_APP_IMAGE=${WORKLOAD_WEB_APP_IMAGE} \
	-v ${KUBECONFIG}:/kube.config:z \
	-v "${PWD}":/workload-web-app \
	-w /workload-web-app \
	${TOOLS_IMAGE} make $(1)

.PHONY: test
test:
	@echo "SUCCESS"

.PHONY: image/build/tools
image/build/tools:
	${CONTAINER_ENGINE} build -t ${TOOLS_IMAGE} -f Dockerfile.tools .

local/deploy: image/build/tools
	$(call in_container,deploy)

local/undeploy: image/build/tools
	$(call in_container,undeploy)

local/build-deploy:
	${CONTAINER_ENGINE} build -t ${WORKLOAD_WEB_APP_IMAGE} .
	${CONTAINER_ENGINE} push ${WORKLOAD_WEB_APP_IMAGE}
	$(call in_container,deploy)

.PHONY: deploy
deploy:
	./deploy/deploy.sh

.PHONY: undeploy
undeploy:
	./deploy/undeploy.sh

.PHONY: build
build:
	#CGO_ENABLED flag is required to make it work in multi-stage build
	CGO_ENABLED=0 go build -o=$(BUILD_TARGET) .

.PHONY: code/fix
code/fix:
	@gofmt -w `find . -type f -name '*.go' -not -path "./vendor/*"`

