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
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
)

// PostgreSQLConfig builds the driver configuration KFP uses to reach
// PostgreSQL.
//
// Unlike MySQL, the driver resolves TLS itself from sslmode and sslrootcert, so
// no client tls.Config is assembled here. Target.Params overrides every setting,
// which is how operators reach connection options KFP does not expose directly.
//
// The returned configuration carries no password. Providers set the credential
// themselves, so it never passes through a connection string that could be
// logged.
func PostgreSQLConfig(t Target) (*pgx.ConnConfig, error) {
	if t.Driver != DriverPostgreSQL {
		return nil, fmt.Errorf("PostgreSQLConfig called with driver %q; use %q", t.Driver, DriverPostgreSQL)
	}
	if t.Host == "" {
		return nil, fmt.Errorf("database host is empty; set the host for driver %q", DriverPostgreSQL)
	}

	settings := map[string]string{
		"host": t.Host,
		"user": t.User,
	}
	if t.TLS != nil {
		settings["sslmode"] = "verify-full"
	}
	if t.Port != "" {
		settings["port"] = t.Port
	}
	if t.DBName != "" {
		settings["database"] = t.DBName
	}
	if t.TLS != nil && t.TLS.CABundlePath != "" {
		settings["sslrootcert"] = t.TLS.CABundlePath
	}
	for key, value := range t.Params {
		settings[key] = value
	}

	// Defaulting this would silently send database traffic unencrypted, so an
	// operator who configures neither a CA bundle nor an explicit sslmode is
	// stopped rather than quietly downgraded. This is the same requirement the
	// pre-provider connection enforces, so an operator meets one rule whichever
	// path they are on.
	sslMode, ok := settings["sslmode"]
	if !ok {
		return nil, fmt.Errorf(`sslmode must be explicitly set: put it in the PostgreSQL extra params -- "disable" ` +
			`for local development, "verify-full" for production -- or configure a database CA bundle, which selects "verify-full"`)
	}

	// A CA bundle is a statement about how this connection is protected, and
	// every ssl* setting merged from the extra params above lands in the same
	// map and can silently win over it. That is not a theoretical downgrade: a
	// provider whose credential is a token presents it as the password, so a
	// connection that is unencrypted, or encrypted but unverified, puts a
	// bearer credential within reach. Reject the conflict rather than resolve
	// it, exactly as MySQLConfig does for tls= and allowFallbackToPlaintext.
	//
	// MySQL needs only those two checks because its tls.Config is assigned
	// after the DSN round-trip, out of the operator's reach. Here there is one
	// map, so the whole ssl* namespace is refused rather than the keys known
	// to be dangerous today -- sslrootcert="" drops certificate verification
	// entirely, sslrootcert="system" swaps the pinned CA for the public web
	// PKI, and the driver is free to add more.
	if t.TLS != nil {
		if !encryptsConnection(sslMode) {
			return nil, fmt.Errorf("database TLS is configured by DB_TLS_CA_PATH, but the connection parameters also set sslmode=%q, "+
				"which permits an unencrypted connection; remove it or set it to one of %q, %q or %q",
				sslMode, "require", "verify-ca", "verify-full")
		}
		for _, key := range sortedKeys(t.Params) {
			if key != "sslmode" && strings.HasPrefix(key, "ssl") {
				return nil, fmt.Errorf("database TLS is configured by DB_TLS_CA_PATH, but the connection parameters also set %q; "+
					"remove it, because the bundle is what configures transport security", key)
			}
		}
	}

	config, err := pgx.ParseConfig(keywordValueString(settings))
	if err != nil {
		return nil, fmt.Errorf("invalid PostgreSQL connection parameters for %s: %w", t.Host, err)
	}
	return config, nil
}

// PostgreSQLConnector applies opts and returns the connector.
//
// Options are the driver's extension point for behavior that is not a settable
// field: a provider that refreshes its credential per connection installs
// stdlib.OptionBeforeConnect here.
func PostgreSQLConnector(config *pgx.ConnConfig, opts ...stdlib.OptionOpenDB) driver.Connector {
	return stdlib.GetConnector(*config, opts...)
}

// encryptsConnection reports whether an sslmode always encrypts. The modes it
// excludes -- disable, allow and prefer -- either refuse TLS or fall back to
// plaintext without saying so, which is what makes them unsafe to carry a
// credential.
func encryptsConnection(sslMode string) bool {
	switch sslMode {
	case "require", "verify-ca", "verify-full":
		return true
	default:
		return false
	}
}

// sortedKeys returns the map's keys in a stable order, so that a message built
// from them names the same key on every run.
func sortedKeys(settings map[string]string) []string {
	keys := make([]string, 0, len(settings))
	for key := range settings {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// keywordValueString renders settings in the keyword/value form the driver
// parses, quoting values so that empty strings and spaces survive.
func keywordValueString(settings map[string]string) string {
	keys := sortedKeys(settings)

	var builder strings.Builder
	for _, key := range keys {
		if builder.Len() > 0 {
			builder.WriteByte(' ')
		}
		value := strings.ReplaceAll(settings[key], `\`, `\\`)
		value = strings.ReplaceAll(value, `'`, `\'`)
		fmt.Fprintf(&builder, "%s='%s'", key, value)
	}
	return builder.String()
}
