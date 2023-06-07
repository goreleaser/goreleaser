# Includes

!!! success "GoReleaser Pro"

    Includes is a [GoReleaser Pro feature](/pro/).

GoReleaser allows you to include other files from a URL or in the current
file system.

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
