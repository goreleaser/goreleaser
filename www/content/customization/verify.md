---
title: "Verify"
weight: 55
---

{{< g_featpro >}}

{{< g_version "v2.17-unreleased" >}}

After a release is published, `verify` re-downloads the published release
assets from their public URLs into `dist/verify` and runs your verification
commands against them.

This catches the failures that happen _after_ everything "succeeded": broken or
truncated uploads, bad signatures, and CDN propagation issues.

## Usage

Verification is opt-in: add a `verify` section to your configuration to enable
it. Once enabled, it runs automatically at the end of `goreleaser release` and
`goreleaser publish` — after everything is published, right before announcing.

You can also run it on its own against a previously prepared `dist` directory:

```sh
goreleaser verify
```

Like `goreleaser continue`, this loads the previous run's state from
`dist/ctx.json` and `dist/artifacts.json`, so make sure the environment
variables used during the release are available to it as well.

To skip verification on a given run, use `--skip=verify` on `release` or
`continue`.

## Configuration

```yaml {filename=".goreleaser.yaml"}
verify:
  # Whether to disable verification entirely.
  #
  # Templates: allowed.
  disable: false

  # The verification commands.
  #
  # Before any command runs, every published release asset (artifacts,
  # checksum files, signatures, certificates) is downloaded into
  # 'dist/verify'. Commands then run sequentially, in order, with
  # 'dist/verify' as their working directory.
  commands:
    # context: dir (the default) runs the command once, in the download
    # directory.
    - context: dir
      # Path to the command.
      cmd: sha256sum

      # Command line arguments for the command.
      #
      # Templates: allowed.
      args: ["-c", "{{ .ProjectName }}_{{ .Version }}_checksums.txt"]

      # List of environment variables passed to the command, as well as to
      # the templates.
      #
      # Templates: allowed.
      env:
        - FOO=bar

      # Whether to run this command.
      #
      # Templates: allowed.
      if: '{{ isEnvSet "VERIFY_CHECKSUMS" }}'

    # context: asset runs the command once per downloaded asset, excluding
    # signatures and certificates (they are inputs to verification, not
    # targets). All files are in the working directory, so sibling files can
    # be referenced by naming convention.
    - context: asset
      cmd: cosign
      args:
        - verify-blob
        - "--signature={{ .artifact }}.sig"
        - "--certificate={{ .artifact }}.pem"
        - "{{ .artifact }}"

      # For asset and image contexts, 'if' is evaluated once per artifact,
      # with the artifact's template fields available.
      if: '{{ eq .ArtifactExt ".tar.gz" }}'

    # context: image runs the command once per published docker
    # image/manifest. Nothing is downloaded: the command (e.g. cosign verify)
    # pulls from the registry itself.
    - context: image
      cmd: cosign
      args: ["verify", "{{ .artifact }}@{{ .digest }}"]

  # Right after publishing, assets may not have propagated through the CDN yet.
  # This controls how downloads are retried.
  retry:
    # Maximum number of attempts.
    attempts: 5
    # Delay between attempts.
    delay: 10s
```

The `retry` block uses the same shape as the
[global retry configuration](/customization/general/retry/).

### Available variable names

`args` and `env` accept templates, and each context exposes extra fields (also
available as `${...}` environment variables):

- `dir` and `asset`: `{{ .dir }}`, the absolute path to the download directory
  (also the working directory).
- `asset`: `{{ .artifact }}`, the asset's file name, relative to the working
  directory. The artifact's regular template fields (`.ArtifactName`, `.Os`,
  `.Arch`, ...) are available as well.
- `image`: `{{ .artifact }}`, the image reference, and `{{ .digest }}`, its
  digest.

## Verifying checksums

Since all assets — including the published checksums file — are downloaded
into the working directory, verifying them is a single `dir` command:

```yaml {filename=".goreleaser.yaml"}
verify:
  commands:
    - cmd: sha256sum
      args: ["-c", "{{ .ProjectName }}_{{ .Version }}_checksums.txt"]
```

This proves that the downloaded assets match the published checksums file. To
also compare against the checksums recorded during the build, point it at the
local file instead — it sits right above the download directory:

```yaml {filename=".goreleaser.yaml"}
verify:
  commands:
    - cmd: sha256sum
      args: ["-c", "../{{ .ProjectName }}_{{ .Version }}_checksums.txt"]
```

## Verifying with cosign

If you sign your checksums and images with [cosign][] — for example, keyless
signing in CI — you can verify them like this:

```yaml {filename=".goreleaser.yaml"}
verify:
  commands:
    - context: asset
      if: '{{ eq .ArtifactName (printf "%s_%s_checksums.txt" .ProjectName .Version) }}'
      cmd: cosign
      args:
        - verify-blob
        - "--signature={{ .artifact }}.sig"
        - "--certificate={{ .artifact }}.pem"
        - "--certificate-identity-regexp=https://github.com/myorg/myrepo"
        - "--certificate-oidc-issuer=https://token.actions.githubusercontent.com"
        - "{{ .artifact }}"
    - context: image
      cmd: cosign
      args:
        - verify
        - "--certificate-identity-regexp=https://github.com/myorg/myrepo"
        - "--certificate-oidc-issuer=https://token.actions.githubusercontent.com"
        - "{{ .artifact }}@{{ .digest }}"
```

## Notes and limitations

- Verification only runs for real releases — it is skipped on `--snapshot`, as
  nothing is published.
- Downloads cover the artifacts published as SCM release assets (GitHub,
  GitLab, Gitea). Artifacts sent only to blob storage, uploads, or other
  package registries are not downloaded yet.
- If any download fails, verification fails before running any command, so
  commands never see an incomplete directory.
- If the release is disabled, there are no release assets to download, so only
  `image` commands run.

[cosign]: https://github.com/sigstore/cosign
