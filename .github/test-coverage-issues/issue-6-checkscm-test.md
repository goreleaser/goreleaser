**Title:** Add test for Repo.CheckSCM() method in pkg/config

**Labels:** good first issue, test coverage, enhancement

**Description:**

The `pkg/config/config.go` file contains a `CheckSCM()` method on the `Repo` type that has 0% test coverage. This method validates that a repository URL is from a supported SCM provider (GitHub, GitLab, or Gitea).

### Affected Method
- `Repo.CheckSCM()` (line 65 in config.go)

### Current Coverage
**46.1%** → target **~48%**

### What needs to be done
1. Add test cases in `pkg/config/config_test.go`
2. Test that valid SCM URLs are accepted (github.com, gitlab.com, gitea.io)
3. Test that invalid SCM URLs return an error
4. Test edge cases (empty URL, malformed URL, etc.)

### Example Test Structure
```go
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
```

### Files to modify
- Modify: `pkg/config/config_test.go`

### Difficulty
**Easy** - Simple validation method that needs basic test cases.
