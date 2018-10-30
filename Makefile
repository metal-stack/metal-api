
BINARY := metal-api
MAINMODULE := git.f-i-ts.de/cloud-native/maas/metal-api/cmd/metal-api
COMMONDIR := $(or ${COMMONDIR},../../common)

include $(COMMONDIR)/Makefile.inc

.PHONY: createmasterdata
createmasterdata:
	@cat masterdata/images.json | jq -r -c -M ".[]" | xargs -d'\n' -L1 -I'{}' curl -XPUT -H "Content-Type: application/json" -d '{}' http://localhost:8080/image
	@cat masterdata/sizes.json | jq -r -c -M ".[]" | xargs -d'\n' -L1 -I'{}' curl -XPUT -H "Content-Type: application/json" -d '{}' http://localhost:8080/size
	@cat masterdata/facilities.json | jq -r -c -M ".[]" | xargs -d'\n' -L1 -I'{}' curl -XPUT -H "Content-Type: application/json" -d '{}' http://localhost:8080/facility

.PHONY: createtestdevices
createtestdevices:
	@cat masterdata/testdevices.json | jq -r -c -M ".[]" | xargs -d'\n' -L1 -I'{}' curl -XPOST -H "Content-Type: application/json" -d '{}' http://localhost:8080/device/register

.PHONY: localbuild
localbuild: bin/$(BINARY) Dockerfile.dev
	docker build -t registry.fi-ts.io/metal/metal-api -f Dockerfile.dev .

.PHONY: restart
restart: localbuild
	docker-compose restart metal-api

.PHONY: spec
spec:
	curl http://localhost:8080/apidocs.json >spec/metal-api.json

.PHONY: generate-client
generate-client:
	GO111MODULE=off swagger generate client -f netbox-api/api.yaml -t netbox-api

