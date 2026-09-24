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

package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"time"

	"encoding/json"

	"github.com/cenkalti/backoff"
	"github.com/golang/glog"
	commonsql "github.com/kubeflow/pipelines/backend/src/apiserver/common/sql"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common/sql/dialect"
	"github.com/kubeflow/pipelines/backend/src/cache/client"
	"github.com/kubeflow/pipelines/backend/src/cache/model"
	"github.com/kubeflow/pipelines/backend/src/cache/storage"
	"github.com/kubeflow/pipelines/backend/src/common/dbcreds"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const (
	DefaultConnectionTimeout = "6m"
)

type ClientManager struct {
	db            *gorm.DB
	cacheStore    storage.ExecutionCacheStoreInterface
	k8sCoreClient client.KubernetesCoreInterface
	time          util.TimeInterface
}

func (c *ClientManager) CacheStore() storage.ExecutionCacheStoreInterface {
	return c.cacheStore
}

func (c *ClientManager) KubernetesCoreClient() client.KubernetesCoreInterface {
	return c.k8sCoreClient
}

func (c *ClientManager) Close() {
	sqlDB, err := c.db.DB()
	if err != nil {
		log.Printf("Failed to retrieve underlying sql.DB: %v", err)
		return
	}
	sqlDB.Close()
}

func (c *ClientManager) init(params WhSvrDBParameters, clientParams util.ClientParameters) {
	timeoutDuration, _ := time.ParseDuration(DefaultConnectionTimeout)
	db, d := initDBClient(params, timeoutDuration)

	c.time = util.NewRealTime()
	c.db = db
	c.cacheStore = storage.NewExecutionCacheStore(db, c.time, d)
	c.k8sCoreClient = client.CreateKubernetesCoreOrFatal(timeoutDuration, clientParams)
}

func initDBClient(params WhSvrDBParameters, initConnectionTimeout time.Duration) (*gorm.DB, dialect.DBDialect) {
	driverName := params.dbDriver

	dbDialect := dialect.NewDBDialect(driverName)

	settings := dbSettings(params)
	util.TerminateIfError(settings.Validate())
	// The standard logger, not glog: the cache server's entrypoint passes no
	// -logtostderr, and glog writes only ERROR and above to stderr.
	if warning := settings.IgnoredProviderWarning(); warning != "" {
		log.Print(warning)
	}
	if warning := settings.IgnoredTLSWarning(); warning != "" {
		log.Print(warning)
	}
	log.Print(settings.Describe(driverName))

	var dialector gorm.Dialector
	if settings.Enabled {
		dialector = initDriverWithProvider(params, settings, dbDialect, initConnectionTimeout)
	} else {
		arg := initDBDriver(params, initConnectionTimeout)
		switch driverName {
		case "mysql":
			// DefaultStringSize dictates non-indexable string fields map to VARCHAR(255) for backward compatibility with GORM v1.
			dialector = mysql.New(mysql.Config{
				DSN:               arg,
				DefaultStringSize: 255,
			})
		case "pgx":
			dialector = postgres.Open(arg)
		default:
			glog.Fatalf("Driver %v is not supported", driverName)
		}
	}

	// db is safe for concurrent use by multiple goroutines
	// and maintains its own pool of idle connections.
	gormDB, err := gorm.Open(dialector, &gorm.Config{})
	util.TerminateIfError(err)

	// Create table
	err = gormDB.AutoMigrate(&model.ExecutionCache{})
	if err != nil {
		glog.Fatalf("Failed to initialize the databases.")
	}

	err = gormDB.Migrator().AlterColumn(&model.ExecutionCache{}, "ExecutionOutput")
	if err != nil {
		glog.Fatalf("Failed to update the execution output type. Error: %s", err)
	}
	err = gormDB.Migrator().AlterColumn(&model.ExecutionCache{}, "ExecutionTemplate")
	if err != nil {
		glog.Fatalf("Failed to update the execution template type. Error: %s", err)
	}

	return gormDB, dbDialect
}

