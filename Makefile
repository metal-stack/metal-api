.ONESHELL:
SHA := $(shell git rev-parse --short=8 HEAD)
GITVERSION := $(shell git describe --long --all)
BUILDDATE := $(shell date -Iseconds)
VERSION := $(or ${VERSION},devel)

BINARY := maas-api

all: $(BINARY);

%:
	cd cmd/$@
	CGO_ENABLE=0 GO111MODULE=on go build -tags netgo -ldflags "-X 'main.version=$(VERSION)' -X 'main.revision=$(GITVERSION)' -X 'main.gitsha1=$(SHA)' -X 'main.builddate=$(BUILDDATE)'" -o bin/$@

up:
	docker-compose up --build

test:
	go test -cover ./... 

test-ci:
	go test ./... -coverprofile=coverage.out && go tool cover -func=coverage.out | tail -n 1; rm coverage.out
