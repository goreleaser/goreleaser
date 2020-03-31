package httpmock

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
)

const regexpPrefix = "=~"

// NoResponderFound is returned when no responders are found for a given HTTP method and URL.
var NoResponderFound = errors.New("no responder found") // nolint: golint

type routeKey struct {
	Method string
	URL    string
}

var noResponder routeKey

func (r routeKey) String() string {
	if r == noResponder {
		return "NO_RESPONDER"
	}
	return r.Method + " " + r.URL
}

// ConnectionFailure is a responder that returns a connection failure.  This is the default
// responder, and is called when no other matching responder is found.
func ConnectionFailure(*http.Request) (*http.Response, error) {
	return nil, NoResponderFound
}

// NewMockTransport creates a new *MockTransport with no responders.
func NewMockTransport() *MockTransport {
	return &MockTransport{
		responders:    make(map[routeKey]Responder),
		callCountInfo: make(map[routeKey]int),
	}
}

type regexpResponder struct {
	origRx    string
	method    string
	rx        *regexp.Regexp
	responder Responder
}

// MockTransport implements http.RoundTripper, which fulfills single http requests issued by
// an http.Client.  This implementation doesn't actually make the call, instead deferring to
// the registered list of responders.
type MockTransport struct {
	mu               sync.RWMutex
	responders       map[routeKey]Responder
	regexpResponders []regexpResponder
	noResponder      Responder
	callCountInfo    map[routeKey]int
	totalCallCount   int
}

// RoundTrip receives HTTP requests and routes them to the appropriate responder.  It is required to
// implement the http.RoundTripper interface.  You will not interact with this directly, instead
// the *http.Client you are using will call it for you.
func (m *MockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	url := req.URL.String()

	method := req.Method
	if method == "" {
		// http.Request.Method is documented to default to GET:
		method = http.MethodGet
	}

	var (
		responder  Responder
		respKey    routeKey
		submatches []string
	)
	key := routeKey{
		Method: method,
	}
	for _, getResponder := range []func(routeKey) (Responder, routeKey, []string){
		m.responderForKey,       // Exact match
		m.regexpResponderForKey, // Regexp match
	} {
		// try and get a responder that matches the method and URL with
		// query params untouched: http://z.tld/path?q...
		key.URL = url
		responder, respKey, submatches = getResponder(key)
		if responder != nil {
			break
		}

		// if we weren't able to find a responder, try with the URL *and*
		// sorted query params
		query := sortedQuery(req.URL.Query())
		if query != "" {
			// Replace unsorted query params by sorted ones:
			//   http://z.tld/path?sorted_q...
			key.URL = strings.Replace(url, req.URL.RawQuery, query, 1)
			responder, respKey, submatches = getResponder(key)
			if responder != nil {
				break
			}
		}

		// if we weren't able to find a responder, try without any query params
		strippedURL := *req.URL
		strippedURL.RawQuery = ""
		strippedURL.Fragment = ""

		// go1.6 does not handle URL.ForceQuery, so in case it is set in go>1.6,
		// remove the "?" manually if present.
		surl := strings.TrimSuffix(strippedURL.String(), "?")

		hasQueryString := url != surl

		// if the URL contains a querystring then we strip off the
		// querystring and try again: http://z.tld/path
		if hasQueryString {
			key.URL = surl
			responder, respKey, submatches = getResponder(key)
			if responder != nil {
				break
			}
		}

		// if we weren't able to find a responder for the full URL, try with
		// the path part only
		pathAlone := req.URL.Path

		// First with unsorted querystring: /path?q...
		if hasQueryString {
			key.URL = pathAlone + strings.TrimPrefix(url, surl) // concat after-path part
			responder, respKey, submatches = getResponder(key)
			if responder != nil {
				break
			}

			// Then with sorted querystring: /path?sorted_q...
			key.URL = pathAlone + "?" + sortedQuery(req.URL.Query())
			if req.URL.Fragment != "" {
				key.URL += "#" + req.URL.Fragment
			}
			responder, respKey, submatches = getResponder(key)
			if responder != nil {
				break
			}
		}

		// Then using path alone: /path
		key.URL = pathAlone
		responder, respKey, submatches = getResponder(key)
		if responder != nil {
			break
		}
	}

	m.mu.Lock()
	// if we found a responder, call it
	if responder != nil {
		m.callCountInfo[key]++
		if key != respKey {
			m.callCountInfo[respKey]++
		}
		m.totalCallCount++
	} else {
		// we didn't find a responder, so fire the 'no responder' responder
		if m.noResponder != nil {
			m.callCountInfo[noResponder]++
			m.totalCallCount++
			responder = m.noResponder
		}
	}
	m.mu.Unlock()

	if responder == nil {
		return ConnectionFailure(req)
	}
	return runCancelable(responder, setSubmatches(req, submatches))
}