// parseDBExtraParams parses the JSON-encoded --db_extra_params flag value into
// a key-value map. An empty input yields an empty map; malformed JSON returns
// an error so startup can fail fast instead of silently ignoring the operator's
// configuration (for example, intended TLS options).
func parseDBExtraParams(dbExtraParams string) (map[string]string, error) {
	extraParams := map[string]string{}
	if dbExtraParams == "" {
		return extraParams, nil
	}
	if err := json.Unmarshal([]byte(dbExtraParams), &extraParams); err != nil {
		return nil, fmt.Errorf("failed to parse db_extra_params %q as JSON: %w", dbExtraParams, err)
	}
	return extraParams, nil
}

func initDBDriver(params WhSvrDBParameters, initConnectionTimeout time.Duration) string {
	switch params.dbDriver {
	case "mysql":
		mysqlExtraParams, err := parseDBExtraParams(params.dbExtraParams)
		if err != nil {
			glog.Fatalf("Failed to parse MySQL extra params: %v", err)
		}
		mysqlConfig := commonsql.CreateMySQLConfig(
			params.dbUser,
			params.dbPwd,
			params.dbHost,
			params.dbPort,
			"",
			params.dbGroupConcatMaxLen,
			mysqlExtraParams,
		)

		var db *sql.DB
		var operation = func() error {
			db, err = sql.Open(params.dbDriver, mysqlConfig.FormatDSN())
			if err != nil {
				return err
			}
			return nil
		}
		b := backoff.NewExponentialBackOff()
		b.MaxElapsedTime = initConnectionTimeout
		err = backoff.Retry(operation, b)

		defer db.Close()
		util.TerminateIfError(err)

		// Create database if not exist
		dbName := params.dbName
		drvDialect := dialect.NewDBDialect(params.dbDriver)
		operation = func() error {
			_, err = db.Exec(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s", drvDialect.QuoteIdentifier(dbName)))
			if err != nil {
				return err
			}
			return nil
		}
		b = backoff.NewExponentialBackOff()
		b.MaxElapsedTime = initConnectionTimeout
		err = backoff.Retry(operation, b)
		util.TerminateIfError(err)

		operation = func() error {
			_, err = db.Exec(fmt.Sprintf("USE %s", drvDialect.QuoteIdentifier(dbName)))
			if err != nil {
				return err
			}
			return nil
		}
		b = backoff.NewExponentialBackOff()
		b.MaxElapsedTime = initConnectionTimeout
		err = backoff.Retry(operation, b)

		util.TerminateIfError(err)
		mysqlConfig.DBName = dbName
		// When updating, return rows matched instead of rows affected. This counts rows that are being
		// set as the same values as before. If updating using a primary key and rows matched is 0, then
		// it means this row is not found.
		// Config reference: https://github.com/go-sql-driver/mysql#clientfoundrows
		mysqlConfig.ClientFoundRows = true
		return mysqlConfig.FormatDSN()
	case "pgx":
		port, err := strconv.Atoi(params.dbPort)
		if err != nil {
			glog.Fatalf("Invalid port for PostgreSQL: %v", err)
		}
		pgxExtraParams, err := parseDBExtraParams(params.dbExtraParams)
		if err != nil {
			glog.Fatalf("Failed to parse PostgreSQL extra params: %v", err)
		}
		// Connect without target DB first
		cfgNoDB, _, err := commonsql.CreatePostgreSQLConfig(params.dbUser, params.dbPwd, params.dbHost, "postgres", uint16(port), pgxExtraParams)
		if err != nil {
			glog.Fatalf("Failed to create PostgreSQL config: %v", err)
		}
		db, err := sql.Open(params.dbDriver, cfgNoDB.ConnString())
		if err != nil {
			glog.Fatalf("Failed to open PostgreSQL connection: %v", err)
		}
		var operation = func() error {
			return db.Ping()
		}
		b := backoff.NewExponentialBackOff()
		b.MaxElapsedTime = initConnectionTimeout
		err = backoff.Retry(operation, b)
		if err != nil {
			db.Close()
			glog.Fatalf("Failed to ping PostgreSQL: %v", err)
		}

		// Create database, ignoring "already exists" error
		pgDialect := dialect.NewDBDialect(params.dbDriver)
		_, err = db.Exec(fmt.Sprintf("CREATE DATABASE %s", pgDialect.QuoteIdentifier(params.dbName)))
		if err != nil && !pgDialect.IsDuplicateDatabaseError(err) {
			db.Close()
			glog.Fatalf("Failed to create database: %v", err)
		}
		db.Close()

		// Return DSN with target DB
		cfg, _, err := commonsql.CreatePostgreSQLConfig(params.dbUser, params.dbPwd, params.dbHost, params.dbName, uint16(port), pgxExtraParams)
		if err != nil {
			glog.Fatalf("Failed to create PostgreSQL config: %v", err)
		}
		return cfg.ConnString()
	default:
		glog.Fatalf("Driver %v is not supported", params.dbDriver)
	}
	return ""
}

func NewClientManager(params WhSvrDBParameters, clientParams util.ClientParameters) ClientManager {
	clientManager := ClientManager{}
	clientManager.init(params, clientParams)

	return clientManager
}

// dbSettings resolves the credential-provider configuration from flags.
// Everything after this point is shared with the API server.
func dbSettings(params WhSvrDBParameters) dbcreds.Settings {
	return dbcreds.Settings{
		Enabled:          dbcreds.ParseEnabled(params.dbProviderEnabled),
		ProviderName:     params.dbCredentialProvider,
		CABundlePath:     params.dbTLSCAPath,
		ProviderSettings: params.dbProviderSettings,
	}
}

// initDriverWithProvider is the opt-in counterpart of initDBDriver. It obtains
// the connection from a credential provider, which allows a credential that is
// regenerated per connection and a verified TLS connection. It returns a
// dialector over a live handle rather than a DSN, because a provider's
// credential cannot be written into a connection string.
//
// The connection lifecycle itself is dbcreds.Bootstrap, shared with the API
// server. What stays here is what is genuinely this binary's: the flags the
// configuration comes from, the standard logger, and the GORM dialector.
func initDriverWithProvider(params WhSvrDBParameters, settings dbcreds.Settings, dbDialect dialect.DBDialect, initConnectionTimeout time.Duration) gorm.Dialector {
	// Warn before building, so that a password left behind is reported even if
	// the provider itself then fails to build.
	if warning := settings.IgnoredPasswordWarning(params.dbPwd); warning != "" {
		log.Print(warning)
	}
	provider, err := settings.NewProvider(params.dbPwd)
	util.TerminateIfError(err)

	db, warning, err := dbcreds.Bootstrap(context.Background(), provider, targetFromParams(params), dbcreds.BootstrapOptions{
		DBName:          params.dbName,
		QuoteIdentifier: dbDialect.QuoteIdentifier,
		// The dialect knows each engine's duplicate-database code, which a
		// substring of the message only approximates. EnsureDatabase probes
		// when creation is refused for lack of privilege instead.
		Tolerate: func(err error) error {
			if err == nil || dbDialect.IsDuplicateDatabaseError(err) {
				return nil
			}
			return err
		},
		Timeout: initConnectionTimeout,
	})
	util.TerminateIfError(err)
	if warning != "" {
		log.Print(warning)
	}

	switch params.dbDriver {
	case mysqlDBDriverDefault:
		// DefaultStringSize dictates non-indexable string fields map to VARCHAR(255) for backward compatibility with GORM v1.
		return mysql.New(mysql.Config{Conn: db, DefaultStringSize: 255})
	default:
		return postgres.New(postgres.Config{Conn: db})
	}
}

// targetFromParams describes the database endpoint to connect to. The resulting
// parameters match those the cache server has always used, so the connection is
// configured identically whichever provider supplies the credential.
//
// The flags are the only argument, because they are the only source: the
// settings this also needs are themselves derived from them, so taking both
// would be taking the same configuration twice and inviting the two copies to
// disagree.
func targetFromParams(params WhSvrDBParameters) dbcreds.Target {
	connectionParams := map[string]string{}
	// group_concat_max_len is a MySQL session variable; the flag documents
	// itself as MySQL-only and PostgreSQL would reject it as a setting.
	if params.dbDriver == mysqlDBDriverDefault {
		connectionParams["group_concat_max_len"] = params.dbGroupConcatMaxLen
	}
	extraParams, err := parseDBExtraParams(params.dbExtraParams)
	if err != nil {
		glog.Fatalf("Failed to parse db extra params: %v", err)
	}
	for key, value := range extraParams {
		connectionParams[key] = value
	}

	return dbcreds.Target{
		Driver: params.dbDriver,
		Host:   params.dbHost,
		Port:   params.dbPort,
		User:   params.dbUser,
		Params: connectionParams,
		TLS:    dbSettings(params).TLSOptions(),
	}
}
