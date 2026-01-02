#!/usr/bin/env python3
"""
Fetch OpenCollective and GitHub sponsors and update the sponsors files.

This script fetches the current list of sponsors and backers from both
OpenCollective and GitHub Sponsors APIs and updates:
- www/docs/sponsors.md
- README.md
- www/docs/overrides/home.html (top sponsors $50+/month only)

Filters applied:
- Only active sponsors
- Recurring contributions: full monthly amount
- One-time contributions: included if within last year, divided by 12
- Only public GitHub sponsorships (excludes private)

Sponsors are grouped into tiers based on their monthly contribution:
- Diamond Sponsors: $500+ per month (128px logo)
- Platinum Sponsors: $250-499 per month (112px logo)
- Gold Sponsors: $100-249 per month (96px logo)
- Silver Sponsors: $50-99 per month (80px logo)
- Bronze Sponsors: $20-49 per month (64px logo)
- Backers: <$20 per month (text list only)

Usage:
    python3 scripts/update-sponsors.py

Environment variables:
    GITHUB_TOKEN - Required for fetching GitHub Sponsors (optional, skips if not set)
"""

import os
import sys
import json
import urllib.request
import urllib.error
from typing import List, Dict, Any, Optional


OPENCOLLECTIVE_API = "https://api.opencollective.com/graphql/v2"
GITHUB_API = "https://api.github.com/graphql"
SPONSORS_FILE = "www/docs/sponsors.md"
README_FILE = "README.md"
HOME_FILE = "www/docs/overrides/home.html"
COLLECTIVE_SLUG = "goreleaser"
GITHUB_USER = "caarlos0"
LOGO_THRESHOLD_USD = 20  # Show logos for sponsors contributing $20+ monthly
HOME_THRESHOLD_USD = 50  # Only show sponsors $50+/month on home page

# Markers for unified sponsors section
SPONSORS_BEGIN_MARKER = "<!-- sponsors:begin -->"
SPONSORS_END_MARKER = "<!-- sponsors:end -->"


def fetch_members() -> List[Dict[str, Any]]:
    """Fetch all active members from OpenCollective using GraphQL."""
    from datetime import datetime, timedelta
    
    query = """
    query collective($slug: String!) {
      collective(slug: $slug) {
        members(role: BACKER) {
          nodes {
            account {
              name
              slug
              website
              imageUrl(height: 96)
            }
            tier {
              name
              amount {
                value
              }
              frequency
            }
            totalDonations {
              value
            }
            since
            isActive
          }
        }
      }
    }
    """
    
    variables = {"slug": COLLECTIVE_SLUG}
    payload = json.dumps({"query": query, "variables": variables}).encode('utf-8')
    
    req = urllib.request.Request(
        OPENCOLLECTIVE_API,
        data=payload,
        headers={"Content-Type": "application/json"}
    )
    
    try:
        with urllib.request.urlopen(req, timeout=30) as response:
            data = json.loads(response.read().decode('utf-8'))
    except urllib.error.HTTPError as e:
        print(f"Error: API returned status code {e.code}", file=sys.stderr)
        print(f"Response: {e.read().decode('utf-8')}", file=sys.stderr)
        sys.exit(1)
    except urllib.error.URLError as e:
        print(f"Error: Failed to connect to API: {e}", file=sys.stderr)
        sys.exit(1)
    
    if "errors" in data:
        print(f"GraphQL errors: {data['errors']}", file=sys.stderr)
        sys.exit(1)
    
    members = data.get("data", {}).get("collective", {}).get("members", {}).get("nodes", [])
    
    # Filter out inactive members
    one_year_ago = datetime.now() - timedelta(days=365)
    active_members = []
    
    for m in members:
        # Must be active
        if not m.get("isActive", False):
            continue
        
        # Must have made contributions
        if m.get("totalDonations", {}).get("value", 0) <= 0:
            continue
        
        # Check if it's a recurring or recent one-time contribution
        tier_info = m.get("tier", {})
        if tier_info:
            frequency = tier_info.get("frequency")
            
            # Include recurring contributions
            if frequency in ["MONTHLY", "YEARLY"]:
                active_members.append(m)
            # Include one-time contributions from the last year
            elif frequency == "ONETIME":
                since_str = m.get("since")
                if since_str:
                    try:
                        # Parse ISO 8601 date
                        since_date = datetime.fromisoformat(since_str.replace('Z', '+00:00'))
                        if since_date.replace(tzinfo=None) >= one_year_ago:
                            active_members.append(m)
                    except (ValueError, AttributeError):
                        # Skip if date parsing fails
                        pass
        
    return active_members


