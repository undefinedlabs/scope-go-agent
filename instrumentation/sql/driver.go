package sql

import (
	"context"
	"database/sql/driver"
	"errors"
	"github.com/opentracing/opentracing-go"
	"go.undefinedlabs.com/scopeagent/instrumentation"
)

type (
	// instrumented driver wrapper
	instrumentedDriver struct {
		driver        driver.Driver
		configuration *driverConfiguration
	}

	driverConfiguration struct {
		t          opentracing.Tracer
		statements bool
	}

	Option func(*instrumentedDriver)
)

// Enable statement instrumentation
func WithStatements() Option {
	return func(d *instrumentedDriver) {
		d.configuration.statements = true
	}
}

// Wraps the current sql driver to add instrumentation
func WrapDriver(d driver.Driver, options ...Option) driver.Driver {
	wrapper := &instrumentedDriver{
		driver: d,
		configuration: &driverConfiguration{
			t:          instrumentation.Tracer(),
			statements: false,
		},
	}
	for _, option := range options {
		option(wrapper)
	}
	return wrapper
}

// Open returns a new connection to the database.
// The name is a string in a driver-specific format.
//
// Open may return a cached connection (one previously
// closed), but doing so is unnecessary; the sql package
// maintains a pool of idle connections for efficient re-use.
//
// The returned connection is only used by one goroutine at a
// time.
func (w *instrumentedDriver) Open(name string) (driver.Conn, error) {
	conn, err := w.driver.Open(name)
	if err != nil {
		return nil, err
	}
	return &instrumentedConn{conn: conn, configuration: w.configuration}, nil
}

// namedValueToValue converts driver arguments of NamedValue format to Value format. Implemented in the same way as in
// database/sql ctxutil.go.
func namedValueToValue(named []driver.NamedValue) ([]driver.Value, error) {
	dargs := make([]driver.Value, len(named))
	for n, param := range named {
		if len(param.Name) > 0 {
			return nil, errors.New("sql: driver does not support the use of Named Parameters")
		}
		dargs[n] = param.Value
	}
	return dargs, nil
}

// newSpan creates a new opentracing.Span instance from the given context.
func (t *driverConfiguration) newSpan(operationName string, ctx context.Context) opentracing.Span {
	var opts []opentracing.StartSpanOption
	parent := opentracing.SpanFromContext(ctx)
	if parent != nil {
		opts = append(opts, opentracing.ChildOf(parent.Context()))
	}
	span := t.t.StartSpan(operationName, opts...)
	return span
}
