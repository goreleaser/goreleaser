# Signing Docker Images and Manifests

Signing Docker Images and Manifests is also possible with GoReleaser.
This pipe was designed based on the common [sign](/customization/sign/) pipe having [cosign](https://github.com/sigstore/cosign) in mind.

!!! info
    Note that this pipe will run only at the end of the GoReleaser execution (in its publish phase), as cosign will change the image in the registry.


To customize the signing pipeline you can use the following options:

```yaml
# .goreleaser.yml
docker_signs:
  -
    # ID of the sign config, must be unique.
    # Only relevant if you want to produce some sort of signature file.
    #
    # Defaults to "default".
    id: foo

    # Name/template of the signature file.
    # Note that with cosign you don't need to use this.
    #
    # Defaults to empty.
    signature: "${artifact}_sig"

    # Path to the signature command
    #
    # Defaults to `cosign`
    cmd: cosign

    # Command line templateable arguments for the command
    #
    # defaults to `["sign", "-key=cosign.key", "${artifact}"]`
    args: ["sign", "-key=cosign.key", "-upload=false", "${artifact}"]


    # Which artifacts to sign
    #
    #   all:       all artifacts
    #   none:      no signing
    #   images:    only docker images
    #   manifests: only docker manifests
    #
    # defaults to `none`
    artifacts: all

    # IDs of the artifacts to sign.
    #
    # Defaults to empty (which implies no ID filtering).
    ids:
      - foo
      - bar

    # Stdin data template to be given to the signature command as stdin.
    # Defaults to empty
    stdin: '{{ .Env.COSIGN_PWD }}'

    # StdinFile file to be given to the signature command as stdin.
    # Defaults to empty
    stdin_file: ./.password

    # Sets a certificate name that your signing command should write to.
    # You can later use `${certificate}` or `.Env.certificate` in the `args` section.
    # This is particularly useful for keyless signing (for instance, with cosign).
    # Note that this should be a name, not a path.
    #
    # Defaults to empty.
    certificate: '{{ trimsuffix .Env.artifactName ".tar.gz" }}.pem'

    # List of environment variables that will be passed to the signing command as well as the templates.
    #
    # Defaults to empty
    env:
    - FOO=bar
    - HONK=honkhonk
```

### Available variable names

These environment variables might be available in the fields that are templateable:

- `${artifact}`: the path to the artifact that will be signed [^1]
- `${artifactID}`: the ID of the artifact that will be signed
- `${certificate}`: the certificate filename, if provided
- `${artifactName}`: the name of the artifact [^1]

[^1]: notice that the this might contain `/` characters, which depending on how you use it migth evaluate to actual paths within the filesystem. Use with care.


## Common usage example

Assuming you have a `cosign.key` in the repository root and a `COSIGN_PWD`
environment variable, the simplest configuration to sign both Docker images
and manifests would look like this:

```yaml
# .goreleaser.yml
docker_signs:
- artifacts: all
  stdin: '{{ .Env.COSIGN_PWD }}'
```

Later on you (and anyone else) can verify the image with:

```sh
cosign verify -key cosign.pub your/image
```
