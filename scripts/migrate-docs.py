#!/usr/bin/env python3
"""
Migrate goreleaser docs from MkDocs (www/docs/) to Hugo (www/content/).

Conversions:
  1. <!-- md:X args --> → {{< X args >}}
  2. {% include-markdown "path" ... %} → {{< include file="..." >}}
  3. === "Tab" blocks → {{< tabs >}}{{< tab "Tab" >}}...{{< /tab >}}{{< /tabs >}}
  4. !!! type "title" → {{< callout type="X" >}}
  5. ??? type "title" → {{< details "Title" >}}
  6. Strip MkDocs-specific frontmatter fields
  7. Convert <!-- more --> → <!--more-->
"""

import os
import re
import shutil
import sys
from pathlib import Path

SRC = Path("www/docs")
DST = Path("www/content")

# Admonition type mapping: MkDocs → Hextra callout type
ADMONITION_TYPE_MAP = {
    "note": "info",
    "info": "info",
    "information": "info",
    "tip": "info",
    "hint": "info",
    "success": "info",
    "check": "info",
    "done": "info",
    "warning": "warning",
    "caution": "warning",
    "attention": "warning",
    "danger": "error",
    "error": "error",
    "failure": "error",
    "fail": "error",
    "bug": "error",
    "important": "warning",
    "example": "info",
    "question": "info",
    "quote": "info",
    "abstract": "info",
    "summary": "info",
    "tldr": "info",
}

# MkDocs-specific frontmatter keys to strip
FRONTMATTER_STRIP_KEYS = {"template", "hide"}


def convert_shortcodes(text: str) -> str:
    """Convert <!-- md:X args --> to {{< X args >}}"""

    def replacer(m):
        kind = m.group(1)
        args = m.group(2).strip()

        # Normalize kind: inline_version → inline_version (underscores ok in Hugo)
        # but Hugo shortcode names can't have underscores... wait, they can.
        # Actually Hugo shortcode names can have hyphens but NOT underscores by convention.
        # Let's keep underscores: Hugo allows them.

        if args:
            # Quote the arg if it looks like a plain token (no spaces, no quotes)
            if " " not in args and not args.startswith('"'):
                return f'{{{{< {kind} "{args}" >}}}}'
            return f'{{{{< {kind} {args} >}}}}'
        return f'{{{{< {kind} >}}}}'

    # Match <!-- md:word optional_args -->
    text = re.sub(
        r'<!--\s*md:(\w+)(.*?)-->',
        replacer,
        text,
        flags=re.DOTALL
    )
    return text


def convert_includes(text: str) -> str:
    """Convert {% include-markdown "path" ... %} → {{< include file="..." >}}"""

    def replacer(m):
        path_str = m.group(1)
        # Normalize path: strip leading ../
        path_str = re.sub(r'^(\.\./)+', '', path_str)
        return f'{{{{< include file="{path_str}" >}}}}'

    text = re.sub(
        r'\{%\s*include-markdown\s+"([^"]+)"[^%]*%\}',
        replacer,
        text
    )
    return text


def convert_tabs(text: str) -> str:
    """Convert === "Tab" blocks to Hugo tabs shortcode."""
    lines = text.split('\n')
    result = []
    i = 0
    while i < len(lines):
        line = lines[i]
        # Detect start of a tab group
        if re.match(r'^=== ', line):
            tab_group = []
            while i < len(lines) and re.match(r'^=== ', lines[i]):
                tab_label_m = re.match(r'^=== "([^"]+)"', lines[i])
                if not tab_label_m:
                    tab_label_m = re.match(r"^=== '([^']+)'", lines[i])
                label = tab_label_m.group(1) if tab_label_m else "Tab"
                i += 1
                # Collect indented content (4 spaces)
                content_lines = []
                while i < len(lines) and (lines[i].startswith('    ') or lines[i] == ''):
                    if lines[i] == '':
                        # Empty line - include it but stop if next non-empty line isn't indented
                        # Look ahead
                        j = i + 1
                        while j < len(lines) and lines[j] == '':
                            j += 1
                        if j < len(lines) and (lines[j].startswith('    ') or re.match(r'^=== ', lines[j])):
                            content_lines.append('')
                            i += 1
                        else:
                            break
                    else:
                        content_lines.append(lines[i][4:])  # de-indent by 4
                        i += 1
                # Strip trailing empty lines from content
                while content_lines and content_lines[-1] == '':
                    content_lines.pop()
                tab_group.append((label, content_lines))

            # Emit tabs shortcode
            result.append('{{< tabs >}}')
            for label, content in tab_group:
                result.append(f'{{{{< tab "{label}" >}}}}')
                result.extend(content)
                result.append('{{< /tab >}}')
            result.append('{{< /tabs >}}')
        else:
            result.append(line)
            i += 1
    return '\n'.join(result)


