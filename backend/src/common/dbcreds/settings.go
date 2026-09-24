// Copyright 2026 The Kubeflow Authors
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

package dbcreds

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/go-sql-driver/mysql"
	"github.com/jackc/pgx/v5/pgconn"
)

// Settings is the resolved credential-provider configuration a binary supplies.
// Each binary reads it from its own source -- the API server from viper, the
// cache server from flags -- and everything after that is shared, so the two
// cannot drift apart.
//
// Nothing here logs or terminates: the two binaries disagree on both, so the
// methods return errors and strings and let the caller decide.
//
// The database password is deliberately not a field. It comes from a Secret,
// while everything here comes from an operator's declared configuration, and
// the two binaries learn it at different moments -- the cache server has one
// password up front, the API server's depends on which engine it is about to
// use. A field would be empty for part of this value's life on one of them,
// so the methods that need the password take it as an argument instead.
type Settings struct {
	// Enabled is the switch. When false the caller must build the connection
	// the way it did before credential providers existed.
	Enabled bool

	// ProviderName selects the provider. Required when Enabled.
	ProviderName string

	// CABundlePath verifies the database server certificate. Empty leaves the
	// connection unencrypted.
	CABundlePath string

	// ProviderSettings is provider-specific configuration, as a JSON object.
	ProviderSettings string
}

// ParseEnabled interprets a switch that reaches the process as a string. The
// value comes from a ConfigMap, so it can be empty or an unexpanded reference
// when the key is absent; neither may be fatal.
func ParseEnabled(value string) bool {
	enabled, err := strconv.ParseBool(strings.TrimSpace(value))
	return err == nil && enabled
}

// Validate rejects a configuration that cannot produce a provider. Callers run
// it before logging anything, so an operator reading the failure is not first
// shown a connection summary naming a provider that does not exist.
func (s Settings) Validate() error {
	if !s.Enabled {
		return nil
	}
	if s.ProviderName == "" {
		return fmt.Errorf("a database credential provider is enabled but none is named; set it to one of: %v", RegisteredNames())
	}
	// Reported here rather than on the way to the database, so that an operator
	// who has configured neither is told about both at once instead of fixing
	// one, redeploying, and meeting the other.
	if s.CABundlePath == "" && factoryRequiresTLS(s.ProviderName) {
		return fmt.Errorf("the %q credential provider requires a verified connection; set DB_TLS_CA_PATH to the database CA bundle", s.ProviderName)
	}
	return nil
}

// Describe renders how the connection is authenticated and protected. Without it
// the active mode is visible only by reading a pod's configuration, which is the
// wrong place to look when a rollout misbehaves.
func (s Settings) Describe(driverName string) string {
	if !s.Enabled {
		return fmt.Sprintf("DB connection: driver=%s, credentials=configured password, TLS=disabled", driverName)
	}
	transport := "disabled"
	if s.CABundlePath != "" {
		transport = "verified against " + s.CABundlePath
	}
	return fmt.Sprintf("DB connection: driver=%s, credentials=%q provider, TLS=%s", driverName, s.ProviderName, transport)
}

// IgnoredPasswordWarning reports a configured password that will not be used,
// or an empty string when there is nothing to say.
func (s Settings) IgnoredPasswordWarning(password string) string {
	if s.Enabled && s.ProviderName != StaticProviderName && password != "" {
		return fmt.Sprintf("a %q credential provider is enabled, so the configured database password is ignored; it can be removed from the database Secret once a rollback to password authentication is no longer wanted", s.ProviderName)
	}
	return ""
}

// IgnoredProviderWarning reports a provider that is named but not switched on.
//
// Naming the provider and forgetting the switch is the likeliest way to
// misconfigure this, and it fails towards the stored password -- the very thing
// the operator was moving away from. Describe already says which mode is in
// force, but a line reading "credentials=configured password" is easy to pass
// over when you believe you configured otherwise, so the contradiction is
// stated outright.
func (s Settings) IgnoredProviderWarning() string {
	if !s.Enabled && s.ProviderName != "" {
		return fmt.Sprintf("the %q database credential provider is configured but not enabled, so the configured password is used instead; enable the credential provider to authenticate with it", s.ProviderName)
	}
	return ""
}

