---
title: Bintray
series: customization
hideFromIndex: true
weight: 120
---

## How it works

Uploading to Bintray is a simple case of [using HTTP PUT](https://goreleaser.com/customization/#HTTP%20Put).

### Pre and post requisites:
* Create a user and/or an org in Bintray
* Create a generic repository in Bintray
* Create a package with a name matching your `ProjectName`
* After publishing, dont' forget to publish the uploaded files (either via UI or [REST API](https://bintray.com/docs/api/#_publish_discard_uploaded_content))

```yaml
puts:
  - name: bintray
    target: https://api.bintray.com/content/user.or.org.name/generic.repo.name/{{ .ProjectName }}/{{ .Version }}/
    username: goreleaser
```

Please see [HTTP Put](https://goreleaser.com/customization/#HTTP%20Put) for more details.
