# Templates

Several fields in GoReleaser's config file support templating.

Those fields are often suffixed with `_template`, but sometimes they may not
be. The documentation of each section should be explicit about which fields
support templating.

## Common Fields

In fields that support templates, these fields are always available:

| Key                | Description                                                                                                |
| ------------------ | ---------------------------------------------------------------------------------------------------------- |
| `.ProjectName`     | the project name                                                                                           |
| `.Version`         | the version being released[^version-prefix]                                                                |
| `.Branch`          | the current git branch                                                                                     |
| `.Tag`             | the current git tag                                                                                        |
| `.PreviousTag`     | the previous git tag, or empty if no previous tags                                                         |
| `.ShortCommit`     | the git commit short hash                                                                                  |
| `.FullCommit`      | the git commit full hash                                                                                   |
| `.Commit`          | the git commit hash (deprecated)                                                                           |
| `.CommitDate`      | the UTC commit date in RFC 3339 format                                                                     |
| `.CommitTimestamp` | the UTC commit date in Unix format                                                                         |
| `.GitURL`          | the git remote url                                                                                         |
| `.GitTreeState`    | either 'clean' or 'dirty'                                                                                  |
| `.IsGitClean`      | whether or not current git state is clean                                                                  |
| `.IsGitDirty`      | whether or not current git state is dirty                                                                  |
| `.Major`           | the major part of the version[^tag-is-semver]                                                              |
| `.Minor`           | the minor part of the version[^tag-is-semver]                                                              |
| `.Patch`           | the patch part of the version[^tag-is-semver]                                                              |
| `.Prerelease`      | the prerelease part of the version, e.g. `beta.1`[^tag-is-semver]                                          |
| `.RawVersion`      | composed of `{Major}.{Minor}.{Patch}` [^tag-is-semver]                                                     |
| `.ReleaseNotes`    | the generated release notes, available after the changelog step has been executed                          |
| `.IsDraft`         | `true` if `release.draft` is set in the configuration, `false` otherwise                                   |
| `.IsSnapshot`      | `true` if `--snapshot` is set, `false` otherwise                                                           |
| `.IsNightly`       | `true` if `--nightly` is set, `false` otherwise                                                            |
| `.IsSingleTarget`  | `true` if `--single-target` is set, `false` otherwise (since v2.3)                                         |
| `.Env`             | a map with system's environment variables                                                                  |
| `.Date`            | current UTC date in RFC 3339 format                                                                        |
| `.Now`             | current UTC date as `time.Time` struct, allows all `time.Time` functions (e.g. `{{ .Now.Format "2006" }}`) |
| `.Timestamp`       | current UTC time in Unix format                                                                            |
| `.ModulePath`      | the go module path, as reported by `go list -m`                                                            |
| `.ReleaseURL`      | the current release download url[^scm-release-url]                                                         |
| `.Summary`         | the git summary, e.g. `v1.0.0-10-g34f56g3`[^git-summary]                                                   |
| `.TagSubject`      | the annotated tag message subject, or the message subject of the commit it points out[^git-tag-subject]    |
| `.TagContents`     | the annotated tag message, or the message of the commit it points out[^git-tag-body]                       |
| `.TagBody`         | the annotated tag message's body, or the message's body of the commit it points out[^git-tag-body]         |
| `.Runtime.Goos`    | equivalent to `runtime.GOOS`                                                                               |
| `.Runtime.Goarch`  | equivalent to `runtime.GOARCH`                                                                             |
| `.Outputs`         | custom outputs (since v2.11)                                                                               |

## Common Fields (Pro)

<!-- md:tmpl_pro -->

