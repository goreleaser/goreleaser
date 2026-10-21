---
title: "Packslip manifests"
weight: 25
---

{{< g_version "v2.19" >}}

[packslip](https://packslip.dev/) is an open format for publishing a signed
release manifest next to your release artifacts. The manifest lists each
artifact's checksums, platform, and executable paths, so installers such as
[mise](https://mise.jdx.dev) can pick the right download and verify it
without guessing from file names.

GoReleaser can create and sign that manifest for the archives and binaries it
produces, and upload it to your release as `packslip.sigstore.json`.

## Usage

The manifest is signed keylessly with [Sigstore](https://www.sigstore.dev/),
using the OIDC identity of your CI job, so there is no key to manage. It is
opt-in:

```yaml {filename=".goreleaser.yaml"}
packslip:
  # Whether to create the manifest.
  #
  # Default: false.
  # Templates: allowed.
  enabled: true

  # IDs of the archives and binaries to describe.
  #
  # Default: all of them. Without a `url_template`, only those the release
  # uploads (see `release.ids`).
  ids:
    - foo
    - bar

  # Name of the manifest file.
  #
  # Default: 'packslip.sigstore.json'.
  # Templates: allowed.
  name_template: "packslip.sigstore.json"

  # The packslip project name, in the form `host/owner/name`.
  #
  # Default: 'github.com/<release.github.owner>/<release.github.name>'.
  # Required when releasing anywhere but GitHub.
  project: github.com/foo/bar

  # Template for the URL each artifact can be downloaded from.
  #
  # Default: the release page of your SCM.
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

## What is described

- Archives and binaries (`UploadableArchive` and `UploadableBinary`
  artifacts), with the platform GoReleaser built them for. A universal macOS
  binary has no architecture.
- Executable paths inside archives, including the wrapping directory.
- Statically linked Linux builds are described as needing no shared
  libraries, so they fit both glibc and musl hosts. Linux builds that link
  dynamically (cgo) are recorded as `gnu`, or `musl` when the build target
  says so.
- Builds for a CPU level other than Go's default, such as `goamd64: v3` or
  `goarm64: v8.2`, carry that level as the packslip `variant` (`v3`, `v8.2`),
  so installers pick the default-level build unless asked for another. A
  `goarm64` option such as `v8.2,lse` becomes `v8.2-lse`, and a `goarm`
  softfloat build for ARMv6 or ARMv7 has the variant `softfloat`.
- The source repository, commit, and tag, taken from the git remote of the
  checkout, which can differ from the repository the release is published to.

The manifest is created after the artifacts are signed and notarized, so it
describes the final bytes that are uploaded. It is not created in
[snapshot]({{< ref "/customization/publish/snapshots" >}}) mode or when
publishing is skipped, since signing records the manifest in a public
transparency log.

You can also skip it with `--skip=packslip`.
