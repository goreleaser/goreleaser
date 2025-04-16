from __future__ import annotations

import posixpath
import re

from mkdocs.config.defaults import MkDocsConfig
from mkdocs.structure.files import File, Files
from mkdocs.structure.pages import Page
from re import Match

# very much stolen/based on https://github.com/squidfunk/mkdocs-material/blob/master/src/overrides/hooks/shortcodes.py

def on_page_markdown(markdown: str, *, page: Page, config: MkDocsConfig, files: Files):
    # Replace callback
    def replace(match: Match):
        type, args = match.groups()
        args = args.strip()
        if type == "version":     return _version_block(args)
        if type == "inline_version":     return _inline_version_block(args)
        elif type == "pro":       return _pro_ad()
        elif type == "tmpl_pro":       return _tmpl_pro_ad()
        elif type == "inline_pro":       return _inline_pro_ad()
        elif type == "featpro":       return _pro_feat_ad()
        elif type == "templates": return _templates_ad()
        elif type == "alpha": return _alpha_block()

        # Otherwise, raise an error
        raise RuntimeError(f"Unknown shortcode: {type}")

    # Find and replace all external asset URLs in current page
    return re.sub(
        r"<!-- md:(\w+)(.*?) -->",
        replace, markdown, flags = re.I | re.M
    )

def _tmpl_pro_ad():
    return "".join([
        f"<details class=\"admonition example\">",
        f"<summary>GoReleaser Pro</summary>",
        f"<p>These template properties are exclusively available with <a href=\"/pro/\">GoReleaser Pro</a>.</p>",
        f"</details>"
    ])

def _pro_feat_ad():
    return "".join([
        f"<div class=\"admonition example\">",
        f"<p class=\"admonition-title\">GoReleaser Pro</p>",
        f"<p>This feature is exclusively available with <a href=\"/pro/\">GoReleaser Pro</a>.</p>",
        f"</div>"
    ])

def _pro_ad():
    return "".join([
        f"<div class=\"admonition example\">",
        f"<p class=\"admonition-title\">GoReleaser Pro</p>",
        f"<p>One or more features are exclusively available with <a href=\"/pro/\">GoReleaser Pro</a>.</p>",
        f"</div>"
    ])

def _inline_pro_ad():
    return f"This feature is only available in GoReleaser Pro."

def _version_block(text: str):
    if "unreleased" in text:
        tag = text.removesuffix("-unreleased")
        return f"> :material-tag-outline: This will be available in the next release ({tag}). Stay tuned!"
    return f"> :material-tag-outline: Since <a href=\"/blog/goreleaser-{text}\">{text}</a>."

def _inline_version_block(text: str):
    if "unreleased" in text:
        tag = text.removesuffix("-unreleased")
        return f"Since: {tag} (unreleased)"
    return f"Since: {text}"

def _alpha_block():
    return f"> :material-flask-outline: This feature is currently in alpha. Feedback is greatly appreciated!"

def _templates_ad():
    return "".join([
        f"<details class=\"tip\">",
        f"<summary>Template Language</summary>",
        f"<p>Discover more about the <a href=\"/customization/templates/\">name template engine</a>.</p>",
        f"</details>"
    ])
