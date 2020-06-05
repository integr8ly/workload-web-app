BUILD_TARGET=workload-app
NAMESPACE=workload-web-app
CONTAINER_ENGINE=docker
TOOLS_IMAGE=quay.io/integreatly/workload-web-app-tools

in_container = ${CONTAINER_ENGINE} run --rm -it \
	-e KUBECONFIG=/kube.config \
	-v "${HOME}/.kube/config":/kube.config:z \
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

