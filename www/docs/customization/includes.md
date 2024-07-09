# Includes

{% include-markdown "../includes/pro.md" comments=false %}

GoReleaser allows you to reuse configuration files by including them from either
a URL or a file path.

Files are included recursively in the order they are declared.

```yaml
# .goreleaser.yaml
includes:
  - from_file:
      path: ./config/goreleaser.yaml
  - from_url:
      url: https://raw.githubusercontent.com/goreleaser/goreleaser/main/.goreleaser.yaml
  - from_url:
      url: caarlos0/goreleaserfiles/main/packages.yml # the https://raw.githubusercontent.com/ prefix may be omitted
  - from_url:
      url: https://api.mycompany.com/configs/goreleaser.yaml
      headers:
        # header values are expanded in case they are environment variables
        x-api-token: "${MYCOMPANY_TOKEN}"
```

With this and the power of templates, you might be able to reuse the same
`.goreleaser.yaml` configuration file in many projects, or create one file for
each "purpose" and compose them in the final project's `.goreleaser.yaml`.
