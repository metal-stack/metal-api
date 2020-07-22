module github.com/metal-stack/metal-api

go 1.13

require (
	github.com/Masterminds/semver/v3 v3.1.0
	github.com/dustin/go-humanize v1.0.0
	github.com/emicklei/go-restful-openapi/v2 v2.2.1
	github.com/emicklei/go-restful/v3 v3.2.0
	github.com/go-openapi/spec v0.19.8
	github.com/go-stack/stack v1.8.0
	github.com/golang/protobuf v1.4.2
	github.com/google/go-cmp v0.5.0
	github.com/metal-stack/go-ipam v1.5.0
	github.com/metal-stack/masterdata-api v0.7.1
	github.com/metal-stack/metal-lib v0.5.0
	github.com/metal-stack/security v0.3.0
	github.com/metal-stack/v v1.0.2
	github.com/nsqio/go-nsq v1.0.8
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.7.1
	github.com/spf13/cobra v1.0.0
	github.com/spf13/viper v1.7.0
	github.com/stretchr/testify v1.6.1
	github.com/testcontainers/testcontainers-go v0.7.0
	go.uber.org/zap v1.15.0
	golang.org/x/crypto v0.0.0-20200709230013-948cd5f35899
	google.golang.org/grpc v1.30.0
	google.golang.org/protobuf v1.25.0
	gopkg.in/rethinkdb/rethinkdb-go.v6 v6.2.1
)

replace (
	//FIXME remove as soon as emicklei has merged our fix
	github.com/emicklei/go-restful-openapi/v2 => ../go-restful-openapi/v2 v2.2.2
	github.com/prometheus/client_golang => github.com/prometheus/client_golang v1.5.1
)
