---
title: Homebrew
---

After releasing to GitHub, GoReleaser can generate and publish a *homebrew-tap*
recipe into a repository that you have access to.

The `brew` section specifies how the formula should be created.
You can check the
[Homebrew documentation](https://github.com/Homebrew/brew/blob/master/docs/How-to-Create-and-Maintain-a-Tap.md)
and the
[formula cookbook](https://github.com/Homebrew/brew/blob/master/docs/Formula-Cookbook.md)
for more details.

```yml
# .goreleaser.yml
brew:
  # Reporitory to push the tap to.
  github:
    owner: user
    name: homebrew-tap

  # Git author used to commit to the repository.
  # Defaults are shown.
  commit_author:
    name: goreleaserbot
    email: goreleaser@carlosbecker.com

  # Folder inside the repository to put the formula.
  # Default is the root folder.
  folder: Formula

  # Caveats for the user of your binary.
  # Default is empty.
  caveats: "How to use this binary"

  # Your app's homepage.
  # Default is empty.
  homepage: "https://example.com/"

  # Your app's description.
  # Default is empty.
  description: "Software to create fast and easy drum rolls."

  # Packages your package depends on.
  dependencies:
    - git
    - zsh

  # Packages that conflict with your package.
  conflicts:
    - svn
    - bash

  # Specify for packages that run as a service.
  # Default is empty.
  plist: |
    <?xml version="1.0" encoding="UTF-8"?>
    ...

  # So you can `brew test` your formula.
  # Default is empty.
  test: |
    system "#{bin}/program --version"
    ...

  # Custom install script for brew.
  # Default is 'bin.install "program"'.
  install: |
    bin.install "program"
    ...
```

By defining the `brew` section, GoReleaser will take care of publishing the
Homebrew tap.
Assuming that the current tag is `v1.2.3`, the above configuration will generate a
`program.rb` formula in the `Formula` folder of `user/homebrew-tap` repository:

```rb
class Program < Formula
  desc "How to use this binary"
  homepage "https://github.com/user/repo"
  url "https://github.com/user/repo/releases/download/v1.2.3/program_v1.2.3_macOs_64bit.zip"
  version "v1.2.3"
  sha256 "9ee30fc358fae8d248a2d7538957089885da321dca3f09e3296fe2058e7fff74"

  depends_on "git"
  depends_on "zsh"

  def install
    bin.install "program"
  end
end
```

**Important**": Note that GoReleaser does not yet generate a valid
homebrew-core formula. The generated formulas are meant to be published as
[homebrew taps](https://docs.brew.sh/brew-tap.html), and in their current
form will not be accepted in any of the official homebrew repositories.
