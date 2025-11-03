#!/bin/bash

# This script creates GitHub issues for test coverage improvements.
# Run this script after authenticating with: gh auth login

set -e

REPO="goreleaser/goreleaser"
ISSUE_DIR="/tmp/issues_to_create"

echo "Creating GitHub issues for test coverage improvements..."
echo "Repository: $REPO"
echo ""

# Check if gh is authenticated
if ! gh auth status &>/dev/null; then
    echo "Error: Not authenticated with GitHub."
    echo "Please run: gh auth login"
    exit 1
fi

# Issue 1: JSONSchema methods
echo "Creating Issue 1: JSONSchema methods..."
gh issue create \
    --repo "$REPO" \
    --title "Add tests for JSONSchema methods in pkg/config" \
    --label "good first issue,test coverage,enhancement" \
    --body "$(cat <<'EOF'
The \`pkg/config/jsonschema.go\` file contains 8 JSONSchema methods that currently have 0% test coverage. These methods are straightforward and would be easy to test.

### Affected Methods
- \`Hook.JSONSchema()\` (line 5)
- \`File.JSONSchema()\` (line 21)
- \`Hooks.JSONSchema()\` (line 37)
- \`FlagArray.JSONSchema()\` (line 53)
- \`StringArray.JSONSchema()\` (line 66)
- \`NixDependency.JSONSchema()\` (line 79)
- \`PullRequestBase.JSONSchema()\` (line 95)
- \`HomebrewDependency.JSONSchema()\` (line 111)

### Current Coverage
**0%** - None of these methods are currently tested.

### What needs to be done
1. Create a new test file \`pkg/config/jsonschema_test.go\`
2. Add test cases for each JSONSchema method
3. Verify that each method returns a valid JSONSchema with the expected structure
4. Test both the string and object/array forms where applicable (OneOf schemas)

### Expected Impact
This will improve coverage in \`pkg/config\` from ~46% to approximately 55-60%.

### Example Test Structure
\`\`\`go
func TestHookJSONSchema(t *testing.T) {
    schema := Hook{}.JSONSchema()
    require.NotNil(t, schema)
    require.NotNil(t, schema.OneOf)
    require.Len(t, schema.OneOf, 2)
    // Verify one option is string type
    // Verify one option is the expanded struct
}
\`\`\`

### Files to modify
- Create: \`pkg/config/jsonschema_test.go\`

### Difficulty
**Easy** - These are straightforward schema generation methods that just need basic validation tests.
EOF
)"

echo "✓ Issue 1 created"

# Issue 2: NFPM conversion methods
echo "Creating Issue 2: NFPM conversion methods..."
gh issue create \
    --repo "$REPO" \
    --title "Add tests for NFPM conversion methods in pkg/config/nfpm.go" \
    --label "good first issue,test coverage,enhancement" \
    --body "$(cat <<'EOF'
The \`pkg/config/nfpm.go\` file contains 2 conversion methods that currently have 0% test coverage. These methods convert GoReleaser config types to NFPM library types.

### Affected Methods
- \`NFPMIPKAlternative.ToNFP()\` (line 5)
- \`NFPMIPK.ToNFPAlts()\` (line 13)

### Current Coverage
**0%** - Neither method is currently tested.

### What needs to be done
1. Create a new test file \`pkg/config/nfpm_test.go\`
2. Add test cases to verify the conversion logic
3. Ensure all fields are properly mapped
4. Test with both empty and populated structures

### Expected Impact
This will further improve coverage in \`pkg/config\` from ~46% to ~48%.

### Example Test Structure
\`\`\`go
func TestNFPMIPKAlternativeToNFP(t *testing.T) {
    alt := NFPMIPKAlternative{
        Priority: 100,
        Target:   "/usr/bin/app",
        LinkName: "app",
    }
    
    result := alt.ToNFP()
    require.Equal(t, 100, result.Priority)
    require.Equal(t, "/usr/bin/app", result.Target)
    require.Equal(t, "app", result.LinkName)
}

func TestNFPMIPKToNFPAlts(t *testing.T) {
    ipk := NFPMIPK{
        Alternatives: []NFPMIPKAlternative{
            {Priority: 100, Target: "/usr/bin/app1", LinkName: "app1"},
            {Priority: 200, Target: "/usr/bin/app2", LinkName: "app2"},
        },
    }
    
    result := ipk.ToNFPAlts()
    require.Len(t, result, 2)
    require.Equal(t, 100, result[0].Priority)
    require.Equal(t, "/usr/bin/app1", result[0].Target)
}
\`\`\`

### Files to modify
- Create: \`pkg/config/nfpm_test.go\`

### Difficulty
**Easy** - Simple conversion methods that just need to verify field mapping.
EOF
)"

echo "✓ Issue 2 created"

# Issue 3: Archive Copy methods
echo "Creating Issue 3: Archive Copy methods..."
gh issue create \
    --repo "$REPO" \
    --title "Add tests for Copy methods in pkg/archive/zip and pkg/archive/targz" \
    --label "good first issue,test coverage,enhancement" \
    --body "$(cat <<'EOF'
Both \`pkg/archive/zip/zip.go\` and \`pkg/archive/targz/targz.go\` have \`Copy\` methods with 0% test coverage. There's even a TODO comment in \`zip_test.go\` line 169 that says "TODO: add copying test".

### Affected Methods
- \`zip.Copy()\` in \`pkg/archive/zip/zip.go\` (line 35)
- \`targz.Copy()\` in \`pkg/archive/targz/targz.go\` (line 30)

### Current Coverage
- \`pkg/archive/zip\`: **54.8%** → target **~75%**
- \`pkg/archive/targz\`: **46.2%** → target **~70%**

### What needs to be done
1. Add test for \`zip.Copy()\` in \`pkg/archive/zip/zip_test.go\`
2. Add test for \`targz.Copy()\` in a test file (create if needed)
3. Tests should:
   - Create a source archive with known contents
   - Copy it to a new archive
   - Verify the copied archive contains the same files
   - Verify file permissions and metadata are preserved

### Example Test Structure
\`\`\`go
func TestZipCopy(t *testing.T) {
    tmp := t.TempDir()
    
    // Create source zip with test files
    sourcePath := filepath.Join(tmp, "source.zip")
    sourceFile, err := os.Create(sourcePath)
    require.NoError(t, err)
    
    sourceArchive := New(sourceFile)
    require.NoError(t, sourceArchive.Add(config.File{
        Source:      "../testdata/foo.txt",
        Destination: "foo.txt",
    }))
    require.NoError(t, sourceArchive.Close())
    require.NoError(t, sourceFile.Close())
    
    // Open source for reading
    sourceFile, err = os.Open(sourcePath)
    require.NoError(t, err)
    defer sourceFile.Close()
    
    // Copy to new archive
    targetPath := filepath.Join(tmp, "target.zip")
    targetFile, err := os.Create(targetPath)
    require.NoError(t, err)
    defer targetFile.Close()
    
    copiedArchive, err := Copy(sourceFile, targetFile)
    require.NoError(t, err)
    require.NoError(t, copiedArchive.Close())
    require.NoError(t, targetFile.Close())
    
    // Verify target archive has same contents
    // ...
}
\`\`\`

### Files to modify
- Modify: \`pkg/archive/zip/zip_test.go\`
- Create or Modify: \`pkg/archive/targz/targz_test.go\`

### Difficulty
**Easy-Medium** - Requires creating archives and verifying their contents, but the pattern exists in other tests.
EOF
)"

echo "✓ Issue 3 created"

# Issue 4: Poetry Target methods
echo "Creating Issue 4: Poetry Target methods..."
gh issue create \
    --repo "$REPO" \
    --title "Add tests for Target methods in internal/builders/poetry" \
    --label "good first issue,test coverage,enhancement" \
    --body "$(cat <<'EOF'
The \`internal/builders/poetry/build.go\` file contains a \`Target\` type with two methods that have 0% test coverage. The same pattern also exists in \`internal/builders/uv\`.

### Affected Methods
- \`Target.Fields()\` (line 62 in build.go)
- \`Target.String()\` (line 70 in build.go)

### Current Coverage
- \`internal/builders/poetry\`: **39.6%** → target **~50%**
- \`internal/builders/uv\`: **39.6%** → target **~50%** (same pattern)

### What needs to be done
1. Add test cases in \`internal/builders/poetry/build_test.go\`
2. Test that \`Target.Fields()\` returns the expected map with "all" values
3. Test that \`Target.String()\` returns "none-any"
4. Optionally apply the same tests to \`internal/builders/uv/build_test.go\`

### Example Test Structure
\`\`\`go
func TestTargetFields(t *testing.T) {
    target := Target{}
    fields := target.Fields()
    require.Equal(t, map[string]string{
        tmpl.KeyOS:   "all",
        tmpl.KeyArch: "all",
    }, fields)
}

func TestTargetString(t *testing.T) {
    target := Target{}
    require.Equal(t, "none-any", target.String())
}
\`\`\`

### Files to modify
- Modify: \`internal/builders/poetry/build_test.go\`
- Optional: Also add to \`internal/builders/uv/build_test.go\` (same pattern)

### Difficulty
**Very Easy** - These are trivial getter methods that just need basic assertion tests.
EOF
)"

echo "✓ Issue 4 created"

# Issue 5: Env utility methods
echo "Creating Issue 5: Env utility methods..."
gh issue create \
    --repo "$REPO" \
    --title "Add tests for Env.Copy() and Env.Strings() methods in pkg/context" \
    --label "good first issue,test coverage,enhancement" \
    --body "$(cat <<'EOF'
The \`pkg/context/context.go\` file contains two utility methods on the \`Env\` type that currently have 0% test coverage.

### Affected Methods
- \`Env.Copy()\` (line 43)
- \`Env.Strings()\` (line 51)

### Current Coverage
**57.9%** → target **~80%**

### What needs to be done
1. Add test cases in \`pkg/context/context_test.go\`
2. Test that \`Env.Copy()\` creates an independent copy (not a reference)
3. Test that \`Env.Strings()\` returns properly formatted "KEY=VALUE" strings
4. Test edge cases (empty env, special characters in values, etc.)

### Example Test Structure
\`\`\`go
func TestEnvCopy(t *testing.T) {
    original := Env{"FOO": "bar", "BAZ": "qux"}
    copied := original.Copy()
    
    // Verify copy has same values
    require.Equal(t, original, copied)
    
    // Verify it's a separate copy (modifications don't affect original)
    copied["FOO"] = "modified"
    require.Equal(t, "bar", original["FOO"])
    require.Equal(t, "modified", copied["FOO"])
}

func TestEnvStrings(t *testing.T) {
    env := Env{"FOO": "bar", "BAZ": "qux"}
    strs := env.Strings()
    
    require.Len(t, strs, 2)
    require.Contains(t, strs, "FOO=bar")
    require.Contains(t, strs, "BAZ=qux")
}

func TestEnvStringsEmpty(t *testing.T) {
    env := Env{}
    strs := env.Strings()
    require.Empty(t, strs)
}
\`\`\`

### Files to modify
- Modify: \`pkg/context/context_test.go\`

### Difficulty
**Very Easy** - Simple utility methods with straightforward testing requirements.
EOF
)"

echo "✓ Issue 5 created"

# Issue 6: CheckSCM method
echo "Creating Issue 6: CheckSCM method..."
gh issue create \
    --repo "$REPO" \
    --title "Add test for Repo.CheckSCM() method in pkg/config" \
    --label "good first issue,test coverage,enhancement" \
    --body "$(cat <<'EOF'
The \`pkg/config/config.go\` file contains a \`CheckSCM()\` method on the \`Repo\` type that has 0% test coverage. This method validates that a repository URL is from a supported SCM provider (GitHub, GitLab, or Gitea).

### Affected Method
- \`Repo.CheckSCM()\` (line 65 in config.go)

### Current Coverage
**46.1%** → target **~48%**

### What needs to be done
1. Add test cases in \`pkg/config/config_test.go\`
2. Test that valid SCM URLs are accepted (github.com, gitlab.com, gitea.io)
3. Test that invalid SCM URLs return an error
4. Test edge cases (empty URL, malformed URL, etc.)

### Example Test Structure
\`\`\`go
func TestRepoCheckSCM(t *testing.T) {
    tests := []struct {
        name    string
        url     string
        wantErr bool
    }{
        {"valid github", "https://github.com/owner/repo", false},
        {"valid gitlab", "https://gitlab.com/owner/repo", false},
        {"valid gitea", "https://gitea.io/owner/repo", false},
        {"invalid scm", "https://example.com/owner/repo", true},
        {"empty url", "", true},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            repo := Repo{
                SCMToken: SCMToken{
                    URL: tt.url,
                },
            }
            err := repo.CheckSCM()
            if tt.wantErr {
                require.Error(t, err)
            } else {
                require.NoError(t, err)
            }
        })
    }
}
\`\`\`

### Files to modify
- Modify: \`pkg/config/config_test.go\`

### Difficulty
**Easy** - Simple validation method that needs basic test cases.
EOF
)"

echo "✓ Issue 6 created"

echo ""
echo "✅ All 6 issues created successfully!"
echo ""
echo "Summary:"
echo "- 17 total uncovered methods identified"
echo "- Expected coverage improvement: 80.7% → 82-83%"
echo "- All issues tagged as 'good first issue'"
