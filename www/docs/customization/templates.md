# Name Templates

Several fields in GoReleaser's config file support templating.

Those fields are often suffixed with `_template`, but sometimes they may not
be. The documentation of each section should be explicit about which fields
support templating.

## Common Fields

On fields that support templating, these fields are always available:

| Key                    | Description                                                                                            |
|------------------------|--------------------------------------------------------------------------------------------------------|
| `.ProjectName`         | the project name                                                                                       |
| `.Version`             | the version being released[^1]                                                                         |
| `.Branch`              | the current git branch                                                                                 |
| `.PrefixedTag`         | the current git tag prefixed with the monorepo config tag prefix (if any)                              |
| `.Tag`                 | the current git tag                                                                                    |
| `.PrefixedPreviousTag` | the previous git tag prefixed with the monorepo config tag prefix (if any)                             |
| `.PreviousTag`         | the previous git tag, or empty if no previous tags                                                     |
| `.ShortCommit`         | the git commit short hash                                                                              |
| `.FullCommit`          | the git commit full hash                                                                               |
| `.Commit`              | the git commit hash (deprecated)                                                                       |
| `.CommitDate`          | the UTC commit date in RFC 3339 format                                                                 |
| `.CommitTimestamp`     | the UTC commit date in Unix format                                                                     |
| `.GitURL`              | the git remote url                                                                                     |
| `.Major`               | the major part of the version[^2]                                                                      |
| `.Minor`               | the minor part of the version[^2]                                                                      |
| `.Patch`               | the patch part of the version[^2]                                                                      |
| `.Prerelease`          | the prerelease part of the version, e.g. `beta`[^2]                                                    |
| `.RawVersion`          | composed of `{Major}.{Minor}.{Patch}` [^2]                                                             |
| `.ReleaseNotes`        | the generated release notes, available after the changelog step has been executed                      |
| `.IsSnapshot`          | `true` if `--snapshot` is set, `false` otherwise                                                       |
| `.IsNightly`           | `true` if `--nightly` is set, `false` otherwise                                                        |
| `.Env`                 | a map with system's environment variables                                                              |
| `.Date`                | current UTC date in RFC 3339 format                                                                    |
| `.Timestamp`           | current UTC time in Unix format                                                                        |
| `.ModulePath`          | the go module path, as reported by `go list -m`                                                        |
| `incpatch "v1.2.4"`    | increments the patch of the given version[^3]                                                          |
| `incminor "v1.2.4"`    | increments the minor of the given version[^3]                                                          |
| `incmajor "v1.2.4"`    | increments the major of the given version[^3]                                                          |
| `.ReleaseURL`          | the current release download url[^4]                                                                   |
| `.Summary`             | the git summary, e.g. `v1.0.0-10-g34f56g3`[^5]                                                         |
| `.PrefixedSummary`     | the git summary prefixed with the monorepo config tag prefix (if any)                                  |
| `.Subject`             | the annotated tag message, or the message of the commit it points out to                               |

[^1]: The `v` prefix is stripped and it might be changed in `snapshot` and `nightly` builds.
[^2]: Assuming `Tag` is a valid a SemVer, otherwise empty/zeroed.
[^3]: Will panic if not a semantic version.
[^4]: Composed from the current SCM's download URL and current tag. For instance, on GitHub, it'll be `https://github.com/{owner}/{repo}/releases/tag/{tag}`.
[^5]: It is generated by `git describe --dirty --always --tags`, the format will be `{Tag}-$N-{CommitSHA}`

## Single-artifact extra fields

On fields that are related to a single artifact (e.g., the binary name), you
may have some extra fields:

| Key             | Description                           |
|-----------------|---------------------------------------|
| `.Os`           | `GOOS`[^6]                            |
| `.Arch`         | `GOARCH`[^6]                          |
| `.Arm`          | `GOARM`[^6]                           |
| `.Mips`         | `GOMIPS`[^6]                          |
| `.Binary`       | binary name                           |
| `.ArtifactName` | archive name                          |
| `.ArtifactPath` | absolute path to artifact             |

[^6]: Might have been replaced by `archives.replacements`.

## nFPM extra fields

On the nFPM name template field, you can use those extra fields as well:

| Key            | Description                                                |
|----------------|------------------------------------------------------------|
| `.Release`     | release from the nfpm config                               |
| `.Epoch`       | epoch from the nfpm config                                 |
| `.PackageName` | package the name. Same as `ProjectName` if not overridden. |
| `.ConventionalFileName` | conventional package file name as provided by nFPM |

## Functions

On all fields, you have these available functions:

| Usage                   | Description                                                                                                                    |
|-------------------------|--------------------------------------------------------------------------------------------------------------------------------|
| `replace "v1.2" "v" ""` | replaces all matches. See [ReplaceAll](https://golang.org/pkg/strings/#ReplaceAll)                                             |
| `time "01/02/2006"`     | current UTC time in the specified format (this is not deterministic, a new time for every call)                                |
| `tolower "V1.2"`        | makes input string lowercase. See [ToLower](https://golang.org/pkg/strings/#ToLower)                                           |
| `toupper "v1.2"`        | makes input string uppercase. See [ToUpper](https://golang.org/pkg/strings/#ToUpper)                                           |
| `trim " v1.2  "`        | removes all leading and trailing white space. See [TrimSpace](https://golang.org/pkg/strings/#TrimSpace)                       |
| `trimprefix "v1.2" "v"` | removes provided leading prefix string, if present. See [TrimPrefix](https://golang.org/pkg/strings/#TrimPrefix)               |
| `trimsuffix "1.2v" "v"` | removes provided trailing suffix string, if present. See [TrimSuffix](https://pkg.go.dev/strings#TrimSuffix)                   |
| `dir .Path`             | returns all but the last element of path, typically the path's directory. See [Dir](https://golang.org/pkg/path/filepath/#Dir) |
| `abs .ArtifactPath`     | returns an absolute representation of path. See [Abs](https://golang.org/pkg/path/filepath/#Abs)                               |

With all those fields, you may be able to compose the name of your artifacts
pretty much the way you want:

```yaml
example_template: '{{ tolower .ProjectName }}_{{ .Env.USER }}_{{ time "2006" }}'
```

For example, if you want to add the go version to some artifact:

```yaml
foo_template: 'foo_{{ .Env.GOVERSION }}'
```

And then you can run:

```sh
GOVERSION_NR=$(go version | awk '{print $3;}') goreleaser
```

!!! warning
    Note that those are hypothetical examples and the fields `foo_template` and
    `example_template` are not valid GoReleaser configurations.

## Custom variables

!!! success "GoReleaser Pro"
     Custom template variables support is a [GoReleaser Pro feature](/pro/).

You can also declare custom variables.
This feature is specially useful with [includes](/customization/includes/), so you can have more generic config files.

Usage is as simple as you would expect:

```yaml
# .goreleaser.yml
variables:
  description: my project description
  somethingElse: yada yada yada
  empty: ""
```

And then you can use those fields as `{{ .description }}`, for example.

!!! warning
    You won't be allowed to override GoReleaser "native" fields.
