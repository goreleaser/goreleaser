# GitHub release upload errors

Here are some common errors that might happen when releasing to GitHub, and some
guidance on how to fix them.

## `Resource:ReleaseAsset Field:name Code:already_exists`

This error essentially means that the file you're trying to upload is already
there.

It usually happens for one of these reasons:

### 1. A GitHub bug, in which it "successfully fails" to upload

Meaning, it says there was an error, but on subsequent tries it replies saying
the file is already there.

There isn't much you can do here, except report to GitHub and maybe try to run
the release from somewhere else.

I already reported multiple times, but it seems themselves are having a hard
time reproducing it.

See also: [this](https://github.com/orgs/community/discussions/14341) and
[this](https://github.com/google/go-github/issues/2113).

### 2. Your configuration is somehow creating more than one file with the same name

A common one here is when your `archives.name` is not specific enough.
You can run your release locally (e.g. `goreleaser release --snapshot`) and
check the `./dist/*.json` files to debug.

### 3. You are running GoReleaser multiple times against the same tag

This one is easier to fix: make sure you are running GoReleaser only on tags,
and only one time per tag.
