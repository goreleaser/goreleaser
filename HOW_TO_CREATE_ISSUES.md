# Creating Fuzzy Testing GitHub Issues

This document provides instructions for creating the 10 GitHub issues identified in the fuzzy testing analysis.

## Quick Start

### Option 1: Use the Python Script (Recommended)

The easiest way to create all 10 issues at once:

```bash
# Install required dependency
pip install requests

# Set your GitHub token (create one at https://github.com/settings/tokens with 'repo' scope)
export GITHUB_TOKEN="your_github_token_here"

# Run the script
python3 scripts/create-fuzzing-issues.py

# Or use dry-run to preview what will be created
python3 scripts/create-fuzzing-issues.py --dry-run
```

### Option 2: Use the Bash Script

If you prefer bash and have GitHub CLI (`gh`) installed:

```bash
# Make sure gh CLI is installed and authenticated
gh auth login

# Run the script
./scripts/create-fuzzing-issues.sh
```

### Option 3: Manual Creation

To create issues one by one manually:

1. Go to https://github.com/goreleaser/goreleaser/issues/new
2. Open one of the template files in `.github/ISSUE_TEMPLATES_FUZZING/`
3. Copy the content (skip the YAML front matter between `---` lines)
4. Use the title from the template file
5. Add the labels specified in the template
6. Submit the issue

## What Will Be Created

The scripts will create 10 GitHub issues:

| # | Title | Labels | Priority |
|---|-------|--------|----------|
| 1 | Add fuzzy testing for YAML configuration parsing | enhancement, testing, security | Critical |
| 2 | Add fuzzy testing for package.json parsing | enhancement, testing, nodejs, bun | High |
| 3 | Add fuzzy testing for pyproject.toml parsing | enhancement, testing, python | High |
| 4 | Add fuzzy testing for Cargo.toml parsing | enhancement, testing, rust | High |
| 5 | Expand fuzzy testing coverage for template engine | enhancement, testing, templates | High |
| 6 | Add fuzzy testing for config loading | enhancement, testing, config, security | Critical |
| 7 | Add fuzzy testing for archive file processing | enhancement, testing, security, files | High |
| 8 | Add fuzzy testing for shell command construction | enhancement, testing, security, critical | Critical |
| 9 | Add fuzzy testing for changelog parsing | enhancement, testing, changelog | Medium |
| 10 | Add fuzzy testing for HTTP client utilities | enhancement, testing, security, http | High |

## Issue Content

Each issue includes:

- **Description**: What needs to be implemented
- **Rationale**: Why this fuzzy testing is important
- **Proposed Implementation**: Specific fuzz tests to add
- **Example Test Structure**: Code snippets to get started
- **Acceptance Criteria**: Checklist for completion
- **Related Files**: Files to modify
- **Priority**: Urgency level

## Priority-Based Implementation

It's recommended to implement these in priority order:

### Phase 1 - Critical (Immediate)
1. Issue #8: Shell Command Construction (security critical)
2. Issue #1: YAML Configuration Parsing (entry point)
3. Issue #6: Config Loading (core functionality)

### Phase 2 - High Priority (Next Sprint)
4. Issue #2: Package.json Parsing
5. Issue #3: Pyproject.toml Parsing
6. Issue #4: Cargo.toml Parsing
7. Issue #5: Template Engine Expansion
8. Issue #7: Archive Files Processing
9. Issue #10: HTTP Client Utilities

### Phase 3 - Medium Priority (Future)
10. Issue #9: Changelog Parsing

## Verification

After running either script, verify the issues were created by visiting:
https://github.com/goreleaser/goreleaser/issues

You should see 10 new issues with the "enhancement" and "testing" labels.

## Troubleshooting

### Python Script

**Error: 'requests' library is required**
```bash
pip install requests
```

**Error: GitHub token required**
- Create a token at https://github.com/settings/tokens
- Select the `repo` scope
- Set it: `export GITHUB_TOKEN="your_token"`

**401 Unauthorized**
- Check your token has the correct permissions
- Verify you have write access to the repository

### Bash Script

**Error: gh CLI is not installed**
- Install from: https://cli.github.com/

**Error: Not authenticated with GitHub CLI**
```bash
gh auth login
```

**Permission denied**
- Verify you have write access to the repository

## References

- Analysis Document: `FUZZY_TESTING_ANALYSIS.md`
- Issue Templates: `.github/ISSUE_TEMPLATES_FUZZING/`
- Python Script: `scripts/create-fuzzing-issues.py`
- Bash Script: `scripts/create-fuzzing-issues.sh`

## Support

If you encounter any issues creating the GitHub issues, please:
1. Check the troubleshooting section above
2. Verify your GitHub permissions
3. Review the issue templates manually
4. Create issues manually if automation fails
