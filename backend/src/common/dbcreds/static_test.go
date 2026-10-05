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

package dbcreds_test

import (
	"context"
	"testing"

	"github.com/kubeflow/pipelines/backend/src/common/dbcreds"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStaticProviderBuildsConnector(t *testing.T) {
	provider, err := dbcreds.NewProvider(dbcreds.StaticProviderName, dbcreds.Config{Password: "hunter2"})
	require.NoError(t, err)

	connector, err := provider.Connector(context.Background(), mysqlTarget())
	require.NoError(t, err)
	assert.NotNil(t, connector)
}

func TestStaticProviderAppliesTLS(t *testing.T) {
	provider, err := dbcreds.NewProvider(dbcreds.StaticProviderName, dbcreds.Config{Password: "hunter2"})
	require.NoError(t, err)

	target := mysqlTarget()
	target.TLS = &dbcreds.TLSOptions{CABundlePath: writeCABundle(t)}

	connector, err := provider.Connector(context.Background(), target)
	require.NoError(t, err)
	assert.NotNil(t, connector)
}

func TestStaticProviderSurfacesTLSFailure(t *testing.T) {
	provider, err := dbcreds.NewProvider(dbcreds.StaticProviderName, dbcreds.Config{})
	require.NoError(t, err)

	target := mysqlTarget()
	target.TLS = &dbcreds.TLSOptions{CABundlePath: "/nonexistent/ca.pem"}

	_, err = provider.Connector(context.Background(), target)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read DB_TLS_CA_PATH")
}

func TestStaticProviderRejectsUnsupportedDriver(t *testing.T) {
	provider, err := dbcreds.NewProvider(dbcreds.StaticProviderName, dbcreds.Config{})
	require.NoError(t, err)

	target := mysqlTarget()
	target.Driver = "sqlite"

	_, err = provider.Connector(context.Background(), target)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `does not support driver "sqlite"`)
}

func TestStaticProviderBuildsPostgresConnector(t *testing.T) {
	provider, err := dbcreds.NewProvider(dbcreds.StaticProviderName, dbcreds.Config{Password: "hunter2"})
	require.NoError(t, err)

	connector, err := provider.Connector(context.Background(), dbcreds.Target{
		Driver: dbcreds.DriverPostgreSQL,
		Host:   "postgresql",
		Port:   "5432",
		User:   "root",
		DBName: "mlpipeline",
		Params: map[string]string{"sslmode": "disable"},
	})
	require.NoError(t, err)
	assert.NotNil(t, connector)
}