| Key                    | Description                                                                |
| ---------------------- | -------------------------------------------------------------------------- |
| `.PrefixedTag`         | the current git tag prefixed with the monorepo config tag prefix (if any)  |
| `.PrefixedPreviousTag` | the previous git tag prefixed with the monorepo config tag prefix (if any) |
| `.PrefixedSummary`     | the git summary prefixed with the monorepo config tag prefix (if any)      |
| `.IsRelease`           | `true` if regular release (not a nightly nor a snapshot) (since v2.8)      |
| `.IsMerging`           | `true` if you are running with `--merge` (since v2.8)                      |
| `.Artifacts`           | [the current artifacts list](#artifacts)                                   |
| `.Metadata`            | [project metadata fields](#metadata) (since v2.13-unreleased)              |

## Metadata

<!-- md:featpro -->

<!-- md:version v2.13-unreleased -->

If you use the `.Metadata` field, it evaluates to the project metadata configuration.
You can access all fields defined in the `metadata` section of your config:

| Key                      | Description                     |
| ------------------------ | ------------------------------- |
| `.Metadata.Description`  | project description             |
| `.Metadata.Homepage`     | project homepage URL            |
| `.Metadata.License`      | project license                 |
| `.Metadata.Maintainers`  | list of project maintainers     |
| `.Metadata.ModTimestamp` | modification timestamp template |

## Artifacts

<!-- md:featpro -->

If you use the `.Artifacts` field, it evaluates to an
[`artifact.Artifact` list](https://pkg.go.dev/github.com/goreleaser/goreleaser@main/internal/artifact#Artifact).
You should be able to use all its fields on each item:

- `.Name`
- `.Path`
- `.Goos`
- `.Goarch`
- `.Goarm`
- `.Gomips`
- `.Goamd64`
- `.Goarm64` (since v2.4)
- `.Gomips64` (since v2.4)
- `.Goppc64` (since v2.4)
- `.Goriscv64` (since v2.4)
- `.Go386` (since v2.4)
- `.Target` (Since v2.5)
- `.Type`
- `.Extra`

## Single-artifact extra fields

On fields that are related to a single artifact (e.g., the binary name), you
may have some extra fields:

| Key             | Description                                   |
| --------------- | --------------------------------------------- |
| `.Os`           | `GOOS`                                        |
| `.Arch`         | `GOARCH`                                      |
| `.Arm`          | `GOARM`                                       |
| `.Mips`         | `GOMIPS`                                      |
| `.Amd64`        | `GOAMD64`                                     |
| `.Arm64`        | `GOARM64` (since v2.4)                        |
| `.Mips64`       | `GOMIPS64` (since v2.4)                       |
| `.Ppc64`        | `GOPPC64` (since v2.4)                        |
| `.Riscv64`      | `GORISCV64` (since v2.4)                      |
| `.I386`         | `GO386` (since v2.4)                          |
| `.Target`       | the whole target (since v2.5)                 |
| `.Binary`       | artifact name                                 |
| `.ArtifactID`   | artifact id (since v2.3[^pro])                |
| `.ArtifactName` | artifact name                                 |
| `.ArtifactPath` | absolute path to artifact                     |
| `.ArtifactExt`  | artifact extension (e.g. `.exe`, `.dmg`, etc) |

## nFPM extra fields

In the nFPM name template field, you can use those extra fields:

| Key                      | Description                                                     |
| ------------------------ | --------------------------------------------------------------- |
| `.Release`               | release from the nfpm config                                    |
| `.Epoch`                 | epoch from the nfpm config                                      |
| `.PackageName`           | package the name. Same as `ProjectName` if not overridden.      |
| `.ConventionalFileName`  | conventional package file name as provided by nFPM.[^arm-names] |
| `.ConventionalExtension` | conventional package extension as provided by nFPM              |
| `.Format`                | package format                                                  |

## Release body extra fields

In the `release.body` field, you can use these extra fields:

| Key          | Description                                                                                                                               |
| ------------ | ----------------------------------------------------------------------------------------------------------------------------------------- |
| `.Checksums` | the current checksum file contents, or a map of filename/checksum contents if `checksum.split` is set. Only available in the release body |

## Functions

On all fields, you have these available functions:

| Usage                             | Description                                                                                                               |
| --------------------------------- | ------------------------------------------------------------------------------------------------------------------------- |
| `replace "v1.2" "v" ""`           | replaces all matches. See [ReplaceAll](https://pkg.go.dev/strings#ReplaceAll)                                             |
| `split "1.2" "."`                 | split string at separator. See [Split](https://pkg.go.dev/strings#Split)                                                  |
| `time "01/02/2006"`               | current UTC time in the specified format (this is not deterministic, a new time for every call)                           |
| `contains "foobar" "foo"`         | checks whether the first string contains the second. See [Contains](https://pkg.go.dev/strings#Contains)                  |
| `tolower "V1.2"`                  | makes input string lowercase. See [ToLower](https://pkg.go.dev/strings#ToLower)                                           |
| `toupper "v1.2"`                  | makes input string uppercase. See [ToUpper](https://pkg.go.dev/strings#ToUpper)                                           |
| `trim " v1.2  "`                  | removes all leading and trailing white space. See [TrimSpace](https://pkg.go.dev/strings#TrimSpace)                       |
| `trimprefix "v1.2" "v"`           | removes provided leading prefix string, if present. See [TrimPrefix](https://pkg.go.dev/strings#TrimPrefix)               |
| `trimsuffix "1.2v" "v"`           | removes provided trailing suffix string, if present. See [TrimSuffix](https://pkg.go.dev/strings#TrimSuffix)              |
| `dir .Path`                       | returns all but the last element of path, typically the path's directory. See [Dir](https://pkg.go.dev/path/filepath#Dir) |
| `base .Path`                      | returns the last element of path. See [Base](https://pkg.go.dev/path/filepath#Base)                                       |
| `abs .ArtifactPath`               | returns an absolute representation of path. See [Abs](https://pkg.go.dev/path/filepath#Abs)                               |
| `filter "text" "regex"`           | keeps only the lines matching the given regex, analogous to `grep -E`                                                     |
| `reverseFilter "text" "regex"`    | keeps only the lines **not** matching the given regex, analogous to `grep -vE`                                            |
| `title "foo"`                     | "titlenize" the string using english as language. See [Title](https://pkg.go.dev/golang.org/x/text/cases#Title)           |
| `mdv2escape "foo"`                | escape characters according to MarkdownV2, especially useful in the Telegram integration                                  |
| `envOrDefault "NAME" "value"`     | either gets the value of the given environment variable, or the given default                                             |
| `isEnvSet "NAME"`                 | returns true if the env is set and not empty, false otherwise                                                             |
| `$m := map "KEY" "VALUE"`         | creates a map from a list of key and value pairs. Both keys and values must be of type `string`                           |
| `indexOrDefault $m "KEY" "value"` | either gets the value of the given key or the given default value from the given map                                      |
| `incpatch "v1.2.4"`               | increments the patch of the given version[^panic-if-not-semver]                                                           |
| `incminor "v1.2.4"`               | increments the minor of the given version[^panic-if-not-semver]                                                           |
| `incmajor "v1.2.4"`               | increments the major of the given version[^panic-if-not-semver]                                                           |
| `urlPathEscape "foo/bar"`         | escapes URL paths. See [PathEscape](https://pkg.go.dev/net/url#PathEscape) (since v2.5)                                   |
| `blake2b .ArtifactPath`           | `blake2b` checksum of the artifact. See [Blake2b](https://pkg.go.dev/golang.org/x/crypto/blake2b) (since v2.9)            |
| `blake2s .ArtifactPath`           | `blake2s` checksum of the artifact. See [Blake2s](https://pkg.go.dev/golang.org/x/crypto/blake2s) (since v2.9)            |
| `crc32 .ArtifactPath`             | `crc32` checksum of the artifact. See [CRC32](https://pkg.go.dev/hash/crc32) (since v2.9)                                 |
| `md5 .ArtifactPath`               | `md5` checksum of the artifact. See [MD5](https://pkg.go.dev/crypto/md5) (since v2.9)                                     |
| `sha224 .ArtifactPath`            | `sha224` checksum of the artifact. See [SHA224](https://pkg.go.dev/crypto/sha256) (since v2.9)                            |
| `sha384 .ArtifactPath`            | `sha384` checksum of the artifact. See [SHA384](https://pkg.go.dev/golang.org/x/crypto/sha3) (since v2.9)                 |
| `sha256 .ArtifactPath`            | `sha256` checksum of the artifact. See [SHA256](https://pkg.go.dev/crypto/sha256) (since v2.9)                            |
| `sha1 .ArtifactPath`              | `sha1` checksum of the artifact. See [SHA1](https://pkg.go.dev/crypto/sha1) (since v2.9)                                  |
| `sha512 .ArtifactPath`            | `sha512` checksum of the artifact. See [SHA512](https://pkg.go.dev/crypto/sha512) (since v2.9)                            |
| `sha3_224 .ArtifactPath`          | `sha3_224` checksum of the artifact. See [SHA3-224](https://pkg.go.dev/golang.org/x/crypto/sha3) (since v2.9)             |
| `sha3_384 .ArtifactPath`          | `sha3_384` checksum of the artifact. See [SHA3-384](https://pkg.go.dev/golang.org/x/crypto/sha3) (since v2.9)             |
| `sha3_256 .ArtifactPath`          | `sha3_256` checksum of the artifact. See [SHA3-256](https://pkg.go.dev/golang.org/x/crypto/sha3) (since v2.9)             |
| `sha3_512 .ArtifactPath`          | `sha3_512` checksum of the artifact. See [SHA3-512](https://pkg.go.dev/golang.org/x/crypto/sha3) (since v2.9)             |
| `mustReadFile "/foo/bar.txt"`     | reads the file contents or fails if it can't be read (since v2.12)                                                        |
| `readFile "/foo/bar.txt"`         | reads the file contents if it it can be read, or return empty string (since v2.12)                                        |

## Functions (Pro)

<!-- md:tmpl_pro -->

| Usage                                | Description                                                                                                                                                                                                             |
| ------------------------------------ | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `list "a" "b" "c"`                   | makes a list of strings                                                                                                                                                                                                 |
| `in (list "a" "b" "c") "b"`          | checks if a slice contains a value                                                                                                                                                                                      |
| `reReplaceAll "(.*)" "foo" "bar-$1"` | compiles the first argument with [`regexp.Compile`](https://pkg.go.dev/regexp#Compile), then uses [`ReplaceAllString`](https://pkg.go.dev/regexp#Regexp.ReplaceAllStringFunc) with the following arguments (since v2.8) |

With all those fields, you may be able to compose the name of your artifacts
pretty much the way you want:

```yaml
example_template: '{{ tolower .ProjectName }}_{{ .Env.USER }}_{{ time "2006" }}'
```

For example, if you want to add the go version to some artifact:

```yaml
foo_template: "foo_{{ .Env.GOVERSION }}"
```

And then you can run:

```sh
GOVERSION=$(go version | awk '{print $3;}') goreleaser
```

!!! warning

    Note that those are hypothetical examples and the fields `foo_template` and
    `example_template` are not valid GoReleaser configurations.

## Custom variables

<!-- md:featpro -->

You can also declare custom variables. This feature is specially useful with
[includes](includes.md), so you can have more generic configuration
files.

Usage is as simple as you would expect:

```yaml title=".goreleaser.yaml"
variables:
  description: my project description
  somethingElse: yada yada yada
  empty: ""
```

And then you can use those fields as `{{ .Var.description }}`, for example.

[^pro]: This feature is only available in [GoReleaser Pro](/pro).

[^version-prefix]:
    The `v` prefix is stripped, and it might be changed in
    `snapshot` and `nightly` builds.

[^tag-is-semver]: Assuming `Tag` is a valid a SemVer, otherwise empty/zeroed.

[^scm-release-url]:
    Composed of the current SCM's download URL and current tag.
    For instance, on GitHub, it'll be
    `https://github.com/{owner}/{repo}/releases/tag/{tag}`.

[^git-summary]:
    It is generated by `git describe --dirty --always --tags`, the
    format will be `{Tag}-$N-{CommitSHA}`
    If using Pro with a `monorepo.tag_prefix`, it will be passed to `describe`
    as `--match={prefix}*`.

[^git-tag-subject]: As reported by `git tag -l --format='%(contents:subject)'`

[^git-tag-body]: As reported by `git tag -l --format='%(contents)'`

[^arm-names]:
    Please beware: some OSs might have the same names for different
    ARM versions, for example, for Debian both ARMv6 and ARMv7 are called `armhf`.
    Make sure that's not your case otherwise you might end up with colliding
    names. It also does not handle multiple GOAMD64 versions.

[^panic-if-not-semver]: Will panic if not a semantic version.
