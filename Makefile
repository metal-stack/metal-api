.ONESHELL:
SHA := $(shell git rev-parse --short=8 HEAD)
GITVERSION := $(shell git describe --long --all)
BUILDDATE := $(shell date -Iseconds)

BINARY := maas-api

all: $(BINARY);

%:
	cd cmd/$@
	CGO_ENABLE=0 GO111MODULE=on go build -tags netgo -ldflags "-X 'main.revision=$(GITVERSION)' -X 'main.builddate=$(BUILDDATE)'" -o bin/$@

up:
	docker-compose up --build