func runCancelable(responder Responder, req *http.Request) (*http.Response, error) {
	ctx := req.Context()
	if req.Cancel == nil && ctx.Done() == nil { // nolint: staticcheck
		resp, err := responder(req)
		return resp, checkStackTracer(req, err)
	}

	// Set up a goroutine that translates a close(req.Cancel) into a
	// "request canceled" error, and another one that runs the
	// responder. Then race them: first to the result channel wins.

	type result struct {
		response *http.Response
		err      error
	}
	resultch := make(chan result, 1)
	done := make(chan struct{}, 1)

	go func() {
		select {
		case <-req.Cancel: // nolint: staticcheck
			resultch <- result{
				response: nil,
				err:      errors.New("request canceled"),
			}
		case <-ctx.Done():
			resultch <- result{
				response: nil,
				err:      ctx.Err(),
			}
		case <-done:
		}
	}()

	go func() {
		defer func() {
			if err := recover(); err != nil {
				resultch <- result{
					response: nil,
					err:      fmt.Errorf("panic in responder: got %q", err),
				}
			}
		}()

		response, err := responder(req)
		resultch <- result{
			response: response,
			err:      err,
		}
	}()

	r := <-resultch

	// if a cancel() issued from context.WithCancel() or a
	// close(req.Cancel) are never coming, we'll need to unblock the
	// first goroutine.
	done <- struct{}{}

	return r.response, checkStackTracer(req, r.err)
}

type stackTracer struct {
	customFn func(...interface{})
	err      error
}

func (n stackTracer) Error() string {
	if n.err == nil {
		return ""
	}
	return n.err.Error()
}

// checkStackTracer checks for specific error returned by
// NewNotFoundResponder function or Debug Responder method.
func checkStackTracer(req *http.Request, err error) error {
	if nf, ok := err.(stackTracer); ok {
		if nf.customFn != nil {
			pc := make([]uintptr, 128)
			npc := runtime.Callers(2, pc)
			pc = pc[:npc]

			var mesg bytes.Buffer
			var netHTTPBegin, netHTTPEnd bool

			// Start recording at first net/http call if any...
			for {
				frames := runtime.CallersFrames(pc)

				var lastFn string
				for {
					frame, more := frames.Next()

					if !netHTTPEnd {
						if netHTTPBegin {
							netHTTPEnd = !strings.HasPrefix(frame.Function, "net/http.")
						} else {
							netHTTPBegin = strings.HasPrefix(frame.Function, "net/http.")
						}
					}

					if netHTTPEnd {
						if lastFn != "" {
							if mesg.Len() == 0 {
								if nf.err != nil {
									mesg.WriteString(nf.err.Error())
								} else {
									fmt.Fprintf(&mesg, "%s %s", req.Method, req.URL)
								}
								mesg.WriteString("\nCalled from ")
							} else {
								mesg.WriteString("\n  ")
							}
							fmt.Fprintf(&mesg, "%s()\n    at %s:%d", lastFn, frame.File, frame.Line)
						}
					}
					lastFn = frame.Function

					if !more {
						break
					}
				}

				// At least one net/http frame found
				if mesg.Len() > 0 {
					break
				}
				netHTTPEnd = true // retry without looking at net/http frames
			}

			nf.customFn(mesg.String())
		}
		err = nf.err
	}
	return err
}

// responderForKey returns a responder for a given key.
func (m *MockTransport) responderForKey(key routeKey) (Responder, routeKey, []string) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.responders[key], key, nil
}

// responderForKeyUsingRegexp returns the first responder matching a given key using regexps.
func (m *MockTransport) regexpResponderForKey(key routeKey) (Responder, routeKey, []string) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, regInfo := range m.regexpResponders {
		if regInfo.method == key.Method {
			if sm := regInfo.rx.FindStringSubmatch(key.URL); sm != nil {
				if len(sm) == 1 {
					sm = nil
				} else {
					sm = sm[1:]
				}
				return regInfo.responder, routeKey{
					Method: key.Method,
					URL:    regInfo.origRx,
				}, sm
			}
		}
	}
	return nil, key, nil
}

