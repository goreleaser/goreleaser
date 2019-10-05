module github.com/goreleaser/goreleaser

go 1.13

require (
	code.gitea.io/gitea v1.10.0-dev.0.20190711052757-a0820e09fbf7
	code.gitea.io/sdk/gitea v0.0.0-20190915142708-a6d0aab59332
	github.com/Masterminds/semver v1.5.0
	github.com/apex/log v1.1.1
	github.com/aws/aws-sdk-go v1.23.20
	github.com/caarlos0/ctrlc v1.0.0
	github.com/campoy/unique v0.0.0-20180121183637-88950e537e7e
	github.com/fatih/color v1.7.0
	github.com/google/go-github/v25 v25.1.3
	github.com/goreleaser/nfpm v0.13.0
	github.com/imdario/mergo v0.3.7
	github.com/jarcoal/httpmock v0.0.0-20180424175123-9c70cfe4a1da
	github.com/kamilsk/retry/v4 v4.3.1
	github.com/mattn/go-zglob v0.0.1
	github.com/mitchellh/go-homedir v1.1.0
	github.com/pkg/errors v0.8.1
	github.com/stretchr/testify v1.4.0
	github.com/xanzy/go-gitlab v0.20.1
	gocloud.dev v0.17.0
	golang.org/x/oauth2 v0.0.0-20190604053449-0f29369cfe45
	golang.org/x/sync v0.0.0-20190423024810-112230192c58
	gopkg.in/alecthomas/kingpin.v2 v2.2.6
	gopkg.in/yaml.v2 v2.2.4
)

// Fix invalid pseudo-version: revision is longer than canonical (6fd6a9bfe14e)
replace github.com/go-macaron/cors => github.com/go-macaron/cors v0.0.0-20190418220122-6fd6a9bfe14e
