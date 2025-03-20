# Signing Binaries

<!-- md:version v2.2 -->

This can be used to archive the binaries with their signatures, instead of
signing the whole archive.

The default is configured to create a detached signature for the checksum files
with [GnuPG](https://www.gnupg.org/), and your default key.

To enable binary signing just add this to your configuration:

```yaml title=".goreleaser.yaml"
binary_signs:
  - {}
```

To customize the binary signing pipeline you can use the following options:

```yaml title=".goreleaser.yaml"
binary_signs:
  - #
    # ID of the sign config, must be unique.
    #
    # Default: 'default'.
    id: foo

    # Name of the signature file.
    #
    # Default: '${artifact}_{{ .Os }}_{{ .Arch }}{{ with .Arm }}v{{ . }}{{ end }}{{ with .Mips }}_{{ . }}{{ end }}{{ if not (eq .Amd64 "v1") }}{{ .Amd64 }}{{ end }}'.
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
    # Default: ["--output", "${signature}", "--detach-sign", "${artifact}"].
    # Templates: allowed.
    args: ["--output", "${signature}", "${artifact}", "{{ .ProjectName }}"]

    # Which artifacts to sign
    #
    # Valid options are:
    # - none        no signing
    # - binary:     the binaries
    #
    # Default: 'binary'.
    artifacts: binary

    # IDs of the artifacts to sign.
    #
    # If `artifacts` is checksum or source, this fields has no effect.
    ids:
      - foo
      - bar

    # Allows to further filter the artifacts.
    #
    # Artifacts that do not match this expression will be ignored.
    #
    # Pro only.
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
    output: true
```

### Available variable names

These environment variables might be available in the fields that accept
templates:

- `${artifact}`: the path to the artifact that will be signed
- `${artifactID}`: the ID of the artifact that will be signed
- `${certificate}`: the certificate filename, if provided
- `${signature}`: the signature filename

### Differences from the default `signs`

The only difference is the artifact filtering and that this pipe also runs in
the build phase.

In `signs`, if you set `artifacts` to `binary`, it'll only work if you also set
`archives` `format` to `binary`.
Here, it'll work anyway.

## Signing with cosign

You can sign your artifacts with [cosign][cosign] as well.

Assuming you have a `cosign.key` in the repository root and a `COSIGN_PWD`
environment variable set, a simple usage example would look like this:

```yaml title=".goreleaser.yaml"
binary_signs:
  - cmd: cosign
    stdin: "{{ .Env.COSIGN_PWD }}"
    args:
      - "sign-blob"
      - "--key=cosign.key"
      - "--output-signature=${signature}"
      - "${artifact}"
      - "--yes" # needed on cosign 2.0.0+
```

Your users can then verify the signature with:

```sh
cosign verify-blob -key cosign.pub -signature binary.sig binary
```

[cosign]: https://github.com/sigstore/cosign
