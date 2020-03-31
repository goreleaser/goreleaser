> # ♻️ retry
>
> The most advanced interruptible mechanism to perform actions repetitively until successful.

[![Build][build.icon]][build.page]
[![Documentation][docs.icon]][docs.page]
[![Quality][quality.icon]][quality.page]
[![Template][template.icon]][template.page]
[![Coverage][coverage.icon]][coverage.page]
[![Awesome][awesome.icon]][awesome.page]

## 💡 Idea

The package based on [Rican7/retry][] but fully reworked and focused on integration
with the 🚧 [breaker][] and the built-in [context][] packages.

Full description of the idea is available [here][design.page].

## 🏆 Motivation

I developed distributed systems at [Lazada][], and later at [Avito][],
which communicate with each other through a network, and I need a package to make
these communications more reliable.

## 🤼‍♂️ How to

### retry.Retry

```go
var response *http.Response

action := func(uint) error {
	var err error
	response, err = http.Get("https://github.com/kamilsk/retry")
	return err
}

if err := retry.Retry(breaker.BreakByTimeout(time.Minute), action, strategy.Limit(3)); err != nil {
	if err == retry.Interrupted {
		// timeout exceeded
	}
	// handle error
}
// work with response
```

### retry.Try

```go
var response *http.Response

action := func(uint) error {
	var err error
	response, err = http.Get("https://github.com/kamilsk/retry")
	return err
}

// you can combine multiple Breakers into one
interrupter := breaker.MultiplexTwo(
	breaker.BreakByTimeout(time.Minute),
	breaker.BreakBySignal(os.Interrupt),
)
defer interrupter.Close()

if err := retry.Try(interrupter, action, strategy.Limit(3)); err != nil {
	if err == retry.Interrupted {
		// timeout exceeded
	}
	// handle error
}
// work with response
```

or use Context

```go
ctx, cancel := context.WithTimeout(request.Context(), time.Minute)
defer cancel()

if err := retry.Try(ctx, action, strategy.Limit(3)); err != nil {
	if err == retry.Interrupted {
		// timeout exceeded
	}
	// handle error
}
// work with response
```

### retry.TryContext

```go
var response *http.Response

action := func(ctx context.Context, _ uint) error {
	req, err := http.NewRequest(http.MethodGet, "https://github.com/kamilsk/retry", nil)
	if err != nil {
		return err
	}
	req = req.WithContext(ctx)
	response, err = http.DefaultClient.Do(req)
	return err
}

// you can combine Context and Breaker together
interrupter, ctx := breaker.WithContext(request.Context())
defer interrupter.Close()

if err := retry.TryContext(ctx, action, strategy.Limit(3)); err != nil {
	if err == retry.Interrupted {
		// timeout exceeded
	}
	// handle error
}
// work with response
```

### Complex example

```go
import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net"
	"time"

	"github.com/kamilsk/retry/v4"
	"github.com/kamilsk/retry/v4/backoff"
	"github.com/kamilsk/retry/v4/jitter"
	"github.com/kamilsk/retry/v4/strategy"
)

func main() {
	what := func(uint) (err error) {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("unexpected panic: %v", r)
			}
		}()
		return SendRequest()
	}

	how := retry.How{
		strategy.Limit(5),
		strategy.BackoffWithJitter(
			backoff.Fibonacci(10*time.Millisecond),
			jitter.NormalDistribution(
				rand.New(rand.NewSource(time.Now().UnixNano())),
				0.25,
			),
		),
		strategy.CheckNetworkError(strategy.Skip),
	}

	ctx, _ := context.WithTimeout(context.Background(), time.Second)
	if err := retry.Try(ctx, what, how...); err != nil {
		log.Fatal(err)
	}
}

func SendRequest() error {
	// communicate with some service
}
```

## 🧩 Integration

The library uses [SemVer](https://semver.org) for versioning, and it is not
[BC](https://en.wikipedia.org/wiki/Backward_compatibility)-safe through major releases.
You can use [go modules](https://github.com/golang/go/wiki/Modules) or
[dep](https://golang.github.io/dep/) to manage its version.

```bash
# inside GOPATH and for old Go versions
$ go get -u github.com/kamilsk/retry
$ dep ensure -add github.com/kamilsk/retry@v4.0.0
# inside Go module, works well since Go 1.11
$ go get -u github.com/kamilsk/retry/v4
```

## 🤲 Outcomes

### Console tool for command execution with retries

This example shows how to repeat console command until successful.

```bash
$ retry -timeout 10m -backoff lin:500ms -- /bin/sh -c 'echo "trying..."; exit $((1 + RANDOM % 10 > 5))'
```

[![asciicast][cli.preview]][cli.demo]

See more details [here][cli].

---

made with ❤️ for everyone

[awesome.icon]:     https://cdn.rawgit.com/sindresorhus/awesome/d7305f38d29fed78fa85652e3a63e154dd8e8829/media/badge.svg
[awesome.page]:     https://github.com/avelino/awesome-go#utilities
[build.icon]:       https://travis-ci.org/kamilsk/retry.svg?branch=v4
[build.page]:       https://travis-ci.org/kamilsk/retry
[coverage.icon]:    https://api.codeclimate.com/v1/badges/ed88afbc0754e49e9d2d/test_coverage
[coverage.page]:    https://codeclimate.com/github/kamilsk/retry/test_coverage
[design.page]:      https://www.notion.so/octolab/retry-cab5722faae445d197e44fbe0225cc98?r=0b753cbf767346f5a6fd51194829a2f3
[docs.page]:        https://pkg.go.dev/github.com/kamilsk/retry/v4
[docs.icon]:        https://img.shields.io/badge/docs-pkg.go.dev-blue
[promo.page]:       https://github.com/kamilsk/retry
[quality.icon]:     https://goreportcard.com/badge/github.com/kamilsk/retry
[quality.page]:     https://goreportcard.com/report/github.com/kamilsk/retry
[template.page]:    https://github.com/octomation/go-module
[template.icon]:    https://img.shields.io/badge/template-go--module-blue

[Avito]:            https://tech.avito.ru
[breaker]:          https://github.com/kamilsk/breaker
[cli]:              https://github.com/kamilsk/try
[cli.demo]:         https://asciinema.org/a/150367
[cli.preview]:      https://asciinema.org/a/150367.png
[context]:          https://pkg.go.dev/context
[Lazada]:           https://github.com/lazada
[Rican7/retry]:     https://github.com/Rican7/retry

[tmp.docs]:         https://nicedoc.io/kamilsk/retry?theme=dark
[tmp.history]:      https://github.githistory.xyz/kamilsk/retry/blob/v4/README.md
