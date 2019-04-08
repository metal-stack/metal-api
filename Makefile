# PLEASE MAKE SURE TO HAVE THE kubectl CONFIG POINT TO MINIKUBE WHEN LOCAL DEVELOPMENT
BINARY := metal-api
MAINMODULE := git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api
COMMONDIR := $(or ${COMMONDIR},../common)
KCTL := kubectl

include $(COMMONDIR)/Makefile.inc

.PHONY: all
all::
	@bin/metal-api dump-swagger >spec/metal-api.json
	go mod tidy

release:: all;

.PHONY: createmasterdata
createmasterdata:
	API_BASE_URL := $(or ${API_BASE_URL}, $(shell minikube service -n default --url metal-api))
	@cat masterdata/images.json | jq -r -c -M ".[]" | xargs -d'\n' -L1 -I'{}' curl -XPUT -H "Content-Type: application/json" -d '{}' $(API_BASE_URL)/v1/image
	@cat masterdata/sizes.json | jq -r -c -M ".[]" | xargs -d'\n' -L1 -I'{}' curl -XPUT -H "Content-Type: application/json" -d '{}' $(API_BASE_URL)/v1/size
	@cat masterdata/partitions.json | jq -r -c -M ".[]" | xargs -d'\n' -L1 -I'{}' curl -XPUT -H "Content-Type: application/json" -d '{}' $(API_BASE_URL)/v1/partition

.PHONY: localbuild
localbuild: bin/$(BINARY)

.PHONY: localbuild-push
localbuild-push: localbuild Dockerfile.dev
	@eval $(shell minikube docker-env)
	docker build -t registry.fi-ts.io/metal/metal-api -f Dockerfile.dev .
	${KCTL} delete pod $(shell ${KCTL} get pods -l app=metal-api --field-selector=status.phase=Running --output=jsonpath={.items..metadata.name})

# the watch target needs https://github.com/cortesi/modd
.PHONY: watch
watch:
	modd -n -f ./modd.conf

# localdev should be started in a fresh shell
.PHONY: localdev
localdev:
	cd ../metal-lab/provision/api && docker-compose pull && cd -
	tmux new-session -d 'cd ../metal-lab/provision/api && docker-compose up -d && docker-compose logs -f'
	tmux split-window -v '$(MAKE) watch'
	tmux attach-session -d

# this must be run as root, kubefwd neets root priv's. inside my vsc-docker-image
# the UID bit is set on the kubefwd binary
.PHONY: local-forward
local-forward:
	kubefwd svc -c $HOME/.kube/minikube

# commands for localkube development. first do a check to make sure we are
# on minikube and do not overwrite other environments by accident.
localkube-install:
	${KCTL} config view | grep minikube && \
	helm install -n rethink localkube/rethinkdb && \
	helm install -n metal localkube/metal-control-plane

localkube-upgrade-rethink:
	${KCTL} config view | grep minikube && \
	helm upgrade --force rethink localkube/rethinkdb

localkube-upgrade-metal:
	${KCTL} config view | grep minikube && \
	helm upgrade --force metal localkube/metal-control-plane

.PHONY: generate-client
generate-client:
	rm -rf netbox-api/*
	cp ../netbox-api-proxy/netbox_api_proxy/api_schemas/v1.yaml netbox-api/v1.yaml
	GO111MODULE=off swagger generate client -f netbox-api/v1.yaml -t netbox-api

.PHONY: redoc
redoc:
	docker run -it --rm -v $(PWD):/work -w /work letsdeal/redoc-cli bundle -o generate/redoc.html /work/spec/metal-api.json
	xdg-open generate/redoc.html
