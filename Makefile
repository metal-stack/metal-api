CGO_ENABLED := 1

SHA := $(shell git rev-parse --short=8 HEAD)
GITVERSION := $(shell git describe --long --all)
# gnu date format iso-8601 is parsable with Go RFC3339
BUILDDATE := $(shell date --iso-8601=seconds)
VERSION := $(or ${VERSION},$(shell git describe --tags --exact-match 2> /dev/null || git symbolic-ref -q --short HEAD || git rev-parse --short HEAD))

MINI_LAB_KUBECONFIG := $(shell pwd)/../mini-lab/.kubeconfig

LINKMODE := -linkmode external -extldflags '-static -s -w' \
		 -X 'github.com/metal-stack/v.Version=$(VERSION)' \
		 -X 'github.com/metal-stack/v.Revision=$(GITVERSION)' \
		 -X 'github.com/metal-stack/v.GitSHA1=$(SHA)' \
		 -X 'github.com/metal-stack/v.BuildDate=$(BUILDDATE)'

.PHONY: release
release: protoc test build

.PHONY: build
build:
	go build \
		-tags 'osusergo netgo static_build' \
		-ldflags \
		"$(LINKMODE)" \
		-o bin/metal-api \
		github.com/metal-stack/metal-api/cmd/metal-api

	md5sum bin/metal-api > bin/metal-api.md5

	bin/metal-api dump-swagger | jq -r -S 'walk(if type == "array" then sort_by(strings) else . end)' > spec/metal-api.json || { echo "jq >=1.6 required"; exit 1; }

.PHONY: test
test: test-unit check-diff

.PHONY: test-unit
test-unit:
	go test -race -cover ./...

.PHONY: test-integration
test-integration:
	go test -v -count=1 -tags=integration -timeout 600s -p 1 ./...

.PHONY: check-diff
check-diff: spec
	git diff --exit-code spec pkg

.PHONY: protoc
protoc:
	rm -rf pkg/api/v1
	make -C proto protoc

.PHONY: mini-lab-push
mini-lab-push:
	make
	docker build -f Dockerfile -t metalstack/metal-api:latest .
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
	docker run --user $$(id -u):$$(id -g) --rm -w /work -v ${PWD}:/work vektra/mockery:v2.21.1 --name MachineManager --dir /work/cmd/metal-api/internal/scaler --output /work/cmd/metal-api/internal/scaler --filename pool_scaler_mock_test.go --testonly --inpackage
