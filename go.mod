module github.com/metal-stack/metal-api

go 1.13

require (
	github.com/Masterminds/semver/v3 v3.0.3
	github.com/dustin/go-humanize v1.0.0
	github.com/emicklei/go-restful v2.12.0+incompatible
	// FIXME need to update to v2
	github.com/emicklei/go-restful-openapi v1.3.0
	github.com/go-openapi/spec v0.19.7
	github.com/go-stack/stack v1.8.0
	github.com/golang/protobuf v1.3.5
	github.com/google/go-cmp v0.4.0
	github.com/metal-stack/go-ipam v1.3.2
	github.com/metal-stack/masterdata-api v0.6.1
	github.com/metal-stack/metal-lib v0.3.4
	github.com/metal-stack/security v0.3.0
	github.com/metal-stack/v v1.0.2
	github.com/nsqio/go-nsq v1.0.8
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.5.1
	github.com/spf13/cobra v1.0.0
	github.com/spf13/viper v1.6.3
	github.com/stretchr/testify v1.5.1
	github.com/testcontainers/testcontainers-go v0.4.0
	go.uber.org/zap v1.14.1
	golang.org/x/crypto v0.0.0-20200406173513-056763e48d71
	google.golang.org/grpc v1.28.0
	gopkg.in/rethinkdb/rethinkdb-go.v6 v6.2.1
)