// IgnoredTLSWarning reports a CA bundle that will not take effect.
func (s Settings) IgnoredTLSWarning() string {
	if !s.Enabled && s.CABundlePath != "" {
		return "a database CA bundle is configured but no credential provider is enabled, so it has no effect"
	}
	return ""
}

// TLSOptions returns nil when no CA bundle is configured, which leaves the
// connection unencrypted as it has always been.
func (s Settings) TLSOptions() *TLSOptions {
	if s.CABundlePath == "" {
		return nil
	}
	return &TLSOptions{CABundlePath: s.CABundlePath}
}

// NewProvider builds the configured provider. The password reaches only a
// provider that authenticates with one; the rest ignore it.
func (s Settings) NewProvider(password string) (Provider, error) {
	settings := map[string]string{}
	if raw := strings.TrimSpace(s.ProviderSettings); raw != "" {
		if err := json.Unmarshal([]byte(raw), &settings); err != nil {
			return nil, fmt.Errorf("the credential provider settings are not a JSON object: %w", err)
		}
	}
	return NewProvider(s.ProviderName, Config{Password: password, Settings: settings})
}

// Open builds the connector for the target and wraps it in a pooled handle. As
// with sql.Open, no connection is made until the handle is first used.
func Open(ctx context.Context, provider Provider, target Target) (*sql.DB, error) {
	connector, err := provider.Connector(ctx, target)
	if err != nil {
		return nil, err
	}
	return sql.OpenDB(connector), nil
}

// Probe reports whether the named database exists and is usable, using the same
// credentials the connection itself will use.
func Probe(ctx context.Context, provider Provider, target Target, dbName string) error {
	target.DBName = dbName
	db, err := Open(ctx, provider, target)
	if err != nil {
		return err
	}
	defer db.Close()
	return db.Ping()
}

// permissionDenied reports whether the database refused the statement for lack
// of privilege. Only then is a reachable database a reason to continue: any
// other failure -- a syntax error, a full disk, an unreachable server, a
// credential the server rejected -- is not answered by the database merely
// existing.
//
// The engines' error codes are matched rather than their messages. MySQL's
// privilege refusal (1044, "Access denied for user ... to database ...") and
// its authentication failure (1045, "Access denied for user ... (using
// password: YES)") both say "access denied", so a substring would treat a wrong
// password or a bad token as a privilege problem and tell the operator to grant
// permissions they already have.
func permissionDenied(err error) bool {
	if err == nil {
		return false
	}
	// MySQL 1044 is ER_DBACCESS_DENIED_ERROR: the user exists and
	// authenticated, but may not touch this database.
	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) {
		return mysqlErr.Number == 1044
	}
	// PostgreSQL 42501 is insufficient_privilege, which is what "permission
	// denied to create database" carries.
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "42501"
	}
	return false
}

// EnsureDatabase creates dbName, or accepts a database that already exists and
// is usable when creation is refused for lack of privilege. It returns a warning
// describing that second outcome, so the caller can record that it proceeded on
// a weaker guarantee than creating the database itself.
//
// The two engines report a refused creation differently. MySQL checks existence
// before privilege, so it says "database exists" and tolerate covers it.
// PostgreSQL checks privilege first, so it says "permission denied" whether or
// not the database is there -- the error alone cannot distinguish a
// pre-provisioned database from a missing one. Rather than guess from the
// message, ask the database: probe opens the target and runs a trivial query.
//
// Probing inside the retry keeps the timeout unchanged. A user without CREATE
// DATABASE on an existing database succeeds on the first attempt instead of
// retrying until the deadline, and a genuinely missing database still fails.
func EnsureDatabase(dbName string, timeout time.Duration, create func() error, tolerate func(error) error, probe func() error) (string, error) {
	b := backoff.NewExponentialBackOff()
	b.MaxElapsedTime = timeout

	var warning string
	err := backoff.Retry(func() error {
		warning = ""
		createErr := tolerate(create())
		if createErr == nil {
			return nil
		}
		// Reachability is a weaker guarantee than having created the database,
		// so it is only accepted for the failure it actually explains.
		if !permissionDenied(createErr) {
			return createErr
		}
		if probeErr := probe(); probeErr != nil {
			return fmt.Errorf("could not create database %q (%w) and could not reach it (%v); "+
				"create the database, or grant the user permission to create it", dbName, createErr, probeErr)
		}
		warning = fmt.Sprintf("database %q was not created (%v); continuing because it already exists and is reachable", dbName, createErr)
		return nil
	}, b)
	return warning, err
}

