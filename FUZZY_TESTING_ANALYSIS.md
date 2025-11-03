# Fuzzy Testing Analysis for GoReleaser

This document identifies the top ten places in the GoReleaser codebase that would most benefit from fuzzy testing.

## Background

Fuzzy testing (or fuzz testing) is a software testing technique that involves providing invalid, unexpected, or random data as inputs to a program. GoReleaser already has some fuzzy tests:
- `internal/artifact/artifact_fuzz_test.go` - Checksum calculation fuzzing
- `internal/tmpl/fuzz_test.go` - Template engine fuzzing

However, there are many critical areas that handle external input and would benefit from fuzzy testing to improve robustness and security.

## Top 10 Fuzzy Testing Candidates

### 1. YAML Configuration Parsing (`internal/yaml`)

**Priority: Critical**

**Rationale:**
- The YAML parser handles GoReleaser's primary configuration files
- Malformed YAML could cause crashes or unexpected behavior
- Both strict and non-strict unmarshaling modes need testing
- Critical for security and stability

**Suggested Tests:**
- Fuzz `UnmarshalStrict()` with various malformed YAML inputs
- Fuzz `Unmarshal()` with edge cases
- Test deeply nested structures
- Test very large files
- Test unicode and special characters

**Files:** `internal/yaml/yaml.go`

---

### 2. Package.json Parsing (`internal/packagejson`)

**Priority: High**

**Rationale:**
- Parses Node.js/Bun package.json files
- JSON parsing vulnerabilities could lead to security issues
- Detection of Bun projects depends on parsing accuracy

**Suggested Tests:**
- Fuzz `Open()` function with malformed JSON
- Test deeply nested dependencies
- Test invalid UTF-8 sequences
- Test extremely large package.json files

**Files:** `internal/packagejson/packagejson.go`

---

### 3. Pyproject.toml Parsing (`internal/pyproject`)

**Priority: High**

**Rationale:**
- Parses Python project configuration files
- TOML parsing errors could break Python project builds
- Poetry detection relies on correct parsing

**Suggested Tests:**
- Fuzz `Open()` function with malformed TOML
- Test invalid project names
- Test edge cases in version strings
- Test malformed tool.poetry sections

**Files:** `internal/pyproject/pyproject.go`

---

### 4. Cargo.toml Parsing (`internal/cargo`)

**Priority: High**

**Rationale:**
- Parses Rust Cargo manifest files
- Workspace member parsing is critical for monorepos
- TOML parsing vulnerabilities could affect builds

**Suggested Tests:**
- Fuzz `Open()` function with malformed TOML
- Test invalid package names
- Test malformed workspace configurations
- Test very large member lists

**Files:** `internal/cargo/cargo.go`

---

### 5. Template Engine (`internal/tmpl`)

**Priority: High**

**Rationale:**
- Already has some fuzzy tests, but coverage could be expanded
- Templates are used throughout GoReleaser for string interpolation
- Template injection vulnerabilities are a real security concern
- Handles user-provided input in many contexts

**Suggested Tests:**
- Expand existing fuzzing to cover more template functions
- Test with artifacts that have unusual field values
- Fuzz template functions with edge cases
- Test nested template expressions
- Test templates with control structures (if/range/etc.)

**Files:** `internal/tmpl/tmpl.go`, `internal/tmpl/fuzz_test.go`

---

### 6. Config Loading (`pkg/config`)

**Priority: Critical**

**Rationale:**
- The main entry point for loading GoReleaser configurations
- Handles version detection and validation
- Very large and complex configuration structure
- Errors here affect all downstream operations

**Suggested Tests:**
- Fuzz `LoadReader()` with various malformed configs
- Test version validation with edge cases
- Test with extremely large configuration files
- Test with mixed valid/invalid sections
- Test Pro config detection

**Files:** `pkg/config/load.go`, `pkg/config/config.go`

---

### 7. Archive Files Processing (`internal/archivefiles`)

**Priority: High**

**Rationale:**
- Handles file path processing and glob patterns
- Path traversal vulnerabilities are possible
- Glob pattern parsing is complex
- File destination calculation is critical

**Suggested Tests:**
- Fuzz `Eval()` with malicious glob patterns
- Test path traversal attempts (../, etc.)
- Test very long file paths
- Test unicode in paths
- Test symbolic links and special characters

**Files:** `internal/archivefiles/archivefiles.go`

---

### 8. Shell Command Construction (`internal/shell`)

**Priority: Critical**

**Rationale:**
- Executes shell commands with user-provided input
- Command injection is a critical security risk
- Handles environment variables and working directories
- Errors could lead to arbitrary command execution

**Suggested Tests:**
- Fuzz command array construction
- Test with shell metacharacters
- Test with very long commands
- Test with unusual environment variable values
- Test directory path edge cases

**Files:** `internal/shell/shell.go`

---

### 9. Changelog Parsing (`internal/pipe/changelog`)

**Priority: Medium**

**Rationale:**
- Uses regular expressions extensively
- Parses git commit messages and generates changelogs
- ReDoS (Regular Expression Denial of Service) vulnerabilities possible
- Handles untrusted input from commit messages

**Suggested Tests:**
- Fuzz regex patterns with pathological inputs
- Test with very long commit messages
- Test with unusual unicode characters
- Test with malformed git log output
- Test filter and sort configurations

**Files:** `internal/pipe/changelog/changelog.go`

---

### 10. HTTP Client Utilities (`internal/http`)

**Priority: High**

**Rationale:**
- Handles HTTP requests and responses
- URL parsing and validation is complex
- File downloads and uploads need security testing
- Certificate handling is security-critical

**Suggested Tests:**
- Fuzz URL parsing
- Test with malformed HTTP responses
- Test with very large payloads
- Test redirect handling
- Test timeout scenarios
- Test TLS certificate edge cases

**Files:** `internal/http/http.go`

---

## Implementation Priority

1. **Critical Priority** (Immediate):
   - YAML Configuration Parsing (security/stability)
   - Shell Command Construction (security)
   - Config Loading (stability)

2. **High Priority** (Next sprint):
   - Package.json Parsing
   - Pyproject.toml Parsing
   - Cargo.toml Parsing
   - Template Engine expansion
   - Archive Files Processing
   - HTTP Client Utilities

3. **Medium Priority** (Future):
   - Changelog Parsing

## Testing Strategy

For each fuzzy test implementation:

1. **Create dedicated fuzz test files** (e.g., `*_fuzz_test.go`)
2. **Use Go's native fuzzing** (`testing.F`)
3. **Add seed corpus** with known edge cases
4. **Set appropriate timeouts** for CI/CD integration
5. **Document expected behaviors** (crash vs. error handling)
6. **Integrate with existing test suite**

## CI/CD Integration

- Add fuzzing to CI pipeline using the existing `scripts/fuzz.sh`
- Run fuzzing for limited time on PRs (e.g., 30 seconds)
- Run extended fuzzing nightly (e.g., 10 minutes per test)
- Store and track discovered issues

## References

- Existing fuzzy tests: `internal/artifact/artifact_fuzz_test.go`, `internal/tmpl/fuzz_test.go`
- Fuzzing script: `scripts/fuzz.sh`
- Go fuzzing documentation: https://go.dev/doc/fuzz/
