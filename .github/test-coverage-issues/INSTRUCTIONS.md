# Test Coverage Analysis - Issues Ready to Create

## Overview

I've analyzed the test coverage of the goreleaser/goreleaser repository and identified **6 specific areas** with low test coverage that could be easily improved. Each area has been documented as a ready-to-create GitHub issue.

### Current State
- **Overall Test Coverage:** 80.7%
- **Total Uncovered Methods Identified:** 17
- **Expected Coverage After All Improvements:** 82-83%

## Issues Documentation

All issue documentation is available in `/tmp/issues_to_create/`:

1. `issue-1-jsonschema-tests.md` - 8 JSONSchema methods (0% coverage)
2. `issue-2-nfpm-tests.md` - 2 NFPM conversion methods (0% coverage)
3. `issue-3-archive-copy-tests.md` - 2 Archive Copy methods (0% coverage) *Has TODO in code!*
4. `issue-4-poetry-target-tests.md` - 2 Poetry Target methods (0% coverage)
5. `issue-5-env-methods-tests.md` - 2 Env utility methods (0% coverage)
6. `issue-6-checkscm-test.md` - 1 CheckSCM method (0% coverage)

## How to Create These Issues

### Option 1: Automated Script (Recommended)

I've created a script that will create all 6 issues automatically:

```bash
# First, authenticate with GitHub
gh auth login

# Then run the script
/tmp/issues_to_create/create_issues.sh
```

The script will create all 6 issues in the goreleaser/goreleaser repository with proper titles, labels, and detailed descriptions.

### Option 2: Manual Creation

If you prefer to create issues manually, each `.md` file contains:
- Issue title (at the top)
- Recommended labels
- Complete description with:
  - Overview
  - List of affected methods with line numbers
  - Current and target coverage
  - Step-by-step implementation guide
  - Example test code
  - Files to modify
  - Difficulty rating

Simply copy the content from each file and paste it into a new GitHub issue.

### Option 3: Review First

If you want to review before creating:

```bash
# View summary of all issues
cat /tmp/issues_to_create/README.md

# View individual issues
cat /tmp/issues_to_create/issue-1-jsonschema-tests.md
cat /tmp/issues_to_create/issue-2-nfpm-tests.md
# ... etc
```

## Priority Recommendation

If you want to create issues in priority order:

1. **Issue 5** (Env methods) - Biggest coverage jump (57.9% → 80%), very easy
2. **Issue 1** (JSONSchema) - Good impact (46% → 55-60%), 8 methods with similar pattern
3. **Issue 4** (Poetry Target) - Quick win, very easy, 5-minute task
4. **Issue 3** (Archive Copy) - Addresses existing TODO comment in code
5. **Issue 2** (NFPM) - Small but easy
6. **Issue 6** (CheckSCM) - Small but easy

## Labels to Use

All issues should be labeled with:
- `good first issue` - They're well-scoped and don't require deep codebase knowledge
- `test coverage` - Clearly indicates the purpose
- `enhancement` - Improving the codebase

## Key Benefits

1. **Easy Wins:** All identified areas are straightforward to test
2. **Good First Issues:** Perfect for new contributors
3. **Well-Documented:** Each issue includes example code and clear instructions
4. **Measurable Impact:** Clear before/after coverage metrics
5. **Existing TODO:** Issue #3 addresses a TODO comment already in the code

## Files Summary

```
/tmp/issues_to_create/
├── README.md                       # This file
├── ISSUES.md                       # Detailed master document with all issues
├── create_issues.sh                # Automated creation script
├── issue-1-jsonschema-tests.md     # JSONSchema methods issue
├── issue-2-nfpm-tests.md           # NFPM conversion issue
├── issue-3-archive-copy-tests.md   # Archive Copy issue
├── issue-4-poetry-target-tests.md  # Poetry Target issue
├── issue-5-env-methods-tests.md    # Env utility methods issue
└── issue-6-checkscm-test.md        # CheckSCM issue
```

## Analysis Methodology

The analysis was conducted by:
1. Running the full test suite with coverage: `go test -coverprofile=coverage.txt ./...`
2. Analyzing coverage with: `go tool cover -func=coverage.txt`
3. Identifying packages with coverage below 70%
4. Examining specific methods with 0% coverage
5. Evaluating testability (can it be easily tested?)
6. Documenting methods that are straightforward to test

## Next Steps

1. Review the issues (optional)
2. Run `/tmp/issues_to_create/create_issues.sh` to create all issues
3. Or manually create issues one by one from the `.md` files
4. Contributors can then pick up these "good first issue" tasks

---

**Note:** Due to system constraints, I cannot directly create GitHub issues programmatically. The script and documentation provided will enable easy creation of these issues with proper authentication.
