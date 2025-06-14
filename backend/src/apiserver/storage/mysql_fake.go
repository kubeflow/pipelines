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
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	mysqlcontainer "github.com/testcontainers/testcontainers-go/modules/mysql"
)

func NewMySqlDBOrFatal(container *mysqlcontainer.MySQLContainer, ctx context.Context) *DB {
	gormDb, err := OpenMysqlDb(ctx, container)
	if err != nil {
		glog.Fatalf(err.Error())
	}
	db, err := ApplyMigration(gormDb)
	return db
}

func LaunchMysqlContainer(ctx context.Context) (*mysqlcontainer.MySQLContainer, error) {
	container, err := mysqlcontainer.Run(ctx, "mysql:8.0")
	if err != nil {
		return nil, fmt.Errorf("error during creating mysql container for testing: %v", err)
	}
	return container, nil
}

func OpenMysqlDb(ctx context.Context, container *mysqlcontainer.MySQLContainer) (*gorm.DB, error) {
	connString, err := container.ConnectionString(ctx)
	if err != nil {
		return nil, fmt.Errorf("error during creating mysql connection string for testing: %v", err)
	}
	db, err := gorm.Open("mysql", connString)
	if err != nil {
		return nil, util.Wrap(err, "Could not create the GORM mysql database for testing")
	}
	return db, nil
}

func ApplyMigration(db *gorm.DB) (*DB, error) {
	dbWithMigration, err := migrate(db)
	if err != nil {
		return nil, util.Wrap(err, "Could not migrate the database")
	}
	return NewDB(dbWithMigration, NewMySQLDialect()), nil
}
