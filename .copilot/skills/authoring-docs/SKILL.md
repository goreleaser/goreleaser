---
name: authoring-docs
description: Use when writing or editing GoReleaser documentation under www/content (Hugo site) — customization reference pages, YAML config examples, "Since vX.Y" / Pro badges, callouts, or documenting a feature that is merged but not yet released.
---

# Authoring GoReleaser Docs

The docs are a [Hugo](https://gohugo.io) site under `www/`. Content lives in
`www/content/`; custom shortcodes in `www/layouts/shortcodes/`.

- Preview locally: `task docs:serve` (runs `docs:generate` then `hugo server`).
- Build / link-check: `task docs:build` / `task docs:test`.

## Page structure

Pages are Markdown with YAML frontmatter:

```markdown
---
title: "Discourse"
weight: 30
---
```

`weight` orders pages within a section (lower = higher in the nav).

## Version & Pro badges

These mark when a feature appeared or that it is Pro-only. **Two forms exist:
block (its own line, renders a callout box) and inline (renders a small badge).**

| Shortcode | Use |
|-----------|-----|
| `{{< g_version "v2.13" >}}` | Block. Top of a new page (right after frontmatter) or under a section heading for a feature added later. |
| `{{< g_inline_version "v2.6" >}}` | Inline. Inside YAML comments and Markdown table cells. |
| `{{< g_featpro >}}` | Block. Pro-only feature note. |
| `{{< g_inline_pro >}}` | Inline. Marks a single field/option as Pro-only. |

The version is the GoReleaser tag the feature shipped in (e.g. `v2.6`,
`v2.15.4`). `g_inline_version` links to the corresponding `/blog/goreleaser-vX.Y`
release post.

## Unreleased features (`-unreleased` suffix)

When documenting something already merged but **not yet released**, append
`-unreleased` to the upcoming version number. Use the next unreleased minor
(currently `v2.17` — check the latest tag / `## A note about MSIX` in
`customization/package/nfpm.md` for the live example).

```yaml
    # {{< g_inline_version "v2.17-unreleased" >}}
```

```markdown
{{< g_version "v2.17-unreleased" >}}
```

Both shortcodes detect the `-unreleased` suffix and render a "beaker"
(experimental) badge — "Since v2.17 (unreleased)" / "This will be available in
the next release". When v2.17 ships, drop the `-unreleased` suffix.

## YAML config examples

Customization pages document fields as a commented `.goreleaser.yaml` block,
fenced with a `filename` attribute:

````markdown
```yaml {filename=".goreleaser.yaml"}
announce:
  discourse:
    # Whether this announcer is enabled or not.
    #
    # Templates: allowed.
    enabled: true

    # Title to use for the Discourse topic.
    #
    # Default: '{{ .ProjectName }} {{ .Tag }} is out!'.
    # Templates: allowed.
    title_template: "..."
```
````

Conventions inside the block:

- One comment block per field, directly above it. Fields separated by a blank line.
- Description sentence(s) first, then a bare `#` separator line, then standardized
  annotations — each on its own line:
  - `# Default: <value>.`
  - `# Templates: allowed.`
  - `# Required.`
  - `# Valid options: a, b, c.`
- Badges go at the **end** of a comment line (e.g.
  `# Templates: allowed. {{< g_inline_version "v2.6" >}}`) or on their own
  comment line right before the field (e.g. `# {{< g_inline_pro >}}`).

## Callouts

Block-level notes, placed in prose with a blank line above and below:

- `{{< g_experimental "https://github.com/.../issues/123" >}}` — experimental
  feature; the URL is the feedback link.
- `{{< g_featpro >}}` — Pro-only feature.
- `{{< g_templates >}}` — "learn more about templates"; place after a config
  block whose fields support templates.

## Shortcode delimiters

- `{{< shortcode >}}` (angle brackets) — default; raw HTML output. Used by all
  badges, callouts, and `{{< tabs >}}` / `{{< tab >}}`.
- `{{% shortcode %}}` (percent) — only when output is Markdown to be rendered:
  `g_include` and `g_button`.

## Reuse & tabs

- Shared YAML snippets live in `www/content/includes/*.md` and are embedded with
  `{{% g_include file="includes/repository.md" %}}` (reused: `repository.md`,
  `prs.md`, `commit_author.md`).
- OSS vs Pro variants:
  `{{< tabs >}}{{< tab "OSS" >}}…{{< /tab >}}{{< tab "Pro" >}}…{{< /tab >}}{{< /tabs >}}`.

## Don't hand-edit (generated)

- `www/content/resources/{contributing,users,eula,security}.md` — copied from
  repo-root files by `task docs:generate`.
- `www/static/schema.json` — generated from `pkg/config/config.go` via `task schema`.
