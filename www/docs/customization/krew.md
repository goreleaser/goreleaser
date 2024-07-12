# Krew Plugin Manifests

After releasing to GitHub or GitLab, GoReleaser can generate and publish a _Krew
Plugin Manifest_ into a repository that you have access to.

Check their [website](https://krew.sigs.k8s.io) for more information.

The `krews` section specifies how the plugins should be created:

```yaml
# .goreleaser.yaml
krews:
  -
    # Name of the recipe
    #
    # Default: the project name.
    # Templates: allowed.
    name: myproject

    # IDs of the archives to use.
    ids:
    - foo
    - bar

    # GOARM to specify which 32-bit arm version to use if there are multiple
    # versions from the build section. Krew plugin supports at this moment only
    # one 32-bit version.
    #
    # Default: 6.
    goarm: 6

    # GOAMD64 to specify which amd64 version to use if there are multiple
    # versions from the build section.
    #
    # Default: 'v1'.
    goamd64: v3

    # NOTE: make sure the url_template, the token and given repo (github or
    # gitlab) owner and name are from the same kind. We will probably unify this
    # in the next major version like it is done with scoop.

    # URL which is determined by the given Token (github or
    # gitlab)
    # Default:
    #   GitHub: 'https://github.com/<repo_owner>/<repo_name>/releases/download/{{ .Tag }}/{{ .ArtifactName }}'
    #   GitLab: 'https://gitlab.com/<repo_owner>/<repo_name>/-/releases/{{ .Tag }}/downloads/{{ .ArtifactName }}'
    #   Gitea: 'https://gitea.com/<repo_owner>/<repo_name>/releases/download/{{ .Tag }}/{{ .ArtifactName }}'
    # Templates: allowed.
    url_template: "http://github.mycompany.com/foo/bar/releases/{{ .Tag }}/{{ .ArtifactName }}"

    # Git author used to commit to the repository.
    commit_author:
      name: goreleaserbot
      email: bot@goreleaser.com

    # The project name and current git tag are used in the format string.
    commit_msg_template: "Krew plugin update for {{ .ProjectName }} version {{ .Tag }}"

    # Your app's homepage.
    #
    # Default: inferred from global metadata.
    homepage: "https://example.com/"

    # Your app's description.
    # The usual guideline for this is to wrap the line at 80 chars.
    #
    # Templates: allowed.
    # Default: inferred from global metadata.
    description: "Software to create fast and easy drum rolls."

    # Your app's short description.
    # The usual guideline for this is to be at most 50 chars long.
    #
    # Templates: allowed.
    # Default: inferred from global metadata.
    short_description: "Software to create fast and easy drum rolls."

    # Caveats for the user of your binary.
    # The usual guideline for this is to wrap the line at 80 chars.
    caveats: "How to use this binary"

    # Setting this will prevent goreleaser to actually try to commit the updated
    # krew plugin - instead, the plugin file will be stored on the dist directory
    # only, leaving the responsibility of publishing it to the user.
    # If set to auto, the release will not be uploaded to the Krew plugin
    # in case there is an indicator for prerelease in the tag e.g. v1.0.0-rc1
    skip_upload: true

{% include-markdown "../includes/repository.md" comments=false %}
```

{% include-markdown "../includes/templates.md" comments=false %}

## Limitations

- Only one binary per archive is allowed;
- Binary releases (when `archives.format` is set to `binary`) are not allowed;
- Only one `GOARM` build is allowed;

{% include-markdown "../includes/prs.md" comments=false %}
