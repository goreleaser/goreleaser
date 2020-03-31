# httpmock [![Build Status](https://travis-ci.org/jarcoal/httpmock.png?branch=v1)](https://travis-ci.org/jarcoal/httpmock) [![Coverage Status](https://coveralls.io/repos/github/jarcoal/httpmock/badge.svg?branch=v1)](https://coveralls.io/github/jarcoal/httpmock?branch=v1) [![GoDoc](https://godoc.org/github.com/jarcoal/httpmock?status.svg)](https://godoc.org/github.com/jarcoal/httpmock) [![Version](https://img.shields.io/github/tag/jarcoal/httpmock.svg)](https://github.com/jarcoal/httpmock/releases) [![Mentioned in Awesome Go](https://awesome.re/mentioned-badge.svg)](https://github.com/avelino/awesome-go/#testing)

Easy mocking of http responses from external resources.

## Install

Currently supports Go 1.7 - 1.14.

`v1` branch has to be used instead of `master`.


### Using go modules (aka. `go mod`)

In your go files, simply use:
```go
import "github.com/jarcoal/httpmock"
```

Then next `go mod tidy` or `go test` invocation will automatically
populate your `go.mod` with the last httpmock release, now
[![Version](https://img.shields.io/github/tag/jarcoal/httpmock.svg)](https://github.com/jarcoal/httpmock/releases).

Note you can use `go mod vendor` to vendor your dependencies.


### Using `$GOPATH`

`v1` branch is configured as the default branch in github, so:
```
go get github.com/jarcoal/httpmock
```

automatically downloads the `v1` branch in `$GOPATH/src`. Then in your
go files use:
```go
import "github.com/jarcoal/httpmock"
```


### Vendoring, using [`govendor`](https://github.com/kardianos/govendor) for example

When vendoring is used, `v1` branch has to be specified. Two choices here:

- preferred way:
  ```
  govendor fetch github.com/jarcoal/httpmock@v1
  ```
  then in go files:
  ```go
  import "github.com/jarcoal/httpmock"
  ```
- old way (before `v1` was set as default branch), use gopkg to read from
  `v1` branch:
  ```
  govendor fetch gopkg.in/jarcoal/httpmock.v1
  ```
  then in go files:
  ```go
  import "gopkg.in/jarcoal/httpmock.v1"
  ```


## Usage

### Simple Example:
```go
func TestFetchArticles(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	// Exact URL match
	httpmock.RegisterResponder("GET", "https://api.mybiz.com/articles",
		httpmock.NewStringResponder(200, `[{"id": 1, "name": "My Great Article"}]`))

	// Regexp match (could use httpmock.RegisterRegexpResponder instead)
	httpmock.RegisterResponder("GET", `=~^https://api\.mybiz\.com/articles/id/\d+\z`,
		httpmock.NewStringResponder(200, `{"id": 1, "name": "My Great Article"}`))

	// do stuff that makes a request to articles
	...

	// get count info
	httpmock.GetTotalCallCount()

	// get the amount of calls for the registered responder
	info := httpmock.GetCallCountInfo()
	info["GET https://api.mybiz.com/articles"] // number of GET calls made to https://api.mybiz.com/articles
	info["GET https://api.mybiz.com/articles/id/12"] // number of GET calls made to https://api.mybiz.com/articles/id/12
	info[`GET =~^https://api\.mybiz\.com/articles/id/\d+\z`] // number of GET calls made to https://api.mybiz.com/articles/id/<any-number>
}
```

### Advanced Example:
```go
func TestFetchArticles(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	// our database of articles
	articles := make([]map[string]interface{}, 0)

	// mock to list out the articles
	httpmock.RegisterResponder("GET", "https://api.mybiz.com/articles",
		func(req *http.Request) (*http.Response, error) {
			resp, err := httpmock.NewJsonResponse(200, articles)
			if err != nil {
				return httpmock.NewStringResponse(500, ""), nil
			}
			return resp, nil
		},
	)

	// return an article related to the request with the help of regexp submatch (\d+)
	httpmock.RegisterResponder("GET", `=~^https://api\.mybiz\.com/articles/id/(\d+)\z`,
		func(req *http.Request) (*http.Response, error) {
			// Get ID from request
			id := httpmock.MustGetSubmatchAsUint(req, 1) // 1=first regexp submatch
			return httpmock.NewJsonResponse(200, map[string]interface{}{
				"id":   id,
				"name": "My Great Article",
			})
		},
	)

	// mock to add a new article
	httpmock.RegisterResponder("POST", "https://api.mybiz.com/articles",
		func(req *http.Request) (*http.Response, error) {
			article := make(map[string]interface{})
			if err := json.NewDecoder(req.Body).Decode(&article); err != nil {
				return httpmock.NewStringResponse(400, ""), nil
			}

			articles = append(articles, article)

			resp, err := httpmock.NewJsonResponse(200, article)
			if err != nil {
				return httpmock.NewStringResponse(500, ""), nil
			}
			return resp, nil
		},
	)

	// do stuff that adds and checks articles
}
```

### Algorithm

When `GET http://example.tld/some/path?b=12&a=foo&a=bar` request is
caught, all standard responders are checked against the following URL
or paths, the first match stops the search:

