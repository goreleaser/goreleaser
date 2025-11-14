---
title: "Add fuzzy testing for changelog parsing"
labels: ["enhancement", "testing", "changelog"]
---

## Description

Add fuzzy testing for the changelog parsing module (`internal/pipe/changelog`) to improve robustness and prevent ReDoS (Regular Expression Denial of Service) vulnerabilities.

## Rationale

The changelog module:
- Uses regular expressions extensively for parsing
- Processes git commit messages (untrusted input)
- ReDoS vulnerabilities are possible with pathological inputs
- Generates changelogs from git log output
- Complex filter and sort configurations

## Proposed Implementation

Create `internal/pipe/changelog/changelog_fuzz_test.go` with the following tests:

### 1. `FuzzRegexPatterns`
Test regular expression matching with pathological inputs:
- Inputs designed to cause catastrophic backtracking
- Very long commit messages
- Nested patterns
- Alternation explosion patterns
- Repeated groupings

### 2. `FuzzCommitMessageParsing`
Test commit message parsing:
- Very long messages (> 1MB)
- Unusual unicode characters
- Control characters and null bytes
- Malformed conventional commit formats
- Multi-line messages
- Messages with ANSI escape codes

### 3. `FuzzFilterConfiguration`
Test changelog filter settings:
- Invalid regex patterns in filters
- Very long filter lists
- Conflicting filters
- Edge cases in include/exclude logic

### 4. `FuzzSortConfiguration`
Test changelog sorting:
- Edge cases in commit ordering
- Malformed commit data
- Missing fields in commit info
- Duplicate commits

## Example Test Structure

```go
func FuzzRegexPatterns(f *testing.F) {
    // Add seed corpus with known ReDoS patterns
    f.Add("(a+)+b", "aaaaaaaaaaaaaaaaaaaaaaaaa")
    f.Add("(a|a)*", "aaaaaaaaaaaaaaaaaaaaaaaaa")
    f.Add("(a|ab)*", "ababababababababababababab")
    
    // Normal patterns
    f.Add("^feat:", "feat: add new feature")
    f.Add("^fix:", "fix: resolve bug")
    
    f.Fuzz(func(t *testing.T, pattern, input string) {
        // Test with timeout to catch ReDoS
        done := make(chan bool, 1)
        go func() {
            defer func() {
                if r := recover(); r != nil {
                    // Panic is acceptable for invalid regex
                }
                done <- true
            }()
            
            // Attempt to compile and match
            re, err := regexp.Compile(pattern)
            if err != nil {
                return // Invalid regex is acceptable
            }
            
            _ = re.MatchString(input)
        }()
        
        select {
        case <-done:
            // Completed successfully
        case <-time.After(100 * time.Millisecond):
            t.Errorf("Regex matching took too long (possible ReDoS): pattern=%q input=%q", pattern, input)
        }
    })
}

func FuzzCommitMessageParsing(f *testing.F) {
    f.Add("feat: add new feature")
    f.Add("fix: resolve issue #123")
    f.Add("docs: update readme\n\nLong description here")
    f.Add(strings.Repeat("a", 1000000)) // Very long message
    
    f.Fuzz(func(t *testing.T, message string) {
        ctx := testctx.NewWithCfg(config.Project{
            ProjectName: "test",
            Changelog: config.Changelog{
                Filters: config.Filters{
                    Exclude: []string{},
                },
            },
        })
        
        // Test parsing logic without panicking
        // This will depend on internal changelog parsing functions
        // Should handle any message gracefully
    })
}
```

## ReDoS Prevention

Key patterns to test for catastrophic backtracking:
- `(a+)+` - Nested quantifiers
- `(a|a)*` - Overlapping alternation
- `(a|ab)*` - Subset alternation
- `([a-zA-Z]+)*` - Repeated character classes
- `(.*)*` - Nested wildcards

## Acceptance Criteria

- [ ] Create `internal/pipe/changelog/changelog_fuzz_test.go`
- [ ] Implement at least 4 fuzz test functions
- [ ] Add seed corpus with ReDoS patterns
- [ ] All regex operations should timeout within 100ms
- [ ] Test with very long commit messages (> 1MB)
- [ ] Test unicode and special character handling
- [ ] Document any discovered performance issues
- [ ] Integrate with existing test suite

## Related Files

- `internal/pipe/changelog/changelog.go`
- `internal/pipe/changelog/changelog_test.go`
- `pkg/config/config.go` (Changelog configuration)

## Priority

**Medium** - Changelog parsing handles untrusted input but is less critical than config/command execution.

## Additional Resources

- [OWASP ReDoS Guide](https://owasp.org/www-community/attacks/Regular_expression_Denial_of_Service_-_ReDoS)
- [Go regexp performance](https://github.com/golang/go/wiki/RegexpBenchmarks)
