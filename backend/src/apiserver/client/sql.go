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
)

func CreateMySQLConfig(user, password string, mysqlServiceHost string,
	mysqlServicePort string, dbName string, mysqlGroupConcatMaxLen string, mysqlExtraParams map[string]string) *mysql.Config {

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
