# Test Coverage Improvement Issues - Summary

This document provides a summary of 6 GitHub issues that should be created to improve test coverage in the goreleaser/goreleaser repository. Each issue targets specific untested or under-tested code that would be easy to improve.

## Quick Stats

- **Current Overall Coverage:** 80.7%
- **Expected Coverage After All Issues:** ~82-83%
- **Total Uncovered Methods:** 17
- **Difficulty:** All issues are rated Easy or Very Easy (good first issues)

## Issues to Create

### Issue 1: JSONSchema Methods (pkg/config)
- **File:** `/tmp/issues_to_create/issue-1-jsonschema-tests.md`
- **Affected:** 8 methods with 0% coverage
- **Impact:** pkg/config: 46% → 55-60%
- **Difficulty:** Easy
- **Labels:** `good first issue`, `test coverage`, `enhancement`

### Issue 2: NFPM Conversion Methods (pkg/config)
- **File:** `/tmp/issues_to_create/issue-2-nfpm-tests.md`
- **Affected:** 2 methods with 0% coverage
- **Impact:** pkg/config: 46% → 48%
- **Difficulty:** Easy
- **Labels:** `good first issue`, `test coverage`, `enhancement`

### Issue 3: Archive Copy Methods
- **File:** `/tmp/issues_to_create/issue-3-archive-copy-tests.md`
- **Affected:** 2 methods with 0% coverage (zip.Copy, targz.Copy)
- **Impact:** pkg/archive/zip: 54.8% → 75%, pkg/archive/targz: 46.2% → 70%
- **Difficulty:** Easy-Medium
- **Labels:** `good first issue`, `test coverage`, `enhancement`
- **Note:** There's already a TODO comment in the code asking for this!

### Issue 4: Poetry Target Methods
- **File:** `/tmp/issues_to_create/issue-4-poetry-target-tests.md`
- **Affected:** 2 methods with 0% coverage
- **Impact:** internal/builders/poetry: 39.6% → 50%
- **Difficulty:** Very Easy
- **Labels:** `good first issue`, `test coverage`, `enhancement`

### Issue 5: Env Utility Methods (pkg/context)
- **File:** `/tmp/issues_to_create/issue-5-env-methods-tests.md`
- **Affected:** 2 methods with 0% coverage (Env.Copy, Env.Strings)
- **Impact:** pkg/context: 57.9% → 80%
- **Difficulty:** Very Easy
- **Labels:** `good first issue`, `test coverage`, `enhancement`

### Issue 6: CheckSCM Method (pkg/config)
- **File:** `/tmp/issues_to_create/issue-6-checkscm-test.md`
- **Affected:** 1 method with 0% coverage
- **Impact:** pkg/config: 46% → 48%
- **Difficulty:** Easy
- **Labels:** `good first issue`, `test coverage`, `enhancement`

## How to Create These Issues

Each issue file in `/tmp/issues_to_create/` contains:
1. **Title** - Copy this as the issue title
2. **Labels** - Apply these labels to the issue
3. **Description** - Complete markdown description with:
   - Overview of the problem
   - List of affected methods
   - Current and target coverage percentages
   - Step-by-step implementation guide
   - Example test code
   - Files to modify
   - Difficulty rating

## Priority Recommendation

If you want to prioritize based on impact and ease:

1. **Highest Impact, Very Easy:** Issue 5 (Env methods) - Big coverage jump for minimal effort
2. **Good Impact, Easy:** Issue 1 (JSONSchema) - 8 methods, all similar pattern
3. **Moderate Impact, Very Easy:** Issue 4 (Poetry Target) - Can be done in 5 minutes
4. **Moderate Impact, Easy-Medium:** Issue 3 (Archive Copy) - Addresses existing TODO
5. **Small Impact, Easy:** Issue 2 (NFPM) - Simple conversion tests
6. **Small Impact, Easy:** Issue 6 (CheckSCM) - Validation tests

## Notes

- All issues are marked as `good first issue` because they're well-scoped and don't require deep codebase knowledge
- Each issue includes example test code to help contributors get started quickly
- Tests should follow existing patterns in the repository (using testify/require)
- These improvements are "low-hanging fruit" - easy wins for test coverage
