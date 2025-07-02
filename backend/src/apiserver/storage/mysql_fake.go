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

// MySQL is tested using the official Testcontainers MySQL module:
// https://golang.testcontainers.org/modules/mysql/
//
// Testcontainers requires a Docker API-compatible container runtime:
// https://golang.testcontainers.org/system_requirements/docker/
//
// Configuration options can be found here:
// https://golang.testcontainers.org/features/configuration/
//
// By default, Testcontainers starts a "ryuk" container to automatically
// clean up other containers and resources after tests.
// This behavior can be customized as described here:
// https://golang.testcontainers.org/features/configuration/#customizing-ryuk-the-resource-reaper
//
// IMPORTANT:
// If your tests run inside a container in your CI environment,
// you need to provide access to the Docker runtime on the host,
// typically by mounting the Docker socket (e.g., /var/run/docker.sock)
// into the test container, so that Testcontainers can manage containers.
//
// Image can be configured using the command-line flag: go test -v -mysql-image <YOUR_MYSQL_IMAGE>

func NewMySqlDBOrFatal(ctx context.Context, container *mysqlcontainer.MySQLContainer) *DB {
	gormDb, err := OpenMySQLDb(ctx, container)
	if err != nil {
		glog.Fatalf(err.Error())
	}
	db, err := ApplyMigration(gormDb)
	return db
}

func LaunchMySQLContainer(ctx context.Context, params *MySQLTestParams) (*mysqlcontainer.MySQLContainer, error) {
	if params == nil {
		return nil, fmt.Errorf("params cannot be nil")
	}
	container, err := mysqlcontainer.Run(ctx, params.Image)
	if err != nil {
		return nil, fmt.Errorf("error during creating MySQL container for testing: %v", err)
	}
	return container, nil
}

func OpenMySQLDb(ctx context.Context, container *mysqlcontainer.MySQLContainer) (*gorm.DB, error) {
	connString, err := container.ConnectionString(ctx)
	if err != nil {
		return nil, fmt.Errorf("error during creating MySQL connection string for testing: %v", err)
	}
	db, err := gorm.Open("mysql", connString)
	if err != nil {
		return nil, util.Wrap(err, "Could not create the GORM MySQL database for testing")
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
