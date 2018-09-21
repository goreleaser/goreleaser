---
title: Scoop
series: customization
hideFromIndex: true
weight: 100
---

After releasing to GitHub, GoReleaser can generate and publish a
_Scoop App Manifest_ into a repository that you have access to.

The `scoop` section specifies how the manifest should be created. See
the commented example bellow:

```yml
# .goreleaser.yml
scoop:
  # Template for the url.
  # Default is "https://github.com/<repo_owner>/<repo_name>/releases/download/{{ .Tag }}/{{ .ArtifactName }}"
  url_template: "http://github.mycompany.com/foo/bar/releases/{{ .Tag }}/{{ .ArtifactName }}"

  # Repository to push the app manifest to.
  bucket:
    owner: user
    name: scoop-bucket

  # Git author used to commit to the repository.
  # Defaults are shown.
  commit_author:
    name: goreleaserbot
    email: goreleaser@carlosbecker.com

  # Your app's homepage.
  # Default is empty.
  homepage: "https://example.com/"

  # Your app's description.
  # Default is empty.
  description: "Software to create fast and easy drum rolls."

  # Your app's license
  # Default is empty.
  license: MIT

  # Persist data between application updates
  persist:
  - "data"
  - "config.toml"
```

By defining the `scoop` section, GoReleaser will take care of publishing the
Scoop app. Assuming that the project name is `drumroll` and the current tag is
`v1.2.3`, the above configuration will generate a `drumroll.json` manifest in
the root of the repository specified in the `bucket` section.

```json
{
  "version": "1.2.3",
  "architecture": {
    "64bit": {
      "url":
        "https://github.com/user/drumroll/releases/download/1.2.3/drumroll_1.2.3_windows_amd64.tar.gz",
      "bin": "drumroll.exe",
      "hash": "86920b1f04173ee08773136df31305c0dae2c9927248ac259e02aafd92b6008a"
    },
    "32bit": {
      "url":
        "https://github.com/user/drumroll/releases/download/1.2.3/drumroll_1.2.3_windows_386.tar.gz",
      "bin": "drumroll.exe",
      "hash": "283faa524ef41987e51c8786c61bb56658a489f63512b32139d222b3ee1d18e6"
    }
  },
  "homepage": "https://example.com/"
}
```

Your users can then install your app by doing:

```sh
scoop bucket add app https://github.com/org/repo.git
scoop install app
```

You can check the
[Scoop documentation](https://github.com/lukesampson/scoop/wiki) for more
details.
