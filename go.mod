module github.com/goreleaser/goreleaser

go 1.13.

require (
	cloud.google.com/go v0.45.1 // indirect
	code.gitea.io/gitea v1.10.0-dev.0.20190711052757-a0820e09fbf7
	code.gitea.io/sdk/gitea v0.0.0-20190802154435-bbad0d915e44
	contrib.go.opencensus.io/exporter/ocagent v0.6.0 // indirect
	github.com/Azure/azure-amqp-common-go v1.1.4 // indirect
	github.com/Azure/azure-pipeline-go v0.2.2 // indirect
	github.com/Azure/azure-sdk-for-go v33.2.0+incompatible // indirect
	github.com/Azure/azure-storage-blob-go v0.8.0 // indirect
	github.com/Azure/go-autorest v13.0.1+incompatible // indirect
	github.com/Azure/go-autorest/autorest v0.9.1 // indirect
	github.com/Azure/go-autorest/autorest/azure/auth v0.3.0 // indirect
	github.com/Azure/go-autorest/autorest/to v0.3.0 // indirect
	github.com/Azure/go-autorest/autorest/validation v0.2.0 // indirect
	github.com/Masterminds/semver v1.4.2
	github.com/alecthomas/template v0.0.0-20190718012654-fb15b899a751 // indirect
	github.com/alecthomas/units v0.0.0-20190910110746-680d30ca3117 // indirect
	github.com/apache/thrift v0.12.0 // indirect
	github.com/apex/log v1.1.1
	github.com/aws/aws-sdk-go v1.23.19
	github.com/blakesmith/ar v0.0.0-20190502131153-809d4375e1fb // indirect
	github.com/caarlos0/ctrlc v1.0.0
	github.com/campoy/unique v0.0.0-20180121183637-88950e537e7e
	github.com/codahale/hdrhistogram v0.0.0-20161010025455-3a0bb77429bd // indirect
	github.com/fatih/color v1.7.0
	github.com/golang/snappy v0.0.1 // indirect
	github.com/google/go-github/v25 v25.1.3
	github.com/goreleaser/nfpm v0.13.0
	github.com/grpc-ecosystem/grpc-gateway v1.11.1 // indirect
	github.com/hashicorp/golang-lru v0.5.3 // indirect
	github.com/imdario/mergo v0.3.7
	github.com/jarcoal/httpmock v0.0.0-20180424175123-9c70cfe4a1da
	github.com/kamilsk/retry/v4 v4.3.1
	github.com/mattn/go-ieproxy v0.0.0-20190805055040-f9202b1cfdeb // indirect
	github.com/mattn/go-isatty v0.0.9 // indirect
	github.com/mattn/go-zglob v0.0.1
	github.com/mitchellh/go-homedir v1.1.0
	github.com/opentracing/opentracing-go v1.0.2 // indirect
	github.com/openzipkin/zipkin-go v0.1.6 // indirect
	github.com/pkg/errors v0.8.1
	github.com/stretchr/testify v1.4.0
	github.com/tidwall/pretty v0.0.0-20190325153808-1166b9ac2b65 // indirect
	github.com/uber-go/atomic v1.3.2 // indirect
	github.com/uber/jaeger-client-go v2.15.0+incompatible // indirect
	github.com/uber/jaeger-lib v1.5.0 // indirect
	github.com/xanzy/go-gitlab v0.20.1
	github.com/xdg/scram v0.0.0-20180814205039-7eeb5667e42c // indirect
	github.com/xdg/stringprep v1.0.0 // indirect
	go.mongodb.org/mongo-driver v1.0.1 // indirect
	go.opencensus.io v0.22.1 // indirect
	go.uber.org/atomic v1.3.2 // indirect
	gocloud.dev v0.17.0
	golang.org/x/crypto v0.0.0-20190909091759-094676da4a83 // indirect
	golang.org/x/net v0.0.0-20190909003024-a7b16738d86b // indirect
	golang.org/x/oauth2 v0.0.0-20190604053449-0f29369cfe45
	golang.org/x/sync v0.0.0-20190423024810-112230192c58
	golang.org/x/sys v0.0.0-20190910064555-bbd175535a8b // indirect
	google.golang.org/api v0.10.0 // indirect
	google.golang.org/appengine v1.6.2 // indirect
	google.golang.org/genproto v0.0.0-20190905072037-92dd089d5514 // indirect
	google.golang.org/grpc v1.23.0 // indirect
	gopkg.in/alecthomas/kingpin.v2 v2.2.6
	gopkg.in/yaml.v2 v2.2.2
)

// related to an invalid pseudo version in code.gitea.io/gitea v1.10.0-dev.0.20190711052757-a0820e09fbf7
replace github.com/go-macaron/cors => github.com/go-macaron/cors v0.0.0-20190418220122-6fd6a9bfe14e
