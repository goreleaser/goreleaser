# DockerHub

> Since v1.25 (pro)

!!! success "GoReleaser Pro"

    DockerHub is a [GoReleaser Pro feature](/pro/).

DockerHub allows you to set an image description, and a full description.
However, this is not possible via `docker push`.
This pipe allows you to configure these fields and ensures they are set when
publishing your releases.

You can also have plenty of customization options:

```yaml
# goreleaser.yaml

dockerhubs:
  - # Your docker.io username.
    #
    # Templates: allowed
    usernme: "{{ .Env.DOCKER_USERNAME }}"

    # Disables the configuration feature in some conditions, for instance, when
    # publishing patch releases.
    # Any value different of 'true' will be considered 'false'.
    #
    # Templates: allowed
    disable: "{{gt .Patch 0}}"

    # Environment variable name to get the push token from.
    # You might want to change it if you have multiple dockerhub configurations.
    #
    # Templates: allowed
    # Default: 'DOCKER_PASSWORD'
    secret_name: MY_ACCOUNT_DOCKER_PASSWORD

    # The short description of the image.
    #
    # Templates: allowed
    description: A short description

    # The full description of the image.
    #
    # It can be a string directly, or you can use `from_url` or `from_file` to
    # source it from somewhere else.
    #
    # Templates: allowed
    full_description:
      # Loads from an URL.
      from_url:
        # Templates: allowed
        url: https://foo.bar/README.md
        headers:
          x-api-token: "${MYCOMPANY_TOKEN}"

      # Loads from a local file.
      # Overrides `from_url`.
      from_file:
        # Templates: allowed
        path: ./README.md
```

!!! tip

    Learn more about the [name template engine](/customization/templates/).