func isRegexpURL(url string) bool {
	return strings.HasPrefix(url, regexpPrefix)
}

// RegisterResponder adds a new responder, associated with a given
// HTTP method and URL (or path).
//
// When a request comes in that matches, the responder will be called
// and the response returned to the client.
//
// If url contains query parameters, their order matters as well as
// their content. All following URLs are here considered as different:
//   http://z.tld?a=1&b=1
//   http://z.tld?b=1&a=1
//   http://z.tld?a&b
//   http://z.tld?a=&b=
//
// If url begins with "=~", the following chars are considered as a
// regular expression. If this regexp can not be compiled, it panics.
// Note that the "=~" prefix remains in statistics returned by
// GetCallCountInfo(). As 2 regexps can match the same URL, the regexp
// responders are tested in the order they are registered. Registering
// an already existing regexp responder (same method & same regexp
// string) replaces its responder but does not change its position.
//
// See RegisterRegexpResponder() to directly pass a *regexp.Regexp.
//
// Example:
// 		func TestFetchArticles(t *testing.T) {
// 			httpmock.Activate()
// 			defer httpmock.DeactivateAndReset()
//
// 			httpmock.RegisterResponder("GET", "http://example.com/",
// 				httpmock.NewStringResponder(200, "hello world"))
//
// 			httpmock.RegisterResponder("GET", "/path/only",
// 				httpmock.NewStringResponder("any host hello world", 200))
//
// 			httpmock.RegisterResponder("GET", `=~^/item/id/\d+\z`,
// 				httpmock.NewStringResponder("any item get", 200))
//
// 			// requests to http://example.com/ will now return "hello world" and
// 			// requests to any host with path /path/only will return "any host hello world"
// 			// requests to any host with path matching ^/item/id/\d+\z regular expression will return "any item get"
// 		}
func (m *MockTransport) RegisterResponder(method, url string, responder Responder) {
	if isRegexpURL(url) {
		m.registerRegexpResponder(regexpResponder{
			origRx:    url,
			method:    method,
			rx:        regexp.MustCompile(url[2:]),
			responder: responder,
		})
		return
	}

	key := routeKey{
		Method: method,
		URL:    url,
	}

	m.mu.Lock()
	m.responders[key] = responder
	m.callCountInfo[key] = 0
	m.mu.Unlock()
}

func (m *MockTransport) registerRegexpResponder(regexpResponder regexpResponder) {
	m.mu.Lock()
	defer m.mu.Unlock()

found:
	for {
		for i, rr := range m.regexpResponders {
			if rr.method == regexpResponder.method && rr.origRx == regexpResponder.origRx {
				m.regexpResponders[i] = regexpResponder
				break found
			}
		}
		m.regexpResponders = append(m.regexpResponders, regexpResponder)
		break // nolint: staticcheck
	}

	m.callCountInfo[routeKey{
		Method: regexpResponder.method,
		URL:    regexpResponder.origRx,
	}] = 0
}

// RegisterRegexpResponder adds a new responder, associated with a given
// HTTP method and URL (or path) regular expression.
//
// When a request comes in that matches, the responder will be called
// and the response returned to the client.
//
// As 2 regexps can match the same URL, the regexp responders are
// tested in the order they are registered. Registering an already
// existing regexp responder (same method & same regexp string)
// replaces its responder but does not change its position.
//
// A "=~" prefix is added to the stringified regexp in the statistics
// returned by GetCallCountInfo().
//
// See RegisterResponder function and the "=~" prefix in its url
// parameter to avoid compiling the regexp by yourself.
func (m *MockTransport) RegisterRegexpResponder(method string, urlRegexp *regexp.Regexp, responder Responder) {
	m.registerRegexpResponder(regexpResponder{
		origRx:    regexpPrefix + urlRegexp.String(),
		method:    method,
		rx:        urlRegexp,
		responder: responder,
	})
}

