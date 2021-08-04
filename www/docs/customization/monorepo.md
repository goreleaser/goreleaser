---
title: Monorepo
---

If you want to use GoReleaser within a monorepo and use tag prefixes to mark "which tags belong to which sub project", GoReleaser got you covered.

## Premise

You create your tags like `subproject1/v1.2.3` and `subproject2/v1.2.3`.

## Usage

You'll need to create a `.goreleaser.yml` for each subproject you want to use GoReleaser in:

```yaml
# subroj1/.goreleaser.yml
project_name: subproj1

monorepo:
  tag_prefix: subproject1/
  dir: subproj1
```

Then, you can release with (from the project's root directory):

```sh
goreleaser release --rm-dist -f ./subproj1/.goreleaser.yml
```

Then, the following is different from a "regular" run:

- GoReleaser will then look if current commit has a tag prefixed with `subproject1`, and also the previous tag with the same prefix;
- Changelog will include only commits that contain changes to files within the `subproj1` directory;
- Release name gets prefixed with `{{ .ProjectName }} ` if empty;
- All build's `dir` setting get set to `monorepo.dir` if empty;
  - if yours is not, you might want to change that manually;
- Extra files on the release, archives, Docker builds, etc are prefixed with `monorepo.dir`;
- On templates, `{{.PrefixedTag}}` will be `monorepo.prefix/tag` (aka the actual tag name), and `{{.Tag}}` has the prefix stripped;

The rest of the release process should work as usual.

!!! info
    Monorepo support is a [GoReleaser Pro feature](/pro/).

!!! warning
    This feature is in beta and might change based on feedback.
    Let me know you think about it after trying it out!
