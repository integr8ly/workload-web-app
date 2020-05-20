NAMESPACE=test-app

define wait_command
	@echo Waiting for $(2) for $(3)...
	@time timeout --foreground $(3) bash -c "until $(1); do echo $(2) not ready yet, trying again in $(4)s...; sleep $(4); done"
	@echo $(2) ready!
endef

.PHONY: test
test:
	@echo "SUCCESS"

.PHONY: deploy
deploy:
	@oc new-project $(NAMESPACE)
	@oc apply -f deploy/dc/deployment_config.yaml
	@oc apply -f deploy/svc/service.yaml
	@oc expose -f deploy/svc/service.yaml
	$(call wait_command, oc get dc -n $(NAMESPACE) -o json | jq '.items[] | .status |.readyReplicas' | grep 1 , available replicas, 5m, 30)
	@oc get route -o json | jq -r '.items[] | .spec.host'