// RegisterResponderWithQuery is same as RegisterResponder, but it
// doesn't depend on query items order.
//
// If query is non-nil, its type can be:
//   url.Values
//   map[string]string
//   string, a query string like "a=12&a=13&b=z&c" (see net/url.ParseQuery function)
//
// If the query type is not recognized or the string cannot be parsed
// using net/url.ParseQuery, a panic() occurs.
//
// Unlike RegisterResponder, path cannot be prefixed by "=~" to say it
// is a regexp. If it is, a panic occurs.
func (m *MockTransport) RegisterResponderWithQuery(method, path string, query interface{}, responder Responder) {
	if isRegexpURL(path) {
		panic(`path begins with "=~", RegisterResponder should be used instead of RegisterResponderWithQuery`)
	}

	var mapQuery url.Values
	switch q := query.(type) {
	case url.Values:
		mapQuery = q

	case map[string]string:
		mapQuery = make(url.Values, len(q))
		for key, e := range q {
			mapQuery[key] = []string{e}
		}

	case string:
		var err error
		mapQuery, err = url.ParseQuery(q)
		if err != nil {
			panic("RegisterResponderWithQuery bad query string: " + err.Error())
		}

	default:
		if query != nil {
			panic(fmt.Sprintf("RegisterResponderWithQuery bad query type %T. Only url.Values, map[string]string and string are allowed", query))
		}
	}

	if queryString := sortedQuery(mapQuery); queryString != "" {
		path += "?" + queryString
	}
	m.RegisterResponder(method, path, responder)
}

func sortedQuery(m url.Values) string {
	if len(m) == 0 {
		return ""
	}

	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b bytes.Buffer
	var values []string // nolint: prealloc

	for _, k := range keys {
		// Do not alter the passed url.Values
		values = append(values, m[k]...)
		sort.Strings(values)

		k = url.QueryEscape(k)

		for _, v := range values {
			if b.Len() > 0 {
				b.WriteByte('&')
			}
			fmt.Fprintf(&b, "%v=%v", k, url.QueryEscape(v))
		}

		values = values[:0]
	}

	return b.String()
}

// RegisterNoResponder is used to register a responder that will be called if no other responder is
// found.  The default is ConnectionFailure.
func (m *MockTransport) RegisterNoResponder(responder Responder) {
	m.mu.Lock()
	m.noResponder = responder
	m.mu.Unlock()
}

// Reset removes all registered responders (including the no
// responder) from the MockTransport. It zeroes call counters too.
func (m *MockTransport) Reset() {
	m.mu.Lock()
	m.responders = make(map[routeKey]Responder)
	m.regexpResponders = nil
	m.noResponder = nil
	m.callCountInfo = make(map[routeKey]int)
	m.totalCallCount = 0
	m.mu.Unlock()
}

// ZeroCallCounters zeroes call counters without touching registered responders.
func (m *MockTransport) ZeroCallCounters() {
	m.mu.Lock()
	for k := range m.callCountInfo {
		m.callCountInfo[k] = 0
	}
	m.totalCallCount = 0
	m.mu.Unlock()
}

// GetCallCountInfo gets the info on all the calls httpmock has caught
// since it was activated or reset. The info is returned as a map of
// the calling keys with the number of calls made to them as their
// value. The key is the method, a space, and the url all concatenated
// together.
//
// As a special case, regexp responders generate 2 entries for each
// call. One for the call caught and the other for the rule that
// matched. For example:
//   RegisterResponder("GET", `=~z\.com\z`, NewStringResponder(200, "body"))
//   http.Get("http://z.com")
//
// will generate the following result:
//   map[string]int{
//     `GET http://z.com`:  1,
//     `GET =~z\.com\z`: 1,
//   }
func (m *MockTransport) GetCallCountInfo() map[string]int {
	res := make(map[string]int, len(m.callCountInfo))
	m.mu.RLock()
	for k, v := range m.callCountInfo {
		res[k.String()] = v
	}
	m.mu.RUnlock()
	return res
}

// GetTotalCallCount returns the totalCallCount.
func (m *MockTransport) GetTotalCallCount() int {
	m.mu.RLock()
	count := m.totalCallCount
	m.mu.RUnlock()
	return count
}

// DefaultTransport is the default mock transport used by Activate, Deactivate, Reset,
// DeactivateAndReset, RegisterResponder, and RegisterNoResponder.
var DefaultTransport = NewMockTransport()

// InitialTransport is a cache of the original transport used so we can put it back
// when Deactivate is called.
var InitialTransport = http.DefaultTransport

// Used to handle custom http clients (i.e clients other than http.DefaultClient)
var oldClients = map[*http.Client]http.RoundTripper{}

