// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy at http://www.apache.org/licenses/LICENSE-2.0

package storage

import (
	"database/sql"
	"fmt"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// OpenTransferDB reuses the API server's connection pool without migrations.
func OpenTransferDB(db *sql.DB, driver string) (*gorm.DB, error) {
	var dialector gorm.Dialector
	switch driver {
	case "mysql":
		dialector = mysql.New(mysql.Config{Conn: db, SkipInitializeWithVersion: true})
	case "pgx", "postgres":
		dialector = postgres.New(postgres.Config{Conn: db})
	case "sqlite":
		dialector = sqlite.New(sqlite.Config{Conn: db})
	default:
		return nil, fmt.Errorf("unsupported transfer database %q", driver)
	}
	return gorm.Open(dialector, &gorm.Config{DisableAutomaticPing: true})
}
