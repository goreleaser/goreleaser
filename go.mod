module github.com/goreleaser/goreleaser

go 1.12

require (
	code.gitea.io/gitea v1.10.0-dev.0.20190711052757-a0820e09fbf7
	code.gitea.io/sdk/gitea v0.0.0-20190802154435-bbad0d915e44
	github.com/Masterminds/semver v1.4.2
	github.com/apex/log v1.1.0
	github.com/aws/aws-sdk-go v1.19.16
	github.com/caarlos0/ctrlc v1.0.0
	github.com/campoy/unique v0.0.0-20180121183637-88950e537e7e
	github.com/fatih/color v1.7.0
	github.com/google/go-github/v25 v25.0.1
	github.com/goreleaser/nfpm v0.13.0
	github.com/imdario/mergo v0.3.6
	github.com/jarcoal/httpmock v0.0.0-20180424175123-9c70cfe4a1da
	github.com/kamilsk/retry/v4 v4.0.0
	github.com/mattn/go-colorable v0.0.9 // indirect
	github.com/mattn/go-zglob v0.0.0-20180803001819-2ea3427bfa53
	github.com/mitchellh/go-homedir v1.1.0
	github.com/pkg/errors v0.8.1
	github.com/stretchr/testify v1.3.0
	github.com/xanzy/go-gitlab v0.19.0
	gocloud.dev v0.15.0
	golang.org/x/net v0.0.0-20190620200207-3b0461eec859 // indirect
	golang.org/x/oauth2 v0.0.0-20190402181905-9f3314589c9a
	golang.org/x/sync v0.0.0-20190423024810-112230192c58
	golang.org/x/sys v0.0.0-20190626221950-04f50cda93cb // indirect
	gopkg.in/alecthomas/kingpin.v2 v2.2.6
	gopkg.in/yaml.v2 v2.2.2
)

// related to an invalid pseudo version in code.gitea.io/gitea v1.10.0-dev.0.20190711052757-a0820e09fbf7
replace github.com/go-macaron/cors => github.com/go-macaron/cors v0.0.0-20190418220122-6fd6a9bfe14e
