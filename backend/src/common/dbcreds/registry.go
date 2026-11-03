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
	"fmt"
	"sort"
	"sync"
)

var (
	registryMu sync.RWMutex
	factories  = map[string]Factory{}
)

// RegisterFactory adds a Factory to the global registry. Typically called from
// a provider package's init function.
//
// Registering two factories under one name is a programming error and panics,
// because it would make the selected provider depend on package import order.
func RegisterFactory(factory Factory) {
	registryMu.Lock()
	defer registryMu.Unlock()
	name := factory.Name()
	if _, exists := factories[name]; exists {
		panic(fmt.Sprintf("database credential provider %q is already registered", name))
	}
	factories[name] = factory
}

// NewProvider builds the provider registered under name.
func NewProvider(name string, cfg Config) (Provider, error) {
	registryMu.RLock()
	factory, exists := factories[name]
	registryMu.RUnlock()
	if !exists {
		return nil, fmt.Errorf("unknown database credential provider %q; set DB_CREDENTIAL_PROVIDER to one of: %v", name, RegisteredNames())
	}
	provider, err := factory.New(cfg)
	if err != nil {
		return nil, fmt.Errorf("build database credential provider %q: %w", name, err)
	}
	return provider, nil
}

// isRegistered reports whether a provider answers to name.
func isRegistered(name string) bool {
	registryMu.RLock()
	defer registryMu.RUnlock()
	_, exists := factories[name]
	return exists
}

// factoryRequiresTLS reports whether the named provider cannot operate without a
// verified connection. An unregistered name is reported by NewProvider instead.
func factoryRequiresTLS(name string) bool {
	registryMu.RLock()
	factory, exists := factories[name]
	registryMu.RUnlock()
	return exists && factory.RequiresTLS()
}

// RegisteredNames returns the registered provider names in sorted order.
func RegisteredNames() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()
	names := make([]string, 0, len(factories))
	for name := range factories {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// resetRegistry drops externally registered factories and reinstates the
// built-in ones. Package init functions do not run again, so the built-ins have
// to be restored here. It is exported to tests by export_test.go and is
// deliberately absent from the package's public surface.
func resetRegistry() {
	registryMu.Lock()
	defer registryMu.Unlock()
	factories = map[string]Factory{}
	registerBuiltinsLocked()
}

// registerBuiltinsLocked installs the providers that are always available. The
// caller must hold registryMu.
func registerBuiltinsLocked() {
	factories[StaticProviderName] = staticFactory{}
}
