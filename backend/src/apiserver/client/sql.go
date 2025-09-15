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

package client

import (
	"fmt"
	"net/url"

	"github.com/go-sql-driver/mysql"
	pgx "github.com/jackc/pgx/v5"
)

func CreateMySQLConfig(user, password, mysqlServiceHost, mysqlServicePort,
	dbName, mysqlGroupConcatMaxLen string, mysqlExtraParams map[string]string,
) *mysql.Config {
	params := map[string]string{
		"charset":              "utf8",
		"parseTime":            "True",
		"loc":                  "Local",
		"group_concat_max_len": mysqlGroupConcatMaxLen,
	}

	for k, v := range mysqlExtraParams {
		params[k] = v
	}

	return &mysql.Config{
		User:                 user,
		Passwd:               password,
		Net:                  "tcp",
		Addr:                 fmt.Sprintf("%s:%s", mysqlServiceHost, mysqlServicePort),
		Params:               params,
		DBName:               dbName,
		AllowNativePasswords: true,
	}
}

func CreatePostgreSQLConfig(user, password, postgresHost, dbName string, postgresPort uint16,
) (*pgx.ConnConfig, string) {
	u := &url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(user, password),
		Host:     fmt.Sprintf("%s:%d", postgresHost, postgresPort),
		Path:     dbName,
		RawQuery: "sslmode=disable",
	}
	cfg, _ := pgx.ParseConfig(u.String())
	redactedDSN := fmt.Sprintf("host=%s port=%d user=%s password=*** database=%s sslmode=disable",
		postgresHost, postgresPort, user, dbName)
	return cfg, redactedDSN
}
