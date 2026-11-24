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

// RunPipelineRefBackfill retries the resumable migration until completion or shutdown.
func (c *ClientManager) RunPipelineRefBackfill(ctx context.Context) {
	for ctx.Err() == nil {
		done, err := runPipelineRefBackfillAttempt(ctx, c.backfillDB, c.dbDialect)
		if done || ctx.Err() != nil {
			return
		}
		if err != nil {
			glog.Warningf("Pipeline reference backfill paused, retrying in one minute: %v", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Minute):
		}
	}
}

func runPipelineRefBackfillAttempt(ctx context.Context, db *gorm.DB, dialect sqldrv.DBDialect) (bool, error) {
	db = db.WithContext(ctx)
	applied, err := migrationApplied(db, backfillPipelineRefsMigration)
	if applied || err != nil {
		return applied, err
	}

	var lockSQL string
	switch dialect.Name() {
	case "mysql":
		lockSQL = "SELECT GET_LOCK(CONCAT('kfp:pipeline-refs:', LEFT(SHA2(DATABASE(), 256), 40)), 0), CONNECTION_ID()"
	case "pgx":
		lockSQL = "SELECT pg_try_advisory_lock(hashtext(current_database()), hashtext(current_schema() || ':kfp:pipeline-refs')), pg_backend_pid()"
	default:
		return false, fmt.Errorf("unsupported pipeline reference backfill dialect %q", dialect.Name())
	}

	var acquired bool
	err = db.Connection(func(pinned *gorm.DB) error {
		// A session lock survives commits. Discard this physical connection
		// before returning it to the pool to release locks and session settings,
		// including when acquiring the lock failed with an ambiguous outcome.
		conn := pinned.Statement.ConnPool.(*sql.Conn)
		defer conn.Raw(func(any) error { return driver.ErrBadConn })
		pinned = pinned.Session(&gorm.Session{})
		var sessionID int64
		if err := pinned.Raw(lockSQL).Row().Scan(&acquired, &sessionID); err != nil || !acquired {
			return err
		}

		// A canceled MySQL client can leave a statement running on the server.
		// Use a second session to interrupt it, and finish before releasing the
		// pinned connection so it cannot be reused for another request.
		queryFinished := make(chan struct{})
		cancelFinished := make(chan struct{})
		go func() {
			defer close(cancelFinished)
			select {
			case <-ctx.Done():
			case <-queryFinished:
				if ctx.Err() == nil {
					return
				}
			}
			cancelCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			control := db.WithContext(cancelCtx)
			var err error
			if dialect.Name() == "mysql" {
				err = control.Exec(fmt.Sprintf("KILL QUERY %d", sessionID)).Error
			} else {
				err = control.Exec("SELECT pg_cancel_backend(?)", sessionID).Error
			}
			if err != nil {
				glog.Warningf("Failed to cancel pipeline reference backfill session %d: %v", sessionID, err)
			}
		}()
		defer func() {
			close(queryFinished)
			<-cancelFinished
		}()

		if dialect.Name() == "mysql" {
			if err := pinned.Exec("SET SESSION innodb_lock_wait_timeout = 2, SESSION lock_wait_timeout = 2").Error; err != nil {
				return err
			}
		}
		return backfillPipelineRefsToRunTable(pinned, dialect)
	})
	return acquired && err == nil, err
}
