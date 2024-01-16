---
date: 2022-05-22
slug: nightly-actions
categories:
  - tutorials
authors:
  - hariso
---

# Nightly builds with GoReleaser and GitHub Actions

![](https://miro.medium.com/v2/resize:fit:4800/format:webp/1*MKGwITFSobVveZhnemzKcA.png)
Our flow for nightly builds

<!-- more -->

Nightly builds offer other benefits like insight into a project’s activity. They also help to highlight the continuous delivery process. In this blog, I’ll go over how my team and I set up nightly builds using [GoReleaser](https://goreleaser.com) and [GitHub](https://github.com). I’ll explain it using a bottom-up approach, so you can better understand what drove our decision and the design of the release procedure.

Before we dive in here’s a quick overview of how the [Conduit](https://github.com/ConduitIO/conduit) team actually performs releases. A release consists of:

1. a GitHub release containing binaries for different platforms and a changelog amongst other artifacts.
1. a [Docker](https://www.docker.com) image (pushed to GitHub’s Container Registry)

## Requirements

One of Conduit’s primary drivers is being developer-friendly, which also means that we want developers (and Conduit users generally) to be able to try out all the latest features and fixes in any way they want: be it a binary they run directly or a Docker image. We also want to be very precise about what exactly is new in a build.

The above means that nightly builds and regular builds should have the same structure. For example, if our regular release contains a changelog, a binary for Linux and a Docker image, then a nightly release should also contain a changelog, a binary for Linux and a Docker image.

Ideally, any change in the release structure should be automatically reflected in both, the regular builds and the nightly builds. Continuing the previous example: if we decide to start supporting Plan9, we ideally change the configuration in one place, and see the change in regular and nightly builds alike. The release structure is defined through a GoReleaser configuration, so for that reason we would like to use a single GoReleaser configuration for both types of builds.

We have a new contribution almost every (work)day, so we want the nightly builds to be scheduled around them. We may get dependency upgrades on weekends, but we’re fine not having a nightly build only for those. And, of course, we want the older releases to be cleaned up. We keep the builds for at most 7 days, so our full list of requirements is as follows:

1. Nightly builds are “full releases” (i.e. include everything a “normal” release includes)
1. The existing GoReleaser configuration is used.
1. Nightly builds are scheduled on each working day
1. Nightly builds older than 7 days are removed.

## Our Process Before Nightly Releases

We chose GoReleaser to automate building the binaries, create a GitHub release, etc. To build the Docker image we use Docker’s GitHub actions and not GoReleaser, because Conduit comes with a built-in UI. This allows for multi-stage Docker builds, while GoRelease only supports single-stage builds.

The two parts we had prior to nightly builds were:

1. A GoReleaser config: [.goreleaser.yml](https://github.com/ConduitIO/conduit/blob/main/.goreleaser.yml)
1. A trigger for the release, in a GitHub workflow, [workflows/release.yml](https://github.com/ConduitIO/conduit/blob/main/.github/workflows/release.yml#L3-L6). The trigger is a tag push.

What we have works well for major, minor and patch releases. What this process clearly doesn’t have is scheduling nightly builds nor a cleanup.

## Implementing Nightly Builds

Now we come to the design of the nightly build process: A scheduled GitHub action is pushing a nightly tag:

```yaml
on:
  schedule:
    # * is a special character in YAML, so you have to quote this string
    # doing builds Tue-Sat, so we have changes from Fri
    # available already on Sat
    - cron: "0 0 * * 2-6"
```

That will trigger the full release. Then, we use a GitHub action to clean up older GitHub releases, and also a GitHub action to clean up older Docker images. All of that together can be found in [workflows/trigger-nightly.yml](https://github.com/ConduitIO/conduit/blob/main/.github/workflows/trigger-nightly.yml). Here’s the big picture of everything together:

1. To create a release, GoReleaser and Docker actions are used.
1. A release is triggered by a tag push.
1. “Normal” (major, minor, patch) releases are triggered by manually pushing a tag.
1. Nightly builds (releases) are triggered by pushing a “nightly tag”
1. A workflow is responsible for pushing the nightly tag and also for the cleanup.

## Don’t forget to clean up

Nightly builds will accumulate over time, but we don’t want to keep all of them. In the end, those are not stable releases and relying on them for a longer period of time is not recommended anyway.

We need to clean up older GitHub releases as well as Docker images. Here’s the relevant code snippet from [workflows/trigger-nightly.yml](https://github.com/ConduitIO/conduit/blob/main/.github/workflows/trigger-nightly.yml):

```yaml
- name: "Clean up nightly releases"
  uses: dev-drprasad/delete-older-releases@v0.2.0
  with:
    keep_latest: 5
    delete_tags: true
    delete_tag_pattern: nightly
  env:
    GITHUB_TOKEN: ${{ secrets.NIGHTLY_BUILD_GH_TOKEN }}
- name: "Delete nightly containers older than a week"
  uses: snok/container-retention-policy@v1
  with:
    image-names: conduit
    cut-off: 1 week ago UTC
    account-type: org
    org-name: ConduitIO
    keep-at-least: 5
    token: ${{ secrets.NIGHTLY_BUILD_GH_TOKEN }}
    filter-tags: "*-nightly*"
```

We’ve been running this for a few weeks now (and also started using in [another repository](https://github.com/meroxa/cli)) and it’s been working fine. One of the improvements we’ve identified so far is to announce the nightly release by using GoReleaser.
