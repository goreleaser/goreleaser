# Model Context Protocol (MCP) Server

<!-- md:version v2.13-unreleased -->

<!-- md:experimental https://github.com/orgs/goreleaser/discussions/6251 -->

After building your binaries, GoReleaser can generate and publish a Model
Context Protocol (MCP) server manifest file (`server.json`).
This manifest makes your tool discoverable and installable by MCP clients,
enabling integration with AI assistants and development tools.

The `mcp` section specifies how the MCP server manifest should be created.
You can check the
[Model Context Protocol specification](https://modelcontextprotocol.io/) for
more details.

```yaml title=".goreleaser.yaml"
mcp:
  # Unique server name in reverse-DNS format.
  # Must contain exactly one forward slash separating namespace from server name.
  #
  # Required.
  # Templates: allowed.
  name: io.github.user/myserver

  # Human-readable title or display name for the MCP server.
  # MCP clients may use this for display purposes.
  #
  # Required.
  # Templates: allowed.
  title: "My MCP Server"

  # Clear human-readable explanation of server functionality.
  # Should focus on capabilities, not implementation details.
  #
  # Templates: allowed.
  description: "MCP server providing weather data and forecasts"

  # URL to the server's homepage, documentation, or project website.
  # Provides a central link for users to learn more about the server.
  #
  # Templates: allowed.
  homepage: "https://example.com/myserver"

  # Authentication configuration for the server.
  auth:
    # Authentication type.
    #
    # Valid values: none, github, github-oidc.
    # Default: none.
    type: github-oidc

    # Authentication token (for github type only).
    #
    # Templates: allowed.
    # Default: '$MCP_GITHUB_TOKEN'.
    token: "{{ .Env.GITHUB_TOKEN }}"

  # Repository metadata for the MCP server source code.
  # Enables users and security experts to inspect the code.
  # Recommended for transparency and security inspection.
  repository:
    # Repository URL for browsing source code.
    # Should support both web browsing and git clone operations.
    #
    # Required if repository is specified.
    # Templates: allowed.
    url: "https://github.com/user/myserver"

    # Repository hosting service identifier.
    # Used by registries to determine validation and API access methods.
    #
    # Required if repository is specified.
    # Examples: github, gitlab, gitea.
    source: github

    # Repository identifier from the hosting service.
    # Should remain stable across repository renames.
    # For GitHub, use: gh api repos/<owner>/<repo> --jq '.id'
    #
    # Templates: allowed.
    id: "123456789"

    # Optional relative path from repository root to the server location
    # within a monorepo or nested package structure.
    #
    # Templates: allowed.
    subfolder: "src/server"

  # Package configurations for different distribution methods.
  packages:
    # Registry type indicating how to download packages.
    # Valid values: oci, npm, pypi, nuget, mcpb.
    - registry_type: npm

      # Package identifier - either a package name (for registries) or URL (for direct downloads).
      #
      # Required.
      # Templates: allowed.
      identifier: "@modelcontextprotocol/server-example"

      # Transport protocol configuration for the package.
      transport:
        # Transport type.
        # Valid values: stdio, streamable-http, sse.
        #
        # Required.
        type: stdio

    # OCI (Docker) registry example
    - registry_type: oci
      identifier: "ghcr.io/user/myserver:{{ .Version }}"
      transport:
        type: stdio

    # NPM registry example
    - registry_type: npm
      identifier: "@myorg/myserver"
      transport:
        type: stdio

  # Setting this will prevent GoReleaser from actually trying to publish the
  # server manifest - instead, the manifest file will be stored in the dist
  # directory only, leaving the responsibility of publishing it to the user.
  # If set to auto, the manifest will not be published in case there is an
  # indicator for prerelease in the tag e.g. v1.0.0-rc1
  #
  # Templates: allowed.
  disable: false
```

<!-- md:templates -->

## How it works

GoReleaser will generate a `server.json` conforming to the
[MCP Schema](https://static.modelcontextprotocol.io/schemas/2025-10-17/server.schema.json).

Then, it'll login using the provided authentication details.

Finally, it'll publish the server.json to the registry.

Follow
[this guide](https://github.com/modelcontextprotocol/registry/blob/main/docs/guides/publishing/publish-server.md)
for more details.

!!! tip

    You don't need to install `mcp-publisher` nor run any commands.
    GoReleaser takes care of it all.

## Tips

If you are using the `oci` transport type, make sure to add the required
label to the image as well:

```yaml
# Docker v2 (new):
dockers_v2:
  - images:
      - ghcr.io/etc/etc
    labels:
      io.modelcontextprotocol.server.name: "io.github.username/server-name"

# Docker (old):
dockers:
  - image_templates:
      - ghcr.io/etc/etc:{{ .Version }}
    build_flag_templates:
      - '--label=io.modelcontextprotocol.server.name="io.github.username/server-name"'
```

If you're using NPM, you can use the `extra` field to set the `mcpName`:

```yaml
npms:
  - name: "@foo/bar"
    extra:
      mcpName: io.github.foo/bar
```

If you don't set these fields, publishing the MCP will fail.

Read
[this page](https://github.com/modelcontextprotocol/registry/blob/main/docs/guides/publishing/publish-server.md)
for more information.
