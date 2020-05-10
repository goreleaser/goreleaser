---
title: Name Templates
---

Several fields in GoReleaser's config file support templating.

Those fields are often suffixed with `_template`, but sometimes they may not
be. The documentation of each section should explicit in which fields
templating is available.

On fields that support templating, this fields are always available:

|      Key       |                   Description                    |
| :------------: | :----------------------------------------------: |
| `.ProjectName` |                 the project name                 |
|   `.Version`   | the version being released (`v` prefix stripped) |
|     `.Tag`     |               the current git tag                |
| `.ShortCommit` |            the git commit short hash             |
| `.FullCommit`  |            the git commit full hash              |
|   `.Commit`    |       the git commit hash (deprecated)           |
|   `.GitURL`    |               the git remote url                 |
|    `.Major`    |          the major part of the version           |
|    `.Minor`    |          the minor part of the version           |
|    `.Patch`    |          the patch part of the version           |
|     `.Env`     |    a map with system's environment variables     |
|    `.Date`     |        current UTC date in RFC3339 format        |
|  `.Timestamp`  |         current UTC time in Unix format          |

On fields that are related to a single artifact (e.g., the binary name), you
may have some extra fields:

|       Key       |              Description              |
| :-------------: | :-----------------------------------: |
|      `.Os`      |  `GOOS` (usually allow replacements)  |
|     `.Arch`     | `GOARCH` (usually allow replacements) |
|     `.Arm`      | `GOARM` (usually allow replacements)  |
|     `.Mips`     | `GOMIPS` (usually allow replacements) |
|    `.Binary`    |              Binary name              |
| `.ArtifactName` |             Archive name              |
| `.ArtifactPath` |       Relative path to artifact       |

On the NFPM name template field, you can use those extra fields as well:

|       Key       |              Description              |
| :-------------: | :-----------------------------------: |
|   `.Release`    |     Release from the nfpm config      |
|    `.Epoch`     |      Epoch from the nfpm config       |

On all fields, you have these available functions:

|        Usage            |               Description                                                                                |
| :--------------------:  | :----------------------------------------------------------------------------------:                     |
| `replace "v1.2" "v" ""` | replaces all matches. See [ReplaceAll](https://golang.org/pkg/strings/#ReplaceAll)                       |
| `time "01/02/2006"`     | current UTC time in the specified format                                                                 |
| `tolower "V1.2"`        | makes input string lowercase. See [ToLower](https://golang.org/pkg/strings/#ToLower)                     |
| `toupper "v1.2"`        | makes input string uppercase. See [ToUpper](https://golang.org/pkg/strings/#ToUpper)                     |
| `trim " v1.2  "`        | removes all leading and trailing white space. See [TrimSpace](https://golang.org/pkg/strings/#TrimSpace) |
| `dir .Path`             | returns all but the last element of path, typically the path's directory. See [Dir](https://golang.org/pkg/path/filepath/#Dir) |
| `abs .ArtifactPath`     | returns an absolute representation of path. See [Abs](https://golang.org/pkg/path/filepath/#Abs) |

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

> Note that those are hypothetical examples and the fields `foo_template` and
> `example_template` are not valid GoReleaser configurations.
