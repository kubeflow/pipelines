package plugins

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var _ HandlerFactory = (*fakeFactory)(nil)

type fakeFactory struct {
	name    string
	enabled bool
	handler TaskPluginHandler
	err     error
}

func (f *fakeFactory) Name() string                       { return f.name }
func (f *fakeFactory) IsEnabled() bool                    { return f.enabled }
func (f *fakeFactory) Create() (TaskPluginHandler, error) { return f.handler, f.err }

func resetRegistryForTest(t *testing.T) {
	t.Helper()
	ResetRegistry()
	t.Cleanup(ResetRegistry)
}

func TestRegisterHandlerFactory_AppearsInSnapshot(t *testing.T) {
	resetRegistryForTest(t)

	factory := &fakeFactory{name: "A", enabled: true}
	RegisterHandlerFactory(factory)

	registered := RegisteredFactories()
	require.Len(t, registered, 1)
	assert.Equal(t, "A", registered[0].Name())
}

func TestRegisteredFactories_SnapshotIsolation(t *testing.T) {
	resetRegistryForTest(t)

	RegisterHandlerFactory(&fakeFactory{name: "A"})
	snapshot := RegisteredFactories()
	snapshot[0] = &fakeFactory{name: "Mutated"}

	assert.Equal(t, "A", RegisteredFactories()[0].Name())
}

func TestResetRegistry_ClearsAll(t *testing.T) {
	resetRegistryForTest(t)

	RegisterHandlerFactory(&fakeFactory{name: "A"})
	ResetRegistry()

	assert.Empty(t, RegisteredFactories())
}

func TestGetPluginDispatcher_NoFactories_ReturnsNoOp(t *testing.T) {
	resetRegistryForTest(t)

	dispatcher, err := GetPluginDispatcher()

	require.NoError(t, err)
	assert.IsType(t, NoOpDispatcher{}, dispatcher)
}

func TestGetPluginDispatcher_AllDisabled_ReturnsNoOp(t *testing.T) {
	resetRegistryForTest(t)
	RegisterHandlerFactory(&fakeFactory{name: "A", enabled: false})

	dispatcher, err := GetPluginDispatcher()

	require.NoError(t, err)
	assert.IsType(t, NoOpDispatcher{}, dispatcher)
}

func TestGetPluginDispatcher_OneEnabled_ReturnsImpl(t *testing.T) {
	resetRegistryForTest(t)
	handler := &fakeHandler{name: "A"}
	RegisterHandlerFactory(&fakeFactory{name: "A", enabled: true, handler: handler})

	dispatcher, err := GetPluginDispatcher()

	require.NoError(t, err)
	assert.IsType(t, &TaskPluginDispatcherImpl{}, dispatcher)
}

func TestGetPluginDispatcher_CreateFails_ReturnsError(t *testing.T) {
	resetRegistryForTest(t)
	RegisterHandlerFactory(&fakeFactory{
		name:    "Broken",
		enabled: true,
		err:     fmt.Errorf("init failed"),
	})

	dispatcher, err := GetPluginDispatcher()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "Broken")
	assert.Nil(t, dispatcher)
}
