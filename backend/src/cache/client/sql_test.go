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
		{
			name: "with password and database name",
			args: args{
				user:                   "pipeline_user",
				password:               "secretpassword",
				host:                   "mysql-service.kubeflow",
				port:                   "3307",
				dbName:                 "cachedb",
				mysqlGroupConcatMaxLen: "4096",
				mysqlExtraParams:       nil,
			},
			want: &mysql.Config{
				User:                 "pipeline_user",
				Passwd:               "secretpassword",
				Net:                  "tcp",
				Addr:                 "mysql-service.kubeflow:3307",
				DBName:               "cachedb",
				Params:               map[string]string{"charset": "utf8", "parseTime": "True", "loc": "Local", "group_concat_max_len": "4096"},
				AllowNativePasswords: true,
			},
		},
		{
			name: "extra params override defaults",
			args: args{
				user:                   "root",
				host:                   "mysql",
				port:                   "3306",
				mysqlGroupConcatMaxLen: "1024",
				mysqlExtraParams:       map[string]string{"charset": "utf8mb4"},
			},
			want: &mysql.Config{
				User:                 "root",
				Net:                  "tcp",
				Addr:                 "mysql:3306",
				Params:               map[string]string{"charset": "utf8mb4", "parseTime": "True", "loc": "Local", "group_concat_max_len": "1024"},
				AllowNativePasswords: true,
			},
		},
		{
			name: "multiple extra parameters",
			args: args{
				user:                   "root",
				host:                   "mysql",
				port:                   "3306",
				mysqlGroupConcatMaxLen: "1024",
				mysqlExtraParams:       map[string]string{"tls": "skip-verify", "timeout": "30s", "readTimeout": "30s"},
			},
			want: &mysql.Config{
				User:                 "root",
				Net:                  "tcp",
				Addr:                 "mysql:3306",
				Params:               map[string]string{"charset": "utf8", "parseTime": "True", "loc": "Local", "group_concat_max_len": "1024", "tls": "skip-verify", "timeout": "30s", "readTimeout": "30s"},
				AllowNativePasswords: true,
			},
		},
		{
			name: "empty extra params map",
			args: args{
				user:                   "root",
				host:                   "localhost",
				port:                   "3306",
				mysqlGroupConcatMaxLen: "1024",
				mysqlExtraParams:       map[string]string{},
			},
			want: &mysql.Config{
				User:                 "root",
				Net:                  "tcp",
				Addr:                 "localhost:3306",
				Params:               map[string]string{"charset": "utf8", "parseTime": "True", "loc": "Local", "group_concat_max_len": "1024"},
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
		name          string
		user          string
		password      string
		host          string
		dbName        string
		port          uint16
		extraParams   map[string]string
		wantTLSNotNil bool
		wantRedacted  string
	}{
		{
			name:         "all fields populated",
			user:         "admin",
			password:     "secret",
			host:         "postgres-host",
			dbName:       "mlpipeline",
			port:         5432,
			wantRedacted: "host=postgres-host port=5432 user=admin password=*** database=mlpipeline sslmode=disable",
		},
		{
			name:         "empty password",
			user:         "admin",
			password:     "",
			host:         "postgres-host",
			dbName:       "mlpipeline",
			port:         5432,
			wantRedacted: "host=postgres-host port=5432 user=admin password=*** database=mlpipeline sslmode=disable",
		},
		{
			name:         "password with spaces",
			user:         "admin",
			password:     "foo bar",
			host:         "postgres-host",
			dbName:       "mlpipeline",
			port:         5432,
			wantRedacted: "host=postgres-host port=5432 user=admin password=*** database=mlpipeline sslmode=disable",
		},
		{
			name:         "password with single quote",
			user:         "admin",
			password:     "pa'ss",
			host:         "postgres-host",
			dbName:       "mlpipeline",
			port:         5432,
			wantRedacted: "host=postgres-host port=5432 user=admin password=*** database=mlpipeline sslmode=disable",
		},
		{
			name:         "password with backslash",
			user:         "admin",
			password:     `p\as`,
			host:         "postgres-host",
			dbName:       "mlpipeline",
			port:         5432,
			wantRedacted: "host=postgres-host port=5432 user=admin password=*** database=mlpipeline sslmode=disable",
		},
		{
			name:         "nil extra params defaults to sslmode=disable",
			user:         "admin",
			password:     "secret",
			host:         "postgres-host",
			dbName:       "mlpipeline",
			port:         5432,
			extraParams:  nil,
			wantRedacted: "host=postgres-host port=5432 user=admin password=*** database=mlpipeline sslmode=disable",
		},
		{
			name:         "empty extra params defaults to sslmode=disable",
			user:         "admin",
			password:     "secret",
			host:         "postgres-host",
			dbName:       "mlpipeline",
			port:         5432,
			extraParams:  map[string]string{},
			wantRedacted: "host=postgres-host port=5432 user=admin password=*** database=mlpipeline sslmode=disable",
		},
		{
			name:          "sslmode=require overrides default",
			user:          "admin",
			password:      "secret",
			host:          "postgres-host",
			dbName:        "mlpipeline",
			port:          5432,
			extraParams:   map[string]string{"sslmode": "require"},
			wantTLSNotNil: true,
			wantRedacted:  "host=postgres-host port=5432 user=admin password=*** database=mlpipeline sslmode=require",
		},
		{
			name:         "application_name with special characters",
			user:         "admin",
			password:     "secret",
			host:         "postgres-host",
			dbName:       "mlpipeline",
			port:         5432,
			extraParams:  map[string]string{"application_name": "kfp pipeline"},
			wantRedacted: "host=postgres-host port=5432 user=admin password=*** database=mlpipeline sslmode=disable",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, gotRedacted, err := CreatePostgreSQLConfig(tt.user, tt.password, tt.host, tt.dbName, tt.port, tt.extraParams)
			if err != nil {
				t.Fatalf("CreatePostgreSQLConfig() returned unexpected error: %v", err)
			}
			if cfg == nil {
				t.Fatal("CreatePostgreSQLConfig() returned nil ConnConfig")
			}
			if cfg.Host != tt.host {
				t.Errorf("cfg.Host = %q, want %q", cfg.Host, tt.host)
			}
			if cfg.Port != tt.port {
				t.Errorf("cfg.Port = %d, want %d", cfg.Port, tt.port)
			}
			if cfg.User != tt.user {
				t.Errorf("cfg.User = %q, want %q", cfg.User, tt.user)
			}
			if cfg.Password != tt.password {
				t.Errorf("cfg.Password = %q, want %q", cfg.Password, tt.password)
			}
			if cfg.Database != tt.dbName {
				t.Errorf("cfg.Database = %q, want %q", cfg.Database, tt.dbName)
			}
			if tt.wantTLSNotNil && cfg.TLSConfig == nil {
				t.Errorf("cfg.TLSConfig = nil, want non-nil for sslmode=%s", tt.extraParams["sslmode"])
			}
			if !tt.wantTLSNotNil && cfg.TLSConfig != nil {
				t.Errorf("cfg.TLSConfig = %v, want nil (sslmode=disable)", cfg.TLSConfig)
			}
			if gotRedacted != tt.wantRedacted {
				t.Errorf("CreatePostgreSQLConfig() redactedDSN = %q, want %q", gotRedacted, tt.wantRedacted)
			}
		})
	}
}
