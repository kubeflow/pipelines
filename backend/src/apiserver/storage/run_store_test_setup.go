// Copyright 2018 The Kubeflow Authors
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
	"context"
	"fmt"

	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/testcontainers/testcontainers-go/modules/mysql"
)

type RuneStoreMysqlTestEnv struct {
	Db             *DB
	MysqlContainer *mysql.MySQLContainer
}

func NewRuneStoreMysqlSetupOrFatal(ctx context.Context) *RuneStoreMysqlTestEnv {
	container, err := LaunchMysqlContainer(ctx)
	if err != nil {
		glog.Fatal("failed to create mysql container:", err)
	}
	gormDb := NewMySqlDBOrFatal(container, ctx)
	return &RuneStoreMysqlTestEnv{Db: gormDb, MysqlContainer: container}
}

func (r *RuneStoreMysqlTestEnv) CleanStorageOrFatal() {
	if err := cleanTables(r.Db); err != nil {
		glog.Fatal("failed to clean tables:", err)
	}
}

func (r *RuneStoreMysqlTestEnv) StopOrFatal(ctx context.Context) {
	defer r.stopContainerOrFatal(ctx)
	if r.Db == nil {
		glog.Fatalf("Close mysql connection failed db is nil")
	}
	if err := r.Db.Close(); err != nil {
		glog.Fatal("Close mysql connection failed:", err)
	}
}

func (r *RuneStoreMysqlTestEnv) stopContainerOrFatal(ctx context.Context) {
	if r.Db == nil {
		glog.Fatalf("Close mysql container failed. container is nil")
	}
	if err := r.MysqlContainer.Terminate(ctx); err != nil {
		glog.Fatalf("Failed to terminate mysql container: %v", err)
	}
}

func (r *RuneStoreMysqlTestEnv) OpenExtraDbOrFatal() *DB {
	return NewMySqlDBOrFatal(r.MysqlContainer, context.Background())
}

func cleanTables(db *DB) error {
	err := cleanTable(db, model.Run{})
	if err != nil {
		return err
	}
	err = cleanTable(db, model.RunMetric{})
	if err != nil {
		return err
	}
	err = cleanTable(db, model.Experiment{})
	if err != nil {
		return err
	}
	err = cleanTable(db, model.ResourceReference{})
	if err != nil {
		return err
	}
	return nil
}

func cleanTable(db *DB, table model.Tabler) error {
	_, err := db.Exec(fmt.Sprintf("DELETE FROM %v", table.TableName()))
	if err != nil {
		return fmt.Errorf("failed to delete all rows from table %v: %v", table.TableName(), err)
	}
	return nil
}

func createRuns(store *RunStore, runs ...*model.Run) error {
	for _, run := range runs {
		if _, err := store.CreateRun(run); err != nil {
			return fmt.Errorf("failed to create run: %v", err)
		}
	}
	return nil
}

func createMetrics(store *RunStore, metrics ...*model.RunMetric) error {
	for _, metric := range metrics {
		if err := store.CreateMetric(metric); err != nil {
			return fmt.Errorf("failed to create metric: %v", err)
		}
	}
	return nil
}
