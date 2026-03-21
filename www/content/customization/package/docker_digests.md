---
title: "Docker Digests"
weight: 160
---

{{< version "v2.12" >}}

Creates a `digests.txt` file with the digests and image names of all images and
manifests published.
This is specially useful if you want to do something with this information, for
instance, send it to
[GitHub's attestation action](https://github.com/actions/attest).

Here's a commented out configuration:

```yaml {filename=".goreleaser.yaml"}
docker_digest:
  # Name of the file.
  #
  # Default: 'digests.txt'
  # Templates: allowed.
  name_template: "{{ProjectName}}_digest.txt"

  # Set this to true if you don't want to create the digest file.
  #
  # Templates: allowed.
  disable: "{{ .Env.NO_DIGEST }}"
```

See [this page](/customization/publish/attestations/) for information on how to use this to attest
images.

{{< templates >}}

> [!WARNING]
> **`sha256:` prefix**
>
>
> GitHub expects the digests without the `sha256:` prefix, so we trim the
> digest up until `:`.
