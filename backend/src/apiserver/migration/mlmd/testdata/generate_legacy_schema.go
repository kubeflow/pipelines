//go:build ignore

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

// Run from the pinned legacy checkout. This captures statements executed by the
// real legacy database initializer on a new, local-only disposable database.
package main

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	mysqlDriver "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	clientmanager "github.com/kubeflow/pipelines/backend/src/apiserver/client_manager"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/spf13/viper"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
)

type recorder struct {
	logger.Interface
	Statements []string
}

func (r *recorder) Trace(ctx context.Context, begin time.Time, statement func() (string, int64), err error) {
	query, _ := statement()
	if err != nil {
		return
	}
	for _, prefix := range []string{"CREATE TABLE", "CREATE INDEX", "CREATE UNIQUE INDEX", "ALTER TABLE", "INSERT INTO"} {
		if strings.HasPrefix(query, prefix) {
			r.Statements = append(r.Statements, query)
			return
		}
	}
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	driver := flag.String("driver", "", "mysql or pgx; connects only to the documented local fixture ports")
	metadataPath := flag.String("metadata", "", "path to the checked-in legacy_source.json")
	flag.Parse()
	password := os.Getenv("KFP_MLMD_FIXTURE_PASSWORD")
	if password == "" {
		panic("set KFP_MLMD_FIXTURE_PASSWORD to the disposable local database password")
	}
	for path, expected := range map[string]string{
		"go.mod": "2017f283e017f3eb68f499bee1367a3a3d1cdb3926c9bc24716a4aba977aafe6",
		"go.sum": "4083b567a2f9aed27dacfdac5d6e7df7eda7dd488e96327e121da1c1c6689e13",
		"backend/src/apiserver/client_manager/client_manager.go": "58cda38014be69d8af6961fc811aefa3dd36ec46b17e3bb0cc1c98d5106427cd",
		"backend/src/apiserver/model/task.go":                    "ccd5ccbffe0eadc61038672e0206bef56347d850a97ba4f052049fc31b2e182f",
	} {
		data, err := os.ReadFile(path)
		check(err)
		if fmt.Sprintf("%x", sha256.Sum256(data)) != expected {
			panic("source differs from pinned 2.17.3: " + path)
		}
	}
	var fixture struct {
		Commit string           `json:"source_commit"`
		Cases  map[string]int64 `json:"cases"`
		Task   model.Task       `json:"legacy_task"`
	}
	data, err := os.ReadFile(*metadataPath)
	check(err)
	check(json.Unmarshal(data, &fixture))
	if fixture.Commit != "a0733cb51b0484a6050cafee989ffe3312f808ad" || fixture.Task.RunID == "" {
		panic("expected the pinned legacy metadata fixture")
	}
	name := "kfp_mlmd_fixture_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	var rootDSN, configPrefix string
	switch *driver {
	case "mysql":
		config := mysqlDriver.NewConfig()
		config.User, config.Passwd = "root", password
		config.Net, config.Addr = "tcp", "127.0.0.1:13306"
		config.ParseTime = true
		rootDSN = config.FormatDSN()
		configPrefix = "DBConfig.MySQLConfig."
		viper.Set(configPrefix+"Port", "13306")
	case "pgx":
		connectionURL := &url.URL{
			Scheme: "postgres", Host: "127.0.0.1:15432", Path: "/postgres",
			User: url.UserPassword("root", password), RawQuery: "sslmode=disable",
		}
		rootDSN = connectionURL.String()
		configPrefix = "DBConfig.PostgreSQLConfig."
		viper.Set(configPrefix+"Port", "15432")
	default:
		panic("specify -driver=mysql or -driver=pgx")
	}
	viper.Set("DBDriverName", *driver)
	viper.Set(configPrefix+"Host", "127.0.0.1")
	viper.Set(configPrefix+"User", "root")
	viper.Set(configPrefix+"Password", password)
	viper.Set(configPrefix+"DBName", name)
	root, err := sql.Open(*driver, rootDSN)
	check(err)
	defer root.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	check(root.PingContext(ctx))
	var exists bool
	query := "SELECT EXISTS(SELECT 1 FROM information_schema.SCHEMATA WHERE SCHEMA_NAME = ?)"
	if *driver == "pgx" {
		query = "SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)"
	}
	check(root.QueryRowContext(ctx, query, name).Scan(&exists))
	if exists {
		panic("refusing to use an existing database")
	}
	// Only this generated database name can be dropped. No configured user DB is used.
	defer func() { _, err := root.Exec("DROP DATABASE " + name); check(err) }()
	trace := &recorder{Interface: logger.Default.LogMode(logger.Silent)}
	logger.Default = trace
	source := clientmanager.InitDBClient(20 * time.Second)
	defer source.Close()
	var dialector gorm.Dialector
	if *driver == "mysql" {
		dialector = mysql.New(mysql.Config{Conn: source.DB, DefaultStringSize: 255})
	} else {
		dialector = postgres.New(postgres.Config{Conn: source.DB})
	}
	db, err := gorm.Open(dialector, &gorm.Config{Logger: trace})
	check(err)
	create := func(value any) { check(db.Omit(clause.Associations).Create(value).Error) }
	create(&model.Experiment{UUID: "fixture-experiment", Name: "Default", Namespace: fixture.Task.Namespace})
	create(&model.Run{UUID: fixture.Task.RunID, Namespace: fixture.Task.Namespace, ExperimentId: "fixture-experiment"})
	fixture.Task.Name = "consumer"
	fixture.Task.PodName = "consumer-pod"
	fixture.Task.State = model.RuntimeStateSucceeded
	fixture.Task.StartedTimestamp = 1700000001
	fixture.Task.FinishedTimestamp = 1700000002
	create(&fixture.Task)
	// Keep one ordinary row and one near the legacy varchar limit. LargeText
	// values also exercise fields that cannot be represented by a short string.
	longTask := fixture.Task
	longTask.UUID = "44444444-4444-4444-8444-444444444444"
	longTask.MLMDExecutionID = fmt.Sprint(fixture.Cases["late"])
	longTask.Name = strings.Repeat("n", 240)
	longTask.PodName = strings.Repeat("p", 240)
	longTask.MLMDInputs = model.LargeText(strings.Repeat("i", 1024))
	longTask.Payload = model.LargeText(strings.Repeat("d", 1024))
	create(&longTask)
	seedTasks := []model.Task{}
	check(db.Where(map[string]any{"UUID": []string{fixture.Task.UUID, longTask.UUID}}).Find(&seedTasks).Error)
	if len(seedTasks) != 2 {
		panic("expected both legacy seed task rows")
	}
	sort.Slice(seedTasks, func(i, j int) bool { return seedTasks[i].UUID < seedTasks[j].UUID })
	var version string
	check(source.QueryRow("SELECT version()").Scan(&version))
	// Read actual catalog values, including engine-specific widths/defaults.
	columnSQL := `SELECT TABLE_NAME, COLUMN_NAME, COLUMN_TYPE, IS_NULLABLE,
		COALESCE(COLUMN_DEFAULT, '<NULL>') FROM information_schema.COLUMNS
		WHERE TABLE_SCHEMA = DATABASE() ORDER BY TABLE_NAME, ORDINAL_POSITION`
	if *driver == "pgx" {
		columnSQL = `SELECT c.relname, a.attname, format_type(a.atttypid,a.atttypmod),
			CASE WHEN a.attnotnull THEN 'NO' ELSE 'YES' END,
			COALESCE(pg_get_expr(d.adbin,d.adrelid),'<NULL>')
			FROM pg_attribute a JOIN pg_class c ON c.oid=a.attrelid
			JOIN pg_namespace n ON n.oid=c.relnamespace
			LEFT JOIN pg_attrdef d ON d.adrelid=a.attrelid AND d.adnum=a.attnum
			WHERE n.nspname='public' AND c.relkind='r' AND a.attnum>0 AND NOT a.attisdropped
			ORDER BY c.relname,a.attnum`
	}
	rows, err := source.Query(columnSQL)
	check(err)
	columns := [][5]string{}
	for rows.Next() {
		var column [5]string
		check(rows.Scan(&column[0], &column[1], &column[2], &column[3], &column[4]))
		columns = append(columns, column)
	}
	check(rows.Err())
	check(rows.Close())
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	check(encoder.Encode(map[string]any{
		"source_commit": fixture.Commit, "source_version": "2.17.3",
		"driver": *driver, "database_version": version, "statements": trace.Statements,
		"seed_tasks":    seedTasks,
		"column_fields": []string{"table", "column", "type", "nullable", "default"},
		"columns":       columns,
	}))
}
