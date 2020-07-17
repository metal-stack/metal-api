BINARY := metal-api
MAINMODULE := github.com/metal-stack/metal-api/cmd/metal-api
COMMONDIR := $(or ${COMMONDIR},../builder)

in-docker: tidy spec protoc gofmt test all;

include $(COMMONDIR)/Makefile.inc

release:: tidy spec protoc gofmt test all ;

.PHONY: spec
spec:
	go build \
    		-tags netgo \
    		-ldflags \
    		"$(LINKMODE)" \
    		-o bin/$(BINARY) \
    		$(MAINMODULE)
	bin/$(BINARY) dump-swagger | jq -r -S 'walk(if type == "array" then sort_by(strings) else . end)' > spec/metal-api.json || { echo "jq >=1.6 required"; exit 1; }

.PHONY: redoc
redoc:
	docker run -it --rm --user $$(id -u):$$(id -g) -v $(PWD):/work -w /work letsdeal/redoc-cli bundle -o generate/index.html /work/spec/metal-api.json
	xdg-open generate/index.html

.PHONY: protoc
protoc:
	docker run -it --rm --user $$(id -u):$$(id -g) -v $(PWD):/work -w /work metalstack/builder protoc -I pkg --go_out plugins=grpc:pkg pkg/api/v1/*.proto
