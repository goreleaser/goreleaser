# Scoop Manifests

After releasing to GitHub, GitLab, or Gitea, GoReleaser can generate and publish a
_Scoop App Manifest_ into a repository that you have access to.

The `scoop` section specifies how the manifest should be created. See the
commented example below:

```yaml
# .goreleaser.yaml
scoop:
  # Template for the url which is determined by the given Token (github or gitlab)
  # Default for github is "https://github.com/<repo_owner>/<repo_name>/releases/download/{{ .Tag }}/{{ .ArtifactName }}"
  # Default for gitlab is "https://gitlab.com/<repo_owner>/<repo_name>/-/releases/{{ .Tag }}/downloads/{{ .ArtifactName }}"
  # Default for gitea is "https://gitea.com/<repo_owner>/<repo_name>/releases/download/{{ .Tag }}/{{ .ArtifactName }}"
  url_template: "http://github.mycompany.com/foo/bar/releases/{{ .Tag }}/{{ .ArtifactName }}"

  # Repository to push the app manifest to.
  bucket:
    # Repository owner template. (templateable)
    owner: user

    # Repository name. (templateable)
    name: scoop-bucket

    # Optionally a branch can be provided. (templateable)
    #
    # Default: the repository default branch
    branch: main

    # Optionally a token can be provided, if it differs from the token provided
    # to GoReleaser
    token: "{{ .Env.SCOOP_TAP_GITHUB_TOKEN }}"

  # Folder inside the repository to put the scoop.
  folder: Scoops

  # Git author used to commit to the repository.
  commit_author:
    name: goreleaserbot
    email: bot@goreleaser.com

  # The project name and current git tag are used in the format string.
  commit_msg_template: "Scoop update for {{ .ProjectName }} version {{ .Tag }}"

  # Your app's homepage.
  homepage: "https://example.com/"

  # Your app's description.
  description: "Software to create fast and easy drum rolls."

  # Your app's license
  license: MIT

  # Setting this will prevent goreleaser to actually try to commit the updated
  # manifest leaving the responsibility of publishing it to the user.
  # If set to auto, the release will not be uploaded to the scoop bucket
  # in case there is an indicator for prerelease in the tag e.g. v1.0.0-rc1
  skip_upload: true

  # Persist data between application updates
  persist:
  - "data"
  - "config.toml"

  # An array of commands to be executed before an application is installed.
  pre_install: ["Write-Host 'Running preinstall command'"]

  # An array of commands to be executed after an application is installed.
  post_install: ["Write-Host 'Running postinstall command'"]

  # An array of dependencies.
  #
  # Since GoReleaser v1.16
  depends: ["git", "foo"]


  # A two-dimensional array of string, specifies the shortcut values to make available in the startmenu.
  # The array has to contain a executable/label pair. The third and fourth element are optional.
  #
  # Since GoReleaser v1.17.0.
  shortcuts: [["drumroll.exe", "drumroll"]]

  # GOAMD64 to specify which amd64 version to use if there are multiple versions
  # from the build section.
  #
  # Default: v1
  goamd64: v3
```

By defining the `scoop` section, GoReleaser will take care of publishing the
Scoop app. Assuming that the project name is `drumroll`, and the current tag is
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
scoop bucket add org https://github.com/org/repo.git
scoop install org/drumroll
```

You can check the
[Scoop documentation](https://github.com/lukesampson/scoop/wiki) for more
details.
