# DockerHub

{% include-markdown "../includes/pro.md" comments=false %}

DockerHub allows you to set an image description and a full description.
However, this is not possible via `docker push`.
This pipe allows you to configure these fields and ensures they are set when
publishing your releases.

You also have plenty of customization options:

```yaml
# goreleaser.yaml

dockerhub:
  - # Your hub.docker.com username. Must have 'editor' permissions
    #
    # Default: "{{ .Env.DOCKER_USERNAME }}".
    # Templates: allowed.
    username: "john.doe"

    # Environment variable name to get the push token from.
    # You might want to change it if you have multiple dockerhub configurations.
    #
    # Templates: allowed.
    # Default: "DOCKER_PASSWORD".
    secret_name: DOCKER_TOKEN

    # Images to apply the description and/or full description to.
    #
    # Templates: allowed.
    images:
      - goreleaser/goreleaser
      - goreleaser/goreleaser-pro

    # Disables the configuration feature in some conditions, for instance, when
    # publishing patch releases.
    # Any value different of 'true' will be considered 'false'.
    #
    # Templates: allowed.
    disable: "{{gt .Patch 0}}"

    # The short description of the image.
    #
    # Templates: allowed.
    description: A short description

    # The full description of the image.
    #
    # It can be a string directly, or you can use `from_url` or `from_file` to
    # source it from somewhere else.
    #
    # Templates: allowed.
    full_description:
      # Loads from an URL.
      from_url:
        # Templates: allowed.
        url: https://foo.bar/README.md
        headers:
          x-api-token: "${MYCOMPANY_TOKEN}"

      # Loads from a local file.
      # Overrides `from_url`.
      from_file:
        # Templates: allowed.
        path: ./README.md
```

{% include-markdown "../includes/templates.md" comments=false %}
