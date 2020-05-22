BUILD_TARGET=workload-app
NAMESPACE=workload-web-app

.PHONY: test
test:
	@echo "SUCCESS"

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

