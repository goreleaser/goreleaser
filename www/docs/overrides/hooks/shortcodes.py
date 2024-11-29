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
        elif type == "pro":       return _pro_ad(page, files)
        elif type == "featpro":       return _pro_feat_ad(page, files)
        elif type == "templates": return _templates_ad()
        elif type == "alpha": return _alpha_block()

        # Otherwise, raise an error
        raise RuntimeError(f"Unknown shortcode: {type}")

    # Find and replace all external asset URLs in current page
    return re.sub(
        r"<!-- md:(\w+)(.*?) -->",
        replace, markdown, flags = re.I | re.M
    )

def _pro_feat_ad(page: Page, files: Files):
    return "".join([
        f"<div class=\"admonition example\">",
        f"<p class=\"admonition-title\">GoReleaser Pro</p>",
        f"<p>This feature is exclusively available with <a href=\"/pro/\">GoReleaser Pro</a>.</p>",
        f"</div>"
    ])

def _pro_ad(page: Page, files: Files):
    return "".join([
        f"<div class=\"admonition example\">",
        f"<p class=\"admonition-title\">GoReleaser Pro</p>",
        f"<p>One or more features are exclusively available with <a href=\"/pro/\">GoReleaser Pro</a>.</p>",
        f"</div>"
    ])

def _version_block(text: str):
    return f"> :material-tag-outline: Since <a href=\"/blog/goreleaser-{text}\">{text}</a>."

def _alpha_block():
    return f"> :material-flask-outline: This feature is currently in alpha. Feedback is greatly appreciated!"

def _templates_ad():
    return "".join([
        f"<div class=\"admonition tip\">",
        f"<p class=\"admonition-title\">Tip</p>",
        f"<p>Discover more about the <a href=\"/customization/templates/\">name template engine</a>.</p>",
        f"</div>"
    ])
