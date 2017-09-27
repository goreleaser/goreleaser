---
title: GitHub Enterprise Support
---

You can use GitHub Enteprise within GoReleaser by providing its URLs in
the `.goreleaer.yml` configuration file:

```yaml
# .goreleaser.yml
github_urls:
    api: api.github.foo.bar
    upload: uploads.github.foo.bar
    download: github.foo.bar
```

If none is set, it will default to the public GitHub's URLs.
