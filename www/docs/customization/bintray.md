---
title: Bintray
---

## How it works

Uploading to Bintray is a simple case of [using HTTP Upload](https://goreleaser.com/customization/upload/).

### Pre and post requisites:
* Create a user and/or an org in Bintray
* Create a generic repository in Bintray
* Create a package with a name matching your `ProjectName`
* After publishing, don't forget to publish the uploaded files (either via UI or [REST API](https://bintray.com/docs/api/#_publish_discard_uploaded_content))

```yaml
uploads:
  - name: bintray
    target: https://api.bintray.com/content/user.or.org.name/generic.repo.name/{{ .ProjectName }}/{{ .Version }}/
    username: goreleaser
```

Please see [HTTP Upload](https://goreleaser.com/customization/upload/) for more details.
