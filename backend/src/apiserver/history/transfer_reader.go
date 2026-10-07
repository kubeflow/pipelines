// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0

package history

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"reflect"

	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/transfer"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ignoredColumn struct{}

func (*ignoredColumn) Scan(any) error { return nil }

// readTransferRows streams rows into the archive budget. JSON columns are read
// again from the current row (not a second query) to bypass the normal float64
// scanner without retaining a second collection of raw JSON.
func readTransferRows[T any](query *gorm.DB, out *[]T, budget *transfer.ExportBudget) error {
	stmt := &gorm.Statement{DB: query}
	if err := stmt.Parse(new(T)); err != nil {
		return err
	}
	rows, err := query.Model(new(T)).Rows()
	if err != nil {
		return err
	}
	defer rows.Close()
	names, err := rows.Columns()
	if err != nil {
		return err
	}
	raw := make([]sql.RawBytes, len(names))
	targets := make([]any, len(names))
	var jsonColumns []int
	for i, name := range names {
		field := stmt.Schema.FieldsByDBName[name]
		if field != nil && (field.FieldType == reflect.TypeOf(model.JSONData{}) || field.FieldType == reflect.TypeOf(model.JSONSlice{})) {
			jsonColumns = append(jsonColumns, i)
			targets[i] = &raw[i]
		} else {
			targets[i] = &ignoredColumn{}
		}
	}
	for rows.Next() {
		var value T
		if err := query.ScanRows(rows, &value); err != nil {
			return err
		}
		if len(jsonColumns) > 0 {
			if err := rows.Scan(targets...); err != nil {
				return err
			}
			rv := reflect.ValueOf(&value).Elem()
			for _, i := range jsonColumns {
				if len(raw[i]) == 0 {
					continue
				}
				field := stmt.Schema.FieldsByDBName[names[i]]
				decoder := json.NewDecoder(bytes.NewReader(raw[i]))
				decoder.UseNumber()
				parsed := reflect.New(field.FieldType)
				if err := decoder.Decode(parsed.Interface()); err != nil {
					return err
				}
				if err := field.Set(query.Statement.Context, rv, parsed.Elem().Interface()); err != nil {
					return err
				}
			}
		}
		if err := budget.Add(value); err != nil {
			return err
		}
		*out = append(*out, value)
	}
	return rows.Err()
}

func findTransferRows[T any](db *gorm.DB, column string, ids []string, out *[]T, budget *transfer.ExportBudget) error {
	if len(ids) == 0 {
		return nil
	}
	values := make([]any, len(ids))
	for i, id := range ids {
		values[i] = id
	}
	return readTransferRows(db.Where(clause.IN{Column: clause.Column{Name: column}, Values: values}), out, budget)
}
