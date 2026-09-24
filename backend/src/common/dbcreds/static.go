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
	"database/sql/driver"
)

// StaticProviderName identifies the provider that authenticates with a fixed
// password.
const StaticProviderName = "static"

func init() {
	RegisterFactory(staticFactory{})
}

type staticFactory struct{}

func (staticFactory) Name() string { return StaticProviderName }

// RequiresTLS reports false: the configured password is what the connection
// would have used anyway, so this provider does not itself require a verified
// connection.
func (staticFactory) RequiresTLS() bool { return false }

func (staticFactory) New(cfg Config) (Provider, error) {
	return &staticProvider{password: cfg.Password}, nil
}

// staticProvider authenticates with the configured password. It is the
// reference implementation of Provider, and lets the provider path be exercised
// without a cloud identity.
type staticProvider struct {
	password string
}

func (p *staticProvider) Name() string { return StaticProviderName }

func (p *staticProvider) Connector(ctx context.Context, t Target) (driver.Connector, error) {
	switch t.Driver {
	case DriverMySQL:
		config, err := MySQLConfig(t)
		if err != nil {
			return nil, err
		}
		config.Passwd = p.password
		return MySQLConnector(config)
	case DriverPostgreSQL:
		config, err := PostgreSQLConfig(t)
		if err != nil {
			return nil, err
		}
		config.Password = p.password
		return PostgreSQLConnector(config), nil
	default:
		return nil, UnsupportedDriverError(StaticProviderName, t.Driver)
	}
}
