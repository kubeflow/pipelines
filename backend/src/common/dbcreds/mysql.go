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
	"database/sql/driver"
	"fmt"
	"net"

	"github.com/go-sql-driver/mysql"
)

// MySQLConfig builds the driver configuration KFP uses to reach MySQL.
//
// Every provider must build its configuration here so that connection
// parameters do not drift between authentication mechanisms. Target.Params
// overrides the defaults, which is how operators reach driver settings KFP does
// not expose directly.
//
// The returned configuration carries no password. Providers set the credential
// themselves, so it never passes through a DSN string that could be logged. It
// does carry Target.TLS, so providers do not apply transport security
// themselves.
func MySQLConfig(t Target) (*mysql.Config, error) {
	if t.Driver != DriverMySQL {
		return nil, fmt.Errorf("MySQLConfig called with driver %q; use %q", t.Driver, DriverMySQL)
	}
	if t.Host == "" {
		return nil, fmt.Errorf("database host is empty; set the host for driver %q", DriverMySQL)
	}

	params := map[string]string{
		"charset":   "utf8",
		"parseTime": "True",
		"loc":       "Local",
	}
	for key, value := range t.Params {
		params[key] = value
	}

	// JoinHostPort brackets IPv6 literals, which the DSN grammar requires.
	addr := net.JoinHostPort(t.Host, t.Port)

	requested := &mysql.Config{
		User:                 t.User,
		Net:                  "tcp",
		Addr:                 addr,
		Params:               params,
		DBName:               t.DBName,
		AllowNativePasswords: true,
	}

	// Normalize through the DSN. Parameters the driver recognizes -- charset,
	// parseTime, loc, and anything an operator supplies through ExtraParams --
	// must land on their typed fields; left in Params they would instead be
	// issued as SET statements at connect time. This is the same normalization
	// sql.Open performs, so a connection opened from the result is configured
	// exactly as it is today.
	config, err := mysql.ParseDSN(requested.FormatDSN())
	if err != nil {
		return nil, fmt.Errorf("invalid MySQL connection parameters for %s: %w", requested.Addr, err)
	}

	// Transport security is resolved here rather than by each provider, so that a
	// provider cannot omit it and connect unencrypted while the operator believes
	// TLS is on. This is also where PostgreSQLConfig resolves TLS, so the two
	// engines behave the same way. It must follow ParseDSN, which returns a fresh
	// configuration.
	if t.TLS != nil {
		// The round-trip above may have set TLS fields from operator-supplied
		// parameters. Assigning config.TLS alone is not enough: tls=preferred
		// also sets AllowFallbackToPlaintext, which lets the driver drop TLS
		// silently when the server does not offer it, and that would defeat a
		// provider's TLS requirement while the credential is still sent. Reject
		// the conflict rather than resolve it silently, so an operator carrying
		// a legacy parameter is told instead of being quietly downgraded.
		if config.TLSConfig != "" {
			return nil, fmt.Errorf("database TLS is configured by DB_TLS_CA_PATH, but the connection parameters also set tls=%q; remove it",
				config.TLSConfig)
		}
		if config.AllowFallbackToPlaintext {
			return nil, fmt.Errorf("database TLS is configured by DB_TLS_CA_PATH, but the connection parameters also allow a plaintext fallback; remove allowFallbackToPlaintext")
		}
		if config.TLS, err = TLSConfig(t.TLS, t.Host); err != nil {
			return nil, err
		}
	}
	return config, nil
}

// MySQLConnector applies opts to config and returns the resulting connector.
//
// Options are the driver's extension point for behavior that is not a settable
// field: a provider that refreshes its credential per connection installs
// mysql.BeforeConnect here.
func MySQLConnector(config *mysql.Config, opts ...mysql.Option) (driver.Connector, error) {
	if err := config.Apply(opts...); err != nil {
		return nil, fmt.Errorf("apply MySQL driver options: %w", err)
	}
	connector, err := mysql.NewConnector(config)
	if err != nil {
		return nil, fmt.Errorf("build MySQL connector for %s: %w", config.Addr, err)
	}
	return connector, nil
}