// Activate starts the mock environment.  This should be called before your tests run.  Under the
// hood this replaces the Transport on the http.DefaultClient with DefaultTransport.
//
// To enable mocks for a test, simply activate at the beginning of a test:
// 		func TestFetchArticles(t *testing.T) {
// 			httpmock.Activate()
// 			// all http requests will now be intercepted
// 		}
//
// If you want all of your tests in a package to be mocked, just call Activate from init():
// 		func init() {
// 			httpmock.Activate()
// 		}
func Activate() {
	if Disabled() {
		return
	}

	// make sure that if Activate is called multiple times it doesn't overwrite the InitialTransport
	// with a mock transport.
	if http.DefaultTransport != DefaultTransport {
		InitialTransport = http.DefaultTransport
	}

	http.DefaultTransport = DefaultTransport
}

// ActivateNonDefault starts the mock environment with a non-default http.Client.
// This emulates the Activate function, but allows for custom clients that do not use
// http.DefaultTransport
//
// To enable mocks for a test using a custom client, activate at the beginning of a test:
// 		client := &http.Client{Transport: &http.Transport{TLSHandshakeTimeout: 60 * time.Second}}
// 		httpmock.ActivateNonDefault(client)
func ActivateNonDefault(client *http.Client) {
	if Disabled() {
		return
	}

	// save the custom client & it's RoundTripper
	if _, ok := oldClients[client]; !ok {
		oldClients[client] = client.Transport
	}
	client.Transport = DefaultTransport
}

// GetCallCountInfo gets the info on all the calls httpmock has caught
// since it was activated or reset. The info is returned as a map of
// the calling keys with the number of calls made to them as their
// value. The key is the method, a space, and the url all concatenated
// together.
//
// As a special case, regexp responders generate 2 entries for each
// call. One for the call caught and the other for the rule that
// matched. For example:
//   RegisterResponder("GET", `=~z\.com\z`, NewStringResponder(200, "body"))
//   http.Get("http://z.com")
//
// will generate the following result:
//   map[string]int{
//     `GET http://z.com`:  1,
//     `GET =~z\.com\z`: 1,
//   }
func GetCallCountInfo() map[string]int {
	return DefaultTransport.GetCallCountInfo()
}

// GetTotalCallCount gets the total number of calls httpmock has taken since it was activated or
// reset.
func GetTotalCallCount() int {
	return DefaultTransport.GetTotalCallCount()
}

// Deactivate shuts down the mock environment.  Any HTTP calls made after this will use a live
// transport.
//
// Usually you'll call it in a defer right after activating the mock environment:
// 		func TestFetchArticles(t *testing.T) {
// 			httpmock.Activate()
// 			defer httpmock.Deactivate()
//
// 			// when this test ends, the mock environment will close
// 		}
func Deactivate() {
	if Disabled() {
		return
	}
	http.DefaultTransport = InitialTransport

	// reset the custom clients to use their original RoundTripper
	for oldClient, oldTransport := range oldClients {
		oldClient.Transport = oldTransport
		delete(oldClients, oldClient)
	}
}

// Reset will remove any registered mocks and return the mock
// environment to it's initial state. It zeroes call counters too.
func Reset() {
	DefaultTransport.Reset()
}

// ZeroCallCounters zeroes call counters without touching registered responders.
func ZeroCallCounters() {
	DefaultTransport.ZeroCallCounters()
}

// DeactivateAndReset is just a convenience method for calling Deactivate() and then Reset().
//
// Happy deferring!
func DeactivateAndReset() {
	Deactivate()
	Reset()
}

// RegisterResponder adds a new responder, associated with a given
// HTTP method and URL (or path).
//
// When a request comes in that matches, the responder will be called
// and the response returned to the client.
//
// If url contains query parameters, their order matters as well as
// their content. All following URLs are here considered as different:
//   http://z.tld?a=1&b=1
//   http://z.tld?b=1&a=1
//   http://z.tld?a&b
//   http://z.tld?a=&b=
//
// If url begins with "=~", the following chars are considered as a
// regular expression. If this regexp can not be compiled, it panics.
// Note that the "=~" prefix remains in statistics returned by
// GetCallCountInfo(). As 2 regexps can match the same URL, the regexp
// responders are tested in the order they are registered. Registering
// an already existing regexp responder (same method & same regexp
// string) replaces its responder but does not change its position.
//
// See RegisterRegexpResponder() to directly pass a *regexp.Regexp.
//
// Example:
// 		func TestFetchArticles(t *testing.T) {
// 			httpmock.Activate()
// 			defer httpmock.DeactivateAndReset()
//
// 			httpmock.RegisterResponder("GET", "http://example.com/",
// 				httpmock.NewStringResponder(200, "hello world"))
//
// 			httpmock.RegisterResponder("GET", "/path/only",
// 				httpmock.NewStringResponder("any host hello world", 200))
//
// 			httpmock.RegisterResponder("GET", `=~^/item/id/\d+\z`,
// 				httpmock.NewStringResponder("any item get", 200))
//
// 			// requests to http://example.com/ will now return "hello world" and
// 			// requests to any host with path /path/only will return "any host hello world"
// 			// requests to any host with path matching ^/item/id/\d+\z regular expression will return "any item get"
// 		}
func RegisterResponder(method, url string, responder Responder) {
	DefaultTransport.RegisterResponder(method, url, responder)
}

