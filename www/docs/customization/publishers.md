---
title: Custom Publishers
---

GoReleaser supports publishing artifacts by executing a custom publisher.

## How it works

You can declare multiple `publishers` instances. Each publisher will be
executed for each (filtered) artifact. For example, there will be a total of
6 executions for 2 publishers with 3 artifacts.

Publishers run sequentially in the order they're defined
and executions are parallelised between all artifacts.
In other words the publisher is expected to be safe to run
in multiple instances in parallel.

If you have only one `publishers` instance, the configuration is as easy as adding
the command to your `.goreleaser.yml` file:

```yaml
publishers:
  - name: my-publisher
    cmd: custom-publisher -version={{ .Version }} {{ abs .ArtifactPath }}
```

### Environment

Commands which are executed as custom publishers do not inherit any environment variables
(unlike existing hooks) as a precaution to avoid leaking sensitive data accidentally
and provide better control of the environment for each individual process
where variable names may overlap unintentionally.

You can however use `.Env.NAME` templating syntax which enables
more explicit inheritance.

```yaml
- cmd: custom-publisher
  env:
    - SECRET_TOKEN={{ .Env.SECRET_TOKEN }}
```

### Variables

Command (`cmd`), workdir (`dir`) and environment variables (`env`) support templating

```yaml
publishers:
  - name: production
    cmd: |
      custom-publisher \
      -product={{ .ProjectName }} \
      -version={{ .Version }} \
      {{ .ArtifactName }}
    dir: "{{ dir .ArtifactPath }}"
    env:
      - TOKEN={{ .Env.CUSTOM_PUBLISHER_TOKEN }}
```

so the above example will execute `custom-publisher -product=goreleaser -version=1.0.0 goreleaser_1.0.0_linux_amd64.zip` in `/path/to/dist` with `TOKEN=token`, assuming that GoReleaser is executed with `CUSTOM_PUBLISHER_TOKEN=token`.

Supported variables:

- `Version`
- `Tag`
- `ProjectName`
- `ArtifactName`
- `ArtifactPath`
- `Os`
- `Arch`
- `Arm`

## Customization

Of course, you can customize a lot of things:

```yaml
# .goreleaser.yml
publishers:
  -
    # Unique name of your publisher. Used for identification
    name: "custom"

    # IDs of the artifacts you want to publish
    ids:
     - foo
     - bar

    # Publish checksums (defaults to false)
    checksum: true

    # Publish signatures (defaults to false)
    signature: true

    # Working directory in which to execute the command
    dir: "/utils"

    # Command to be executed
    cmd: custom-publisher -product={{ .ProjectName }} -version={{ .Version }} {{ .ArtifactPath }}

    # Environment variables
    env:
      - API_TOKEN=secret-token
```

These settings should allow you to push your artifacts to any number of endpoints
which may require non-trivial authentication or has otherwise complex requirements.

!!! tip
    Learn more about the [name template engine](/customization/templates).
