
BINARY := metal-api
MAINMODULE := git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api
COMMONDIR := $(or ${COMMONDIR},../common)

include $(COMMONDIR)/Makefile.inc

.PHONY: all
all::
	bin/metal-api dump-swagger >spec/metal-api.json

.PHONY: createmasterdata
createmasterdata:
	@cat masterdata/images.json | jq -r -c -M ".[]" | xargs -d'\n' -L1 -I'{}' curl -XPUT -H "Content-Type: application/json" -d '{}' http://localhost:8080/image
	@cat masterdata/sizes.json | jq -r -c -M ".[]" | xargs -d'\n' -L1 -I'{}' curl -XPUT -H "Content-Type: application/json" -d '{}' http://localhost:8080/size
	@cat masterdata/sites.json | jq -r -c -M ".[]" | xargs -d'\n' -L1 -I'{}' curl -XPUT -H "Content-Type: application/json" -d '{}' http://localhost:8080/site

.PHONY: createtestdevices
createtestdevices:
	@cat masterdata/testdevices.json | jq -r -c -M ".[]" | xargs -d'\n' -L1 -I'{}' curl -XPOST -H "Content-Type: application/json" -d '{}' http://localhost:8080/device/register

.PHONY: localbuild
localbuild: bin/$(BINARY) Dockerfile.dev
	docker build -t registry.fi-ts.io/metal/metal-api -f Dockerfile.dev .

.PHONY: restart
restart: localbuild
	docker-compose restart metal-api

.PHONY: generate-client
generate-client:
	rm -rf netbox-api/*
	cp ../netbox-api-proxy/netbox_api_proxy/api_schemas/v1.yaml netbox-api/v1.yaml
	GO111MODULE=off swagger generate client -f netbox-api/v1.yaml -t netbox-api

