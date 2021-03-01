module github.com/goreleaser/goreleaser

go 1.16

require (
	code.gitea.io/sdk/gitea v0.13.2
	github.com/Djarvur/go-err113 v0.1.0 // indirect
	github.com/Masterminds/semver/v3 v3.1.1
	github.com/apex/log v1.9.0
	github.com/caarlos0/ctrlc v1.0.0
	github.com/campoy/unique v0.0.0-20180121183637-88950e537e7e
	github.com/client9/misspell v0.3.4
	github.com/fatih/color v1.10.0
	github.com/golangci/golangci-lint v1.36.0
	github.com/golangci/misspell v0.3.5 // indirect
	github.com/golangci/revgrep v0.0.0-20180812185044-276a5c0a1039 // indirect
	github.com/google/go-github/v28 v28.1.1
	github.com/goreleaser/fileglob v1.2.0
	github.com/goreleaser/nfpm/v2 v2.3.1
	github.com/gostaticanalysis/analysisutil v0.6.1 // indirect
	github.com/imdario/mergo v0.3.11
	github.com/jarcoal/httpmock v1.0.8
	github.com/jirfag/go-printf-func-name v0.0.0-20200119135958-7558a9eaa5af // indirect
	github.com/matoous/godox v0.0.0-20200801072554-4fb83dc2941e // indirect
	github.com/mattn/go-shellwords v1.0.10
	github.com/mitchellh/go-homedir v1.1.0
	github.com/quasilyte/go-ruleguard v0.2.1 // indirect
	github.com/quasilyte/regex/syntax v0.0.0-20200805063351-8f842688393c // indirect
	github.com/spf13/afero v1.5.1 // indirect
	github.com/spf13/cobra v1.1.3
	github.com/stretchr/objx v0.3.0 // indirect
	github.com/stretchr/testify v1.7.0
	github.com/tdakkota/asciicheck v0.0.0-20200416200610-e657995f937b // indirect
	github.com/timakin/bodyclose v0.0.0-20200424151742-cb6215831a94 // indirect
	github.com/tomarrell/wrapcheck v0.0.0-20201130113247-1683564d9756 // indirect
	github.com/ulikunitz/xz v0.5.10
	github.com/xanzy/go-gitlab v0.44.0
	gocloud.dev v0.22.0
	golang.org/x/oauth2 v0.0.0-20201203001011-0b49973bad19
	golang.org/x/sync v0.0.0-20201020160332-67f06af15bc9
	gopkg.in/yaml.v2 v2.4.0
)

// https://github.com/mattn/go-shellwords/pull/39
replace github.com/mattn/go-shellwords => github.com/caarlos0/go-shellwords v1.0.11
