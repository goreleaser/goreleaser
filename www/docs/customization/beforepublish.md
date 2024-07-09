# Before Publish Hooks

> Since v2.1 (Pro).

{% include-markdown "../includes/pro.md" comments=false %}

You can use the `before_publish` hooks to run command against artifacts before
the publishing step kicks in.

This should allow you to run it through a scanner, for example, or do pretty
much anything else you need.

It'll run as the last step before the `publish` phase kicks in when running
`goreleaser release`, and after the `build` phase when running
`goreleaser build`.

Here's the list of options available:

```yaml
# .goreleaser.yaml
before_publish:
  - # IDs of the artifacts to filter for.
    #
    # If `artifacts` is checksum or source, this fields has no effect.
    #
    # Default: no filter.
    ids:
      - foo
      - bar

    # Which artifacts to filter for.
    #
    # Valid options are:
    # - checksum:   checksum files
    # - source:     source archive
    # - package:    Linux packages (deb, rpm, apk, etc)
    # - installer:  Windows MSI installers
    # - diskimage:  macOS DMG disk images
    # - archive:    archives from archive pipe
    # - binary:     binaries output from the build stage
    # - sbom:       any SBOMs generated for other artifacts
    # - image:      any Docker Images
    #
    # Default: no filter.
    artifacts: all

    # The command to run.
    #
    # Templates: allowed.
    cmd: "./scan {{.ArtifactPath}}"

    # Always prints command output.
    #
    # Default: false.
    output: true

    # Base directory to run the commands from.
    #
    # Default: current working directory..
    dir: ./submodule # specify command working directory

    # Additional environment variables to set.
    env:
      - "FILE_TO_TOUCH=something-{{ .ProjectName }}" # specify hook level environment variables
```

{% include-markdown "../includes/templates.md" comments=false %}
