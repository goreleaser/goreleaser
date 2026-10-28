---
title: "Packslip manifests"
weight: 25
---

{{< g_version "v2.20" >}}

[packslip](https://packslip.dev/) is an open format for publishing a signed
release manifest next to your release artifacts. The manifest lists each
artifact's checksums, platform, and executable paths, so installers such as
[mise](https://mise.jdx.dev) can pick the right download and verify it
without guessing from file names.

GoReleaser can create and sign that manifest for the archives and binaries it
produces, and upload it to your release as `packslip.sigstore.json`. A
repository that releases several tools can create one manifest per tool.

## Usage

The manifest is signed keylessly with [Sigstore](https://www.sigstore.dev/),
using the OIDC identity of your CI job, so there is no key to manage. It is
opt-in: add an entry to `packslips` to create a manifest. For a repository
that releases one tool, an empty entry is enough:

```yaml {filename=".goreleaser.yaml"}
packslips:
  - {}
```

All the options:

```yaml {filename=".goreleaser.yaml"}
packslips:
  - # Whether to skip this manifest.
    #
    # Templates: allowed.
    disable: "{{ .IsNightly }}"

    # IDs of the archives and binaries to describe.
    #
    # Default: all of them. Without a `url_template`, only those the release
    # uploads (see `release.ids`).
    ids:
      - foo

    # The packslip project name, in the form `host/owner/name`, with a
    # subpath for one tool of a monorepo (`github.com/owner/name/tool`).
    #
    # Default: 'github.com/<release.github.owner>/<release.github.name>'.
    # Required when releasing anywhere but GitHub.
    project: github.com/owner/name/foo

    # Name of the manifest file.
    #
    # Default: 'packslip.sigstore.json', or 'packslip.<subpath>.sigstore.json'
    # for a project under the release repository, with `/` in the subpath
    # replaced by `-` (`packslip.foo.sigstore.json` here).
    # Templates: allowed.
    name_template: "packslip.foo.sigstore.json"

    # Template for the URL each artifact can be downloaded from.
    #
    # Default: the release page of your SCM.
    # Required when `release.skip_upload` is set, as the release then has no
    # assets to link to.
    # Templates: allowed.
    url_template: "https://dl.example.com/{{ .Tag }}/{{ .ArtifactName }}"
```

On GitHub Actions, the job needs the `id-token: write` permission:

```yaml {filename=".github/workflows/release.yml"}
permissions:
  contents: write
  id-token: write
```

Outside of GitHub Actions, set `SIGSTORE_ID_TOKEN` to an OIDC token whose
identity is a verified email address.

## Monorepos

When one release ships several tools, add an entry per tool, select its
artifacts with `ids`, and name it with a subpath of the repository. For a
repository `acme/tools` whose archives have the IDs `foo` and `bar`:

```yaml {filename=".goreleaser.yaml"}
packslips:
  - ids: [foo]
    project: github.com/acme/tools/foo
  - ids: [bar]
    project: github.com/acme/tools/cmd/bar
```

This uploads `packslip.foo.sigstore.json` and `packslip.cmd-bar.sigstore.json`,
each describing only its tool. Installers pick the manifest by the signed
project name, not the file name. An entry without `ids` describes every
archive and binary, so set `ids` on each entry when a release has several
tools. Two entries cannot have the same project or file name, and a file
name cannot be the name of an archive, checksum, or other artifact.

When each tool has its own release and configuration file instead, a single
entry with the tool's `project` is enough.

## What is described

- Archives and binaries (`UploadableArchive` and `UploadableBinary`
  artifacts), with the platform GoReleaser built them for. A universal macOS
  binary has no architecture.
- Executable paths inside archives, including the wrapping directory.
- Statically linked Linux builds are described as needing no shared
  libraries, so they fit both glibc and musl hosts. For dynamically linked
  Linux builds, GoReleaser reads the ELF program interpreter before optional
  packing and records
  `gnu` or `musl`; it stops rather than publishing a manifest when the runtime
  cannot be determined.
- Builds for a CPU level other than Go's default, such as `goamd64: v3` or
  `goarm64: v8.2`, carry that level as the packslip `variant` (`v3`, `v8.2`),
  so installers pick the default-level build unless asked for another. A
  `goarm64` option such as `v8.2,lse` becomes `v8.2-lse`, and a `goarm`
  softfloat build for ARMv6 or ARMv7 has the variant `softfloat`.
- The source repository, commit, and tag, taken from the git remote of the
  checkout, which can differ from the repository the release is published to.
  On GitHub Actions, a GitHub `project` must name the caller repository
  (or one of its monorepo subprojects), because the keyless certificate is
  bound to that repository. This also works with reusable workflows: the
  reusable workflow can live in another repository. The workflow may still
  check out a different source repository; that checkout is recorded in the
  source metadata.

The manifests are created after the artifacts are signed and notarized, so they
describe the final bytes that are uploaded. They are not created in
[snapshot]({{< ref "/customization/publish/snapshots" >}}) mode or when
publishing is skipped, since signing records each manifest in a public
transparency log.

You can also skip it with `--skip=packslip`.
