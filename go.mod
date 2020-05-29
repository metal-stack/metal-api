module github.com/metal-stack/metal-api

go 1.13

require (
	github.com/Masterminds/semver/v3 v3.1.0
	github.com/Microsoft/hcsshim v0.8.9 // indirect
	github.com/containerd/continuity v0.0.0-20200413184840-d3ef23f19fbb // indirect
	github.com/dustin/go-humanize v1.0.0
	github.com/emicklei/go-restful-openapi/v2 v2.1.0
	github.com/emicklei/go-restful/v3 v3.1.0
	github.com/go-openapi/spec v0.19.8
	github.com/go-stack/stack v1.8.0
	github.com/google/go-cmp v0.4.0
	github.com/gopherjs/gopherjs v0.0.0-20200217142428-fce0ec30dd00 // indirect
	github.com/gorilla/mux v1.7.4 // indirect
	github.com/mattn/go-sqlite3 v2.0.3+incompatible // indirect
	github.com/metal-stack/go-ipam v1.5.0
	github.com/metal-stack/masterdata-api v0.7.1
	github.com/metal-stack/metal-lib v0.4.0
	github.com/metal-stack/security v0.3.0
	github.com/metal-stack/v v1.0.2
	github.com/morikuni/aec v1.0.0 // indirect
	github.com/nsqio/go-nsq v1.0.8
	github.com/pelletier/go-toml v1.8.0 // indirect
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.6.0
	github.com/smartystreets/assertions v1.1.0 // indirect
	github.com/spf13/afero v1.2.2 // indirect
	github.com/spf13/cast v1.3.1 // indirect
	github.com/spf13/cobra v1.0.0
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/spf13/viper v1.7.0
	github.com/stretchr/testify v1.5.1
	github.com/testcontainers/testcontainers-go v0.5.1
	go.uber.org/zap v1.15.0
	golang.org/x/crypto v0.0.0-20200510223506-06a226fb4e37
	golang.org/x/time v0.0.0-20200416051211-89c76fbcd5d1 // indirect
	gopkg.in/ini.v1 v1.56.0 // indirect
	gopkg.in/rethinkdb/rethinkdb-go.v6 v6.2.1
)

replace (
	// FIXME updating prometheus client leads to stack overflow on make spec
	// the newer prometheus client comes with protobuf 1.4.x which creates a recursion somehow.
	github.com/golang/protobuf => github.com/golang/protobuf v1.3.5
	github.com/metal-stack/metal-lib => ../metal-lib
	github.com/prometheus/client_golang => github.com/prometheus/client_golang v1.5.1
)
