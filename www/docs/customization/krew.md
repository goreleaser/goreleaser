# Krew Plugin Manifests

After releasing to GitHub or GitLab, GoReleaser can generate and publish a _Krew Plugin Manifest_ into a repository that you have access to.

Check their [website](https://krew.sigs.k8s.io) for more information.

The `krews` section specifies how the plugins should be created:

```yaml
# .goreleaser.yaml
krews:
  -
    # Name template of the recipe
    # Default to project name
    name: myproject

    # IDs of the archives to use.
    # Defaults to all.
    ids:
    - foo
    - bar

    # GOARM to specify which 32-bit arm version to use if there are multiple versions
    # from the build section. Krew plugin supports at this moment only one 32-bit version.
    # Default is 6 for all artifacts or each id if there a multiple versions.
    goarm: 6

    # NOTE: make sure the url_template, the token and given repo (github or gitlab) owner and name are from the
    # same kind. We will probably unify this in the next major version like it is done with scoop.

    # GitHub/GitLab repository to push the Krew plugin to
    # Gitea is not supported yet, but the support coming
    index:
      owner: repo-owner
      name: krew-plugins
      # Optionally a branch can be provided. If the branch does not exist, it
      # will be created. If no branch is listed, the default branch will be used
      branch: main
      # Optionally a token can be provided, if it differs from the token provided to GoReleaser
      token: "{{ .Env.KREW_GITHUB_TOKEN }}"

    # Template for the url which is determined by the given Token (github or gitlab)
    # Default for github is "https://github.com/<repo_owner>/<repo_name>/releases/download/{{ .Tag }}/{{ .ArtifactName }}"
    # Default for gitlab is "https://gitlab.com/<repo_owner>/<repo_name>/-/releases/{{ .Tag }}/downloads/{{ .ArtifactName }}"
    # Default for gitea is "https://gitea.com/<repo_owner>/<repo_name>/releases/download/{{ .Tag }}/{{ .ArtifactName }}"
    url_template: "http://github.mycompany.com/foo/bar/releases/{{ .Tag }}/{{ .ArtifactName }}"

    # Git author used to commit to the repository.
    # Defaults are shown.
    commit_author:
      name: goreleaserbot
      email: goreleaser@carlosbecker.com

    # The project name and current git tag are used in the format string.
    commit_msg_template: "Krew plugin update for {{ .ProjectName }} version {{ .Tag }}"

    # Your app's homepage.
    # Default is empty.
    homepage: "https://example.com/"

    # Template of your app's description.
    # The usual guideline for this is to wrap the line at 80 chars.
    #
    # Default is empty.
    description: "Software to create fast and easy drum rolls."

    # Template of your app's short description.
    # The usual guideline for this is to be at most 50 chars long.
    #
    # Default is empty.
    short_description: "Software to create fast and easy drum rolls."

    # Caveats for the user of your binary.
    # The usual guideline for this is to wrap the line at 80 chars.
    #
    # Default is empty.
    caveats: "How to use this binary"

    # Setting this will prevent goreleaser to actually try to commit the updated
    # krew plugin - instead, the plugin file will be stored on the dist folder only,
    # leaving the responsibility of publishing it to the user.
    # If set to auto, the release will not be uploaded to the Krew plugin
    # in case there is an indicator for prerelease in the tag e.g. v1.0.0-rc1
    # Default is false.
    skip_upload: true
```

!!! tip
    Learn more about the [name template engine](/customization/templates/).

## Limitations

- Only one binary per archive is allowed;
- Binary releases (when `archives.format` is set to `binary`) are not allowed;
- Only one `GOARM` build is allowed;
