# Includes

!!! success "GoReleaser Pro"
    Includes is a [GoReleaser Pro feature](/pro/).

GoReleaser allows you to include other files from an URL or in the current filesystem.

Files are included recursively in the order they are declared.

```yaml
# .goreleaser.yml
includes:
  - from_file:
      path: ./config/goreleaser.yml
  - from_url:
      url: https://raw.githubusercontent.com/goreleaser/goreleaser/master/.goreleaser.yml
  - from_url:
      url: caarlos0/goreleaserfiles/main/packages.yml # the https://raw.githubusercontent.com/ prefix may be ommited
  - from_url:
      url: https://api.mycompany.com/configs/goreleaser.yml
      headers:
        # header values are expanded in case they are environment variables
        x-api-token: "${MYCOMPANY_TOKEN}"
```
