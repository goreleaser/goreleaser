---
title: "Signing archives, installers, packages, and checksums"
linkTitle: "Archives, installers, packages, and checksums"
weight: 20
---

Signing lets your users verify that you generated the artifacts, by comparing
the signature against your public signing key.

GoReleaser can sign both executables and archives.

## Usage

Signing works in combination with checksum files, and it is generally enough
to sign the checksum files only.

The default is configured to create a detached signature for the checksum files
with [GnuPG](https://www.gnupg.org/), and your default key.

To enable signing just add this to your configuration:

```yaml {filename=".goreleaser.yaml"}
signs:
  - artifacts: checksum
```

To customize the signing pipeline you can use the following options:

```yaml {filename=".goreleaser.yaml"}
signs:
  - #
    # ID of the sign config, must be unique.
    #
    # Default: 'default'.
    id: foo

    # Name of the signature file.
    #
    # Default: '${artifact}.sig'.
    # Templates: allowed.
    signature: "${artifact}_sig"

    # Path to the signature command
    #
    # Default: 'gpg'.
    cmd: gpg2

    # Command line arguments for the command
    #
    # to sign with a specific key use
    # args: ["-u", "<key id, fingerprint, email, ...>", "--output", "${signature}", "--detach-sign", "${artifact}"]
    #
    # Default: ["--output", "${signature}", "--detach-sig", "${artifact}"].
    # Templates: allowed.
    args: ["--output", "${signature}", "${artifact}", "{{ .ProjectName }}"]

    # Which artifacts to sign
    #
    # Valid options are:
    # - none        no signing
    # - all:        all artifacts
    # - checksum:   checksum files
    # - source:     source archive
    # - package:    Linux packages (deb, rpm, apk, etc)
    # - installer:  Installers (MSI, NSIS, macOS Pkgs) {{< g_inline_pro >}}
    # - diskimage:  macOS DMG disk images {{< g_inline_pro >}}
    # - archive:    archives from archive pipe
    # - sbom:       any SBOMs generated for other artifacts
    # - binary:     binaries (only when `archives.format` is 'binary', use binary_signs otherwise)
    #
    # Default: 'none'.
    artifacts: all

    # IDs of the artifacts to sign.
    #
    # If `artifacts` is checksum or source, this field has no effect.
    ids:
      - foo
      - bar

    # Allows to further filter the artifacts.
    #
    # Artifacts that do not match this expression will be ignored.
    #
    # {{< g_inline_pro >}}
    # {{< g_inline_version "v2.2" >}}
    # Templates: allowed.
    if: '{{ eq .Os "linux" }}'

    # Stdin data to be given to the signature command as stdin.
    #
    # Templates: allowed.
    stdin: "{{ .Env.GPG_PASSWORD }}"

    # StdinFile file to be given to the signature command as stdin.
    stdin_file: ./.password

    # Sets a certificate that your signing command should write to.
    #
    # You can later use `${certificate}` or `.Env.certificate` in the `args` section.
    #
    # This is particularly useful for keyless signing with cosign, and should
    # not usually be used otherwise.
    #
    # Note that this should be a name, not a path.
    #
    # Templates: allowed.
    certificate: '{{ trimsuffix .Env.artifact ".tar.gz" }}.pem'

    # List of environment variables that will be passed to the signing command
    # as well as the templates.
    env:
      - FOO=bar
      - HONK=honkhonk

    # By default, the stdout and stderr of the signing cmd are discarded unless
    # GoReleaser is running with `--verbose` set.
    # You can set this to true if you want them to be displayed regardless.
    #
    # Templates: allowed. {{< g_inline_version "v2.13" >}}
    output: true
```

### Available variable names

These environment variables might be available in the fields that accept
templates:

- `${artifact}`: the path to the artifact that will be signed
- `${artifactID}`: the ID of the artifact that will be signed
- `${certificate}`: the certificate filename, if provided
- `${signature}`: the signature filename

## Signing with cosign

You can sign your artifacts with [cosign][] as well.

Cosign uses the `--bundle` flag, which combines the certificate and
signature into a single `.sigstore.json` file. This example signs the checksum
file, not each archive:

```yaml {filename=".goreleaser.yaml"}
checksum:
  name_template: checksums.txt

signs:
  - cmd: cosign
    signature: "${artifact}.sigstore.json"
    args:
      - "sign-blob"
      - "--bundle=${signature}"
      - "${artifact}"
      - "--yes"
    artifacts: checksum
```

Download `checksums.txt`, `checksums.txt.sigstore.json`, and the artifacts you
want to verify into the same directory.

First, verify the checksum file's signature against the publisher's expected
identity and OIDC issuer. For a GitHub Actions release, replace `<owner>`,
`<repository>`, the workflow path, and the tag with the publisher's actual
values:

```sh
cosign verify-blob \
  --certificate-identity 'https://github.com/<owner>/<repository>/.github/workflows/release.yml@refs/tags/v1.2.3' \
  --certificate-oidc-issuer 'https://token.actions.githubusercontent.com' \
  --bundle checksums.txt.sigstore.json checksums.txt
```

Only after that succeeds, verify the downloaded artifacts with GNU `sha256sum`:

```sh
sha256sum --check --ignore-missing checksums.txt
```

Use the publisher's checksum filename if it differs from this example.

## Signing and notarizing macOS executables

For signing and notarizing macOS executables, please refer to
[Notarize macOS applications](/customization/sign/notarize/).

## Signing Docker images and manifests

Please refer to [Docker Images Signing](/customization/sign/docker_sign/).

## Limitations

You can sign with any command that either writes a file or modifies the file
being signed.

If you set `signature` — or `certificate` — that file should exist once the
command finishes. GoReleaser warns with `the signer did not write <path>` when
it does not, because the artifact it records then points to nothing, and that
only fails much later, while uploading it.

> [!WARNING]
> This is a warning in v2, and will be an error in v3.

If you want to sign with something that writes to `STDOUT` instead of a file,
you can wrap the command inside a `sh -c` execution, for instance:

```yaml {filename=".goreleaser.yaml"}
signs:
  - cmd: sh
    args:
      - "-c"
      - 'echo "${artifact} is signed and I can prove it" | tee ${signature}'
    artifacts: all
```

And it will work just fine. Just make sure to always use the `${signature}`
template variable as the result file name and `${artifact}` as the origin file.

[cosign]: https://github.com/sigstore/cosign
