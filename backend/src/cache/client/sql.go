// Copyright 2020 The Kubeflow Authors
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
	"bytes"
	"fmt"
	"strings"

	"github.com/go-sql-driver/mysql"
)

// quoteDSNValue quotes a value for use in a libpq keyword/value connection string.
// Per libpq rules: wrap in single quotes, escape \ and ' with a backslash.
func quoteDSNValue(v string) string {
	v = strings.ReplaceAll(v, `\`, `\\`)
	v = strings.ReplaceAll(v, `'`, `\'`)
	return "'" + v + "'"
}

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

func CreatePostgreSQLConfig(user, password, postgresHost, dbName string, postgresPort uint16,
) string {
	var b bytes.Buffer
	if dbName != "" {
		fmt.Fprintf(&b, "database=%s ", quoteDSNValue(dbName))
	}
	if user != "" {
		fmt.Fprintf(&b, "user=%s ", quoteDSNValue(user))
	}
	if password != "" {
		fmt.Fprintf(&b, "password=%s ", quoteDSNValue(password))
	}
	if postgresHost != "" {
		fmt.Fprintf(&b, "host=%s ", quoteDSNValue(postgresHost))
	}
	if postgresPort != 0 {
		fmt.Fprintf(&b, "port=%d ", postgresPort)
	}
	fmt.Fprint(&b, "sslmode=disable")

	return b.String()
}
