module github.com/goreleaser/goreleaser

go 1.13

require (
	code.gitea.io/gitea v1.10.0-dev.0.20190711052757-a0820e09fbf7
	code.gitea.io/sdk/gitea v0.0.0-20190915142708-a6d0aab59332
	github.com/Masterminds/semver/v3 v3.0.1
	github.com/apex/log v1.1.1
	github.com/aws/aws-sdk-go v1.25.11
	github.com/caarlos0/ctrlc v1.0.0
	github.com/campoy/unique v0.0.0-20180121183637-88950e537e7e
	github.com/fatih/color v1.7.0
	github.com/google/go-github/v28 v28.1.1
	github.com/goreleaser/nfpm v1.0.1-0.20191022035611-07dfa5b67a4a
	github.com/imdario/mergo v0.3.8
	github.com/jarcoal/httpmock v1.0.4
	github.com/kamilsk/retry/v4 v4.3.1
	github.com/mattn/go-zglob v0.0.1
	github.com/mitchellh/go-homedir v1.1.0
	github.com/pkg/errors v0.8.1
	github.com/stretchr/testify v1.4.0
	github.com/xanzy/go-gitlab v0.20.1
	gocloud.dev v0.17.0
	golang.org/x/net v0.0.0-20191003171128-d98b1b443823 // indirect
	golang.org/x/oauth2 v0.0.0-20190604053449-0f29369cfe45
	golang.org/x/sync v0.0.0-20190911185100-cd5d95a43a6e
	google.golang.org/appengine v1.6.4 // indirect
	gopkg.in/alecthomas/kingpin.v2 v2.2.6
	gopkg.in/yaml.v2 v2.2.4
)

// Fix invalid pseudo-version: revision is longer than canonical (b0274f40d4c7)
replace github.com/go-macaron/cors => github.com/go-macaron/cors v0.0.0-20190925001837-b0274f40d4c7
