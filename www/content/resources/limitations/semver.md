---
title: "Semantic Versioning"
weight: 20
---

GoReleaser enforces semantic versioning and will error on non-compliant tags.

Your tag **should** be a valid [semantic version](http://semver.org/).
If it is not, GoReleaser will error.

The `v` prefix is not mandatory. You can check the
[templating](/customization/general/templates/) documentation to see how to use the
tag or each part of the semantic version in name templates.

## Monorepo support

A common practice for monorepos is to have tags prefixed with their component,
e.g. `foo/v1.2.3` and `bar/v2.3.4`.

This works only on [GoReleaser Pro](/pro/).
You can read more about it [here](/customization/monorepo/).
