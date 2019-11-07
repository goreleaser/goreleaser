module github.com/goreleaser/goreleaser

go 1.13

require (
	code.gitea.io/sdk/gitea v0.0.0-20191013013401-e41e9ea72caa
	github.com/Masterminds/semver/v3 v3.0.1
	github.com/apex/log v1.1.1
	github.com/aws/aws-sdk-go v1.25.11
	github.com/caarlos0/ctrlc v1.0.0
	github.com/campoy/unique v0.0.0-20180121183637-88950e537e7e
	github.com/fatih/color v1.7.0
	github.com/golang/protobuf v1.3.2 // indirect
	github.com/google/go-github/v28 v28.1.1
	github.com/goreleaser/nfpm v1.1.5
	github.com/imdario/mergo v0.3.8
	github.com/jarcoal/httpmock v1.0.4
	github.com/kamilsk/retry/v4 v4.3.1
	github.com/mattn/go-zglob v0.0.1
	github.com/mitchellh/go-homedir v1.1.0
	github.com/pkg/errors v0.8.1
	github.com/stretchr/testify v1.4.0
	github.com/xanzy/go-gitlab v0.21.0
	gocloud.dev v0.17.0
	golang.org/x/net v0.0.0-20191028085509-fe3aa8a45271 // indirect
	golang.org/x/oauth2 v0.0.0-20190604053449-0f29369cfe45
	golang.org/x/sync v0.0.0-20190911185100-cd5d95a43a6e
	google.golang.org/appengine v1.6.5 // indirect
	gopkg.in/alecthomas/kingpin.v2 v2.2.6
	gopkg.in/yaml.v2 v2.2.5
)

// TODO: remove this when https://github.com/google/rpmpack/pull/33 gets merged in.
replace github.com/google/rpmpack => github.com/caarlos0/rpmpack v0.0.0-20191106130752-24a815bfaee0