def convert_admonitions(text: str) -> str:
    """Convert !!! type and ??? type admonition blocks."""
    lines = text.split('\n')
    result = []
    i = 0
    while i < len(lines):
        line = lines[i]
        # Match !!! or ??? or ???+
        m = re.match(r'^(\?{3}\+?|!{3})\s+(\w+)(?:\s+"([^"]*)")?', line)
        if m:
            is_collapsible = m.group(1).startswith('?')
            admon_type = m.group(2).lower()
            title = m.group(3)
            i += 1

            # Collect indented content
            content_lines = []
            while i < len(lines) and (lines[i].startswith('    ') or lines[i] == ''):
                if lines[i] == '':
                    j = i + 1
                    while j < len(lines) and lines[j] == '':
                        j += 1
                    if j < len(lines) and lines[j].startswith('    '):
                        content_lines.append('')
                        i += 1
                    else:
                        break
                else:
                    content_lines.append(lines[i][4:])  # de-indent
                    i += 1

            while content_lines and content_lines[-1] == '':
                content_lines.pop()

            if is_collapsible:
                display_title = title or admon_type.capitalize()
                result.append(f'{{{{< details "{display_title}" >}}}}')
                result.extend(content_lines)
                result.append('{{< /details >}}')
            else:
                callout_type = ADMONITION_TYPE_MAP.get(admon_type, "info")
                if title:
                    result.append(f'{{{{< callout type="{callout_type}" >}}}}')
                    result.append(f'**{title}**')
                    result.append('')
                    result.extend(content_lines)
                    result.append('{{< /callout >}}')
                else:
                    result.append(f'{{{{< callout type="{callout_type}" >}}}}')
                    result.extend(content_lines)
                    result.append('{{< /callout >}}')
        else:
            result.append(line)
            i += 1
    return '\n'.join(result)


def strip_frontmatter(text: str) -> tuple[dict, str]:
    """Parse and return (frontmatter_dict, body). Strips MkDocs-specific keys."""
    if not text.startswith('---'):
        return {}, text
    end = text.find('\n---', 3)
    if end == -1:
        return {}, text
    fm_text = text[3:end].strip()
    body = text[end + 4:].lstrip('\n')
    return fm_text, body


def process_frontmatter(text: str, is_include: bool = False) -> str:
    """Remove MkDocs-specific frontmatter fields, keep the rest."""
    if not text.startswith('---'):
        return text

    end = text.find('\n---', 3)
    if end == -1:
        return text

    fm_text = text[3:end]
    body = text[end + 4:]

    # Remove specific MkDocs keys (template:, hide:)
    lines = fm_text.split('\n')
    new_lines = []
    skip = False
    for line in lines:
        stripped = line.lstrip()
        # Check if this line starts a key we want to remove
        key_match = re.match(r'^(\w+)\s*:', line)
        if key_match and key_match.group(1) in FRONTMATTER_STRIP_KEYS:
            skip = True
            continue
        # If we're skipping and this line is a continuation (indented), skip it
        if skip and (line.startswith('  ') or line.startswith('\t')):
            continue
        skip = False
        new_lines.append(line)

    new_fm = '\n'.join(new_lines).strip()

    if new_fm:
        return f'---\n{new_fm}\n---\n{body}'
    else:
        return body.lstrip('\n')


def convert_more_tag(text: str) -> str:
    """Convert <!-- more --> to <!--more--> (Hugo format)."""
    return re.sub(r'<!--\s*more\s*-->', '<!--more-->', text)


def convert_file(src_path: Path, dst_path: Path) -> None:
    """Process a single markdown file."""
    dst_path.parent.mkdir(parents=True, exist_ok=True)

    if src_path.suffix != '.md':
        shutil.copy2(src_path, dst_path)
        return

    text = src_path.read_text(encoding='utf-8')

    # Apply conversions in order
    text = process_frontmatter(text)
    text = convert_more_tag(text)
    text = convert_shortcodes(text)
    text = convert_includes(text)
    text = convert_tabs(text)
    text = convert_admonitions(text)

    dst_path.write_text(text, encoding='utf-8')


def main():
    if not SRC.exists():
        print(f"Error: {SRC} does not exist", file=sys.stderr)
        sys.exit(1)

    # Clean destination (but keep layouts/, static/, hugo.yaml, go.mod etc.)
    if DST.exists():
        shutil.rmtree(DST)
    DST.mkdir(parents=True)

    # Walk and convert
    count = 0
    for src_file in SRC.rglob('*'):
        if src_file.is_file():
            rel = src_file.relative_to(SRC)
            dst_file = DST / rel
            convert_file(src_file, dst_file)
            count += 1

    print(f"Migrated {count} files from {SRC} to {DST}")

    # Rename index.md → _index.md for Hugo sections
    rename_section_indexes()

    # Move static dir (www/docs/static → www/static is not needed since we
    # already have www/static; but docs/static content needs to go to www/static)
    consolidate_static()

    print("Done!")


def rename_section_indexes():
    """Rename index.md files that are section indexes to _index.md."""
    # In MkDocs, index.md is both section index AND a page.
    # In Hugo, _index.md is the section list page.
    # Files named index.md that are inside a directory that has other .md files
    # should become _index.md.
    for index_file in DST.rglob('index.md'):
        parent = index_file.parent
        # Don't rename root blog/index.md as it's a simple page
        siblings = list(parent.glob('*.md'))
        if len(siblings) > 1:
            new_path = parent / '_index.md'
            if not new_path.exists():
                index_file.rename(new_path)
                print(f"  Renamed {index_file.relative_to(DST)} → _index.md")


def consolidate_static():
    """Move www/content/static/ back to www/static/ (already the right place)."""
    content_static = DST / 'static'
    www_static = Path('www/static')
    if content_static.exists():
        www_static.mkdir(exist_ok=True)
        for item in content_static.iterdir():
            dest = www_static / item.name
            if not dest.exists():
                shutil.move(str(item), str(dest))
        # Remove the static dir from content if empty
        try:
            content_static.rmdir()
        except OSError:
            # Not empty, leave remaining files
            pass


if __name__ == '__main__':
    main()
