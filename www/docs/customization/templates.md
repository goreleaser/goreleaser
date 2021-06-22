---
title: Name Templates
---

Several fields in GoReleaser's config file support templating.

Those fields are often suffixed with `_template`, but sometimes they may not
be. The documentation of each section should be explicit about which fields
support templating.

On fields that support templating, these fields are always available:

| Key                | Description                                                                                                                  |
|--------------------|------------------------------------------------------------------------------------------------------------------------------|
| `.ProjectName`     | the project name                                                                                                             |
| `.Version`         | the version being released (`v` prefix stripped),<br>or `{{ .Tag }}-SNAPSHOT-{{ .ShortCommit }}` in case of snapshot release |
| `.Branch`          | the current git branch                                                                                                       |
| `.PrefixedTag`     | the current git tag prefixed with the monorepo config tag prefix (if any)                                                    |
| `.Tag`             | the current git tag                                                                                                          |
| `.ShortCommit`     | the git commit short hash                                                                                                    |
| `.FullCommit`      | the git commit full hash                                                                                                     |
| `.Commit`          | the git commit hash (deprecated)                                                                                             |
| `.CommitDate`      | the UTC commit date in RFC 3339 format                                                                                       |
| `.CommitTimestamp` | the UTC commit date in Unix format                                                                                           |
| `.GitURL`          | the git remote url                                                                                                           |
| `.Major`           | the major part of the version (assuming `Tag` is a valid semver, else `0`)                                                   |
| `.Minor`           | the minor part of the version (assuming `Tag` is a valid semver, else `0`)                                                   |
| `.Patch`           | the patch part of the version (assuming `Tag` is a valid semver, else `0`)                                                   |
| `.Prerelease`      | the prerelease part of the version, e.g. `beta` (assuming `Tag` is a valid semver)                                           |
| `.RawVersion`      | Major.Minor.Patch (assuming `Tag` is a valid semver, else `0.0.0`)                                                           |
| `.IsSnapshot`      | `true` if a snapshot is being released, `false` otherwise                                                                    |
| `.Env`             | a map with system's environment variables                                                                                    |
| `.Date`            | current UTC date in RFC 3339 format                                                                                          |
| `.Timestamp`       | current UTC time in Unix format                                                                                              |
| `.ModulePath`      | the go module path, as reported by `go list -m`                                                                              |

On fields that are related to a single artifact (e.g., the binary name), you
may have some extra fields:

| Key             | Description                           |
|-----------------|---------------------------------------|
| `.Os`           | `GOOS` (usually allow replacements)   |
| `.Arch`         | `GOARCH` (usually allow replacements) |
| `.Arm`          | `GOARM` (usually allow replacements)  |
| `.Mips`         | `GOMIPS` (usually allow replacements) |
| `.Binary`       | Binary name                           |
| `.ArtifactName` | Archive name                          |
| `.ArtifactPath` | Absolute path to artifact             |

On the NFPM name template field, you can use those extra fields as well:

| Key            | Description                                                |
|----------------|------------------------------------------------------------|
| `.Release`     | Release from the nfpm config                               |
| `.Epoch`       | Epoch from the nfpm config                                 |
| `.PackageName` | Package the name. Same as `ProjectName` if not overridden. |

On all fields, you have these available functions:

| Usage                   | Description                                                                                                                    |
|-------------------------|--------------------------------------------------------------------------------------------------------------------------------|
| `replace "v1.2" "v" ""` | replaces all matches. See [ReplaceAll](https://golang.org/pkg/strings/#ReplaceAll)                                             |
| `time "01/02/2006"`     | current UTC time in the specified format (this is not deterministic, a new time for every call)                                |
| `tolower "V1.2"`        | makes input string lowercase. See [ToLower](https://golang.org/pkg/strings/#ToLower)                                           |
| `toupper "v1.2"`        | makes input string uppercase. See [ToUpper](https://golang.org/pkg/strings/#ToUpper)                                           |
| `trim " v1.2  "`        | removes all leading and trailing white space. See [TrimSpace](https://golang.org/pkg/strings/#TrimSpace)                       |
| `trimprefix "v1.2" "v"` | removes provided leading prefix string, if present. See [TrimPrefix](https://golang.org/pkg/strings/#TrimPrefix)                |
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

On [GoReleaser Pro](/pro/) you can also declare custom variables.
This feature is specially useful with [includes](/customization/includes/), so you can have more generic config files.

Usage is as simple as you would expect:

```yaml
# .goreleaser.yml
variables:
  description: my project description
  somethingElse: yada yada yada
```

And then you can use those fields as `{{ .description }}`, for example.

!!! warning
    You won't be allowed to override GoReleaser "native" fields.

!!! info
    Custom variables is a [GoReleaser Pro feature](/pro/).
