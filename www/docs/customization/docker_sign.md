# Signing Docker Images and Manifests

Signing Docker Images and Manifests is also possible with GoReleaser.
This pipe was designed based on the common [sign](sign.md) pipe
having [cosign](https://github.com/sigstore/cosign) in mind.

!!! info

    Note that this pipe will run only at the end of the GoReleaser execution (in
    its publishing phase), as cosign will change the image in the registry.

To customize the signing pipeline you can use the following options:

```yaml
# .goreleaser.yml
docker_signs:
  - # ID of the sign config, must be unique.
    # Only relevant if you want to produce some sort of signature file.
    #
    # Default: 'default'.
    id: foo

    # Path to the signature command.
    #
    # Default: 'cosign'.
    cmd: cosign

    # Command line arguments for the command.
    #
    # Default: ["sign", "--key=cosign.key", "${artifact}", "--yes"].
    # Templates: allowed.
    args:
      - "sign"
      - "--key=cosign.key"
      - "--upload=false"
      - "${artifact}"
      - "--yes" # needed on cosign 2.0.0+

    # Which artifacts to sign.
    #
    #   all:       all artifacts
    #   none:      no signing
    #   images:    only docker images
    #   manifests: only docker manifests
    #
    # Default: 'none'.
    artifacts: all

    # IDs of the artifacts to sign.
    ids:
      - foo
      - bar

    # Stdin data to be given to the signature command as stdin.
    #
    # Templates: allowed.
    stdin: "{{ .Env.COSIGN_PWD }}"

    # StdinFile file to be given to the signature command as stdin.
    stdin_file: ./.password

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

These environment variables might be available in the fields that are templateable:

- `${artifact}`[^1]: the path to the artifact that will be signed (including the
  digest[^2])
- `${digest}`[^2]: the digest of the image/manifest that will be signed
- `${artifactID}`: the ID of the artifact that will be signed
- `${certificate}`: the certificate file name, if provided

[^1]:
    notice that this might contain `/` characters, which depending on how
    you use it might evaluate to actual paths within the file system. Use with
    care.

[^2]:
    those are extracted automatically when running Docker push from within
    GoReleaser. Using the digest helps making sure you're signing the right image
    and avoid concurrency issues.

## Common usage example

Assuming you have a `cosign.key` in the repository root and a `COSIGN_PWD`
environment variable, the simplest configuration to sign both Docker images
and manifests would look like this:

```yaml
# .goreleaser.yml
docker_signs:
  - artifacts: all
    stdin: "{{ .Env.COSIGN_PWD }}"
```

Later on you (and anyone else) can verify the image with:

```bash
cosign verify --key cosign.pub your/image
```
