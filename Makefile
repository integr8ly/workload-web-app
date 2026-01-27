BUILD_TARGET?=workload-app
NAMESPACE?=workload-web-app

# Default to Docker for CI/CD compatibility, but allow override
CONTAINER_ENGINE?=docker

CONTAINER_PLATFORM?=linux/amd64
TOOLS_IMAGE?=quay.io/integreatly/workload-web-app-tools
WORKLOAD_WEB_APP_IMAGE?= # Alternative image 
KUBECONFIG?=${HOME}/.kube/config

# Podman-specific parameters for better compatibility
ifeq ($(CONTAINER_ENGINE),podman)
    ADDITIONAL_CONTAINER_ENGINE_PARAMS?=--privileged
else
    ADDITIONAL_CONTAINER_ENGINE_PARAMS?=
endif

in_container = ${CONTAINER_ENGINE} run --rm -it ${ADDITIONAL_CONTAINER_ENGINE_PARAMS} \
	-e KUBECONFIG=/kube.config \
	-e RHOAMI=${RHOAMI} \
	-e GRAFANA_DASHBOARD=${GRAFANA_DASHBOARD} \
	-e USERSSO_NAMESPACE=${USERSSO_NAMESPACE} \
	-e THREESCALE_NAMESPACE=${THREESCALE_NAMESPACE} \
	-e WORKLOAD_WEB_APP_IMAGE=${WORKLOAD_WEB_APP_IMAGE} \
	-v ${KUBECONFIG}:/kube.config:z \
	-v "${PWD}":/workload-web-app \
	-w /workload-web-app \
	${TOOLS_IMAGE} make $(1)

.PHONY: test
test:
	@echo "SUCCESS"

.PHONY: container-engine
container-engine:
	@echo "Container Engine: $(CONTAINER_ENGINE)"
	@echo "Platform: $(CONTAINER_PLATFORM)"
	@echo "Additional Parameters: $(ADDITIONAL_CONTAINER_ENGINE_PARAMS)"
	@$(CONTAINER_ENGINE) --version

.PHONY: validate-engine
validate-engine:
	@command -v $(CONTAINER_ENGINE) >/dev/null 2>&1 || { echo "$(CONTAINER_ENGINE) not found. Please install $(CONTAINER_ENGINE) or use CONTAINER_ENGINE=<engine> to specify another."; exit 1; }
	@echo "✓ $(CONTAINER_ENGINE) is available"

.PHONY: image/build/tools
image/build/tools: validate-engine
	${CONTAINER_ENGINE} build --platform=$(CONTAINER_PLATFORM) -t ${TOOLS_IMAGE} -f Dockerfile.tools .

local/deploy: image/build/tools
	$(call in_container,deploy)

local/undeploy: image/build/tools
	$(call in_container,undeploy)

local/build-deploy: validate-engine
	${CONTAINER_ENGINE} build --platform=$(CONTAINER_PLATFORM) -t ${WORKLOAD_WEB_APP_IMAGE} .
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