// RegisterRegexpResponder adds a new responder, associated with a given
// HTTP method and URL (or path) regular expression.
//
// When a request comes in that matches, the responder will be called
// and the response returned to the client.
//
// As 2 regexps can match the same URL, the regexp responders are
// tested in the order they are registered. Registering an already
// existing regexp responder (same method & same regexp string)
// replaces its responder but does not change its position.
//
// A "=~" prefix is added to the stringified regexp in the statistics
// returned by GetCallCountInfo().
//
// See RegisterResponder function and the "=~" prefix in its url
// parameter to avoid compiling the regexp by yourself.
func RegisterRegexpResponder(method string, urlRegexp *regexp.Regexp, responder Responder) {
	DefaultTransport.RegisterRegexpResponder(method, urlRegexp, responder)
}

// RegisterResponderWithQuery it is same as RegisterResponder, but
// doesn't depends on query items order.
//
// query type can be:
//   url.Values
//   map[string]string
//   string, a query string like "a=12&a=13&b=z&c" (see net/url.ParseQuery function)
//
// If the query type is not recognized or the string cannot be parsed
// using net/url.ParseQuery, a panic() occurs.
//
// Example using a net/url.Values:
// 		func TestFetchArticles(t *testing.T) {
// 			httpmock.Activate()
// 			defer httpmock.DeactivateAndReset()
//
// 			expectedQuery := net.Values{
// 				"a": []string{"3", "1", "8"},
//				"b": []string{"4", "2"},
//			}
// 			httpmock.RegisterResponderWithQueryValues("GET", "http://example.com/", expectedQuery,
// 				httpmock.NewStringResponder("hello world", 200))
//
//			// requests to http://example.com?a=1&a=3&a=8&b=2&b=4
//			//      and to http://example.com?b=4&a=2&b=2&a=8&a=1
//			// will now return 'hello world'
// 		}
//
// or using a map[string]string:
// 		func TestFetchArticles(t *testing.T) {
// 			httpmock.Activate()
// 			defer httpmock.DeactivateAndReset()
//
// 			expectedQuery := map[string]string{
//				"a": "1",
//				"b": "2"
//			}
// 			httpmock.RegisterResponderWithQuery("GET", "http://example.com/", expectedQuery,
// 				httpmock.NewStringResponder("hello world", 200))
//
//			// requests to http://example.com?a=1&b=2 and http://example.com?b=2&a=1 will now return 'hello world'
// 		}
//
// or using a query string:
// 		func TestFetchArticles(t *testing.T) {
// 			httpmock.Activate()
// 			defer httpmock.DeactivateAndReset()
//
// 			expectedQuery := "a=3&b=4&b=2&a=1&a=8"
// 			httpmock.RegisterResponderWithQueryValues("GET", "http://example.com/", expectedQuery,
// 				httpmock.NewStringResponder("hello world", 200))
//
//			// requests to http://example.com?a=1&a=3&a=8&b=2&b=4
//			//      and to http://example.com?b=4&a=2&b=2&a=8&a=1
//			// will now return 'hello world'
// 		}
func RegisterResponderWithQuery(method, path string, query interface{}, responder Responder) {
	DefaultTransport.RegisterResponderWithQuery(method, path, query, responder)
}

// RegisterNoResponder adds a mock that will be called whenever a request for an unregistered URL
// is received.  The default behavior is to return a connection error.
//
// In some cases you may not want all URLs to be mocked, in which case you can do this:
// 		func TestFetchArticles(t *testing.T) {
// 			httpmock.Activate()
// 			defer httpmock.DeactivateAndReset()
//			httpmock.RegisterNoResponder(httpmock.InitialTransport.RoundTrip)
//
// 			// any requests that don't have a registered URL will be fetched normally
// 		}
func RegisterNoResponder(responder Responder) {
	DefaultTransport.RegisterNoResponder(responder)
}

