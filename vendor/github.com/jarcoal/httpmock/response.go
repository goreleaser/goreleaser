package httpmock

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

// Responder is a callback that receives and http request and returns
// a mocked response.
type Responder func(*http.Request) (*http.Response, error)

func (r Responder) times(name string, n int, fn ...func(...interface{})) Responder {
	count := 0
	return func(req *http.Request) (*http.Response, error) {
		count++
		if count > n {
			err := stackTracer{
				err: fmt.Errorf("Responder not found for %s %s (coz %s and already called %d times)", req.Method, req.URL, name, count),
			}
			if len(fn) > 0 {
				err.customFn = fn[0]
			}
			return nil, err
		}
		return r(req)
	}
}

// Times returns a Responder callable n times before returning an
// error. If the Responder is called more than n times and fn is
// passed and non-nil, it acts as the fn parameter of
// NewNotFoundResponder, allowing to dump the stack trace to localize
// the origin of the call.
func (r Responder) Times(n int, fn ...func(...interface{})) Responder {
	return r.times("Times", n, fn...)
}

// Once returns a new Responder callable once before returning an
// error. If the Responder is called 2 or more times and fn is passed
// and non-nil, it acts as the fn parameter of NewNotFoundResponder,
// allowing to dump the stack trace to localize the origin of the
// call.
func (r Responder) Once(fn ...func(...interface{})) Responder {
	return r.times("Once", 1, fn...)
}

// Trace returns a new Responder that allow to easily trace the calls
// of the original Responder using fn. It can be used in conjunction
// with the testing package as in the example below with the help of
// (*testing.T).Log method:
//   import "testing"
//   ...
//   func TestMyApp(t *testing.T) {
//   	...
//   	httpmock.RegisterResponder("GET", "/foo/bar",
//    	httpmock.NewStringResponder(200, "{}").Trace(t.Log),
//   	)
func (r Responder) Trace(fn func(...interface{})) Responder {
	return func(req *http.Request) (*http.Response, error) {
		resp, err := r(req)
		return resp, stackTracer{
			customFn: fn,
			err:      err,
		}
	}
}

// ResponderFromResponse wraps an *http.Response in a Responder
func ResponderFromResponse(resp *http.Response) Responder {
	return func(req *http.Request) (*http.Response, error) {
		res := new(http.Response)
		*res = *resp
		res.Request = req
		return res, nil
	}
}

// NewErrorResponder creates a Responder that returns an empty request and the
// given error. This can be used to e.g. imitate more deep http errors for the
// client.
func NewErrorResponder(err error) Responder {
	return func(req *http.Request) (*http.Response, error) {
		return nil, err
	}
}

// NewNotFoundResponder creates a Responder typically used in
// conjunction with RegisterNoResponder() function and testing
// package, to be proactive when a Responder is not found. fn is
// called with a unique string parameter containing the name of the
// missing route and the stack trace to localize the origin of the
// call. If fn returns (= if it does not panic), the responder returns
// an error of the form: "Responder not found for GET http://foo.bar/path".
// Note that fn can be nil.
//
// It is useful when writing tests to ensure that all routes have been
// mocked.
//
// Example of use:
//   import "testing"
//   ...
//   func TestMyApp(t *testing.T) {
//   	...
//   	// Calls testing.Fatal with the name of Responder-less route and
//   	// the stack trace of the call.
//   	httpmock.RegisterNoResponder(httpmock.NewNotFoundResponder(t.Fatal))
//
// Will abort the current test and print something like:
//   transport_test.go:735: Called from net/http.Get()
//         at /go/src/github.com/jarcoal/httpmock/transport_test.go:714
//       github.com/jarcoal/httpmock.TestCheckStackTracer()
//         at /go/src/testing/testing.go:865
//       testing.tRunner()
//         at /go/src/runtime/asm_amd64.s:1337
func NewNotFoundResponder(fn func(...interface{})) Responder {
	return func(req *http.Request) (*http.Response, error) {
		return nil, stackTracer{
			customFn: fn,
			err:      fmt.Errorf("Responder not found for %s %s", req.Method, req.URL),
		}
	}
}

// NewStringResponse creates an *http.Response with a body based on the given string.  Also accepts
// an http status code.
func NewStringResponse(status int, body string) *http.Response {
	return &http.Response{
		Status:        strconv.Itoa(status),
		StatusCode:    status,
		Body:          NewRespBodyFromString(body),
		Header:        http.Header{},
		ContentLength: -1,
	}
}

