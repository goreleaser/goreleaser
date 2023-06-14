# Winget

> Since: v1.19

After releasing to GitHub, GitLab, or Gitea, GoReleaser can generate and publish
a _winget manifest_ and commit to a git repository, and PR it to `winget-pkgs`
if instructed to.

The `winget` section specifies how the **manifests** should be created:

```yaml
# .goreleaser.yaml
winget:
  - # Name of the recipe
    #
    # Default: ProjectName
    # Templates: allowed
    name: myproject

    # Publisher name.
    #
    # Templates: allowed
    # Required.
    publisher: Foo Inc

    # Your app's description.
    #
    # Templates: allowed
    # Required.
    short_description: "Software to create fast and easy drum rolls."

    # License name.
    # Required.
    license: "mit"

    # Publisher URL.
    #
    # Templates: allowed
    publisher_url: https://goreleaser.com

    # Package identifier.
    #
    # Default: Publisher.ProjectName
    # Templates: allowed
    package_identifier: myproject.myproject

    # IDs of the archives to use.
    # Empty means all IDs.
    ids:
      - foo
      - bar

    # GOAMD64 to specify which amd64 version to use if there are multiple
    # versions from the build section.
    #
    # Default: v1
    goamd64: v1

    # URL which is determined by the given Token (github, gitlab or gitea).
    #
    # Default depends on the client.
    # Templates: allowed
    url_template: "https://github.mycompany.com/foo/bar/releases/download/{{ .Tag }}/{{ .ArtifactName }}"

    # Git author used to commit to the repository.
    commit_author:
      name: goreleaserbot
      email: bot@goreleaser.com

    # The project name and current git tag are used in the format string.
    #
    # Templates: allowed
    commit_msg_template: "{{ .ProjectName }}: {{ .Tag }}"

    # Path for the file inside the repository.
    #
    # Default: manifests/<lowercased first char of publisher>/<publisher>/<version>
    path: manifests/g/goreleaser/1.19

    # Your app's homepage.
    homepage: "https://example.com/"

    # Your app's long description.
    #
    # Templates: allowed
    description: "Software to create fast and easy drum rolls."

    # License URL.
    license_url: "https://goreleaser.com/license"

    # Copyright.
    copyright: "Becker Software LTDA"

    # Setting this will prevent goreleaser to actually try to commit the updated
    # package - instead, it will be stored on the dist folder only,
    # leaving the responsibility of publishing it to the user.
    #
    # If set to auto, the release will not be uploaded to the repository
    # in case there is an indicator for prerelease in the tag e.g. v1.0.0-rc1
    #
    # Templates: allowed
    skip_upload: true

{% include-markdown "../includes/repository.md" comments=false %}
```

!!! tip

    Learn more about the [name template engine](/customization/templates/).