type submatchesKeyType struct{}

var submatchesKey submatchesKeyType

func setSubmatches(req *http.Request, submatches []string) *http.Request {
	if len(submatches) > 0 {
		return req.WithContext(context.WithValue(req.Context(), submatchesKey, submatches))
	}
	return req
}

// ErrSubmatchNotFound is the error returned by GetSubmatch* functions
// when the given submatch index cannot be found.
var ErrSubmatchNotFound = errors.New("submatch not found")

// GetSubmatch has to be used in Responders installed by
// RegisterRegexpResponder or RegisterResponder + "=~" URL prefix. It
// allows to retrieve the n-th submatch of the matching regexp, as a
// string. Example:
// 	RegisterResponder("GET", `=~^/item/name/([^/]+)\z`,
// 		func(req *http.Request) (*http.Response, error) {
// 			name, err := GetSubmatch(req, 1) // 1=first regexp submatch
// 			if err != nil {
// 				return nil, err
// 			}
// 			return NewJsonResponse(200, map[string]interface{}{
// 				"id":   123,
// 				"name": name,
// 			})
// 		})
//
// It panics if n < 1. See MustGetSubmatch to avoid testing the
// returned error.
func GetSubmatch(req *http.Request, n int) (string, error) {
	if n <= 0 {
		panic(fmt.Sprintf("getting submatches starts at 1, not %d", n))
	}
	n--

	submatches, ok := req.Context().Value(submatchesKey).([]string)
	if !ok || n >= len(submatches) {
		return "", ErrSubmatchNotFound
	}
	return submatches[n], nil
}

// GetSubmatchAsInt has to be used in Responders installed by
// RegisterRegexpResponder or RegisterResponder + "=~" URL prefix. It
// allows to retrieve the n-th submatch of the matching regexp, as an
// int64. Example:
// 	RegisterResponder("GET", `=~^/item/id/(\d+)\z`,
// 		func(req *http.Request) (*http.Response, error) {
// 			id, err := GetSubmatchAsInt(req, 1) // 1=first regexp submatch
// 			if err != nil {
// 				return nil, err
// 			}
// 			return NewJsonResponse(200, map[string]interface{}{
// 				"id":   id,
// 				"name": "The beautiful name",
// 			})
// 		})
//
// It panics if n < 1. See MustGetSubmatchAsInt to avoid testing the
// returned error.
func GetSubmatchAsInt(req *http.Request, n int) (int64, error) {
	sm, err := GetSubmatch(req, n)
	if err != nil {
		return 0, err
	}
	return strconv.ParseInt(sm, 10, 64)
}

// GetSubmatchAsUint has to be used in Responders installed by
// RegisterRegexpResponder or RegisterResponder + "=~" URL prefix. It
// allows to retrieve the n-th submatch of the matching regexp, as a
// uint64. Example:
// 	RegisterResponder("GET", `=~^/item/id/(\d+)\z`,
// 		func(req *http.Request) (*http.Response, error) {
// 			id, err := GetSubmatchAsUint(req, 1) // 1=first regexp submatch
// 			if err != nil {
// 				return nil, err
// 			}
// 			return NewJsonResponse(200, map[string]interface{}{
// 				"id":   id,
// 				"name": "The beautiful name",
// 			})
// 		})
//
// It panics if n < 1. See MustGetSubmatchAsUint to avoid testing the
// returned error.
func GetSubmatchAsUint(req *http.Request, n int) (uint64, error) {
	sm, err := GetSubmatch(req, n)
	if err != nil {
		return 0, err
	}
	return strconv.ParseUint(sm, 10, 64)
}

// GetSubmatchAsFloat has to be used in Responders installed by
// RegisterRegexpResponder or RegisterResponder + "=~" URL prefix. It
// allows to retrieve the n-th submatch of the matching regexp, as a
// float64. Example:
// 	RegisterResponder("PATCH", `=~^/item/id/\d+\?height=(\d+(?:\.\d*)?)\z`,
// 		func(req *http.Request) (*http.Response, error) {
// 			height, err := GetSubmatchAsFloat(req, 1) // 1=first regexp submatch
// 			if err != nil {
// 				return nil, err
// 			}
// 			return NewJsonResponse(200, map[string]interface{}{
// 				"id":     id,
// 				"name":   "The beautiful name",
// 				"height": height,
// 			})
// 		})
//
// It panics if n < 1. See MustGetSubmatchAsFloat to avoid testing the
// returned error.
func GetSubmatchAsFloat(req *http.Request, n int) (float64, error) {
	sm, err := GetSubmatch(req, n)
	if err != nil {
		return 0, err
	}
	return strconv.ParseFloat(sm, 64)
}

