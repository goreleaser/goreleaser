---
date: 2023-03-06
slug: goreleaser-v1.16
categories:
  - announcements
authors:
  - caarlos0
---

# Announcing GoReleaser v1.16 — the late February release

The February release got a little late... _better later than even later, I guess!_ 😄

<!-- more -->

![goreleaser healthcheck](https://carlosbecker.com/posts/goreleaser-v1.16/img.png)

It is packed with some juicy features and tons of bug fixes and quality-of-life
improvements.

Let's take a look:

### Highlights

- On GoReleaser Pro you can now add dividers between groups in the changelog.
  [Documentation](https://goreleaser.com/customization/changelog/).
- GoReleaser Pro also gets a new template variable: `{{ .Artifacts }}`, which
  you can iterate over to build, for instance, custom scripts. Oh, how can you
  build custom scripts? I'm glad you asked!
  [Documentation](https://goreleaser.com/customization/templates/#artifacts).
- Concluding our Pro-exclusive feature-set for this release: template files! You
  can template entire files, and they'll get added to the release!
  [Documentation](https://goreleaser.com/customization/templatefiles/).
- All GoReleaser distributions get a new subcommand: `healthcheck`.
  [Documentation](https://goreleaser.com/cmd/goreleaser_healthcheck/).
- The single `build` statement has been undocumented for many years, and has
  been deprecated.
- You can now announce to OpenCollective.
  [Documentation](https://goreleaser.com/customization/announce/opencollective/).
- Templating updates: `nfpms` get `{{ .ConventionalExtension }}`, new
  templateable fields, `base` (as in `filepath.Base`) template function, and
  more.
- When running on a new project, with no `project_name` set, and no git remotes,
  GoReleaser will now try to infer the project name from the `go.mod` file
  instead of erroring.
- GoReleaser is now built with Go 1.20.
- As always, a lot of bug fixes, dependency updates and improvements!

You can [install][] the same way you always do, and you can see the full release
notes [here][oss-rel] and [here (for Pro)][pro-rel].

[install]: https://goreleaser.com/install
[pro-rel]: https://github.com/goreleaser/goreleaser-pro/releases/tag/v1.16.0-pro
[oss-rel]: https://github.com/goreleaser/goreleaser/releases/tag/v1.16.0

### Other news

- We have a whole lot of example repositories, including Zig, GoReleaser-Cross,
  GoReleaser Pro features, and more.
  [Check it out](https://github.com/orgs/goreleaser/repositories?q=example)!
- GoReleaser now has ~11.3k stars and 327 contributors! Thanks, everyone!
- We eventually discuss new features in our Discord server. 
  [Join the conversation](https://goreleaser.com/discord)!
- nFPM had new releases as well, 
  [check it out](https://github.com/goreleaser/nfpm/releases).
