# Fuzzy Testing Enhancement Project

This directory contains a comprehensive analysis and tooling for adding fuzzy testing to GoReleaser's codebase.

## 📋 What's Included

### 📊 Analysis
- **[FUZZY_TESTING_ANALYSIS.md](FUZZY_TESTING_ANALYSIS.md)** - Detailed analysis of top 10 fuzzy testing candidates
- Identifies critical security and stability improvements
- Provides implementation recommendations
- Includes priority rankings

### 📝 GitHub Issue Templates
- **[.github/ISSUE_TEMPLATES_FUZZING/](.github/ISSUE_TEMPLATES_FUZZING/)** - 10 ready-to-use issue templates
- Each template includes:
  - Problem description and rationale
  - Proposed implementation with code examples
  - Acceptance criteria checklist
  - Related files and priority level

### 🔧 Automation Scripts

#### Python Script (Cross-platform)
```bash
pip install requests
export GITHUB_TOKEN="your_token"
python3 scripts/create-fuzzing-issues.py
```
**Features:**
- Works on any platform with Python 3
- Dry-run mode available
- Detailed error reporting
- **File**: `scripts/create-fuzzing-issues.py`

#### Bash Script (Linux/Mac)
```bash
gh auth login
./scripts/create-fuzzing-issues.sh
```
**Features:**
- Uses GitHub CLI (gh)
- Simple and fast
- Automatic authentication
- **File**: `scripts/create-fuzzing-issues.sh`

### 📖 Documentation
- **[HOW_TO_CREATE_ISSUES.md](HOW_TO_CREATE_ISSUES.md)** - Complete guide for creating issues
- **[MANUAL_ISSUE_CREATION.md](MANUAL_ISSUE_CREATION.md)** - Step-by-step manual guide
- **[.github/ISSUE_TEMPLATES_FUZZING/README.md](.github/ISSUE_TEMPLATES_FUZZING/README.md)** - Template documentation

## 🚀 Quick Start

### Automated (Recommended)

1. **Set up authentication**:
   ```bash
   # Create a token at https://github.com/settings/tokens with 'repo' scope
   export GITHUB_TOKEN="your_github_personal_access_token"
   ```

2. **Run the Python script**:
   ```bash
   pip install requests
   python3 scripts/create-fuzzing-issues.py
   ```

3. **Verify**:
   Visit https://github.com/goreleaser/goreleaser/issues

### Manual

If you prefer manual creation or if scripts don't work:

1. Open [MANUAL_ISSUE_CREATION.md](MANUAL_ISSUE_CREATION.md)
2. Follow the copy-paste instructions for each issue
3. Each issue takes about 2 minutes to create manually

## 📊 The 10 Fuzzy Testing Candidates

| Priority | Issue | Module | Security Impact |
|----------|-------|--------|-----------------|
| 🔴 Critical | Shell Command Construction | `internal/shell` | Command injection risk |
| 🔴 Critical | YAML Configuration Parsing | `internal/yaml` | Entry point for all configs |
| 🔴 Critical | Config Loading | `pkg/config` | Core functionality |
| 🟡 High | Package.json Parsing | `internal/packagejson` | Node/Bun support |
| 🟡 High | Pyproject.toml Parsing | `internal/pyproject` | Python support |
| 🟡 High | Cargo.toml Parsing | `internal/cargo` | Rust support |
| 🟡 High | Template Engine | `internal/tmpl` | Template injection risk |
| 🟡 High | Archive Files | `internal/archivefiles` | Path traversal risk |
| 🟡 High | HTTP Client | `internal/http` | Multiple security risks |
| 🟢 Medium | Changelog Parsing | `internal/pipe/changelog` | ReDoS risk |

## 🎯 Why Fuzzy Testing?

Fuzzy testing helps discover:
- **Security vulnerabilities** - Command injection, path traversal, etc.
- **Crash bugs** - Unexpected input that causes panics
- **Edge cases** - Unusual inputs that aren't covered by unit tests
- **Performance issues** - Inputs that cause excessive resource use (ReDoS, etc.)

GoReleaser already has some fuzzy tests:
- ✅ `internal/artifact/artifact_fuzz_test.go` - Checksum calculation
- ✅ `internal/tmpl/fuzz_test.go` - Template engine (basic coverage)

This project identifies 10 additional areas that would benefit from fuzzing.

## 📁 File Structure

```
.
├── FUZZY_TESTING_ANALYSIS.md          # Main analysis document
├── HOW_TO_CREATE_ISSUES.md            # Issue creation guide
├── MANUAL_ISSUE_CREATION.md           # Manual creation guide
├── README_FUZZING_PROJECT.md          # This file
│
├── .github/
│   └── ISSUE_TEMPLATES_FUZZING/       # Issue templates
│       ├── README.md
│       ├── 01-yaml-parsing.md
│       ├── 02-packagejson-parsing.md
│       ├── 03-pyproject-parsing.md
│       ├── 04-cargo-parsing.md
│       ├── 05-template-engine.md
│       ├── 06-config-loading.md
│       ├── 07-archive-files.md
│       ├── 08-shell-commands.md
│       ├── 09-changelog-parsing.md
│       └── 10-http-handling.md
│
└── scripts/
    ├── create-fuzzing-issues.py       # Python automation script
    ├── create-fuzzing-issues.sh       # Bash automation script
    └── fuzz.sh                        # Existing fuzz test runner
```

## 🔐 Security Considerations

Several of these fuzzy tests target security-critical areas:

- **Shell Command Construction** (Issue #8) - Prevents command injection
- **Archive Files Processing** (Issue #7) - Prevents path traversal
- **Template Engine** (Issue #5) - Prevents template injection
- **HTTP Client** (Issue #10) - Prevents SSRF, header injection
- **YAML/Config Parsing** (Issues #1, #6) - Prevents config-based attacks

## 📈 Expected Impact

Implementing these fuzzy tests will:
- ✅ **Improve security** - Catch injection and traversal vulnerabilities
- ✅ **Increase stability** - Find crash bugs before users do
- ✅ **Better error handling** - Discover edge cases in error paths
- ✅ **CI/CD integration** - Continuous fuzzing in pipeline
- ✅ **Documentation** - Fuzzy tests serve as edge case documentation

## 🤝 Contributing

To implement a fuzzy test:

1. Choose an issue from the priority list
2. Read the corresponding template in `.github/ISSUE_TEMPLATES_FUZZING/`
3. Follow the implementation guidelines
4. See existing examples:
   - `internal/artifact/artifact_fuzz_test.go`
   - `internal/tmpl/fuzz_test.go`
5. Submit a PR with your implementation

## 📚 Resources

- [Go Fuzzing Tutorial](https://go.dev/doc/tutorial/fuzz)
- [Go Fuzzing Documentation](https://go.dev/doc/fuzz/)
- [Google Fuzzing Best Practices](https://github.com/google/fuzzing/blob/master/docs/good-fuzz-target.md)
- [OWASP Testing Guide](https://owasp.org/www-project-web-security-testing-guide/)

## 🐛 Issues?

If you encounter problems with the scripts:
1. Check [HOW_TO_CREATE_ISSUES.md](HOW_TO_CREATE_ISSUES.md) troubleshooting section
2. Verify your GitHub token has `repo` scope
3. Try the manual approach in [MANUAL_ISSUE_CREATION.md](MANUAL_ISSUE_CREATION.md)
4. Open a discussion if problems persist

---

**Created**: 2025-11-03  
**Purpose**: Enhance GoReleaser's test coverage and security through fuzzy testing  
**Status**: Ready for issue creation and implementation
