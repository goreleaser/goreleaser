---
title: Semantic Versioning
weight: 11
menu: true
---

GoReleaser enforces semantic versioning and will error on non compliant tags.

Your tag **should** be a valid [semantic version](http://semver.org/).
If it is not, GoReleaser will error.

The `v` prefix is not mandatory. You can check the [templating](/templates)
documentation to see how to use the tag or each part of the semantic version
in name templates.
