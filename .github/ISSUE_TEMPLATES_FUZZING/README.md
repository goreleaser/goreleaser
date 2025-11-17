# Fuzzy Testing Issue Templates

This directory contains GitHub issue templates for adding fuzzy testing to critical areas of the GoReleaser codebase.

## Overview

These templates were created as part of an analysis to identify the top 10 places in GoReleaser that would most benefit from fuzzy testing. Each template is a complete GitHub issue ready to be created.

## Issue Templates

1. **01-yaml-parsing.md** - Add fuzzy testing for YAML configuration parsing
2. **02-packagejson-parsing.md** - Add fuzzy testing for package.json parsing
3. **03-pyproject-parsing.md** - Add fuzzy testing for pyproject.toml parsing
4. **04-cargo-parsing.md** - Add fuzzy testing for Cargo.toml parsing
5. **05-template-engine.md** - Expand fuzzy testing coverage for template engine
6. **06-config-loading.md** - Add fuzzy testing for config loading
7. **07-archive-files.md** - Add fuzzy testing for archive file processing
8. **08-shell-commands.md** - Add fuzzy testing for shell command construction
9. **09-changelog-parsing.md** - Add fuzzy testing for changelog parsing
10. **10-http-handling.md** - Add fuzzy testing for HTTP client utilities

## Creating the Issues

### Option 1: Use the Script (Automated)

Run the provided script to create all issues at once:

```bash
./scripts/create-fuzzing-issues.sh
```

**Prerequisites:**
- GitHub CLI (`gh`) must be installed and authenticated
- You must have write access to the repository

### Option 2: Manual Creation

To create issues manually:

1. Go to https://github.com/goreleaser/goreleaser/issues/new
2. Copy the content from one of the template files (skip the YAML front matter)
3. Use the title and labels specified in the front matter
4. Submit the issue

### Option 3: Using GitHub CLI Directly

For each template file:

```bash
# Extract title, labels, and body from the template
# Then create the issue
gh issue create --repo goreleaser/goreleaser \
  --title "Add fuzzy testing for YAML configuration parsing" \
  --label "enhancement,testing,security" \
  --body-file <(sed -n '/^---$/,/^---$/!p' .github/ISSUE_TEMPLATES_FUZZING/01-yaml-parsing.md | tail -n +2)
```

## Priority Order

The issues are numbered by priority:

**Critical (Immediate):**
- 01 - YAML Configuration Parsing
- 06 - Config Loading
- 08 - Shell Command Construction

**High (Next Sprint):**
- 02 - Package.json Parsing
- 03 - Pyproject.toml Parsing
- 04 - Cargo.toml Parsing
- 05 - Template Engine
- 07 - Archive Files Processing
- 10 - HTTP Client Utilities

**Medium (Future):**
- 09 - Changelog Parsing

## Implementation Guidelines

Each issue includes:
- **Description** - What needs to be implemented
- **Rationale** - Why this is important
- **Proposed Implementation** - Specific fuzz tests to add
- **Example Test Structure** - Code examples to get started
- **Acceptance Criteria** - Checklist for completion
- **Related Files** - Which files to modify
- **Priority** - Urgency level

## Background

For more details on the analysis that led to these recommendations, see:
- `FUZZY_TESTING_ANALYSIS.md` - Comprehensive analysis document
- `internal/artifact/artifact_fuzz_test.go` - Existing fuzzy test example
- `internal/tmpl/fuzz_test.go` - Existing template fuzzy tests
- `scripts/fuzz.sh` - Fuzzing test runner

## References

- [Go Fuzzing Tutorial](https://go.dev/doc/tutorial/fuzz)
- [Go Fuzzing Documentation](https://go.dev/doc/fuzz/)
- [Fuzzing Best Practices](https://github.com/google/fuzzing/blob/master/docs/good-fuzz-target.md)
