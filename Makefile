.ONESHELL:
SHA := $(shell git rev-parse --short=8 HEAD)
GITVERSION := $(shell git describe --long --all)
BUILDDATE := $(shell date -Iseconds)
VERSION := $(or ${VERSION},devel)

BINARY := metal-api
MODULE := git.f-i-ts.de/cloud-native/maas/metal-api
GOSRC = $(shell find cmd/ -type f -name '*.go') $(shell find pkg/ -type f -name '*.go')

.PHONY: all test up test-ci createmasterdata createtestdevices

all: bin/$(BINARY);

bin/$(BINARY): $(GOSRC)
	CGO_ENABLE=0 GO111MODULE=on go build -tags netgo -ldflags "-X 'main.version=$(VERSION)' -X 'main.revision=$(GITVERSION)' -X 'main.gitsha1=$(SHA)' -X 'main.builddate=$(BUILDDATE)'" -o bin/$(BINARY) $(MODULE)/cmd/$(BINARY)

up:
	docker-compose up --build

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
