module git.f-i-ts.de/cloud-native/metal/metal-api

require (
	git.f-i-ts.de/cloud-native/metallib v0.2.6
	github.com/Microsoft/go-winio v0.4.14 // indirect
	github.com/bitly/go-hostpool v0.0.0-20171023180738-a3a6125de932 // indirect
	github.com/bmizerany/assert v0.0.0-20160611221934-b7ed37b82869 // indirect
	github.com/dustin/go-humanize v1.0.0
	github.com/emicklei/go-restful v2.9.6+incompatible
	github.com/emicklei/go-restful-openapi v1.2.0
	github.com/go-openapi/spec v0.19.3
	github.com/go-stack/stack v1.8.0
	github.com/gogo/protobuf v1.2.2-0.20190723190241-65acae22fc9d // indirect
	github.com/golang/snappy v0.0.1 // indirect
	github.com/google/go-cmp v0.3.1
	github.com/gorilla/mux v1.7.3 // indirect
	github.com/konsorten/go-windows-terminal-sequences v1.0.2 // indirect
	github.com/magiconair/properties v1.8.1 // indirect
	github.com/metal-pod/go-ipam v1.3.0
	github.com/metal-pod/security v0.0.0-20190920091500-ed81ae92725b
	github.com/metal-pod/v v0.0.2
	github.com/nsqio/go-nsq v1.0.7
	github.com/opentracing/opentracing-go v1.1.0 // indirect
	github.com/pelletier/go-toml v1.4.0 // indirect
	github.com/pkg/errors v0.8.1
	github.com/prometheus/client_golang v1.2.0
	github.com/spf13/afero v1.2.2 // indirect
	github.com/spf13/cobra v0.0.5
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/spf13/viper v1.4.0
	github.com/stretchr/testify v1.4.0
	github.com/testcontainers/testcontainers-go v0.0.5
	go.uber.org/zap v1.10.0
	golang.org/x/crypto v0.0.0-20191107222254-f4817d981bb6
	google.golang.org/genproto v0.0.0-20190819201941-24fa4b261c55 // indirect
	google.golang.org/grpc v1.23.0 // indirect
	gopkg.in/rethinkdb/rethinkdb-go.v5 v5.0.1
	gopkg.in/yaml.v2 v2.2.4 // indirect
)

exclude github.com/emicklei/go-restful-openapi v1.0.0

// replace git.f-i-ts.de/cloud-native/metallib => ../../metallib

// required because by default viper depends on etcd v3.3.10 which has a corrupt sum
replace github.com/coreos/etcd => github.com/coreos/etcd v3.3.15+incompatible

go 1.13
