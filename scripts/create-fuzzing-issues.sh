#!/bin/bash
# Script to create GitHub issues for fuzzy testing enhancements
# 
# This script reads the issue template files and creates GitHub issues
# for each fuzzy testing candidate identified in the analysis.
#
# Usage:
#   ./create-fuzzing-issues.sh
#
# Prerequisites:
#   - GitHub CLI (gh) must be installed and authenticated
#   - You must have write access to the repository

set -e

REPO="goreleaser/goreleaser"
ISSUE_DIR=".github/ISSUE_TEMPLATES_FUZZING"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}Creating GitHub issues for fuzzy testing enhancements${NC}"
echo "Repository: $REPO"
echo "Issue templates directory: $ISSUE_DIR"
echo ""

# Check if gh CLI is installed
if ! command -v gh &> /dev/null; then
    echo -e "${RED}Error: GitHub CLI (gh) is not installed${NC}"
    echo "Please install it from: https://cli.github.com/"
    exit 1
fi

# Check if authenticated
if ! gh auth status &> /dev/null; then
    echo -e "${RED}Error: Not authenticated with GitHub CLI${NC}"
    echo "Please run: gh auth login"
    exit 1
fi

# Counter for created issues
created=0
failed=0

# Function to extract title from markdown file
get_title() {
    local file="$1"
    grep "^title:" "$file" | sed 's/title: *"\(.*\)"/\1/' | sed "s/title: *'\(.*\)'/\1/" | sed 's/title: *//'
}

# Function to extract labels from markdown file
get_labels() {
    local file="$1"
    grep "^labels:" "$file" | sed 's/labels: *\[\(.*\)\]/\1/' | tr ',' '\n' | sed 's/"//g' | sed "s/'//g" | sed 's/^ *//' | sed 's/ *$//' | paste -sd, -
}

# Function to extract body (everything after the front matter)
get_body() {
    local file="$1"
    # Skip the YAML front matter (between --- markers) and get the rest
    awk '/^---$/{i++; next} i==2' "$file"
}

# Process each issue template
for template in "$ISSUE_DIR"/*.md; do
    if [ ! -f "$template" ]; then
        echo -e "${YELLOW}No issue templates found in $ISSUE_DIR${NC}"
        exit 0
    fi
    
    filename=$(basename "$template")
    echo -e "${YELLOW}Processing: $filename${NC}"
    
    # Extract metadata
    title=$(get_title "$template")
    labels=$(get_labels "$template")
    body=$(get_body "$template")
    
    if [ -z "$title" ]; then
        echo -e "${RED}  Error: Could not extract title from $filename${NC}"
        ((failed++))
        continue
    fi
    
    echo "  Title: $title"
    echo "  Labels: $labels"
    
    # Create the issue
    if [ -n "$labels" ]; then
        if gh issue create --repo "$REPO" --title "$title" --body "$body" --label "$labels" > /dev/null 2>&1; then
            echo -e "${GREEN}  ✓ Issue created successfully${NC}"
            ((created++))
        else
            echo -e "${RED}  ✗ Failed to create issue${NC}"
            ((failed++))
        fi
    else
        if gh issue create --repo "$REPO" --title "$title" --body "$body" > /dev/null 2>&1; then
            echo -e "${GREEN}  ✓ Issue created successfully${NC}"
            ((created++))
        else
            echo -e "${RED}  ✗ Failed to create issue${NC}"
            ((failed++))
        fi
    fi
    
    echo ""
    
    # Small delay to avoid rate limiting
    sleep 1
done

echo -e "${GREEN}Summary:${NC}"
echo "  Created: $created issues"
if [ $failed -gt 0 ]; then
    echo -e "  ${RED}Failed: $failed issues${NC}"
fi

if [ $created -gt 0 ]; then
    echo ""
    echo -e "${GREEN}All issues created successfully!${NC}"
    echo "View them at: https://github.com/$REPO/issues"
fi
