package sql

import (
	"database/sql/driver"
	"github.com/opentracing/opentracing-go"
)

// conn defines a tracing wrapper for driver.Tx.
type instrumentedTx struct {
	tx            driver.Tx
	configuration *driverConfiguration
	span          opentracing.Span
}

// Commit implements driver.Tx Commit.
func (t *instrumentedTx) Commit() error {
	if t.span != nil {
		defer t.span.Finish()
	}
	return t.tx.Commit()
}

// Rollback implements driver.Tx Rollback.
func (t *instrumentedTx) Rollback() error {
	if t.span != nil {
		defer t.span.Finish()
	}
	return t.tx.Rollback()
}
