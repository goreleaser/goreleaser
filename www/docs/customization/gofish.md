# GoFish

After releasing to GitHub or GitLab, GoReleaser can generate and publish
a _Fish Food_ Cookbook into a repository that you have access to.

The `rigs` section specifies how the fish food should be created.
You can check the
[GoFish documentation](https://gofi.sh/#intro)
and the
[Fish food cookbook](https://gofi.sh/#cookbook)
for more details.

```yaml
# .goreleaser.yaml
rigs:
  -
    # Name template of the recipe
    # Default to project name
    name: myproject

    # IDs of the archives to use.
    # Defaults to all.
    ids:
    - foo
    - bar

    # GOARM to specify which 32-bit arm version to use if there are multiple versions
    # from the build section. GoFish fish food support atm only one 32-bit version.
    # Default is 6 for all artifacts or each id if there a multiple versions.
    goarm: 6

    # NOTE: make sure the url_template, the token and given repo (github or gitlab) owner and name are from the
    # same kind. We will probably unify this in the next major version like it is done with scoop.

    # GitHub/GitLab repository to push the fish food to
    # Gitea is not supported yet, but the support coming
    rig:
      owner: repo-owner
      name: gofish-rig
      # Optionally a branch can be provided. If the branch does not exist, it
      # will be created. If no branch is listed, the default branch will be used
      branch: main
      # Optionally a token can be provided, if it differs from the token provided to GoReleaser
      token: "{{ .Env.GOFISH_RIG_GITHUB_TOKEN }}"

    # Template for the url which is determined by the given Token (github or gitlab)
    # Default for github is "https://github.com/<repo_owner>/<repo_name>/releases/download/{{ .Tag }}/{{ .ArtifactName }}"
    # Default for gitlab is "https://gitlab.com/<repo_owner>/<repo_name>/-/releases/{{ .Tag }}/downloads/{{ .ArtifactName }}"
    # Default for gitea is "https://gitea.com/<repo_owner>/<repo_name>/releases/download/{{ .Tag }}/{{ .ArtifactName }}"
    url_template: "http://github.mycompany.com/foo/bar/releases/{{ .Tag }}/{{ .ArtifactName }}"

    # Git author used to commit to the repository.
    # Defaults are shown.
    commit_author:
      name: goreleaserbot
      email: goreleaser@carlosbecker.com

    # The project name and current git tag are used in the format string.
    commit_msg_template: "GoFish fish food update for {{ .ProjectName }} version {{ .Tag }}"

    # Your app's homepage.
    # Default is empty.
    homepage: "https://example.com/"

    # Template of your app's description.
    # Default is empty.
    description: "Software to create fast and easy drum rolls."

    # SPDX identifier of your app's license.
    # Default is empty.
    license: "MIT"

    # Setting this will prevent goreleaser to actually try to commit the updated
    # fish food - instead, the fish food file will be stored on the dist folder only,
    # leaving the responsibility of publishing it to the user.
    # If set to auto, the release will not be uploaded to the GoFish rig
    # in case there is an indicator for prerelease in the tag e.g. v1.0.0-rc1
    # Default is false.
    skip_upload: true
```

!!! tip
    Learn more about the [name template engine](/customization/templates/).

By defining the `rigs` section, GoReleaser will take care of publishing the
GoFish rig.
Assuming that the current tag is `v1.2.3`, the above configuration will generate a
`program.lua` fish food in the `Food` folder of `user/gofish-rig` repository:

```lua
local name = "Program"
local version = "1.2.3"

food = {
  name = name,
  description = "How to use this binary",
  license = "MIT",
  homepage = "https://github.com/user/repo",
  version = version,
  packages = {
    {
      os = "darwin",
      arch = "amd64",
      url = "https://github.com/user/repo/releases/download/v1.2.3/program_v1.2.3_macOs_64bit.zip",
      sha256 = "9ee30fc358fae8d248a2d7538957089885da321dca3f09e3296fe2058e7fff74",
      resources = {
        {
          path = name,
          installpath = "bin/" .. name,
          executable = true
        }
      }
    }
  }
}
```

## Limitations

- Only one `GOARM` build is allowed;
