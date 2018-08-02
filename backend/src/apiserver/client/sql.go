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

package client

import (
	"fmt"

	"github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
)

func CreateMySQLConfig(user string, mysqlServiceHost string,
		mysqlServicePort string, dbName string) *mysql.Config {
	return &mysql.Config{
		User:                 user,
		Net:                  "tcp",
		Addr:                 fmt.Sprintf("%s:%s", mysqlServiceHost, mysqlServicePort),
		Params:               map[string]string{"charset": "utf8", "parseTime": "True", "loc": "Local"},
		DBName:               dbName,
		AllowNativePasswords: true,
	}
}

func CreateGormClient(driverName string, sqliteDatasourceName string, user string,
		mysqlServiceHost string, mysqlServicePort string, dbName string) (*gorm.DB, error) {

	var dataSourceName string

	switch driverName {
	case "mysql":
		dataSourceName = CreateMySQLConfig(user, mysqlServiceHost, mysqlServicePort, dbName).FormatDSN()
	case "sqlite3":
		dataSourceName = sqliteDatasourceName
	default:
		return nil, errors.Errorf("Driver %v is not supported", driverName)
	}

	return gorm.Open(driverName, dataSourceName)
}
