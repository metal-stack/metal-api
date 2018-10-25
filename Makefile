.ONESHELL:
SHA := $(shell git rev-parse --short=8 HEAD)
GITVERSION := $(shell git describe --long --all)
BUILDDATE := $(shell date -Iseconds)
VERSION := $(or ${VERSION},devel)

BINARY := metal-api
MODULE := git.f-i-ts.de/cloud-native/maas/metal-api
GOSRC = $(shell find cmd/ -type f -name '*.go') $(shell find pkg/ -type f -name '*.go')

export GOPROXY := https://gomods.fi-ts.io
export GO111MODULE := on
export CGO_ENABLED := 0


.PHONY: all test up test-ci createmasterdata createtestdevices spec generate-client clean

all: bin/$(BINARY);

bin/$(BINARY): $(GOSRC)
	go build -tags netgo -ldflags \
		"-X 'main.version=$(VERSION)' \
		 -X 'main.revision=$(GITVERSION)' \
		 -X 'main.gitsha1=$(SHA)' \
		 -X 'main.builddate=$(BUILDDATE)'" \
		-o bin/$(BINARY) \
		$(MODULE)/cmd/$(BINARY)

clean:
	rm -rf bin/$(BINARY)
up:
	docker-compose up --build

spec:
	curl http://localhost:8080/apidocs.json >spec/metal-api.json

generate-client:
	swagger generate client -f netbox-api/api.yaml -t netbox-api

test:
	go test -cover ./...

test-ci:
	go test ./... -coverprofile=coverage.out && go tool cover -func=coverage.out | tail -n 1; rm coverage.out

createmasterdata:
	@cat masterdata/images.json | jq -r -c -M ".[]" | xargs -d'\n' -L1 -I'{}' curl -XPUT -H "Content-Type: application/json" -d '{}' http://localhost:8080/image
	@cat masterdata/sizes.json | jq -r -c -M ".[]" | xargs -d'\n' -L1 -I'{}' curl -XPUT -H "Content-Type: application/json" -d '{}' http://localhost:8080/size
	@cat masterdata/facilities.json | jq -r -c -M ".[]" | xargs -d'\n' -L1 -I'{}' curl -XPUT -H "Content-Type: application/json" -d '{}' http://localhost:8080/facility

createtestdevices:
	@cat masterdata/testdevices.json | jq -r -c -M ".[]" | xargs -d'\n' -L1 -I'{}' curl -XPOST -H "Content-Type: application/json" -d '{}' http://localhost:8080/device/register