// NewStringResponder creates a Responder from a given body (as a string) and status code.
func NewStringResponder(status int, body string) Responder {
	return ResponderFromResponse(NewStringResponse(status, body))
}

// NewBytesResponse creates an *http.Response with a body based on the given bytes.  Also accepts
// an http status code.
func NewBytesResponse(status int, body []byte) *http.Response {
	return &http.Response{
		Status:        strconv.Itoa(status),
		StatusCode:    status,
		Body:          NewRespBodyFromBytes(body),
		Header:        http.Header{},
		ContentLength: -1,
	}
}

// NewBytesResponder creates a Responder from a given body (as a byte slice) and status code.
func NewBytesResponder(status int, body []byte) Responder {
	return ResponderFromResponse(NewBytesResponse(status, body))
}

// NewJsonResponse creates an *http.Response with a body that is a json encoded representation of
// the given interface{}.  Also accepts an http status code.
func NewJsonResponse(status int, body interface{}) (*http.Response, error) { // nolint: golint
	encoded, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	response := NewBytesResponse(status, encoded)
	response.Header.Set("Content-Type", "application/json")
	return response, nil
}

// NewJsonResponder creates a Responder from a given body (as an interface{} that is encoded to
// json) and status code.
func NewJsonResponder(status int, body interface{}) (Responder, error) { // nolint: golint
	resp, err := NewJsonResponse(status, body)
	if err != nil {
		return nil, err
	}
	return ResponderFromResponse(resp), nil
}

// NewJsonResponderOrPanic is like NewJsonResponder but panics in case of error.
//
// It simplifies the call of RegisterResponder, avoiding the use of a
// temporary variable and an error check, and so can be used as
// NewStringResponder or NewBytesResponder in such context:
//   RegisterResponder(
//     "GET",
//     "/test/path",
//     NewJSONResponderOrPanic(200, &MyBody),
//   )
func NewJsonResponderOrPanic(status int, body interface{}) Responder { // nolint: golint
	responder, err := NewJsonResponder(status, body)
	if err != nil {
		panic(err)
	}
	return responder
}

// NewXmlResponse creates an *http.Response with a body that is an xml encoded representation
// of the given interface{}.  Also accepts an http status code.
func NewXmlResponse(status int, body interface{}) (*http.Response, error) { // nolint: golint
	encoded, err := xml.Marshal(body)
	if err != nil {
		return nil, err
	}
	response := NewBytesResponse(status, encoded)
	response.Header.Set("Content-Type", "application/xml")
	return response, nil
}

// NewXmlResponder creates a Responder from a given body (as an interface{} that is encoded to xml)
// and status code.
func NewXmlResponder(status int, body interface{}) (Responder, error) { // nolint: golint
	resp, err := NewXmlResponse(status, body)
	if err != nil {
		return nil, err
	}
	return ResponderFromResponse(resp), nil
}

// NewXmlResponderOrPanic is like NewXmlResponder but panics in case of error.
//
// It simplifies the call of RegisterResponder, avoiding the use of a
// temporary variable and an error check, and so can be used as
// NewStringResponder or NewBytesResponder in such context:
//   RegisterResponder(
//     "GET",
//     "/test/path",
//     NewXmlResponderOrPanic(200, &MyBody),
//   )
func NewXmlResponderOrPanic(status int, body interface{}) Responder { // nolint: golint
	responder, err := NewXmlResponder(status, body)
	if err != nil {
		panic(err)
	}
	return responder
}

// NewRespBodyFromString creates an io.ReadCloser from a string that is suitable for use as an
// http response body.
func NewRespBodyFromString(body string) io.ReadCloser {
	return &dummyReadCloser{strings.NewReader(body)}
}

// NewRespBodyFromBytes creates an io.ReadCloser from a byte slice that is suitable for use as an
// http response body.
func NewRespBodyFromBytes(body []byte) io.ReadCloser {
	return &dummyReadCloser{bytes.NewReader(body)}
}

type dummyReadCloser struct {
	body io.ReadSeeker
}

func (d *dummyReadCloser) Read(p []byte) (n int, err error) {
	n, err = d.body.Read(p)
	if err == io.EOF {
		d.body.Seek(0, 0) // nolint: errcheck
	}
	return n, err
}

func (d *dummyReadCloser) Close() error {
	d.body.Seek(0, 0) // nolint: errcheck
	return nil
}
