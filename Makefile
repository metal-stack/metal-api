BINARY := metal-api
MAINMODULE := github.com/metal-stack/metal-api/cmd/metal-api
COMMONDIR := $(or ${COMMONDIR},../builder)
MINI_LAB_KUBECONFIG := $(shell pwd)/../mini-lab/.kubeconfig

include $(COMMONDIR)/Makefile.inc

release:: spec check-diff all ;

.PHONY: spec
spec: all
	bin/$(BINARY) dump-swagger | jq -r -S 'walk(if type == "array" then sort_by(strings) else . end)' > spec/metal-api.json || { echo "jq >=1.6 required"; exit 1; }

.PHONY: check-diff
check-diff: spec
	git diff --exit-code spec pkg

.PHONY: redoc
redoc:
	docker run --rm --user $$(id -u):$$(id -g) -v $(PWD):/work -w /work letsdeal/redoc-cli bundle -o generate/index.html /work/spec/metal-api.json
	xdg-open generate/index.html

.PHONY: protoc
protoc:
	rm -rf pkg/api/v1
	make -C proto protoc

.PHONY: protoc-docker
protoc-docker:
	rm -rf pkg/api/v1
	docker pull bufbuild/buf:1.5.0
	docker run --rm --user $$(id -u):$$(id -g) -v $(PWD):/work --tmpfs /.cache -w /work/proto bufbuild/buf:1.5.0 generate -v

.PHONY: mini-lab-push
mini-lab-push:
	docker build -t metalstack/metal-api:latest .
	kind --name metal-control-plane load docker-image metalstack/metal-api:latest
	kubectl --kubeconfig=$(MINI_LAB_KUBECONFIG) patch deployments.apps -n metal-control-plane metal-api --patch='{"spec":{"template":{"spec":{"containers":[{"name": "metal-api","imagePullPolicy":"IfNotPresent","image":"metalstack/metal-api:latest"}]}}}}'
	kubectl --kubeconfig=$(MINI_LAB_KUBECONFIG) delete pod -n metal-control-plane -l app=metal-api

.PHONY: visualize-fsm
visualize-fsm:
	cd cmd/metal-api/internal/tools/visualize_fsm
	go run main.go
	dot -Tsvg fsm.dot > fsm.svg

.PHONY: mocks
mocks:
	docker run --user $$(id -u):$$(id -g) --rm -w /work -v ${PWD}:/work vektra/mockery:v2.14.0 --name MachineManager --dir /work/cmd/metal-api/internal/scaler --output /work/cmd/metal-api/internal/scaler --filename pool_scaler_mock_test.go --testonly --inpackage