---
title: GitHub Enterprise
---

You can use GoReleaser with GitHub Enterprise by providing its URLs in
the `.goreleaser.yml` configuration file:

```yaml
# .goreleaser.yml
github_urls:
    api: api.github.foo.bar
    upload: uploads.github.foo.bar
    download: github.foo.bar
```

If none are set, they default to GitHub's public URLs.