def fetch_github_sponsors(token: Optional[str]) -> List[Dict[str, Any]]:
    """Fetch active, recurring, public GitHub sponsors and recent one-time sponsors."""
    from datetime import datetime, timedelta
    
    if not token:
        print("⚠ Skipping GitHub Sponsors (GITHUB_TOKEN not set)", file=sys.stderr)
        return []
    
    query = """
    query {
      user(login: "%s") {
        sponsorshipsAsMaintainer(first: 100, activeOnly: true) {
          nodes {
            sponsorEntity {
              ... on User {
                login
                name
                url
                avatarUrl
              }
              ... on Organization {
                login
                name
                url
                avatarUrl
              }
            }
            tier {
              name
              monthlyPriceInDollars
              isOneTime
            }
            privacyLevel
            createdAt
          }
        }
      }
    }
    """ % GITHUB_USER
    
    payload = json.dumps({"query": query}).encode('utf-8')
    
    req = urllib.request.Request(
        GITHUB_API,
        data=payload,
        headers={
            "Content-Type": "application/json",
            "Authorization": f"Bearer {token}",
            "User-Agent": "goreleaser-sponsors-script"
        }
    )
    
    try:
        with urllib.request.urlopen(req, timeout=30) as response:
            data = json.loads(response.read().decode('utf-8'))
    except urllib.error.HTTPError as e:
        print(f"Error: GitHub API returned status code {e.code}", file=sys.stderr)
        print(f"Response: {e.read().decode('utf-8')}", file=sys.stderr)
        return []
    except urllib.error.URLError as e:
        print(f"Error: Failed to connect to GitHub API: {e}", file=sys.stderr)
        return []
    
    if "errors" in data:
        print(f"GraphQL errors: {data['errors']}", file=sys.stderr)
        return []
    
    sponsorships = data.get("data", {}).get("user", {}).get("sponsorshipsAsMaintainer", {}).get("nodes", [])
    
    # Convert to similar format as OpenCollective
    one_year_ago = datetime.now() - timedelta(days=365)
    sponsors = []
    
    for s in sponsorships:
        entity = s.get("sponsorEntity", {})
        tier = s.get("tier", {})
        privacy_level = s.get("privacyLevel", "PUBLIC")
        created_at_str = s.get("createdAt")
        
        if not entity:
            continue
        
        # Skip private sponsors
        if privacy_level != "PUBLIC":
            continue
        
        is_one_time = tier.get("isOneTime", False)
        monthly_price = tier.get("monthlyPriceInDollars", 0)
        
        # For one-time sponsors, check if within last year
        if is_one_time:
            if not created_at_str:
                continue
            try:
                created_at = datetime.fromisoformat(created_at_str.replace('Z', '+00:00'))
                if created_at.replace(tzinfo=None) < one_year_ago:
                    continue
                # Divide by 12 for one-time contributions
                monthly_price = monthly_price / 12
            except (ValueError, AttributeError):
                continue
        
        sponsors.append({
            "account": {
                "name": entity.get("name") or entity.get("login", "Anonymous"),
                "slug": entity.get("login", ""),
                "website": entity.get("url", ""),
                "imageUrl": entity.get("avatarUrl", "")
            },
            "tier": {
                "name": tier.get("name", "Sponsor"),
                "amount": {
                    "value": monthly_price
                },
                "frequency": "ONETIME" if is_one_time else "MONTHLY"
            },
            "totalDonations": {
                "value": monthly_price  # Approximate
            },
            "since": created_at_str,
            "isActive": True
        })
    
    return sponsors


