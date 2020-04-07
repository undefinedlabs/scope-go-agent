package sql

import (
	"context"
	"database/sql/driver"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/opentracing/opentracing-go"

	"go.undefinedlabs.com/scopeagent/config"
	scopeerrors "go.undefinedlabs.com/scopeagent/errors"
	"go.undefinedlabs.com/scopeagent/instrumentation"
)

type (
	// instrumented driver wrapper
	instrumentedDriver struct {
		driver        driver.Driver
		configuration *driverConfiguration
	}

	driverConfiguration struct {
		t               opentracing.Tracer
		statementValues bool
		stacktrace      bool
		connString      string
		componentName   string
		peerService     string
		user            string
		port            string
		instance        string
		host            string
	}

	Option func(*instrumentedDriver)
)

// Enable statement values instrumentation
func WithStatementValues() Option {
	return func(d *instrumentedDriver) {
		d.configuration.statementValues = true
	}
}

// Enable span stacktrace
func WithStacktrace() Option {
	return func(d *instrumentedDriver) {
		d.configuration.stacktrace = true
	}
}

// Wraps the current sql driver to add instrumentation
func WrapDriver(d driver.Driver, options ...Option) driver.Driver {
	wrapper := &instrumentedDriver{
		driver: d,
		configuration: &driverConfiguration{
			t:               instrumentation.Tracer(),
			statementValues: false,
		},
	}
	for _, option := range options {
		option(wrapper)
	}
	cfg := config.Get()
	wrapper.configuration.statementValues = wrapper.configuration.statementValues || (cfg.Instrumentation.DB.StatementValues != nil && *cfg.Instrumentation.DB.StatementValues)
	wrapper.configuration.stacktrace = wrapper.configuration.stacktrace || (cfg.Instrumentation.DB.StatementValues != nil && *cfg.Instrumentation.DB.StatementValues)
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
	w.callVendorsExtensions(name)
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
func (t *driverConfiguration) newSpan(operationName string, query string, args []driver.NamedValue, c *driverConfiguration, ctx context.Context) opentracing.Span {
	var opts []opentracing.StartSpanOption
	parent := opentracing.SpanFromContext(ctx)
	if parent != nil {
		opts = append(opts, opentracing.ChildOf(parent.Context()))
	}
	opts = append(opts, opentracing.Tags{
		"db.type":       "sql",
		"span.kind":     "client",
		"component":     c.componentName,
		"db.conn":       c.connString,
		"peer.service":  c.peerService,
		"db.user":       c.user,
		"peer.port":     c.port,
		"db.instance":   c.instance,
		"peer.hostname": c.host,
	})
	if t.stacktrace {
		opts = append(opts, opentracing.Tags{
			"stacktrace": scopeerrors.GetCurrentStackTrace(2),
		})
	}
	if query != "" {
		stIndex := strings.IndexRune(query, ' ')
		var method string
		if stIndex >= 0 {
			method = strings.ToUpper(query[:stIndex])
		}
		opts = append(opts, opentracing.Tags{
			"db.prepare_statement": query,
			"db.method":            method,
		})
		operationName = fmt.Sprintf("%s:%s", c.peerService, method)
	} else {
		operationName = fmt.Sprintf("%s:%s", c.peerService, strings.ToUpper(operationName))
	}
	if c.statementValues && args != nil && len(args) > 0 {
		dbParams := map[string]interface{}{}
		for _, item := range args {
			name := item.Name
			if name == "" {
				name = fmt.Sprintf("$%v", item.Ordinal)
			}
			dbParams[name] = map[string]interface{}{
				"type":  reflect.TypeOf(item.Value).String(),
				"value": item.Value,
			}
		}
		opts = append(opts, opentracing.Tags{
			"db.params": dbParams,
		})
	}
	span := t.t.StartSpan(operationName, opts...)
	return span
}

func (w *instrumentedDriver) callVendorsExtensions(name string) {
	w.configuration.connString = name
	w.configuration.componentName = reflect.TypeOf(w.driver).Elem().String()
	for _, vendor := range vendorExtensions {
		if vendor.IsCompatible(w.configuration.componentName) {
			vendor.ProcessConnectionString(name, w.configuration)
			break
		}
	}
}
