# go-agent

Scope agent for Go

## Compatibility

The Scope Go agent is compatible with the following versions of Go:

| Language    | Version |
| ----------- | :-----: |
| Go          |  1.11+  |

The Scope iOS agent is compatible with the following libraries:

| Name                                                         | Span/event creation | Extract | Inject |
| ------------------------------------------------------------ | :-----------------: | :-----: | :----: |
| [`testing`](https://golang.org/pkg/testing/)                 |          ✓          |         |        |
| [`net/http`](https://golang.org/pkg/net/http/)               |          ✓          |    ✓    |    ✓   |

> Do you use a language or library not listed here? Please [let us know](https://home.codescope.com/goto/support)!

## Installation

Installation of the Scope Agent is done via `go get`:

```bash
go get -u github.com/undefinedlabs/go-agent
```

## Usage

In order to instrument your tests that use Go's native [`testing`](https://golang.org/pkg/testing/) package, you
have to follow these steps:
 
1. Write a `TestMain(m *testing.M)` function that calls `GlobalAgent.Stop()` before exiting.
2. Wrap each test using a helper function called `InstrumentTest`.

For example:

```go
import (
    scopeagent "github.com/undefinedlabs/go-agent"
    "testing"
)

func TestMain(m *testing.M) {
    result := m.Run()
    scopeagent.GlobalAgent.Stop()  // This will ensure that we flush all pending results before exiting
    os.Exit(result)
}

func TestExample(t *testing.T) {
    scopeagent.InstrumentTest(t, func(t *testing.T) {
        // ... test code here
    })
}
```

You can also use [OpenTracing's Go API](https://github.com/opentracing/opentracing-go/blob/master/README.md) to add your
own custom spans and events. The Scope Agent's tracer will be registered as the global tracer automatically.


## HTTP instrumentation

### Instrumenting the HTTP client

The Scope Go agent automatically instruments the default HTTP client at `http.DefaultClient`. If you create a custom
`http.Client` instance, you must use the Scope Go agent transport:

```go
import (
    "github.com/undefinedlabs/go-agent/instrumentation/nethttp"
    "net/http"
)

func main() {
    client := &http.Client{Transport: &nethttp.Transport{}}
    // ...
}
```


#### Injecting the trace information to an outgoing request

In order for the Scope Go agent to trace an outgoing request, you must attach the context to the it. For example:

```go
import (
    "context"
    "net/http"
)

func makeRequest(url string, ctx context.Context) err {
    req, err := http.NewRequest("GET", url, nil)
    if err != nil {
        return err
    }
    req = req.WithContext(ctx)
    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return err
    }
    // ...
}
```


### Instrumenting the HTTP server

In order to instrument an HTTP server using the Scope Go agent, wrap the `http.Handler` you are serving with `nethttp.Middleware(h http.Handler)`.

For example, if using the default handler (`http.DefaultServeMux`):

```go
import (
    "github.com/undefinedlabs/go-agent/instrumentation/nethttp"
    "net/http"
    "io"
)

func main() {
    http.HandleFunc("/hello", func(w http.ResponseWriter, req *http.Request) {
        io.WriteString(w, "Hello, world!\n")
    })
    
    err := http.ListenAndServe(":8080", nethttp.Middleware(nil))
    if err != nil {
        panic(err)
    }
}
```


If you are using a custom handler, pass it to `nethttp.Middleware(h http.Handler)`:

```go
import (
    "github.com/undefinedlabs/go-agent/instrumentation/nethttp"
    "net/http"
    "io"
)

func main() {
    handler := http.NewServeMux()
    handler.HandleFunc("/hello", func(w http.ResponseWriter, req *http.Request) {
        io.WriteString(w, "Hello, world!\n")
    })
    
    err := http.ListenAndServe(":8080", nethttp.Middleware(handler))
    if err != nil {
        panic(err)
    }
}
```


## CI provider configuration

The following environment variables need to be configured in your CI provider:

| Environment variable | Description |
|---|---|
| `$SCOPE_APIKEY` | API key to use when sending data to Scope |
| `$SCOPE_API_ENDPOINT` | API endpoint of the Scope installation to send data to |


The following optional parameters can also be configured:

| Environment variable  | Default | Description |
|---|---|---|
| `$SCOPE_SERVICE` | `default` | Service name to use when sending data to Scope |
| `$SCOPE_COMMIT_SHA` | Autodetected | Commit hash to use when sending data to Scope |
| `$SCOPE_REPOSITORY` | Autodetected | Repository URL to use when sending data to Scope |
| `$SCOPE_SOURCE_ROOT` | Autodetected | Repository root path |

Autodetection of git information works if either tests run on Jenkins, CircleCI, Travis or GitLab.
