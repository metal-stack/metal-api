.PHONY: protoc
protoc: protolint
	buf generate -v

.PHONY: protolint
protolint:
	buf format -w .
	buf lint -v
