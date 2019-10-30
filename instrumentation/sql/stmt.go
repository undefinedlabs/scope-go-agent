package sql

import (
	"context"
	"database/sql/driver"
)

type instrumentedStmt struct {
	stmt          driver.Stmt
	configuration *driverConfiguration
}

// Close closes the statement.
//
// As of Go 1.1, a Stmt will not be closed if it's in use
// by any queries.
func (s *instrumentedStmt) Close() error {
	return s.stmt.Close()
}

// NumInput returns the number of placeholder parameters.
//
// If NumInput returns >= 0, the sql package will sanity check
// argument counts from callers and return errors to the caller
// before the statement's Exec or Query methods are called.
//
// NumInput may also return -1, if the driver doesn't know
// its number of placeholders. In that case, the sql package
// will not sanity check Exec or Query argument counts.
func (s *instrumentedStmt) NumInput() int {
	return s.stmt.NumInput()
}

// Exec executes a query that doesn't return rows, such
// as an INSERT or UPDATE.
//
// Deprecated: Drivers should implement StmtExecContext instead (or additionally).
func (s *instrumentedStmt) Exec(args []driver.Value) (driver.Result, error) {
	return s.stmt.Exec(args)
}

// Query executes a query that may return rows, such as a
// SELECT.
//
// Deprecated: Drivers should implement StmtQueryContext instead (or additionally).
func (s *instrumentedStmt) Query(args []driver.Value) (driver.Rows, error) {
	return s.stmt.Query(args)
}

// ExecContext executes a query that doesn't return rows, such
// as an INSERT or UPDATE.
//
// ExecContext must honor the context timeout and return when it is canceled.
func (s *instrumentedStmt) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	span := s.configuration.newSpan("ExecContext", query, s.configuration, ctx)
	span.SetTag("query", query)
	defer span.Finish()
	if execerContext, ok := s.stmt.(driver.ExecerContext); ok {
		return execerContext.ExecContext(ctx, query, args)
	}
	values, err := namedValueToValue(args)
	if err != nil {
		return nil, err
	}
	return s.Exec(values)
}

// QueryContext executes a query that may return rows, such as a
// SELECT.
//
// QueryContext must honor the context timeout and return when it is canceled.
func (s *instrumentedStmt) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (rows driver.Rows, err error) {
	span := s.configuration.newSpan("QueryContext", query, s.configuration, ctx)
	span.SetTag("query", query)
	defer span.Finish()
	if queryerContext, ok := s.stmt.(driver.QueryerContext); ok {
		rows, err := queryerContext.QueryContext(ctx, query, args)
		return rows, err
	}
	values, err := namedValueToValue(args)
	if err != nil {
		return nil, err
	}
	return s.Query(values)
}
