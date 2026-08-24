---
title: "Release Summary"
weight: 37
---

{{< g_version "v2.18" >}}

When running inside [GitHub Actions](/customization/ci/actions/), GoReleaser
writes a summary of what your release published to the [job summary
view][job-summary], so you can see the outcome of a run without reading its
logs.

There is nothing to configure. GoReleaser appends to the file named by the
`GITHUB_STEP_SUMMARY` environment variable, which GitHub Actions sets for every
step.

## Example

Each notable action becomes one bullet, with a link when there is one:

```text
- Published v1.2.3 to GitHub with 12 assets: https://github.com/user/repo/releases/tag/v1.2.3
- Pushed Docker image `user/foo:v1.2.3`
- Updated homebrew formula `foo.rb` in `user/homebrew-tap`
- Opened pull request to `user/nur` (nixpkg `foo.nix`): https://github.com/user/nur/pull/12
- Pushed `foo` to npm
- Uploaded 8 files to `s3://my-bucket`
```

## What gets reported

GoReleaser reports an action only after it succeeds, so the summary describes
what actually happened, not what was configured:

- Releases published to [GitHub, GitLab, and Gitea](/customization/publish/scm/),
  and [milestones](/customization/publish/milestone/) closed.
- [Homebrew formulas](/customization/publish/homebrew_formulas/) and
  [casks](/customization/publish/homebrew_casks/),
  [Scoop](/customization/publish/scoop/) manifests,
  [winget](/customization/publish/winget/) packages,
  [krew](/customization/publish/krew/) plugins, and
  [Nix](/customization/publish/nix/) packages, whether GoReleaser pushed them
  directly or opened a pull request.
- [AUR](/customization/publish/aur/) and [AUR
  source](/customization/publish/aursources/) packages pushed to their
  repositories.
- Container images pushed by [Docker](/customization/package/docker/),
  [Docker v2](/customization/package/dockers_v2/), and
  [ko](/customization/package/ko/), the
  [manifests](/customization/package/docker_manifest/) that reference them,
  their [signatures](/customization/sign/docker_sign/), and
  [Docker Hub](/customization/publish/dockerhub/) descriptions.
- Packages pushed to [npm](/customization/publish/npm/),
  [Chocolatey](/customization/package/chocolatey/),
  [Snapcraft](/customization/package/snapcraft/),
  [GemFury](/customization/publish/gemfury/), and
  [Cloudsmith](/customization/publish/cloudsmith/).
- Files sent to [blob storage](/customization/publish/blob/) and by [HTTP
  upload](/customization/publish/upload/), including
  [Artifactory](/customization/publish/artifactory/).
- Servers published to the [MCP registry](/customization/publish/mcp/), Custom
  Apps published to [Iru](/customization/publish/iru/), and
  [custom publishers](/customization/publish/publishers/) that ran.

## When nothing is written

GoReleaser writes nothing when `GITHUB_STEP_SUMMARY` is not set, so runs outside
GitHub Actions produce no extra output. The regular logs already cover the same
ground.

It also writes nothing when a run publishes nothing, for example `goreleaser
build`, a [snapshot](/customization/publish/snapshots/) release, or a run that only
creates a [draft release](/customization/publish/scm/github/). A draft is not
reported as published, because you still have to publish it yourself.

[job-summary]: https://docs.github.com/en/actions/reference/workflow-commands-for-github-actions#adding-a-job-summary
