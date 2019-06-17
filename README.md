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
| [`testing`](https://golang.org/pkg/testing/)                   |          ✓          |         |        |

> Do you use a language or library not listed here? Please [let us know](https://home.codescope.com/goto/support)!

## Installation

Installation of the Scope Agent is done via `go get`:

```bash
go get -u github.com/undefinedlabs/go-agent
```

## Usage

In order to instrument your tests that use Go's native [`testing`](https://golang.org/pkg/testing/) package, you
have to wrap each test using a helper function called `InstrumentTest`:

```go
import (
	scopeagent "github.com/undefinedlabs/go-agent"
	"testing"
)

func TestPass(t *testing.T) {
	scopeagent.InstrumentTest(t, func(t *testing.T) {
		// ... test code here
	})
}
```

You can also use [OpenTracing's Go API](https://github.com/opentracing/opentracing-go/blob/master/README.md) to add your
own custom spans and events. The Scope Agent's tracer will be registered as the global tracer automatically.


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