def group_members_by_tier(members: List[Dict[str, Any]]) -> Dict[str, List[Dict[str, Any]]]:
    """Group members by their contribution tier/amount."""
    tiers = {}
    seen_members = {}  # Track by slug to deduplicate
    
    for member in members:
        account = member.get("account", {})
        tier_info = member.get("tier", {})
        total_donations = member.get("totalDonations", {}).get("value", 0)
        
        # Skip if no account info
        if not account.get("name"):
            continue
        
        slug = account.get("slug", "")
        # Skip duplicates (keep the one with higher donations)
        if slug in seen_members:
            if seen_members[slug]["total_donations"] >= total_donations:
                continue
        
        # Get monthly amount from tier
        monthly_amount = 0
        if tier_info:
            amount_info = tier_info.get("amount", {})
            if amount_info:
                monthly_amount = amount_info.get("value", 0)
            
            frequency = tier_info.get("frequency")
            # Convert yearly to monthly
            if frequency == "YEARLY" and monthly_amount > 0:
                monthly_amount = monthly_amount / 12
            # One-time contributions already divided by 12 in fetch functions
            # so no additional processing needed here
        
        # Skip if no valid amount
        if monthly_amount <= 0:
            continue
        
        tier_name = tier_info.get("name", "Backers") if tier_info else "Backers"
        
        member_data = {
            "name": account.get("name", "Anonymous"),
            "slug": slug,
            "website": account.get("website", ""),
            "imageUrl": account.get("imageUrl", ""),
            "monthly_amount": monthly_amount,
            "total_donations": total_donations,
            "tier_name": tier_name
        }
        
        # Group by amount ranges
        if monthly_amount >= 500:
            tier_key = "Diamond Sponsors"
        elif monthly_amount >= 250:
            tier_key = "Platinum Sponsors"
        elif monthly_amount >= 100:
            tier_key = "Gold Sponsors"
        elif monthly_amount >= 50:
            tier_key = "Silver Sponsors"
        elif monthly_amount >= LOGO_THRESHOLD_USD:
            tier_key = "Bronze Sponsors"
        else:
            tier_key = "Backers"
        
        # Remove from previous tier if exists
        if slug in seen_members:
            prev_tier = seen_members[slug]["tier"]
            if prev_tier in tiers:
                tiers[prev_tier] = [m for m in tiers[prev_tier] if m["slug"] != slug]
        
        if tier_key not in tiers:
            tiers[tier_key] = []
        tiers[tier_key].append(member_data)
        
        # Track this member
        seen_members[slug] = {"total_donations": total_donations, "tier": tier_key}
    
    # Sort members within each tier by total donations (descending)
    for tier in tiers.values():
        tier.sort(key=lambda x: x["total_donations"], reverse=True)
    
    return tiers


def generate_markdown(tiers: Dict[str, List[Dict[str, Any]]]) -> str:
    """Generate markdown for sponsors list."""
    from datetime import datetime
    
    lines = []
    lines.append(f"<!-- This list is auto-generated by scripts/update-sponsors.py -->")
    lines.append(f"<!-- Last updated: {datetime.now().strftime('%Y-%m-%d %H:%M:%S UTC')} -->")
    lines.append("")
    
    # Define tier order and logo sizes
    tier_config = {
        "Diamond Sponsors": 128,
        "Platinum Sponsors": 112,
        "Gold Sponsors": 96,
        "Silver Sponsors": 80,
        "Bronze Sponsors": 64,
        "Backers": 0  # Text only
    }
    
    for tier_name, logo_size in tier_config.items():
        if tier_name not in tiers:
            continue
        
        members = tiers[tier_name]
        if not members:
            continue
        
        lines.append(f"### {tier_name}")
        lines.append("")
        
        # Show logos for sponsors >= $20/month
        if tier_name != "Backers":
            # Grid layout with logos
            lines.append('<div align="center">')
            lines.append("")
            for member in members:
                url = member["website"] or f"https://opencollective.com/{member['slug']}"
                if member["imageUrl"]:
                    lines.append(
                        f'  <a href="{url}" target="_blank" rel="noopener sponsored">'
                        f'<img src="{member["imageUrl"]}" alt="{member["name"]}" width="{logo_size}" height="{logo_size}" style="border-radius: 8px; margin: 8px;"></a>'
                    )
            lines.append("")
            lines.append("</div>")
            lines.append("")
        else:
            # Text list for backers
            for member in members:
                url = member["website"] or f"https://opencollective.com/{member['slug']}"
                lines.append(f"- [{member['name']}]({url})")
            lines.append("")
    
    return "\n".join(lines)


