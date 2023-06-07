# Chocolatey Packages

> Since: v1.13

GoReleaser can also generate `nupkg` packages.
[Chocolatey](http://chocolatey.org/) are packages based on `nupkg` format, that
will let you publish your project directly to the Chocolatey Repository. From
there it will be able to install locally or in Windows distributions.

You can read more about it in the [chocolatey docs](https://docs.chocolatey.org/).

Available options:

```yaml
# .goreleaser.yaml
chocolateys:
  - # Your app's package name.
    # The value may not contain spaces or character that are not valid for a URL.
    # If you want a good separator for words, use '-', not  '.'.
    #
    # Default: ProjectName
    name: foo

    # IDs of the archives to use.
    # Empty means all IDs.
    ids:
      - foo
      - bar

    # Your app's owner.
    # It basically means your.
    owners: Drum Roll Inc

    # The app's title.
    # A human-friendly title of the package.
    #
    # Default: ProjectName
    title: Foo Bar

    # Your app's authors (probably you).
    authors: Drummer

    # Your app's project url.
    # It is a required field.
    project_url: https://example.com/

    # URL which is determined by the given Token (github,
    # gitlab or gitea).
    #
    # Default: depends on the git remote
    # Templates: allowed
    url_template: "https://github.com/foo/bar/releases/download/{{ .Tag }}/{{ .ArtifactName }}"

    # App's icon.
    icon_url: "https://rawcdn.githack.com/foo/bar/efbdc760-395b-43f1-bf69-ba25c374d473/icon.png"

    # Your app's copyright details.
    copyright: 2022 Drummer Roll Inc

    # App's license information url.
    license_url: https://github.com/foo/bar/blob/main/LICENSE

    # Your apps's require license acceptance:
    # Specify whether the client must prompt the consumer to accept the package
    # license before installing.
    require_license_acceptance: false

    # Your app's source url.
    project_source_url: https://github.com/foo/bar

    # Your app's documentation url.
    docs_url: https://github.com/foo/bar/blob/main/README.md

    # App's bugtracker url.
    bug_tracker_url: https://github.com/foo/barr/issues

    # Your app's tag list.
    tags: "foo bar baz"

    # Your app's summary:
    summary: Software to create fast and easy drum rolls.

    # This the description of your chocolatey package.
    # Supports markdown.
    description: |
      {{ .ProjectName }} installer package.
      Software to create fast and easy drum rolls.

    # Your app's release notes.
    # A description of the changes made in this release of the package.
    # Supports markdown. To prevent the need to continually update this field,
    # providing a URL to an external list of Release Notes is perfectly
    # acceptable.
    release_notes: "https://github.com/foo/bar/releases/tag/v{{ .Version }}"

    # App's dependencies
    # The version is not required.
    dependencies:
      - id: nfpm
        version: 2.20.0

    # The api key that should be used to push to the chocolatey repository.
    #
    # WARNING: do not expose your api key in the configuration file!
    api_key: "{{ .Env.CHOCOLATEY_API_KEY }}"

    # The source repository that will push the package to.
    source_repo: "https://push.chocolatey.org/"

    # Setting this will prevent goreleaser to actually try to push the package
    # to chocolatey repository, leaving the responsibility of publishing it to
    # the user.
    skip_publish: false

    # GOAMD64 to specify which amd64 version to use if there are multiple
    # versions from the build section.
    #
    # Default: 'v1'
    goamd64: v1
```

!!! tip

    Learn more about the [name template engine](/customization/templates/).

!!! note

    GoReleaser will not install `chocolatey`/`choco` nor any of its dependencies
    for you.
