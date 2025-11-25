from __future__ import annotations

import re

from mkdocs.config.defaults import MkDocsConfig
from mkdocs.structure.files import Files
from mkdocs.structure.pages import Page
from re import Match

# very much stolen/based on https://github.com/squidfunk/mkdocs-material/blob/master/src/overrides/hooks/shortcodes.py


def on_page_markdown(markdown: str, *, page: Page, config: MkDocsConfig, files: Files):
    # Replace callback
    def replace(match: Match):
        type, args = match.groups()
        args = args.strip()
        if type == "version":
            return _version_block(args)
        elif type == "inline_version":
            return _inline_version_block(args)
        elif type == "pro":
            return _pro_ad()
        elif type == "tmpl_pro":
            return _tmpl_pro_ad()
        elif type == "inline_pro":
            return _inline_pro_ad()
        elif type == "featpro":
            return _pro_feat_ad()
        elif type == "templates":
            return _templates_ad()
        elif type == "experimental":
            return _experimental_block(args)
        elif type == "community":
            return _community_badge()

        # Otherwise, raise an error
        raise RuntimeError(f"Unknown shortcode: {type}")

    # Find and replace all external asset URLs in current page
    return re.sub(r"<!-- md:(\w+)(.*?) -->", replace, markdown, flags=re.I | re.M)


def _tmpl_pro_ad():
    return "".join(
        [
            '<details class="admonition example">',
            "<summary>GoReleaser Pro</summary>",
            '<p>These template properties are exclusively available with <a href="/pro/">GoReleaser Pro</a>.</p>',
            "</details>",
        ]
    )


def _pro_feat_ad():
    return "".join(
        [
            '<div class="admonition example">',
            '<p class="admonition-title">GoReleaser Pro</p>',
            '<p>This feature is exclusively available with <a href="/pro/">GoReleaser Pro</a>.</p>',
            "</div>",
        ]
    )


def _pro_ad():
    return "".join(
        [
            '<div class="admonition example">',
            '<p class="admonition-title">GoReleaser Pro</p>',
            '<p>One or more features are exclusively available with <a href="/pro/">GoReleaser Pro</a>.</p>',
            "</div>",
        ]
    )


def _inline_pro_ad():
    return "This feature is only available in GoReleaser Pro"


def _version_block(text: str):
    if "unreleased" in text:
        tag = text.removesuffix("-unreleased")
        return f"> :material-tag-outline: This will be available in the next release ({tag}). Stay tuned!"
    return (
        f'> :material-tag-outline: Since <a href="/blog/goreleaser-{text}">{text}</a>.'
    )


def _inline_version_block(text: str):
    if "unreleased" in text:
        tag = text.removesuffix("-unreleased")
        return f"Since: {tag} (unreleased)"
    return f"Since: {text}"


def _experimental_block(link: str):
    maybeLink = "Feedback"
    if link != "":
        maybeLink = f'<a href="{link}">Feedback</a>'
    return f"> :material-flask-outline: This feature is currently experimental. {maybeLink} is greatly appreciated!"


def _templates_ad():
    return "".join(
        [
            '<details class="tip">',
            "<summary>Template Language</summary>",
            '<p>Discover more about the <a href="/customization/templates/">name template engine</a>.</p>',
            "</details>",
        ]
    )


def _badge(icon: str, text: str = "", type: str = ""):
    classes = f"mdx-badge mdx-badge--{type}" if type else "mdx-badge"
    return "".join(
        [
            f'<span class="{classes}">',
            *([f'<span class="mdx-badge__icon">{icon}</span>'] if icon else []),
            *([f'<span class="mdx-badge__text">{text}</span>'] if text else []),
            "</span>",
        ]
    )


def _community_badge():
    return _badge(
        icon="[:octicons-people-16:](#community 'Community Owned')",
        text="",
        type="right",
    )
