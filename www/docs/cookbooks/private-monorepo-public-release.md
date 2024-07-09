# Building in a private monorepo, publishing in to a public repository

One fairly common usecase is on open-core projects is to have the code in a
private monorepo, but publish its binaries to a public repository.

This cookbook gives some suggestions on how to handle that.

{% include-markdown "../includes/pro.md" comments=false %}

Usually, you'll rely on tag prefixes for each sub-project within your monorepo.
GoReleaser can handle that within its [monorepo configuration][Monorepo]:

```yaml
monorepo:
  tag_prefix: app1/
  dir: ./app1/
```

With that you can already push a tag `app1/v1.0.0`, for example, and GoReleaser
should gracefully handling everything.

But, if you want the release to happen in another repository, you'll also need
to add some [release][Release] settings:

```yaml
release:
  github:
    owner: myorg
    name: myrepo
```

When you release now, it'll create the `app1/v1.0.0` tag and respective release
in `myorg/myrepo`.

## Removing the `myapp/` prefix

Maybe you'll create one public repository to release each of the projects in
your monorepo. In that case, the tag prefix on the public repository makes no
sense.

You can remove it by setting the `release.tag` field:

```yaml
release:
  tag: "{{ .Tag }}"
  github:
    owner: myorg
    name: app1
```

!!! info

    On GoReleaser Pro, `{{.Tag}}` is the tag without the prefix, and the
    prefixed tag can be accessed with `{{.PrefixedTag}}`. Check the
    [documentation][Template variables] for more information.

## Learning more

Make sure to take a look at the following documentation pages:

- [Monorepo](../customization/monorepo.md)
- [Release](../customization/release.md)
- [Template variables](../customization/templates.md)