// MustGetSubmatch works as GetSubmatch except that it panics in case
// of error (submatch not found). It has to be used in Responders
// installed by RegisterRegexpResponder or RegisterResponder + "=~"
// URL prefix. It allows to retrieve the n-th submatch of the matching
// regexp, as a string. Example:
// 	RegisterResponder("GET", `=~^/item/name/([^/]+)\z`,
// 		func(req *http.Request) (*http.Response, error) {
// 			name := MustGetSubmatch(req, 1) // 1=first regexp submatch
// 			return NewJsonResponse(200, map[string]interface{}{
// 				"id":   123,
// 				"name": name,
// 			})
// 		})
//
// It panics if n < 1.
func MustGetSubmatch(req *http.Request, n int) string {
	s, err := GetSubmatch(req, n)
	if err != nil {
		panic("GetSubmatch failed: " + err.Error())
	}
	return s
}

// MustGetSubmatchAsInt works as GetSubmatchAsInt except that it
// panics in case of error (submatch not found or invalid int64
// format). It has to be used in Responders installed by
// RegisterRegexpResponder or RegisterResponder + "=~" URL prefix. It
// allows to retrieve the n-th submatch of the matching regexp, as an
// int64. Example:
// 	RegisterResponder("GET", `=~^/item/id/(\d+)\z`,
// 		func(req *http.Request) (*http.Response, error) {
// 			id := MustGetSubmatchAsInt(req, 1) // 1=first regexp submatch
// 			return NewJsonResponse(200, map[string]interface{}{
// 				"id":   id,
// 				"name": "The beautiful name",
// 			})
// 		})
//
// It panics if n < 1.
func MustGetSubmatchAsInt(req *http.Request, n int) int64 {
	i, err := GetSubmatchAsInt(req, n)
	if err != nil {
		panic("GetSubmatchAsInt failed: " + err.Error())
	}
	return i
}

// MustGetSubmatchAsUint works as GetSubmatchAsUint except that it
// panics in case of error (submatch not found or invalid uint64
// format). It has to be used in Responders installed by
// RegisterRegexpResponder or RegisterResponder + "=~" URL prefix. It
// allows to retrieve the n-th submatch of the matching regexp, as a
// uint64. Example:
// 	RegisterResponder("GET", `=~^/item/id/(\d+)\z`,
// 		func(req *http.Request) (*http.Response, error) {
// 			id, err := MustGetSubmatchAsUint(req, 1) // 1=first regexp submatch
// 			return NewJsonResponse(200, map[string]interface{}{
// 				"id":   id,
// 				"name": "The beautiful name",
// 			})
// 		})
//
// It panics if n < 1.
func MustGetSubmatchAsUint(req *http.Request, n int) uint64 {
	u, err := GetSubmatchAsUint(req, n)
	if err != nil {
		panic("GetSubmatchAsUint failed: " + err.Error())
	}
	return u
}

// MustGetSubmatchAsFloat works as GetSubmatchAsFloat except that it
// panics in case of error (submatch not found or invalid float64
// format). It has to be used in Responders installed by
// RegisterRegexpResponder or RegisterResponder + "=~" URL prefix. It
// allows to retrieve the n-th submatch of the matching regexp, as a
// float64. Example:
// 	RegisterResponder("PATCH", `=~^/item/id/\d+\?height=(\d+(?:\.\d*)?)\z`,
// 		func(req *http.Request) (*http.Response, error) {
// 			height := MustGetSubmatchAsFloat(req, 1) // 1=first regexp submatch
// 			return NewJsonResponse(200, map[string]interface{}{
// 				"id":     id,
// 				"name":   "The beautiful name",
// 				"height": height,
// 			})
// 		})
//
// It panics if n < 1.
func MustGetSubmatchAsFloat(req *http.Request, n int) float64 {
	f, err := GetSubmatchAsFloat(req, n)
	if err != nil {
		panic("GetSubmatchAsFloat failed: " + err.Error())
	}
	return f
}
