---
date: 2023-09-27
slug: release-cadence
categories:
  - announcements
authors:
  - caarlos0
---

# Version strategy and release cadence: the future

A couple of weeks ago, I got a couple of complaints about the way GoReleaser
is being versioned - more precisely, the fact that deprecated options are
removed in **minor** instead of **major** versions.

Those complaints are valid, and today I'm announcing how I plan to move forward.

<!-- more -->

## History

Before we do that, I feel like I should explain how we got in this position
first.

GoReleaser was in a [0ver](https://0ver.org)[^joke] scheme for _almost 5 years_.
We had a whopping _468 `v0.x.x` releases_.
Let me repeat: **four hundred and sixty eight v zeroes**.

[^joke]: yes, I know this is kind of a joke.

In fact, `v1` was launched less than 2 years ago, in [November, 2022][v1].

Like many other things, the way we handle deprecations was from that time.
A time in which GoReleaser never had major releases, because there wasn't
one: we would add the deprecation notice to the
[deprecations page][deprecations] and warn about it in the release's output
if you use any of them, and, after roughly 6 months, remove the deprecated
option and move on with life.

[Breaking changes are allowed in v0][semver-si4], so, there were no broken
promises.

On the other hand, `v1` is a _major_ version, so it should not introduce
breaking changes.

In retrospect, my mistake was never stopping to think about it again after `v1`.

[semver-si4]: https://semver.org/#spec-item-4

## Going forward

Thankfully, I was nudged in the right direction, so, from now on, we'll do
things properly.

The plan is as follows:

1. We'll have 1 _minor_ release 1-2 months (when we have some material);
1. Bug fixes will still be released as _patch_ releases in the latest _minor_;
1. Deprecations will continue to be added in _minor_ releases (but **not
   removed**);
1. When we have a good amount of deprecations, we'll launch a new _major_,
   removing them completely.
   I think this will probably happen about once a year.

So, if you lock your CI to get `v1.x.x`, you might get new deprecation
warnings, but no breaking changes.

You can then better plan when to upgrade your apps to the latest _major_,
without having to lock to a specific _minor_/_patch_ release.

## Supporting old versions

I understand GoReleaser has become the _de-facto_ tool to release Go projects.
I don't know how many users we have (because we don't track you), but judging by
some code searches on GitHub, there are thousands of repositories using it.

GoReleaser is also a big project.
Maintaining it could already be a full-time job, but, it isn't.
I work on it on my free time, which is limited - just like yours.

All that being said, I understand that big companies and teams rely on
GoReleaser, and some don't release as often.

With that in mind, [GoReleaser Pro][gpro] customers will have more time to
update: I'll keep launching _patch_ releases of the latest _major_ containing
relevant bug fixes and security-related fixes.

If you use the OSS version, you can either pin to the previous major, or update
to the new one.
You can also get a [GoReleaser Pro][gpro] license, and help fund this project.

I haven't yet developed the exact rules for which bug fixes will get backported
or not, but I'm quite confident that it'll be a mix of "fixes for really bad
bugs" and "a customer is experiencing it and asked it to be fixed", and, of
course, any security-related fixes.

## So, when v2?

Probably in a couple of months.

Stay tuned! 📰

## Early access

Since a couple of weeks ago, we're building a new nightly automatically every
week.

You can already use _nightly_ as the version in [our GitHub Action][gha] if
you can't wait for a new feature that's already on `main`.

This works for both the Pro and OSS distributions.

## Summing up

The _TLDR_:

- new _major_ version ~yearly;
- new _minor_ version every ~two months;
- new _patch_ versions whenever its needed, in the latest _minor_ only;
- [Pro][gpro] latest version keep getting security updates and relevant bug fixes;
- _nigtlies_ weekly for both Pro and OSS;

## Thank you notes

I would like to publicly thank everyone who commented and shared both their
pains and their experiences in [the discussion that started all this][dis],
and specially, [@LandonTClipp](https://github.com/LandonTClipp), who created it.

All of us that do OpenSource know how easy a conversation like this could have
gone south. Thankfully, this was not the case here. 💌

Thank you, for real.
Thank you for patience, for the contributions, and for pointing out ways I can
make GoReleaser better.

See you all soon!

[v1]: ./2021-11-14-goreleaser-v1.md
[deprecations]: ../../deprecations.md
[dis]: https://github.com/orgs/goreleaser/discussions/4169
[gpro]: ../../pro.md
[gha]: https://github.com/goreleaser/goreleaser-action
[Sponsors]: https://github.com/caarlos0