def generate_home_html(tiers: Dict[str, List[Dict[str, Any]]], min_monthly_amount: float = 50.0) -> str:
    """Generate HTML for home page sponsor cards grouped by tier."""
    from datetime import datetime
    
    lines = []
    lines.append(f"<!-- This list is auto-generated by scripts/update-sponsors.py -->")
    lines.append(f"<!-- Last updated: {datetime.now().strftime('%Y-%m-%d %H:%M:%S UTC')} -->")
    
    # Define tier order and logo sizes for home page (only $50+)
    tier_config = {
        "Diamond Sponsors": 128,
        "Platinum Sponsors": 112,
        "Gold Sponsors": 96,
        "Silver Sponsors": 80,
    }
    
    for tier_name, logo_size in tier_config.items():
        if tier_name not in tiers:
            continue
        
        # Get members for this tier that have logos
        members = [m for m in tiers[tier_name] if m["imageUrl"]]
        if not members:
            continue
        
        # Start a new grid for this tier
        lines.append(f'\t\t\t\t<div class="grid cards">')
        
        for member in members:
            url = member["website"] or f"https://opencollective.com/{member['slug']}"
            lines.append(f'\t\t\t\t\t<p class="card">')
            lines.append(f'\t\t\t\t\t\t<a href="{url}" target="_blank" rel="noopener sponsored">')
            lines.append(f'\t\t\t\t\t\t\t<img src="{member["imageUrl"]}" alt="{member["name"]}" width="{logo_size}" height="{logo_size}" style="border-radius: 8px;">')
            lines.append(f'\t\t\t\t\t\t</a>')
            lines.append(f'\t\t\t\t\t</p>')
        
        lines.append(f'\t\t\t\t</div>')
    
    return "\n".join(lines)


def update_file_with_markers(file_path: str, new_content: str, begin_marker: str, end_marker: str):
    """Update a file between begin_marker and end_marker."""
    with open(file_path, "r") as f:
        content = f.read()
    
    start_idx = content.find(begin_marker)
    if start_idx == -1:
        print(f"Error: Could not find {begin_marker} in {file_path}", file=sys.stderr)
        sys.exit(1)
    
    end_idx = content.find(end_marker, start_idx)
    if end_idx == -1:
        print(f"Error: Could not find {end_marker} in {file_path}", file=sys.stderr)
        sys.exit(1)
    
    # Build new content
    new_file_content = (
        content[:start_idx + len(begin_marker)] + 
        "\n" + 
        new_content + 
        "\n" + 
        content[end_idx:]
    )
    
    with open(file_path, "w") as f:
        f.write(new_file_content)
    
    print(f"✓ Updated {file_path} ({begin_marker})")


def main():
    """Main function."""
    all_sponsors = []
    
    # Fetch OpenCollective sponsors
    print("Fetching sponsors from OpenCollective...")
    oc_members = fetch_members()
    print(f"✓ Found {len(oc_members)} active OpenCollective sponsors/backers")
    all_sponsors.extend(oc_members)
    
    # Fetch GitHub Sponsors
    github_token = os.environ.get("GITHUB_TOKEN")
    
    if github_token:
        print("Fetching sponsors from GitHub...")
        gh_members = fetch_github_sponsors(github_token)
        print(f"✓ Found {len(gh_members)} active GitHub sponsors (recurring + one-time from last year)")
        all_sponsors.extend(gh_members)
    else:
        print("⚠ Skipping GitHub Sponsors (GITHUB_TOKEN not set)")
    
    # Group all sponsors together by tier
    print(f"\nGrouping {len(all_sponsors)} total sponsors by tier...")
    unified_tiers = group_members_by_tier(all_sponsors)
    for tier_name, members_list in unified_tiers.items():
        print(f"  {tier_name}: {len(members_list)} member(s)")
    
    # Generate unified markdown
    print("\nGenerating unified sponsors markdown...")
    markdown = generate_markdown(unified_tiers)
    
    print("Updating sponsors.md...")
    update_file_with_markers(SPONSORS_FILE, markdown, SPONSORS_BEGIN_MARKER, SPONSORS_END_MARKER)
    
    print("Updating README.md...")
    update_file_with_markers(README_FILE, markdown, SPONSORS_BEGIN_MARKER, SPONSORS_END_MARKER)
    
    # Generate home page HTML for top sponsors ($50+/month)
    print("\nGenerating home page HTML for top sponsors ($50+/month)...")
    home_html = generate_home_html(unified_tiers, min_monthly_amount=HOME_THRESHOLD_USD)
    
    # Count how many top sponsors we have
    top_count = sum(1 for tier_name, members in unified_tiers.items() 
                    if tier_name != "Backers" 
                    for member in members 
                    if member["monthly_amount"] >= HOME_THRESHOLD_USD and member["imageUrl"])
    print(f"  Found {top_count} sponsor(s) for home page")
    
    print("Updating home.html...")
    update_file_with_markers(HOME_FILE, home_html, SPONSORS_BEGIN_MARKER, SPONSORS_END_MARKER)
    
    print("\n✨ Done! Sponsors lists updated successfully.")


if __name__ == "__main__":
    main()
