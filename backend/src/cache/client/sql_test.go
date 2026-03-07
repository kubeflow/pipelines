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
	"reflect"
	"testing"

	"github.com/go-sql-driver/mysql"
)

func TestCreateMySQLConfig(t *testing.T) {
	type args struct {
		user                   string
		password               string
		host                   string
		port                   string
		dbName                 string
		mysqlGroupConcatMaxLen string
		mysqlExtraParams       map[string]string
	}
	tests := []struct {
		name string
		args args
		want *mysql.Config
	}{
		{
			name: "default config",
			args: args{
				user:                   "root",
				host:                   "mysql",
				port:                   "3306",
				mysqlGroupConcatMaxLen: "1024",
				mysqlExtraParams:       nil,
			},
			want: &mysql.Config{
				User:                 "root",
				Net:                  "tcp",
				Addr:                 "mysql:3306",
				Params:               map[string]string{"charset": "utf8", "parseTime": "True", "loc": "Local", "group_concat_max_len": "1024"},
				AllowNativePasswords: true,
			},
		},
		{
			name: "extra parameters",
			args: args{
				user:                   "root",
				host:                   "mysql",
				port:                   "3306",
				mysqlGroupConcatMaxLen: "1024",
				mysqlExtraParams:       map[string]string{"tls": "true"},
			},
			want: &mysql.Config{
				User:                 "root",
				Net:                  "tcp",
				Addr:                 "mysql:3306",
				Params:               map[string]string{"charset": "utf8", "parseTime": "True", "loc": "Local", "group_concat_max_len": "1024", "tls": "true"},
				AllowNativePasswords: true,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CreateMySQLConfig(tt.args.user, tt.args.password, tt.args.host, tt.args.port, tt.args.dbName, tt.args.mysqlGroupConcatMaxLen, tt.args.mysqlExtraParams); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CreateMySQLConfig() = %#v, want %v", got, tt.want)
			}
		})
	}
}

func TestCreatePostgreSQLConfig(t *testing.T) {
	tests := []struct {
		name         string
		user         string
		password     string
		host         string
		dbName       string
		port         uint16
		want         string
		wantRedacted string
	}{
		{
			name:         "all fields populated",
			user:         "admin",
			password:     "secret",
			host:         "postgres-host",
			dbName:       "mlpipeline",
			port:         5432,
			want:         "database=mlpipeline user=admin password=secret host=postgres-host port=5432 sslmode=disable",
			wantRedacted: "database=mlpipeline user=admin password=*** host=postgres-host port=5432 sslmode=disable",
		},
		{
			name:         "empty password",
			user:         "admin",
			password:     "",
			host:         "postgres-host",
			dbName:       "mlpipeline",
			port:         5432,
			want:         "database=mlpipeline user=admin host=postgres-host port=5432 sslmode=disable",
			wantRedacted: "database=mlpipeline user=admin host=postgres-host port=5432 sslmode=disable",
		},
		{
			name:         "empty user",
			user:         "",
			password:     "secret",
			host:         "postgres-host",
			dbName:       "mlpipeline",
			port:         5432,
			want:         "database=mlpipeline password=secret host=postgres-host port=5432 sslmode=disable",
			wantRedacted: "database=mlpipeline password=*** host=postgres-host port=5432 sslmode=disable",
		},
		{
			name:         "empty database name",
			user:         "admin",
			password:     "secret",
			host:         "postgres-host",
			dbName:       "",
			port:         5432,
			want:         "user=admin password=secret host=postgres-host port=5432 sslmode=disable",
			wantRedacted: "user=admin password=*** host=postgres-host port=5432 sslmode=disable",
		},
		{
			name:         "empty host",
			user:         "admin",
			password:     "secret",
			host:         "",
			dbName:       "mlpipeline",
			port:         5432,
			want:         "database=mlpipeline user=admin password=secret port=5432 sslmode=disable",
			wantRedacted: "database=mlpipeline user=admin password=*** port=5432 sslmode=disable",
		},
		{
			name:         "zero port",
			user:         "admin",
			password:     "secret",
			host:         "postgres-host",
			dbName:       "mlpipeline",
			port:         0,
			want:         "database=mlpipeline user=admin password=secret host=postgres-host sslmode=disable",
			wantRedacted: "database=mlpipeline user=admin password=*** host=postgres-host sslmode=disable",
		},
		{
			name:         "all fields empty",
			user:         "",
			password:     "",
			host:         "",
			dbName:       "",
			port:         0,
			want:         "sslmode=disable",
			wantRedacted: "sslmode=disable",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotRedacted := CreatePostgreSQLConfig(tt.user, tt.password, tt.host, tt.dbName, tt.port)
			if got != tt.want {
				t.Errorf("CreatePostgreSQLConfig() dsn = %q, want %q", got, tt.want)
			}
			if gotRedacted != tt.wantRedacted {
				t.Errorf("CreatePostgreSQLConfig() redactedDSN = %q, want %q", gotRedacted, tt.wantRedacted)
			}
		})
	}
}
