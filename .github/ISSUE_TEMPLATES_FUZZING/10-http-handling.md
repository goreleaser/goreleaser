---
title: "Add fuzzy testing for HTTP client utilities"
labels: ["enhancement", "testing", "security", "http"]
---

## Description

Add comprehensive fuzzy testing for the HTTP client utilities module (`internal/http`) to improve security and robustness when handling HTTP operations.

## Rationale

The HTTP module:
- Handles HTTP requests and responses
- URL parsing and validation is complex
- Processes file downloads and uploads
- Certificate handling is security-critical
- Redirect handling can be exploited
- Handles external network input

## Proposed Implementation

Create `internal/http/http_fuzz_test.go` with the following tests:

### 1. `FuzzURLParsing`
Test URL parsing and validation:
- Malformed URLs
- URLs with special characters
- Very long URLs (> 2000 chars)
- Unicode in URLs
- URL encoding edge cases
- Scheme-less URLs
- URLs with authentication info
- IPv6 addresses

### 2. `FuzzHTTPHeaders`
Test HTTP header processing:
- Very long header values
- Headers with newlines (header injection)
- Invalid header names
- Unicode in headers
- Duplicate headers
- Headers with null bytes

### 3. `FuzzResponseParsing`
Test HTTP response handling:
- Malformed response bodies
- Very large responses
- Invalid content-length
- Chunked encoding edge cases
- Mixed encodings
- Truncated responses

### 4. `FuzzRedirectHandling`
Test redirect following:
- Redirect loops
- Very long redirect chains
- Redirects to different schemes
- Open redirect vulnerabilities
- Redirects with malformed Location headers

### 5. `FuzzTLSConfiguration`
Test TLS/certificate handling:
- Invalid certificates
- Expired certificates
- Self-signed certificates
- Certificate chain validation
- Hostname verification edge cases

## Example Test Structure

```go
func FuzzURLParsing(f *testing.F) {
    // Add seed corpus
    f.Add("https://example.com/path")
    f.Add("http://user:pass@host:8080/path?query=value#fragment")
    f.Add("https://[::1]:8080/")
    f.Add("//example.com/path") // Scheme-less
    f.Add("https://example.com/" + strings.Repeat("a", 10000))
    
    f.Fuzz(func(t *testing.T, urlStr string) {
        // Test URL parsing without panicking
        u, err := url.Parse(urlStr)
        if err != nil {
            return // Invalid URLs are expected
        }
        
        // If URL parses successfully, validate it doesn't lead to issues
        _ = u.String()
        _ = u.Host
        _ = u.Path
    })
}

func FuzzHTTPHeaders(f *testing.F) {
    f.Add("Content-Type", "application/json")
    f.Add("X-Custom", "value\r\nInjected: header")
    f.Add("Long-Value", strings.Repeat("a", 100000))
    f.Add("Unicode", "日本語")
    
    f.Fuzz(func(t *testing.T, name, value string) {
        req := httptest.NewRequest("GET", "http://example.com", nil)
        
        // Test header setting - should handle or reject invalid headers
        req.Header.Set(name, value)
        
        // Verify no header injection occurred
        headers := req.Header
        for key := range headers {
            require.NotContains(t, key, "\r")
            require.NotContains(t, key, "\n")
        }
    })
}

func FuzzResponseParsing(f *testing.F) {
    f.Add([]byte("HTTP/1.1 200 OK\r\nContent-Length: 5\r\n\r\nHello"))
    f.Add([]byte("HTTP/1.1 200 OK\r\nTransfer-Encoding: chunked\r\n\r\n5\r\nHello\r\n0\r\n\r\n"))
    f.Add([]byte("HTTP/1.1 200 OK\r\n\r\n" + strings.Repeat("a", 1000000)))
    
    f.Fuzz(func(t *testing.T, responseData []byte) {
        // Create a test server that returns the fuzzed response
        // or parse response directly if possible
        
        // Should handle malformed responses gracefully
        // without panicking or hanging
    })
}
```

## Security Considerations

Test for these specific vulnerabilities:
- **SSRF** (Server-Side Request Forgery) - malicious URLs
- **Header Injection** - newlines in headers
- **Open Redirects** - unvalidated redirect targets
- **DoS** - very large responses/headers
- **TLS downgrade** - insecure configurations

## Acceptance Criteria

- [ ] Create `internal/http/http_fuzz_test.go`
- [ ] Implement at least 5 fuzz test functions
- [ ] Add security-focused seed corpus
- [ ] Test URL parsing edge cases
- [ ] Test header injection prevention
- [ ] Test redirect security
- [ ] Test TLS validation
- [ ] Test timeout and size limits
- [ ] Integrate with existing test suite

## Related Files

- `internal/http/http.go`
- `internal/http/http_test.go`
- `internal/http/testdata/`

## Priority

**High** - HTTP handling involves external input and has multiple security implications.

## Additional Resources

- [OWASP SSRF Prevention](https://cheatsheetseries.owasp.org/cheatsheets/Server_Side_Request_Forgery_Prevention_Cheat_Sheet.html)
- [HTTP Header Injection](https://owasp.org/www-community/attacks/HTTP_Response_Splitting)
