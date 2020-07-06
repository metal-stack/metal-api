BINARY := metal-api
MAINMODULE := github.com/metal-stack/metal-api/cmd/metal-api
COMMONDIR := $(or ${COMMONDIR},../builder)
MINI_LAB_KUBECONFIG := $(shell pwd)/../mini-lab/.kubeconfig

include $(COMMONDIR)/Makefile.inc

.PHONY: all
all:: gofmt
	go mod tidy

release:: all;

.PHONY: spec
spec: all
	bin/metal-api dump-swagger | jq -r -S 'walk(if type == "array" then sort_by(strings) else . end)' > spec/metal-api.json || { echo "jq >=1.6 required"; exit 1; }

.PHONY: redoc
redoc:
	docker run -it --rm -v $(PWD):/work -w /work letsdeal/redoc-cli bundle -o generate/index.html /work/spec/metal-api.json
	xdg-open generate/index.html

.PHONY: mini-lab-push
mini-lab-push:
	docker build -t metalstack/metal-api:latest .
	kind --name metal-control-plane load docker-image metalstack/metal-api:latest
	kubectl --kubeconfig=$(MINI_LAB_KUBECONFIG) patch deployments.apps -n metal-control-plane metal-api --patch='{"spec":{"template":{"spec":{"containers":[{"name": "metal-api","imagePullPolicy":"IfNotPresent","image":"metalstack/metal-api:latest"}]}}}}'
	kubectl --kubeconfig=$(MINI_LAB_KUBECONFIG) delete pod -n metal-control-plane -l app=metal-api
