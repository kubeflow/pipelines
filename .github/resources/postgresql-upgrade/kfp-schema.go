// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"fmt"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	cm "github.com/kubeflow/pipelines/backend/src/apiserver/client_manager"
	"github.com/spf13/viper"
)

func main() {
	viper.Set("DBDriverName", "pgx")
	viper.Set("DBConfig.PostgreSQLConfig.DBName", "mlpipeline")
	viper.Set("DBConfig.PostgreSQLConfig.Host", "127.0.0.1")
	viper.Set("DBConfig.PostgreSQLConfig.Port", 15432)
	viper.Set("DBConfig.PostgreSQLConfig.User", "user")
	viper.Set("DBConfig.PostgreSQLConfig.Password", "password")
	viper.Set("DBConfig.PostgreSQLConfig.ExtraParams", map[string]string{"sslmode": "disable"})
	db, dialect, _ := cm.InitDBClient(time.Minute)
	defer db.Close()
	if dialect.Name() != "pgx" {
		panic("wrong dialect")
	}
	if os.Args[1] == "seed" {
		_, err := db.Exec(`INSERT INTO experiments ("UUID","Name","Description","CreatedAtInSec","LastRunCreatedAtInSec","Namespace","StorageState") VALUES ('upgrade-check','preserved experiment','before PostgreSQL patch',1,0,'kubeflow','AVAILABLE')`)
		if err != nil {
			panic(err)
		}
	}
	var name, description string
	if err := db.QueryRow(`SELECT "Name","Description" FROM experiments WHERE "UUID"='upgrade-check'`).Scan(&name, &description); err != nil {
		panic(err)
	}
	if name != "preserved experiment" || description != "before PostgreSQL patch" {
		panic("experiment changed")
	}
	if _, err := db.Exec(`UPDATE experiments SET "LastRunCreatedAtInSec"="LastRunCreatedAtInSec"+1 WHERE "UUID"='upgrade-check'`); err != nil {
		panic(err)
	}
	var tables int
	if err := db.QueryRow(`SELECT count(*) FROM information_schema.tables WHERE table_schema='public'`).Scan(&tables); err != nil {
		panic(err)
	}
	if tables < 10 {
		panic("KFP schema missing")
	}
	fmt.Printf("KFP InitDBClient migration + persisted experiment read/write passed; %d tables\n", tables)
}
