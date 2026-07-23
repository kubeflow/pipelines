package plugins

import (
	"fmt"
	"sync"

	"github.com/golang/glog"
)

// HandlerFactory knows how to check whether a plugin is enabled and how to
// construct its TaskPluginHandler. Each plugin package registers a factory
// at init time via RegisterHandlerFactory.
type HandlerFactory interface {
	// Name returns the unique identifier for this plugin factory.
	Name() string
	// IsEnabled reports whether the plugin should be activated in the current environment.
	IsEnabled() bool
	// Create constructs and returns a ready-to-use TaskPluginHandler.
	Create() (TaskPluginHandler, error)
}

type RuntimeArgsHandlerFactory interface {
	IsEnabledWithRuntimeArgs(runtimeArgs map[string]string) bool
	CreateWithRuntimeArgs(runtimeArgs map[string]string) (TaskPluginHandler, error)
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

// GetPluginDispatcher builds a TaskPluginDispatcher from all registered and
// enabled handler factories. Returns a NoOpDispatcher when no plugins are active.
func GetPluginDispatcher() (TaskPluginDispatcher, error) {
	return GetPluginDispatcherWithRuntimeArgs(nil)
}

// GetPluginDispatcherWithRuntimeArgs builds a TaskPluginDispatcher using
// explicit driver runtime arguments when a factory supports them.
func GetPluginDispatcherWithRuntimeArgs(runtimeArgs map[string]string) (TaskPluginDispatcher, error) {
	var handlers []TaskPluginHandler

	for _, factory := range RegisteredFactories() {
		var (
			handler TaskPluginHandler
			err     error
		)
		if runtimeFactory, ok := factory.(RuntimeArgsHandlerFactory); ok {
			if !runtimeFactory.IsEnabledWithRuntimeArgs(runtimeArgs) {
				continue
			}
			handler, err = runtimeFactory.CreateWithRuntimeArgs(runtimeArgs)
		} else {
			if !factory.IsEnabled() {
				continue
			}
			handler, err = factory.Create()
		}
		if err != nil {
			return NoOpDispatcher{}, fmt.Errorf("failed to initialize %s task plugin handler: %v", factory.Name(), err)
		}
		handlers = append(handlers, handler)
	}

	if len(handlers) == 0 {
		glog.Infof("No task-level plugins enabled, returning no-op dispatcher")
		return NoOpDispatcher{}, nil
	}

	return NewTaskPluginDispatcherImpl(handlers)
}
