# Resource not accessible by integration

When using GitHub Actions, you might feel tempted to use the action-bound `GITHUB_TOKEN`.

While it is a good practice, and should work for most cases, if your workflow writes a file in another repository, you may see this error:

```sh
   ⨯ release failed after 430.85s error=homebrew tap formula: failed to publish artifacts: PUT https://api.github.com/repos/user/homebrew-tap/contents/Formula/scorecard.rb: 403 Resource not accessible by integration []
```

Integrations that may cause this:

- Homebrew Tap
- Gofish Rigs
- Krew Plugins

## Fixing it

You have basically two options:

### 1. Use a Personal Access Token (PAT) for the entire process

You can create a PAT and use it for the entire GoReleaser action run.
You'll need to add it as secret and pass it to the action, for instance:

```yaml
# .github/workflows/goreleaser.yaml
# ...
      - uses: goreleaser/goreleaser-action@v2
        env:
          GITHUB_TOKEN: ${{ secrets.GH_PAT }}
# ...
```

### 2. Use a Personal Access Token (PAT) specifically for the integration

You can also create a PAT for each integration.

Let's see, for example, how it would look like for Homebrew Taps.

We would need to change the workflow file:

```yaml
# .github/workflows/goreleaser.yaml
# ...
      - uses: goreleaser/goreleaser-action@v2
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          TAP_GITHUB_TOKEN: ${{ secrets.TAP_GITHUB_TOKEN }}
# ...
```

And also the `.goreleaser.yaml` file:

```yaml
# .goreleaser.yaml
# ...
brews:
- name: myproject
  tap:
    owner: user
    name: homebrew-tap
    token: "{{ .Env.TAP_GITHUB_TOKEN }}"
# ...
```

## Learning more

Read the documentation for each topic:

- [GitHub](/scm/github/)
- [GitHub Actions](/ci/actions/)
- [Homebrew Tap](/customization/homebrew/)
- [Gofish Rigs](/customization/gofish/)
- [Krew Plugin Manifests](/customization/krew/)
- [Scoop Manifests](/customization/scoop/)