// BootstrapOptions carries what Bootstrap needs beyond the target itself.
type BootstrapOptions struct {
	// DBName is the application database to create and connect to.
	DBName string

	// QuoteIdentifier quotes DBName for the CREATE DATABASE statement. Each
	// binary already holds its engine's dialect, so the quoting rule is passed
	// in rather than reimplemented here.
	QuoteIdentifier func(string) string

	// Tolerate turns an expected creation failure into success -- a database
	// that already exists. It is the caller's because the statement's exact
	// wording, and therefore which failures are expected, is the caller's.
	Tolerate func(error) error

	// Timeout bounds the retry that also waits for the server to accept
	// connections, because sql.Open does not dial.
	Timeout time.Duration
}

// Bootstrap opens the application database through provider, creating it first
// if it does not exist, and returns a pooled handle to it.
//
// This is the whole connection lifecycle: open a connection that does not name
// the database, create the database, then open the one the caller will use. It
// lives here rather than in each binary because it is identical for both and is
// the part most likely to drift -- the binaries differ only in where their
// configuration comes from, how they log, and how they wrap the handle for
// GORM, and all three stay with them.
//
// The returned string is a warning worth surfacing, or empty.
func Bootstrap(ctx context.Context, provider Provider, target Target, options BootstrapOptions) (*sql.DB, string, error) {
	bootstrapTarget := target
	switch target.Driver {
	case DriverMySQL:
		// MySQL connects without naming a database, which is what the
		// bootstrap connection wants: the database may not exist yet.
		bootstrapTarget.DBName = ""
	case DriverPostgreSQL:
		// PostgreSQL has no connectionless state, so it bootstraps through the
		// maintenance database.
		bootstrapTarget.DBName = "postgres"
	default:
		return nil, "", UnsupportedDriverError(provider.Name(), target.Driver)
	}

	bootstrap, err := Open(ctx, provider, bootstrapTarget)
	if err != nil {
		return nil, "", err
	}
	defer bootstrap.Close()

	quoted := options.QuoteIdentifier(options.DBName)
	warning, err := EnsureDatabase(options.DBName, options.Timeout,
		func() error {
			_, execErr := bootstrap.Exec(fmt.Sprintf("CREATE DATABASE %s", quoted))
			return execErr
		},
		options.Tolerate,
		func() error { return Probe(ctx, provider, bootstrapTarget, options.DBName) })
	if err != nil {
		return nil, "", err
	}

	applicationTarget := target
	applicationTarget.DBName = options.DBName
	if target.Driver == DriverMySQL {
		// When updating, return rows matched instead of rows affected. This counts rows that are being
		// set as the same values as before. If updating using a primary key and rows matched is 0, then
		// it means this row is not found.
		// Config reference: https://github.com/go-sql-driver/mysql#clientfoundrows
		//
		// Copied rather than assigned into, so that Bootstrap does not mutate
		// the caller's map.
		params := make(map[string]string, len(target.Params)+1)
		for key, value := range target.Params {
			params[key] = value
		}
		params["clientFoundRows"] = "true"
		applicationTarget.Params = params
	}

	db, err := Open(ctx, provider, applicationTarget)
	if err != nil {
		return nil, warning, err
	}
	return db, warning, nil
}
