package sql

import (
	"context"
	"database/sql/driver"
	"errors"
)

type instrumentedConn struct {
	conn          driver.Conn
	configuration *driverConfiguration
}

// ErrUnsupported is an error returned when the underlying driver doesn't provide a given function.
var ErrUnsupported = errors.New("operation unsupported by the underlying driver")

// Prepare returns a prepared statement, bound to this connection.
func (c *instrumentedConn) Prepare(query string) (driver.Stmt, error) {
	stmt, err := c.conn.Prepare(query)
	if err != nil {
		return nil, err
	}
	return &instrumentedStmt{
		stmt:          stmt,
		configuration: c.configuration,
	}, nil
}

// PrepareContext returns a prepared statement, bound to this connection.
// context is for the preparation of the statement,
// it must not store the context within the statement itself.
func (c *instrumentedConn) PrepareContext(ctx context.Context, query string) (driver.Stmt, error) {
	if connPrepareContext, ok := c.conn.(driver.ConnPrepareContext); ok {
		stmt, err := connPrepareContext.PrepareContext(ctx, query)
		if err != nil {
			return nil, err
		}
		return &instrumentedStmt{
			stmt:          stmt,
			configuration: c.configuration,
		}, nil
	}
	return c.conn.Prepare(query)
}

// Close invalidates and potentially stops any current
// prepared statements and transactions, marking this
// connection as no longer in use.
//
// Because the sql package maintains a free pool of
// connections and only calls Close when there's a surplus of
// idle connections, it shouldn't be necessary for drivers to
// do their own connection caching.
func (c *instrumentedConn) Close() error {
	return c.conn.Close()
}

// Begin starts and returns a new transaction.
//
// Deprecated: Drivers should implement ConnBeginTx instead (or additionally).
func (c *instrumentedConn) Begin() (driver.Tx, error) {
	tx, err := c.conn.Begin()
	if err != nil {
		return nil, err
	}
	return &instrumentedTx{
		tx:            tx,
		configuration: c.configuration,
		span:          nil,
	}, nil
}

// BeginTx starts and returns a new transaction.
// If the context is canceled by the user the sql package will
// call Tx.Rollback before discarding and closing the connection.
//
// This must check opts.Isolation to determine if there is a set
// isolation level. If the driver does not support a non-default
// level and one is set or if there is a non-default isolation level
// that is not supported, an error must be returned.
//
// This must also check opts.ReadOnly to determine if the read-only
// value is true to either set the read-only transaction property if supported
// or return an error if it is not supported.
func (c *instrumentedConn) BeginTx(ctx context.Context, opts driver.TxOptions) (driver.Tx, error) {
	s := c.configuration.newSpan("BeginTx", "", c.configuration, ctx)
	if connBeginTx, ok := c.conn.(driver.ConnBeginTx); ok {
		tx, err := connBeginTx.BeginTx(ctx, opts)
		if err != nil {
			return nil, err
		}
		return &instrumentedTx{
			tx:            tx,
			configuration: c.configuration,
			span:          s,
		}, nil
	}
	return c.conn.Begin()
}

// Execer is an optional interface that may be implemented by a Conn.
//
// If a Conn implements neither ExecerContext nor Execer,
// the sql package's DB.Exec will first prepare a query, execute the statement,
// and then close the statement.
//
// Exec may return ErrSkip.
//
// Deprecated: Drivers should implement ExecerContext instead.
func (c *instrumentedConn) Exec(query string, args []driver.Value) (driver.Result, error) {
	if execer, ok := c.conn.(driver.Execer); ok {
		return execer.Exec(query, args)
	}
	return nil, ErrUnsupported
}

// ExecerContext is an optional interface that may be implemented by a Conn.
//
// If a Conn does not implement ExecerContext, the sql package's DB.Exec
// will fall back to Execer; if the Conn does not implement Execer either,
// DB.Exec will first prepare a query, execute the statement, and then
// close the statement.
//
// ExecerContext may return ErrSkip.
//
// ExecerContext must honor the context timeout and return when the context is canceled.
func (c *instrumentedConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	s := c.configuration.newSpan("ExecContext", query, c.configuration, ctx)
	s.SetTag("query", query)
	defer s.Finish()
	if execerContext, ok := c.conn.(driver.ExecerContext); ok {
		r, err := execerContext.ExecContext(ctx, query, args)
		return r, err
	}
	values, err := namedValueToValue(args)
	if err != nil {
		return nil, err
	}
	return c.Exec(query, values)
}

// Pinger is an optional interface that may be implemented by a Conn.
//
// If a Conn does not implement Pinger, the sql package's DB.Ping and
// DB.PingContext will check if there is at least one Conn available.
//
// If Conn.Ping returns ErrBadConn, DB.Ping and DB.PingContext will remove
// the Conn from pool.
func (c *instrumentedConn) Ping(ctx context.Context) error {
	if pinger, ok := c.conn.(driver.Pinger); ok {
		s := c.configuration.newSpan("Ping", "", c.configuration, ctx)
		defer s.Finish()
		return pinger.Ping(ctx)
	}
	return ErrUnsupported
}

// Queryer is an optional interface that may be implemented by a Conn.
//
// If a Conn implements neither QueryerContext nor Queryer,
// the sql package's DB.Query will first prepare a query, execute the statement,
// and then close the statement.
//
// Query may return ErrSkip.
//
// Deprecated: Drivers should implement QueryerContext instead.
func (c *instrumentedConn) Query(query string, args []driver.Value) (driver.Rows, error) {
	if queryer, ok := c.conn.(driver.Queryer); ok {
		return queryer.Query(query, args)
	}
	return nil, ErrUnsupported
}

// QueryerContext is an optional interface that may be implemented by a Conn.
//
// If a Conn does not implement QueryerContext, the sql package's DB.Query
// will fall back to Queryer; if the Conn does not implement Queryer either,
// DB.Query will first prepare a query, execute the statement, and then
// close the statement.
//
// QueryerContext may return ErrSkip.
//
// QueryerContext must honor the context timeout and return when the context is canceled.
func (c *instrumentedConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (rows driver.Rows, err error) {
	s := c.configuration.newSpan("QueryContext", query, c.configuration, ctx)
	s.SetTag("query", query)
	defer s.Finish()
	if queryerContext, ok := c.conn.(driver.QueryerContext); ok {
		rows, err := queryerContext.QueryContext(ctx, query, args)
		return rows, err
	}
	values, err := namedValueToValue(args)
	if err != nil {
		return nil, err
	}
	return c.Query(query, values)
}
