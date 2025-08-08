// Copyright 2025 The Kubeflow Authors
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
// see the License for the specific language governing permissions and
// limitations under the License.

// Package clientmanager provides tools for managing database clients, initialization flows,
// and dialect-specific helpers such as identifier quoting and SQL function mapping.
package clientmanager

import (
	sq "github.com/Masterminds/squirrel"
)

type SQLDialect struct {
	Name                 string
	QuoteIdentifier      func(string) string
	LengthFunc           string
	StatementBuilder     sq.StatementBuilderType
	ExistDatabaseErrHint string
}

func GetDialect(name string) SQLDialect {
	switch name {
	case "mysql":
		return SQLDialect{
			Name:                 "mysql",
			QuoteIdentifier:      func(id string) string { return "`" + id + "`" },
			LengthFunc:           "CHAR_LENGTH",
			StatementBuilder:     sq.StatementBuilder.PlaceholderFormat(sq.Question),
			ExistDatabaseErrHint: "database exists",
		}
	case "pgx":
		return SQLDialect{
			Name:                 "pgx",
			QuoteIdentifier:      func(id string) string { return `"` + id + `"` },
			LengthFunc:           "CHAR_LENGTH",
			StatementBuilder:     sq.StatementBuilder.PlaceholderFormat(sq.Dollar),
			ExistDatabaseErrHint: "already exists",
		}
	case "sqlite":
		// Only for test
		return SQLDialect{
			Name:                 "sqlite",
			QuoteIdentifier:      func(id string) string { return `"` + id + `"` },
			LengthFunc:           "LENGTH",
			StatementBuilder:     sq.StatementBuilder.PlaceholderFormat(sq.Question),
			ExistDatabaseErrHint: "",
		}
	default:
		panic("Unsupported dialect: " + name)
	}
}
