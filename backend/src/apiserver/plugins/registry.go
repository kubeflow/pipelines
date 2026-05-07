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

package plugins

import (
	"fmt"
	"sync"
)

type HandlerFactory interface {

	// Name returns the unique identifier for this plugin factory.
	Name() string
	// IsEnabled reports whether the plugin should be activated in the current environment.
	IsEnabled() bool
	// Create constructs and returns a ready-to-use RunPluginHandler.
	Create() (RunPluginHandler, error)
}

var (
	registryMu sync.RWMutex
	factories  []HandlerFactory
)

// RegisterHandlerFactory adds a HandlerFactory to the global registry.
// Typically called from a plugin package's init() function.
func RegisterHandlerFactory(factory HandlerFactory) {
	registryMu.Lock()
	defer registryMu.Unlock()
	factories = append(factories, factory)
}

// RegisteredFactories returns a snapshot of all registered handler factories.
func RegisteredFactories() []HandlerFactory {
	registryMu.RLock()
	defer registryMu.RUnlock()
	result := make([]HandlerFactory, len(factories))
	copy(result, factories)
	return result
}

// ResetRegistry clears all registered factories. Intended for use in tests only.
func ResetRegistry() {
	registryMu.Lock()
	defer registryMu.Unlock()
	factories = nil
}

func GetPluginDispatcher(kubeClients KubeClientProvider, runOutputStore RunPluginOutputStore) (RunPluginDispatcher, error) {
	var handlers []RunPluginHandler

	for _, factory := range RegisteredFactories() {
		if !factory.IsEnabled() {
			continue
		}
		handler, err := factory.Create()
		if err != nil {
			return NoOpDispatcher{}, fmt.Errorf("failed to initialize %s plugin handler: %v", factory.Name(), err)
		}
		handlers = append(handlers, handler)
	}
	if len(handlers) == 0 {
		return NoOpDispatcher{}, nil
	}
	return NewRunPluginDispatcherImpl(handlers, kubeClients, runOutputStore)
}
