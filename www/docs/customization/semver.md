# SemVer

<!-- md:version v2.11-unreleased -->

This allows you to change some [SemVer][] behavior.

```yaml title=".goreleaser.yaml"
semver:
  # This allows you to use a template output as your version.
  # You can use this to trim a prefix, for example, or to hard-code some version
  # for any reason.
  # The result of this expression should be a valid semver.
  #
  # Templates: allowed.
  version_template: '{{ trimPrefix .Tag "myprefix/" }}'
```

[SemVer]: http://semver.org

<!-- md:templates -->
