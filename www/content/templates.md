---
title: Name Templates
series: customization
hideFromIndex: true
weight: 25
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
|    `.Binary`    |              Binary name              |
| `.ArtifactName` |             Archive name              |

On all fields, you have these available functions:

|        Usage        |               Description                |
| :-----------------: | :--------------------------------------: |
| `time "01/02/2006"` | current UTC time in the specified format |

With all those fields, you may be able to compose the name of your artifacts
pretty much the way you want:

```yaml
example_template: '{{ .ProjectName }}_{{ .Env.USER }}_{{ time "2006" }}'
```

For example, if you want to add the go version to some artifact:

```yaml
foo_template: 'foo_{{ .Env.GOVERSION }}'
```

And then you can run:

```console
GOVERSION_NR=$(go version | awk '{print $3;}') goreleaser
```

> Note that those are hypothetical examples and the fields `foo_template` and
> `example_template` are not valid GoReleaser configurations.