1. `http://example.tld/some/path?b=12&a=foo&a=bar` (original URL)
1. `http://example.tld/some/path?a=bar&a=foo&b=12` (sorted query params)
1. `http://example.tld/some/path` (without query params)
1. `/some/path?b=12&a=foo&a=bar` (original URL without scheme and host)
1. `/some/path?a=bar&a=foo&b=12` (same, but sorted query params)
1. `/some/path` (path only)

If no standard responder matched, the regexp responders are checked,
in the same order, the first match stops the search.


### [Ginkgo](https://onsi.github.io/ginkgo/) Example:
```go
// article_suite_test.go

import (
	// ...
	"github.com/jarcoal/httpmock"
)
// ...
var _ = BeforeSuite(func() {
	// block all HTTP requests
	httpmock.Activate()
})

var _ = BeforeEach(func() {
	// remove any mocks
	httpmock.Reset()
})

var _ = AfterSuite(func() {
	httpmock.DeactivateAndReset()
})


// article_test.go

import (
	// ...
	"github.com/jarcoal/httpmock"
)

var _ = Describe("Articles", func() {
	It("returns a list of articles", func() {
		httpmock.RegisterResponder("GET", "https://api.mybiz.com/articles.json",
			httpmock.NewStringResponder(200, `[{"id": 1, "name": "My Great Article"}]`))

		// do stuff that makes a request to articles.json
	})
})
```

### [Ginkgo](https://onsi.github.io/ginkgo/) + [Resty](https://github.com/go-resty/resty) Example:
```go
// article_suite_test.go

import (
	// ...
	"github.com/jarcoal/httpmock"
	"github.com/go-resty/resty"
)
// ...
var _ = BeforeSuite(func() {
	// block all HTTP requests
	httpmock.ActivateNonDefault(resty.DefaultClient.GetClient())
})

var _ = BeforeEach(func() {
	// remove any mocks
	httpmock.Reset()
})

var _ = AfterSuite(func() {
	httpmock.DeactivateAndReset()
})


// article_test.go

import (
	// ...
	"github.com/jarcoal/httpmock"
	"github.com/go-resty/resty"
)

var _ = Describe("Articles", func() {
	It("returns a list of articles", func() {
		fixture := `{"status":{"message": "Your message", "code": 200}}`
		responder := httpmock.NewStringResponder(200, fixture)
		fakeUrl := "https://api.mybiz.com/articles.json"
		httpmock.RegisterResponder("GET", fakeUrl, responder)

		// fetch the article into struct
		articleObject := &models.Article{}
		_, err := resty.R().SetResult(articleObject).Get(fakeUrl)

		// do stuff with the article object ...
	})
})
```
