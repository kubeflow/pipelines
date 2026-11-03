// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0

package storage

import (
	"fmt"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// TransferDB reuses the API server pool; transfer never accepts database credentials.
func (s *DBStatusStore) TransferDB() (*gorm.DB, error) {
	var driver gorm.Dialector
	switch s.dbDialect.Name() {
	case "mysql":
		driver = mysql.New(mysql.Config{Conn: s.db, SkipInitializeWithVersion: true})
	case "pgx":
		driver = postgres.New(postgres.Config{Conn: s.db})
	case "sqlite":
		driver = sqlite.New(sqlite.Config{Conn: s.db})
	default:
		return nil, fmt.Errorf("unsupported transfer database")
	}
	return gorm.Open(driver, &gorm.Config{DisableAutomaticPing: true, Logger: logger.Default.LogMode(logger.Silent)})
}
