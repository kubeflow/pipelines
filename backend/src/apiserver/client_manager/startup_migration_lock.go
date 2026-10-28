// Copyright 2026 The Kubeflow Authors
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

package clientmanager

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"time"

	"github.com/golang/glog"
	sqldrv "github.com/kubeflow/pipelines/backend/src/apiserver/common/sql/dialect"
	"gorm.io/gorm"
)

const startupMigrationLockRetryPeriod = time.Second

// startupMigrationLockStatements are scoped to the current database (MySQL) or
// database and schema (PostgreSQL), so installations sharing a server do not
// wait on each other. The key differs from the retention index lock.
type startupMigrationLockStatements struct {
	tryAcquire string
	release    string
}

func startupMigrationLockStatementsFor(dialect sqldrv.DBDialect) (startupMigrationLockStatements, error) {
	switch dialect.Name() {
	case "mysql":
		// GET_LOCK names are limited to 64 characters.
		const lockName = `CONCAT('kfp:startup-migration:', LEFT(SHA2(DATABASE(), 256), 40))`
		return startupMigrationLockStatements{
			tryAcquire: `SELECT GET_LOCK(` + lockName + `, 0)`,
			release:    `SELECT RELEASE_LOCK(` + lockName + `)`,
		}, nil
	case "pgx":
		const lockKeys = `hashtext(current_database()), hashtext(current_schema() || ':kfp:startup-migration')`
		return startupMigrationLockStatements{
			tryAcquire: `SELECT pg_try_advisory_lock(` + lockKeys + `)`,
			release:    `SELECT pg_advisory_unlock(` + lockKeys + `)`,
		}, nil
	default:
		return startupMigrationLockStatements{}, fmt.Errorf("startup migration lock does not support database dialect %q: use mysql or pgx", dialect.Name())
	}
}

// withStartupMigrationLock runs migrate while this API server holds a database
// session lock, so replicas that start together do not run the same DDL at the
// same time. A replica that finds the lock held waits up to lockWaitTimeout.
//
// migrate receives a session bound to the connection that holds the lock: if
// that connection is lost, the database releases the lock and the migration
// statements stop with it. migrate must not use the connection pool directly,
// for example through gorm's DB method.
func withStartupMigrationLock(
	database *gorm.DB,
	dialect sqldrv.DBDialect,
	lockWaitTimeout time.Duration,
	migrate func(lockedSession *gorm.DB) error,
) error {
	statements, statementsError := startupMigrationLockStatementsFor(dialect)
	if statementsError != nil {
		return statementsError
	}
	connectionPool, poolError := database.DB()
	if poolError != nil {
		return fmt.Errorf("startup migration lock: get the database connection pool: %w", poolError)
	}
	connection, connectionError := connectionPool.Conn(context.Background())
	if connectionError != nil {
		return fmt.Errorf("startup migration lock: open a database connection: %w", connectionError)
	}
	defer connection.Close()

	if acquireError := acquireStartupMigrationLock(connection, statements, lockWaitTimeout); acquireError != nil {
		return acquireError
	}
	defer releaseStartupMigrationLock(connection, statements)

	lockedSession := database.Session(&gorm.Session{NewDB: true})
	lockedSession.Statement.ConnPool = connection
	return migrate(lockedSession)
}

func acquireStartupMigrationLock(connection *sql.Conn, statements startupMigrationLockStatements, lockWaitTimeout time.Duration) error {
	deadline := time.Now().Add(lockWaitTimeout)
	loggedWait := false
	for {
		// MySQL returns NULL from GET_LOCK on an error; a nil result is not acquired.
		var acquired sql.NullBool
		if queryError := connection.QueryRowContext(context.Background(), statements.tryAcquire).Scan(&acquired); queryError != nil {
			return fmt.Errorf("startup migration lock: acquire: %w", queryError)
		}
		if acquired.Valid && acquired.Bool {
			if loggedWait {
				glog.Info("Acquired the startup migration lock")
			}
			return nil
		}
		if !time.Now().Before(deadline) {
			return fmt.Errorf("startup migration lock: another API server held the lock for longer than %v: "+
				"wait for its database migration to finish, or check for a stuck API server Pod, then restart this Pod", lockWaitTimeout)
		}
		if !loggedWait {
			glog.Infof("Waiting up to %v for another API server to finish database migrations", lockWaitTimeout)
			loggedWait = true
		}
		time.Sleep(startupMigrationLockRetryPeriod)
	}
}

// releaseStartupMigrationLock discards the connection if the lock cannot be
// released, because a connection that still holds the lock must not return to
// the pool. Closing the session releases the lock.
func releaseStartupMigrationLock(connection *sql.Conn, statements startupMigrationLockStatements) {
	var released sql.NullBool
	releaseError := connection.QueryRowContext(context.Background(), statements.release).Scan(&released)
	if releaseError == nil && released.Valid && released.Bool {
		return
	}
	glog.Warningf("Failed to release the startup migration lock (released: %v, error: %v); discarding its connection", released, releaseError)
	_ = connection.Raw(func(any) error { return driver.ErrBadConn })
}
