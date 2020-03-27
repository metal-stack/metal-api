BINARY := metal-api
MAINMODULE := github.com/metal-stack/metal-api/cmd/metal-api
COMMONDIR := $(or ${COMMONDIR},../builder)

include $(COMMONDIR)/Makefile.inc

.PHONY: all
all:: gofmt
	go mod tidy

release:: all;

.PHONY: spec
spec: all
	@$(info spec=$$(bin/metal-api dump-swagger | jq -S 'walk(if type == "array" then sort_by(strings) else . end)' 2>/dev/null) && echo "$${spec}" > spec/metal-api.json)
	@spec=`bin/metal-api dump-swagger | jq -S 'walk(if type == "array" then sort_by(strings) else . end)' 2>/dev/null` && echo "$${spec}" > spec/metal-api.json || { echo "jq >1.6 required"; exit 1; }

.PHONY: redoc
redoc:
	docker run -it --rm -v $(PWD):/work -w /work letsdeal/redoc-cli bundle -o generate/index.html /work/spec/metal-api.json
	xdg-open generate/index.html
