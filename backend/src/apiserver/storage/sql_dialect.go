// Copyright 2018 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package storage

import (
	"bytes"
	"fmt"
	"strings"
)

// SQLDialect abstracts common sql queries which vary in different dialect.
// It is used to bridge the difference between mysql (production) and sqlite
// (test).
type SQLDialect interface {
	// GroupConcat builds query to group concatenate `expr` in each row and use `separator`
	// to join rows in a group.
	GroupConcat(expr string, separator string) string

	// Concat builds query to concatenete a list of `exprs` into a single string with
	// a separator in between.
	Concat(exprs []string, separator string) string
}

// MySQLDialect implements SQLDialect with mysql dialect implementation.
type MySQLDialect struct{}

func (d MySQLDialect) GroupConcat(expr string, separator string) string {
	var buffer bytes.Buffer
	buffer.WriteString("GROUP_CONCAT(")
	buffer.WriteString(expr)
	if separator != "" {
		buffer.WriteString(fmt.Sprintf(" SEPARATOR \"%s\"", separator))
	}
	buffer.WriteString(")")
	return buffer.String()
}

func (d MySQLDialect) Concat(exprs []string, separator string) string {
	separatorSQL := ","
	if separator != "" {
		separatorSQL = fmt.Sprintf(`,"%s",`, separator)
	}
	return fmt.Sprintf("CONCAT(%s)", strings.Join(exprs, separatorSQL))
}

// SQLiteDialect implements SQLDialect with sqlite dialect implementation.
type SQLiteDialect struct{}

func (d SQLiteDialect) GroupConcat(expr string, separator string) string {
	var buffer bytes.Buffer
	buffer.WriteString("GROUP_CONCAT(")
	buffer.WriteString(expr)
	if separator != "" {
		buffer.WriteString(fmt.Sprintf(", \"%s\"", separator))
	}
	buffer.WriteString(")")
	return buffer.String()
}

func (d SQLiteDialect) Concat(exprs []string, separator string) string {
	separatorSQL := "||"
	if separator != "" {
		separatorSQL = fmt.Sprintf(`||"%s"||`, separator)
	}
	return strings.Join(exprs, separatorSQL)
}

func NewMySQLDialect() MySQLDialect {
	return MySQLDialect{}
}

func NewSQLiteDialect() SQLiteDialect {
	return SQLiteDialect{}
}
