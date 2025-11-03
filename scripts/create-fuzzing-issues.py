#!/usr/bin/env python3
"""
Create GitHub Issues for Fuzzy Testing Enhancements

This script creates GitHub issues from the templates in .github/ISSUE_TEMPLATES_FUZZING/
using the GitHub REST API.

Usage:
    python3 scripts/create-fuzzing-issues.py

Environment Variables:
    GITHUB_TOKEN - GitHub personal access token with repo scope
    
Alternatively, you can pass the token as an argument:
    python3 scripts/create-fuzzing-issues.py --token YOUR_TOKEN
"""

import argparse
import json
import os
import re
import sys
from pathlib import Path
from typing import Dict, List, Tuple

try:
    import requests
except ImportError:
    print("Error: 'requests' library is required. Install with: pip install requests")
    sys.exit(1)


def parse_template(file_path: Path) -> Tuple[str, List[str], str]:
    """
    Parse a markdown template file to extract title, labels, and body.
    
    Args:
        file_path: Path to the template markdown file
        
    Returns:
        Tuple of (title, labels, body)
    """
    content = file_path.read_text()
    
    # Extract front matter (YAML between --- markers)
    front_matter_match = re.match(r'^---\n(.*?)\n---\n(.*)', content, re.DOTALL)
    
    if not front_matter_match:
        # No front matter, use entire content as body
        return "", [], content
    
    front_matter, body = front_matter_match.groups()
    
    # Extract title
    title_match = re.search(r'^title:\s*["\']?(.+?)["\']?\s*$', front_matter, re.MULTILINE)
    title = title_match.group(1) if title_match else ""
    
    # Extract labels
    labels_match = re.search(r'^labels:\s*\[(.*?)\]', front_matter, re.MULTILINE)
    labels = []
    if labels_match:
        labels_str = labels_match.group(1)
        # Parse labels, removing quotes and whitespace
        labels = [
            label.strip().strip('"').strip("'")
            for label in labels_str.split(',')
        ]
    
    return title, labels, body.strip()


def create_issue(
    repo: str,
    title: str,
    body: str,
    labels: List[str],
    token: str
) -> Dict:
    """
    Create a GitHub issue using the REST API.
    
    Args:
        repo: Repository in format "owner/repo"
        title: Issue title
        body: Issue body (markdown)
        labels: List of label names
        token: GitHub personal access token
        
    Returns:
        Response JSON from GitHub API
    """
    url = f"https://api.github.com/repos/{repo}/issues"
    
    headers = {
        "Authorization": f"token {token}",
        "Accept": "application/vnd.github.v3+json",
    }
    
    data = {
        "title": title,
        "body": body,
        "labels": labels,
    }
    
    response = requests.post(url, headers=headers, json=data)
    response.raise_for_status()
    
    return response.json()


def main():
    parser = argparse.ArgumentParser(
        description="Create GitHub issues for fuzzy testing enhancements"
    )
    parser.add_argument(
        "--token",
        help="GitHub personal access token (or set GITHUB_TOKEN env var)",
    )
    parser.add_argument(
        "--repo",
        default="goreleaser/goreleaser",
        help="Repository in format owner/repo (default: goreleaser/goreleaser)",
    )
    parser.add_argument(
        "--dry-run",
        action="store_true",
        help="Show what would be created without actually creating issues",
    )
    
    args = parser.parse_args()
    
    # Get token
    token = args.token or os.environ.get("GITHUB_TOKEN")
    if not token:
        print("Error: GitHub token required. Set GITHUB_TOKEN env var or use --token")
        print("Create a token at: https://github.com/settings/tokens")
        print("Required scope: repo")
        sys.exit(1)
    
    # Find template directory
    script_dir = Path(__file__).parent
    repo_root = script_dir.parent
    template_dir = repo_root / ".github" / "ISSUE_TEMPLATES_FUZZING"
    
    if not template_dir.exists():
        print(f"Error: Template directory not found: {template_dir}")
        sys.exit(1)
    
    # Get all template files
    template_files = sorted(template_dir.glob("*.md"))
    template_files = [f for f in template_files if f.name != "README.md"]
    
    if not template_files:
        print(f"Error: No template files found in {template_dir}")
        sys.exit(1)
    
    print(f"Found {len(template_files)} issue templates")
    print(f"Repository: {args.repo}")
    print(f"Dry run: {args.dry_run}")
    print()
    
    created = 0
    failed = 0
    
    for template_file in template_files:
        print(f"Processing: {template_file.name}")
        
        try:
            title, labels, body = parse_template(template_file)
            
            if not title:
                print(f"  ✗ Error: Could not extract title")
                failed += 1
                continue
            
            print(f"  Title: {title}")
            print(f"  Labels: {', '.join(labels) if labels else 'none'}")
            
            if args.dry_run:
                print(f"  [DRY RUN] Would create issue")
                created += 1
            else:
                result = create_issue(args.repo, title, body, labels, token)
                issue_number = result["number"]
                issue_url = result["html_url"]
                print(f"  ✓ Created issue #{issue_number}: {issue_url}")
                created += 1
                
        except Exception as e:
            print(f"  ✗ Error: {e}")
            failed += 1
        
        print()
    
    print("=" * 60)
    print(f"Summary:")
    print(f"  Created: {created} issues")
    if failed > 0:
        print(f"  Failed: {failed} issues")
    
    if created > 0 and not args.dry_run:
        print()
        print(f"View all issues at: https://github.com/{args.repo}/issues")


if __name__ == "__main__":
    main()
