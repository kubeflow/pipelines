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
	"database/sql/driver"
	"fmt"
	"testing"

	"github.com/kubeflow/pipelines/backend/src/common/dbcreds"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeFactory struct {
	name string
	err  error
}

func (f fakeFactory) Name() string { return f.name }

func (fakeFactory) RequiresTLS() bool { return false }

func (f fakeFactory) New(cfg dbcreds.Config) (dbcreds.Provider, error) {
	if f.err != nil {
		return nil, f.err
	}
	return fakeProvider{name: f.name, region: cfg.Setting(dbcreds.SettingRegion)}, nil
}

type fakeProvider struct {
	name   string
	region string
}

func (p fakeProvider) Name() string { return p.name }

func (p fakeProvider) Connector(context.Context, dbcreds.Target) (driver.Connector, error) {
	return nil, nil
}

// restoreRegistry drops test factories before and after a test so they do not
// leak into one another.
func restoreRegistry(t *testing.T) {
	t.Helper()
	t.Cleanup(dbcreds.ResetRegistry)
	dbcreds.ResetRegistry()
}

func TestNewProvider(t *testing.T) {
	restoreRegistry(t)
	dbcreds.RegisterFactory(fakeFactory{name: "fake"})

	provider, err := dbcreds.NewProvider("fake", dbcreds.Config{Settings: map[string]string{dbcreds.SettingRegion: "us-east-1"}})
	require.NoError(t, err)
	assert.Equal(t, "fake", provider.Name())
	assert.Equal(t, "us-east-1", provider.(fakeProvider).region)
}

func TestNewProviderUnknownNameListsAlternatives(t *testing.T) {
	restoreRegistry(t)
	dbcreds.RegisterFactory(fakeFactory{name: "fake"})

	_, err := dbcreds.NewProvider("nope", dbcreds.Config{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), `unknown database credential provider "nope"`)
	assert.Contains(t, err.Error(), "fake", "the error must tell the operator what is available")
}

func TestNewProviderPropagatesFactoryError(t *testing.T) {
	restoreRegistry(t)
	dbcreds.RegisterFactory(fakeFactory{name: "broken", err: fmt.Errorf("no region configured")})

	_, err := dbcreds.NewProvider("broken", dbcreds.Config{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no region configured")
}

func TestRegisterFactoryRejectsDuplicates(t *testing.T) {
	restoreRegistry(t)
	dbcreds.RegisterFactory(fakeFactory{name: "fake"})

	assert.PanicsWithValue(t,
		`database credential provider "fake" is already registered`,
		func() { dbcreds.RegisterFactory(fakeFactory{name: "fake"}) },
		"duplicate registration would make the active provider depend on import order")
}

func TestRegisteredNamesIsSorted(t *testing.T) {
	restoreRegistry(t)
	dbcreds.RegisterFactory(fakeFactory{name: "zulu"})
	dbcreds.RegisterFactory(fakeFactory{name: "alpha"})

	assert.Equal(t, []string{"alpha", dbcreds.StaticProviderName, "zulu"}, dbcreds.RegisteredNames())
}

func TestStaticProviderIsRegisteredByDefault(t *testing.T) {
	assert.Contains(t, dbcreds.RegisteredNames(), dbcreds.StaticProviderName)
}
