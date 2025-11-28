---
title: "Add fuzzy testing for shell command construction"
labels: ["enhancement", "testing", "security", "critical"]
---

## Description

Add comprehensive fuzzy testing for the shell command construction module (`internal/shell`) to prevent command injection vulnerabilities and improve robustness.

## Rationale

The shell module:
- Executes shell commands with user-provided input
- Command injection is a **critical security risk**
- Handles environment variables and working directories
- Errors could lead to arbitrary command execution
- Currently no fuzzy tests exist for this security-critical module

## Security Considerations

This module is particularly sensitive because:
- Commands are executed using `exec.CommandContext`
- User input flows into command arrays
- Environment variables are set from user config
- Working directories can be controlled by users
- Improper handling could allow shell metacharacter injection

## Proposed Implementation

Create `internal/shell/shell_fuzz_test.go` with the following tests:

### 1. `FuzzRun`
Test command execution with various inputs:
- Shell metacharacters (`;`, `|`, `&`, `$`, `` ` ``, etc.)
- Command injection attempts
- Very long command strings
- Unicode in commands
- Null bytes and control characters
- Newlines and carriage returns

### 2. `FuzzCommandArray`
Test command array construction:
- Empty arrays
- Arrays with empty strings
- Arrays with special characters
- Very long argument lists
- Arguments with quotes and escapes

### 3. `FuzzEnvironmentVariables`
Test environment variable handling:
- Variables with special characters
- Very long variable values
- Variables with newlines
- Malicious variable names
- Export attempts

### 4. `FuzzWorkingDirectory`
Test directory handling:
- Path traversal attempts
- Non-existent directories
- Very long paths
- Special characters in paths
- Symbolic links

## Example Test Structure

```go
func FuzzRun(f *testing.F) {
    // Add seed corpus with known dangerous patterns
    f.Add([]string{"echo", "hello"}, []string{}, ".")
    f.Add([]string{"sh", "-c", "echo test; rm -rf /"}, []string{}, ".")
    f.Add([]string{"cmd"}, []string{"PATH=/tmp"}, "/tmp")
    f.Add([]string{"test", "$(malicious)"}, []string{}, ".")
    
    f.Fuzz(func(t *testing.T, cmdSlice []string, env []string, dir string) {
        if len(cmdSlice) == 0 {
            t.Skip() // Empty commands are explicitly handled
        }
        
        ctx := testctx.New()
        
        // For fuzzing, use a safe command wrapper or mock exec
        // to prevent actual execution of malicious commands
        
        // Test should verify:
        // 1. No panics
        // 2. Proper error handling
        // 3. No shell interpretation of metacharacters
        // 4. Arguments are properly separated
        
        err := Run(ctx, dir, cmdSlice, env, false)
        // Expected to fail for non-existent commands, but should not panic
        _ = err
    })
}
```

## Safety Measures for Fuzzing

Since this module executes commands, the fuzz tests should:
1. Use a restricted test environment
2. Mock command execution when possible
3. Use safe test commands (e.g., `/bin/true`, `echo`)
4. Validate command construction without execution
5. Run in isolated containers during CI

## Acceptance Criteria

- [ ] Create `internal/shell/shell_fuzz_test.go`
- [ ] Implement at least 4 fuzz test functions
- [ ] Add security-focused seed corpus with injection attempts
- [ ] Tests must not execute dangerous commands
- [ ] Verify command arrays are properly constructed
- [ ] Test environment variable isolation
- [ ] Document any discovered vulnerabilities
- [ ] Add safeguards to prevent actual malicious execution during fuzzing

## Related Files

- `internal/shell/shell.go`
- `internal/shell/shell_test.go`
- `internal/exec/exec.go`

## Priority

**Critical** - Command execution is the highest security risk in the codebase.

## Additional Notes

Consider using `os/exec.CommandContext` properties that prevent shell interpretation:
- Commands are executed directly, not through a shell
- Arguments are passed as separate strings
- No shell metacharacter interpretation by default

The fuzzing should verify these safety properties are maintained.
