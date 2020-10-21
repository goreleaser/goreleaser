module github.com/goreleaser/goreleaser

go 1.15

require (
	code.gitea.io/sdk/gitea v0.13.1
	github.com/Masterminds/semver/v3 v3.1.0
	github.com/apex/log v1.9.0
	github.com/caarlos0/ctrlc v1.0.0
	github.com/campoy/unique v0.0.0-20180121183637-88950e537e7e
	github.com/client9/misspell v0.3.4
	github.com/fatih/color v1.9.0
	github.com/golangci/golangci-lint v1.31.0
	github.com/google/go-github/v28 v28.1.1
	github.com/goreleaser/nfpm v1.8.0
	github.com/hashicorp/go-version v1.2.1 // indirect
	github.com/imdario/mergo v0.3.11
	github.com/jarcoal/httpmock v1.0.6
	github.com/mattn/go-shellwords v1.0.10
	github.com/mattn/go-zglob v0.0.3
	github.com/mitchellh/go-homedir v1.1.0
	github.com/spf13/cobra v1.1.1
	github.com/stretchr/testify v1.6.1
	github.com/ulikunitz/xz v0.5.8
	github.com/xanzy/go-gitlab v0.38.1
	gocloud.dev v0.20.0
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d
	golang.org/x/sync v0.0.0-20200625203802-6e8e738ad208
	gopkg.in/yaml.v2 v2.3.0
)

// https://github.com/mattn/go-shellwords/pull/39
replace github.com/mattn/go-shellwords => github.com/caarlos0/go-shellwords v1.0.11
