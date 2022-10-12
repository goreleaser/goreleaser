# Monorepo

!!! success "GoReleaser Pro"
    The monorepo support is a [GoReleaser Pro feature](/pro/).

If you want to use GoReleaser within a monorepo and use tag prefixes to mark
"which tags belong to which sub project", GoReleaser has you covered.

## Premises

You project falls into either one of these categories:

1. tags are like `subproject1/v1.2.3` and `subproject2/v1.2.3`;
1. tags are like `@user/thing@v1.2.3` (for a NPM package, for example)
  and `v1.2.3` for the rest of the (Go) code.

## Usage

### Category 1

You'll need to create a `.goreleaser.yaml` for each subproject you want to use
GoReleaser in:

```yaml
# subroj1/.goreleaser.yaml
project_name: subproj1

monorepo:
  tag_prefix: subproject1/
  dir: subproj1
```

Then, you can release with (from the project's root directory):

```bash
goreleaser release --rm-dist -f ./subproj1/.goreleaser.yaml
```

Then, the following is different from a "regular" run:

- GoReleaser will then look if current commit has a tag prefixed with
  `subproject1`, and the previous tag with the same prefix;
- Changelog will include only commits that contain changes to files within the
  `subproj1` directory;
- Release name gets prefixed with `{{ .ProjectName }} ` if empty;
- All build's `dir` setting get set to `monorepo.dir` if empty;
  - if yours is not, you might want to change that manually;
- Extra files on the release, archives, Docker builds, etc are prefixed with
  `monorepo.dir`;
- If using `changelog.use: git`, only commits matching files in `monorepo.dir`
  will be included in the changelog.
- On templates, `{{.PrefixedTag}}` will be `monorepo.prefix/tag` (aka the actual
  tag name), and `{{.Tag}}` has the prefix stripped;

The rest of the release process should work as usual.


### Category 2

You'll need to create a `.goreleaser.yaml` for your Go code in the root of the
project:

```yaml
# .goreleaser.yaml
monorepo:
  tag_prefix: v
```

Then, you can release with:

```bash
goreleaser release --rm-dist
```

GoReleaser will then ignore the tags that are not prefixed with `v`, and it
should work as expected from there on.
