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
	"flag"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	cm "github.com/kubeflow/pipelines/backend/src/apiserver/client_manager"
	"github.com/spf13/viper"
)

func main() {
	phase := flag.String("phase", "", "seed or verify")
	host := flag.String("host", "", "PostgreSQL host")
	port := flag.Int("port", 0, "PostgreSQL port")
	database := flag.String("database", "", "KFP database")
	user := flag.String("user", "", "PostgreSQL user")
	password := flag.String("password", "", "Disposable fixture password")
	flag.Parse()
	if (*phase != "seed" && *phase != "verify") || *host == "" || *port <= 0 || *database == "" || *user == "" || flag.NArg() != 0 {
		panic("pass --phase seed|verify, --host, --port, --database, --user and --password for the disposable database")
	}
	viper.Set("DBDriverName", "pgx")
	viper.Set("DBConfig.PostgreSQLConfig.DBName", *database)
	viper.Set("DBConfig.PostgreSQLConfig.Host", *host)
	viper.Set("DBConfig.PostgreSQLConfig.Port", *port)
	viper.Set("DBConfig.PostgreSQLConfig.User", *user)
	viper.Set("DBConfig.PostgreSQLConfig.Password", *password)
	viper.Set("DBConfig.PostgreSQLConfig.ExtraParams", map[string]string{"sslmode": "disable"})
	db, dialect, _ := cm.InitDBClient(time.Minute)
	defer db.Close()
	if dialect.Name() != "pgx" {
		panic("wrong dialect")
	}
	if *phase == "seed" {
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
	if tables != 17 {
		panic(fmt.Sprintf("expected all 17 KFP tables, found %d; inspect the retained schema", tables))
	}
	fmt.Printf("KFP InitDBClient migration + persisted experiment read/write passed; %d tables\n", tables)
}
