module github.com/goreleaser/goreleaser

go 1.15

require (
	code.gitea.io/sdk/gitea v0.13.2
	github.com/Masterminds/semver/v3 v3.1.1
	github.com/apex/log v1.9.0
	github.com/caarlos0/ctrlc v1.0.0
	github.com/campoy/unique v0.0.0-20180121183637-88950e537e7e
	github.com/client9/misspell v0.3.4
	github.com/fatih/color v1.10.0
	github.com/golangci/golangci-lint v1.36.0
	github.com/google/go-github/v28 v28.1.1
	github.com/goreleaser/fileglob v0.3.1
	github.com/goreleaser/nfpm/v2 v2.2.2
	github.com/imdario/mergo v0.3.11
	github.com/jarcoal/httpmock v1.0.8
	github.com/mattn/go-shellwords v1.0.10
	github.com/mitchellh/go-homedir v1.1.0
	github.com/spf13/cobra v1.1.1
	github.com/stretchr/testify v1.7.0
	github.com/ulikunitz/xz v0.5.10
	github.com/xanzy/go-gitlab v0.43.0
	gocloud.dev v0.21.0
	golang.org/x/oauth2 v0.0.0-20201203001011-0b49973bad19
	golang.org/x/sync v0.0.0-20201020160332-67f06af15bc9
	gopkg.in/yaml.v2 v2.4.0
)

// https://github.com/mattn/go-shellwords/pull/39
replace github.com/mattn/go-shellwords => github.com/caarlos0/go-shellwords v1.0.11
