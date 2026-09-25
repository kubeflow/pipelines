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

// Package dbcreds builds the driver connectors KFP uses to reach its database.
//
// A Provider owns credential acquisition for one authentication mechanism. It
// returns a driver.Connector rather than a password so that mechanisms which do
// not authenticate with a string, such as client certificates, can be added
// without changing this interface.
//
// # Adding a provider
//
// A provider for another cloud lives in its own package under this one and
// needs no changes to the API server or the cache server:
//
//  1. Implement Factory and Provider. New receives the operator's settings;
//     read what you need with Config.Setting, and fail there rather than at
//     connect time if something required is missing. RequiresTLS answers
//     whether the credential is safe to send over an unverified connection;
//     a token presented as a password is not, so such a provider returns true
//     and the configuration is rejected before any call to the cloud.
//  2. Build the connector with MySQLConfig and MySQLConnector, or
//     PostgreSQLConfig and PostgreSQLConnector. These are the only supported
//     way to construct one, so that every mechanism reaches the database with
//     the same base configuration. Set the credential on the configuration they
//     return; do not assemble a connection string yourself.
//  3. For a credential that expires, install the driver's per-connection hook
//     rather than resolving the credential in Connector. mysql.BeforeConnect
//     and stdlib.OptionBeforeConnect both run for every new connection with
//     their own copy of the configuration, which is what keeps an expired
//     credential out of the pool.
//  4. Register the factory from init, and blank-import the package from
//     dbcreds/all.
//
// Anything cloud-specific belongs in the provider package. Nothing in this
// package refers to a particular cloud, and it should stay that way.
package dbcreds

import (
	"context"
	"database/sql/driver"
	"fmt"
)

// Supported values for Target.Driver. These match the driver names KFP accepts
// in the DBDriverName configuration key.
const (
	DriverMySQL      = "mysql"
	DriverPostgreSQL = "pgx"
)

// Target describes the database endpoint a connector must reach. It is
// engine-neutral; a Provider inspects Driver to decide what to build.
type Target struct {
	Driver string
	Host   string
	Port   string
	User   string
	DBName string

	// Params holds driver-specific connection parameters. For MySQL these are
	// DSN parameters, and they override the defaults set by MySQLConfig.
	Params map[string]string

	// TLS configures server certificate verification. Nil leaves the driver's
	// default behavior, which for MySQL is an unencrypted connection.
	TLS *TLSOptions
}

// Well-known Settings keys. Providers are free to read others, but these have
// the same meaning everywhere they are used.
const (
	// SettingRegion is the cloud region a provider signs credentials in.
	SettingRegion = "region"
)

// Config carries the resolved settings a Factory needs to build a Provider.
type Config struct {
	// Password authenticates providers that use a fixed credential.
	Password string

	// Settings carries provider-specific configuration. Operators populate it
	// from DB_CREDENTIAL_PROVIDER_SETTINGS, so a new provider can take the
	// configuration it needs without changes outside its own package.
	Settings map[string]string
}

// Setting returns the named setting, or the empty string when it is absent.
func (c Config) Setting(name string) string {
	return c.Settings[name]
}

// Provider builds the driver connector used to reach a database.
type Provider interface {
	// Name returns the identifier this provider is registered under.
	Name() string

	// Connector returns a connector for the target. Implementations that
	// acquire a credential per connection install it as a driver hook rather
	// than resolving it here, so that every new connection gets a fresh one.
	Connector(ctx context.Context, t Target) (driver.Connector, error)
}

// Factory constructs a Provider from resolved configuration. Implementations
// register themselves from init.
type Factory interface {
	Name() string
	New(cfg Config) (Provider, error)

	// RequiresTLS reports whether the provider's credential is unsafe to send
	// over an unverified connection, so that the requirement is checked while
	// validating configuration rather than on the way to the database.
	//
	// It is part of the interface rather than an optional one a factory may
	// implement, because silence would mean "no TLS needed" -- failing open on
	// a security question, and on the method an author is most likely to skip.
	// The compiler asks instead.
	RequiresTLS() bool
}

// UnsupportedDriverError reports a driver a provider cannot serve. Providers
// outside this package use it so the wording does not drift.
func UnsupportedDriverError(provider, driverName string) error {
	return fmt.Errorf("database credential provider %q does not support driver %q; use %q or %q", provider, driverName, DriverMySQL, DriverPostgreSQL)
}
