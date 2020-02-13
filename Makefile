# PLEASE MAKE SURE TO HAVE THE kubectl CONFIG POINT TO MINIKUBE WHEN LOCAL DEVELOPMENT
BINARY := metal-api
MAINMODULE := git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api
COMMONDIR := $(or ${COMMONDIR},../common)
KCTL := kubectl

include $(COMMONDIR)/Makefile.inc

.PHONY: all
all::
	go mod tidy

release:: all;

.PHONY: spec
spec: all
	bin/metal-api dump-swagger | python -c "$$PYTHON_DEEP_SORT" >spec/metal-api.json

.PHONY: createmasterdata
createmasterdata:
	@metalctl image apply -f masterdata/images.yaml
	@metalctl size apply -f masterdata/sizes.yaml
	@metalctl partition apply -f masterdata/partitions.yaml

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

# this must be run as root, kubefwd neets root priv's. inside my vsc-docker-image
# the SUID bit is set on the kubefwd binary
.PHONY: local-forward
local-forward:
	kubefwd svc

# commands for localkube development. first do a check to make sure we are
# on minikube and do not overwrite other environments by accident.
localkube-install:
	${KCTL} config view | grep minikube && \
	helm install rethink localkube/rethinkdb && \
	helm install metal localkube/metal-control-plane

localkube-upgrade-rethink:
	${KCTL} config view | grep minikube && \
	helm upgrade --force rethink localkube/rethinkdb

localkube-upgrade-metal:
	${KCTL} config view | grep minikube && \
	helm upgrade --force metal localkube/metal-control-plane

.PHONY: redoc
redoc:
	docker run -it --rm -v $(PWD):/work -w /work letsdeal/redoc-cli bundle -o generate/index.html /work/spec/metal-api.json
	xdg-open generate/index.html
