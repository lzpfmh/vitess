// Copyright 2015, Google Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vtgateconn

import (
	"database/sql/driver"
	"fmt"
	"io"

	mproto "github.com/youtube/vitess/go/mysql/proto"
	"github.com/youtube/vitess/go/sqltypes"
)

// rows creates a database/sql/driver compliant Row iterator
// for a non-streaming QueryResult.
type rows struct {
	qr    *mproto.QueryResult
	index int
}

// NewRows creates a new rows from qr.
func NewRows(qr *mproto.QueryResult) driver.Rows {
	return &rows{qr: qr}
}

func (ri *rows) Columns() []string {
	cols := make([]string, 0, len(ri.qr.Fields))
	for _, field := range ri.qr.Fields {
		cols = append(cols, field.Name)
	}
	return cols
}

func (ri *rows) Close() error {
	return nil
}

func (ri *rows) Next(dest []driver.Value) error {
	if ri.index == len(ri.qr.Rows) {
		return io.EOF
	}
	err := populateRow(dest, ri.qr.Fields, ri.qr.Rows[ri.index])
	ri.index++
	return err
}

func populateRow(dest []driver.Value, fields []mproto.Field, row []sqltypes.Value) error {
	if len(dest) != len(fields) {
		return fmt.Errorf("length mismatch: dest is %d, fields are %d", len(dest), len(fields))
	}
	if len(fields) != len(row) {
		return fmt.Errorf("internal error: length mismatch: dest is %d, fields are %d", len(fields), len(row))
	}
	var err error
	for i := range dest {
		dest[i], err = mproto.Convert(fields[i].Type, row[i])
		if err != nil {
			return fmt.Errorf("conversion error: field: %v, val: %v: %v", fields[i], row[i], err)
		}
	}
	return nil
}
