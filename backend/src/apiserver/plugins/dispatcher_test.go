package plugins

import (
	"context"
	"fmt"
	"testing"
	"time"

	workflowapi "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	fakeclientset "k8s.io/client-go/kubernetes/fake"
)

var _ RunPluginHandler = (*fakeHandler)(nil)

type fakeHandler struct {
	name                      string
	pluginConfig              *PluginConfig
	pluginOutput              *apiv2beta1.PluginOutput
	startErr                  error
	endErr                    error
	endBool                   bool
	envVars                   []corev1.EnvVar
	resolveInputErr           error
	resolveInputEnabled       bool
	resolveConfigErr          error
	onBeforeRunCreationCalled bool
}

func (f *fakeHandler) Name() string { return f.name }

func (f *fakeHandler) ResolveRunPluginInput(pluginsInputString *string) (input interface{}, isEnabled bool, err error) {
	if f.resolveInputErr != nil {
		return nil, false, f.resolveInputErr
	}
	return nil, f.resolveInputEnabled, nil
}

func (f *fakeHandler) GetGenericFailedPluginOutput(runID string, message string, pluginInput interface{}) *apiv2beta1.PluginOutput {
	return &apiv2beta1.PluginOutput{
		State:        apiv2beta1.PluginState_PLUGIN_FAILED,
		StateMessage: message,
	}
}

func (f *fakeHandler) ResolveRunPluginConfig(ctx context.Context, clientSet kubernetes.Interface, launcherNamespaceCfg string, namespace string) (interface{}, error) {
	if f.resolveConfigErr != nil {
		return nil, f.resolveConfigErr
	}
	// Important: return nil interface (not typed nil) when pluginConfig is nil
	// to properly trigger the nil check in dispatcher.go:139
	if f.pluginConfig == nil {
		return nil, nil
	}
	return f.pluginConfig, nil
}
func (f *fakeHandler) OnBeforeRunCreation(ctx context.Context, run *PendingRun, runCfg interface{}, pluginInput interface{}) (*apiv2beta1.PluginOutput, []corev1.EnvVar, error) {
	f.onBeforeRunCreationCalled = true
	return f.pluginOutput, f.envVars, f.startErr
}

func (f *fakeHandler) HandleRetry(ctx context.Context, run *PersistedRun, runCfg interface{}) error {
	return nil
}

func (f *fakeHandler) OnRunEnd(ctx context.Context, run *PersistedRun, runCfg interface{}) (bool, error) {
	return f.endBool, f.endErr
}

func (f *fakeHandler) GetPluginOperationTimeout(config interface{}) time.Duration {
	return 30 * time.Second
}

var pendingRun = &PendingRun{
	RunID:     "run-123",
	Namespace: "test-ns",
}

var persistedRun = &PersistedRun{
	RunID:         "run-123",
	Namespace:     "test-ns",
	State:         "SUCCEEDED",
	PluginsOutput: map[string]*apiv2beta1.PluginOutput{},
}

func newFakeExecutionSpec() util.ExecutionSpec {
	return util.NewWorkflow(&workflowapi.Workflow{
		TypeMeta:   metav1.TypeMeta{Kind: "Workflow", APIVersion: "argoproj.io/v1alpha1"},
		ObjectMeta: metav1.ObjectMeta{Name: "test-wf", Namespace: "test-ns"},
	})
}

func newFakeDispatcher(handlers []RunPluginHandler) (*RunPluginDispatcherImpl, error) {
	return NewRunPluginDispatcherImpl(handlers, &fakeKubeClientProvider{}, &fakeRunPluginOutputStore{})
}

func TestNewRunPluginDispatcherImpl_SingleHandler_Success(t *testing.T) {
	handler := &fakeHandler{name: "FakePlugin"}
	dispatcher, err := newFakeDispatcher([]RunPluginHandler{handler})

	require.NoError(t, err)
	require.NotNil(t, dispatcher)
	require.NotNil(t, dispatcher.handlers)
	require.Len(t, dispatcher.handlers, 1)
}

func TestNewRunPluginDispatcherImpl_NilHandlers_Failure(t *testing.T) {
	dispatcher, err := newFakeDispatcher(nil)

	require.Nil(t, dispatcher)
	require.Error(t, err)
	assert.Equal(t, "NewRunPluginDispatcherImpl requires non-nil slice containing minimum one handler", err.Error())
}

func TestNewRunPluginDispatcherImpl_EmptyHandlers_Failure(t *testing.T) {
	dispatcher, err := newFakeDispatcher([]RunPluginHandler{})

	require.Nil(t, dispatcher)
	require.Error(t, err)
	assert.Equal(t, "NewRunPluginDispatcherImpl requires non-nil slice containing minimum one handler", err.Error())
}

func TestOnBeforeRunCreation_SingleHandler_Success(t *testing.T) {
	handler := &fakeHandler{
		name:         "FakePlugin",
		pluginOutput: &apiv2beta1.PluginOutput{},
	}
	dispatcher, _ := newFakeDispatcher([]RunPluginHandler{handler})

	err := dispatcher.OnBeforeRunCreation(context.Background(), pendingRun, newFakeExecutionSpec())

	require.NoError(t, err)
}

func TestOnBeforeRunCreation_MultipleHandlers_Success(t *testing.T) {
	handler1 := &fakeHandler{
		name:         "FakePluginA",
		pluginOutput: &apiv2beta1.PluginOutput{},
	}
	handler2 := &fakeHandler{
		name:         "FakePluginB",
		pluginOutput: &apiv2beta1.PluginOutput{},
	}
	dispatcher, _ := newFakeDispatcher([]RunPluginHandler{handler1, handler2})

	err := dispatcher.OnBeforeRunCreation(context.Background(), pendingRun, newFakeExecutionSpec())

	require.NoError(t, err)
}

func TestOnBeforeRunCreation_NilDispatcher_Failure(t *testing.T) {
	var dispatcher *RunPluginDispatcherImpl

	err := dispatcher.OnBeforeRunCreation(context.Background(), pendingRun, newFakeExecutionSpec())

	require.Error(t, err)
	assert.Equal(t, "dispatcher, run, and executionSpec must be non-nil", err.Error())
}

func TestOnBeforeRunCreation_NilRun_Failure(t *testing.T) {
	handler := &fakeHandler{name: "FakePlugin", pluginOutput: &apiv2beta1.PluginOutput{}}
	dispatcher, _ := newFakeDispatcher([]RunPluginHandler{handler})

	err := dispatcher.OnBeforeRunCreation(context.Background(), nil, newFakeExecutionSpec())

	require.Error(t, err)
	assert.Equal(t, "dispatcher, run, and executionSpec must be non-nil", err.Error())
}

func TestOnBeforeRunCreation_NilExecutionSpec_Failure(t *testing.T) {
	handler := &fakeHandler{name: "FakePlugin", pluginOutput: &apiv2beta1.PluginOutput{}}
	dispatcher, _ := newFakeDispatcher([]RunPluginHandler{handler})

	err := dispatcher.OnBeforeRunCreation(context.Background(), pendingRun, nil)

	require.Error(t, err)
	assert.Equal(t, "dispatcher, run, and executionSpec must be non-nil", err.Error())
}

func TestOnBeforeRunCreation_ResolveRunPluginInputFailure_ReturnsBadRequestError(t *testing.T) {
	handler := &fakeHandler{
		name:            "FakePlugin",
		resolveInputErr: fmt.Errorf("invalid plugin input format"),
	}
	dispatcher, _ := newFakeDispatcher([]RunPluginHandler{handler})

	err := dispatcher.OnBeforeRunCreation(context.Background(), pendingRun, newFakeExecutionSpec())

	require.Error(t, err)
	// Verify it's a UserError with the expected message
	userErr, ok := err.(*util.UserError)
	require.True(t, ok, "expected error to be *util.UserError")
	assert.Contains(t, userErr.ExternalMessage(), "Failed to create a run due to invalid FakePlugin plugin input")
	assert.Contains(t, userErr.Error(), "BadRequestError")
}

func TestOnBeforeRunCreation_ResolveRunPluginConfigFailure_ContinuesWithoutCallingHandler(t *testing.T) {
	handler := &fakeHandler{
		name:                "FakePlugin",
		pluginConfig:        &PluginConfig{Endpoint: "http://test"},
		resolveInputEnabled: true,
		resolveConfigErr:    fmt.Errorf("failed to resolve plugin config"),
		pluginOutput:        &apiv2beta1.PluginOutput{State: apiv2beta1.PluginState_PLUGIN_SUCCEEDED},
	}
	dispatcher, _ := newFakeDispatcher([]RunPluginHandler{handler})

	run := &PendingRun{
		RunID:     "run-123",
		Namespace: "test-ns",
	}

	err := dispatcher.OnBeforeRunCreation(context.Background(), run, newFakeExecutionSpec())

	// Run creation should continue (no error returned)
	require.NoError(t, err)

	// Handler's OnBeforeRunCreation should NOT have been called
	assert.False(t, handler.onBeforeRunCreationCalled, "handler.OnBeforeRunCreation should not be called when ResolveRunPluginConfig fails")

	// Plugin output should be set with failure state
	require.NotNil(t, run.PluginsOutput)
	require.NotNil(t, *run.PluginsOutput)
	assert.Contains(t, *run.PluginsOutput, "FakePlugin config resolution failed")
}

func TestOnBeforeRunCreation_NoConfigMapOverride_SucceedsWithoutCallingHandler(t *testing.T) {
	handler := &fakeHandler{
		name:                "FakePlugin",
		pluginConfig:        nil, // Handler returns nil config when no configmap override exists
		resolveInputEnabled: true,
		pluginOutput:        &apiv2beta1.PluginOutput{State: apiv2beta1.PluginState_PLUGIN_SUCCEEDED},
	}
	dispatcher, _ := newFakeDispatcher([]RunPluginHandler{handler})

	run := &PendingRun{
		RunID:     "run-123",
		Namespace: "test-ns",
	}

	// RetrieveMultiUserModeConfigOverrides returns nil/empty map when:
	// 1. Multi-user mode is disabled (returns nil, nil)
	// 2. Multi-user mode is enabled but no configmap exists (returns nil, nil)
	// In both cases, launcherNamespacePluginCfgs[handler.Name()] will be ""
	// and handler.ResolveRunPluginConfig is called with empty string.
	// When the handler returns nil config, OnBeforeRunCreation should return early.

	err := dispatcher.OnBeforeRunCreation(context.Background(), run, newFakeExecutionSpec())

	// Run creation should succeed (no error returned)
	require.NoError(t, err)

	// Handler's OnBeforeRunCreation should NOT have been called because resolved config is nil
	assert.False(t, handler.onBeforeRunCreationCalled, "handler.OnBeforeRunCreation should not be called when ResolveRunPluginConfig returns nil")

	// No plugin output should be set since the handler was skipped (not an error case)
	if run.PluginsOutput != nil && *run.PluginsOutput != "" {
		t.Errorf("Expected no plugin output when handler is skipped with nil config, but got: %s", *run.PluginsOutput)
	}
}

func TestOnBeforeRunCreation_WithValidConfig_SuccessfullyCallsHandler(t *testing.T) {
	handler := &fakeHandler{
		name:                "FakePlugin",
		pluginConfig:        &PluginConfig{Endpoint: "http://test-endpoint"},
		resolveInputEnabled: true,
		pluginOutput:        &apiv2beta1.PluginOutput{State: apiv2beta1.PluginState_PLUGIN_SUCCEEDED},
	}
	dispatcher, _ := newFakeDispatcher([]RunPluginHandler{handler})

	run := &PendingRun{
		RunID:     "run-456",
		Namespace: "test-ns",
	}

	err := dispatcher.OnBeforeRunCreation(context.Background(), run, newFakeExecutionSpec())

	// Run creation should succeed
	require.NoError(t, err)

	// Handler's OnBeforeRunCreation SHOULD have been called with valid config
	assert.True(t, handler.onBeforeRunCreationCalled, "handler.OnBeforeRunCreation should be called when valid config exists")

	// Plugin output should be set
	require.NotNil(t, run.PluginsOutput)
	require.NotNil(t, *run.PluginsOutput)
	assert.Contains(t, *run.PluginsOutput, "FakePlugin")
}

func TestOnRunEnd_SingleHandler_Success(t *testing.T) {
	handler := &fakeHandler{name: "FakePlugin"}
	dispatcher, _ := newFakeDispatcher([]RunPluginHandler{handler})

	ok := dispatcher.OnRunEnd(context.Background(), persistedRun)

	assert.True(t, ok)
}

func TestOnRunEnd_MultipleHandlers_Success(t *testing.T) {
	handler1 := &fakeHandler{name: "FakePluginA"}
	handler2 := &fakeHandler{name: "FakePluginB"}
	dispatcher, _ := newFakeDispatcher([]RunPluginHandler{handler1, handler2})

	ok := dispatcher.OnRunEnd(context.Background(), persistedRun)

	assert.True(t, ok)
}

func TestOnRunEnd_NilDispatcher_Failure(t *testing.T) {
	var dispatcher *RunPluginDispatcherImpl

	ok := dispatcher.OnRunEnd(context.Background(), persistedRun)

	assert.False(t, ok)
}

func TestOnRunEnd_NilRun_Failure(t *testing.T) {
	handler := &fakeHandler{name: "FakePlugin"}
	dispatcher, _ := newFakeDispatcher([]RunPluginHandler{handler})

	ok := dispatcher.OnRunEnd(context.Background(), nil)

	assert.False(t, ok)
}

func TestOnRunRetry_SingleHandler_Success(t *testing.T) {
	handler := &fakeHandler{name: "FakePlugin"}
	dispatcher, _ := newFakeDispatcher([]RunPluginHandler{handler})

	err := dispatcher.OnRunRetry(context.Background(), persistedRun)

	require.NoError(t, err)
}

func TestOnRunRetry_MultipleHandlers_Success(t *testing.T) {
	handler1 := &fakeHandler{name: "FakePluginA"}
	handler2 := &fakeHandler{name: "FakePluginB"}
	dispatcher, _ := newFakeDispatcher([]RunPluginHandler{handler1, handler2})

	err := dispatcher.OnRunRetry(context.Background(), persistedRun)

	require.NoError(t, err)
}

func TestOnRunRetryNilDispatcher_Failure(t *testing.T) {
	var dispatcher *RunPluginDispatcherImpl

	err := dispatcher.OnRunRetry(context.Background(), persistedRun)

	require.Error(t, err)
	assert.Equal(t, "dispatcher and run must be non-nil", err.Error())
}

func TestOnRunRetry_NilRun_Failure(t *testing.T) {
	handler := &fakeHandler{name: "FakePlugin"}
	dispatcher, _ := newFakeDispatcher([]RunPluginHandler{handler})

	err := dispatcher.OnRunRetry(context.Background(), nil)

	require.Error(t, err)
	assert.Equal(t, "dispatcher and run must be non-nil", err.Error())
}

// fakeRunPluginOutputStoreWithError is a mock store that can be configured to fail.
type fakeRunPluginOutputStoreWithError struct {
	shouldFail bool
	callCount  int
}

func (f *fakeRunPluginOutputStoreWithError) UpdateRunPluginsOutput(_ string, _ *model.LargeText) error {
	f.callCount++
	if f.shouldFail {
		return fmt.Errorf("mock persistence error")
	}
	return nil
}

func TestExecutePostAction_NilRunCfg_SuccessfulPersistence(t *testing.T) {
	handler := &fakeHandler{name: "TestPlugin"}
	store := &fakeRunPluginOutputStoreWithError{shouldFail: false}
	dispatcher := &RunPluginDispatcherImpl{
		handlers:       []RunPluginHandler{handler},
		kubeClients:    &fakeKubeClientProvider{},
		runOutputStore: store,
	}

	run := &PersistedRun{
		RunID:     "run-123",
		Namespace: "test-ns",
		State:     "SUCCEEDED",
		PluginsOutput: map[string]*apiv2beta1.PluginOutput{
			"TestPlugin": {
				State:        apiv2beta1.PluginState_PLUGIN_SUCCEEDED,
				StateMessage: "initial state",
			},
		},
	}

	invokeCalled := false
	pluginSyncOK, retryRequested, persisted := dispatcher.executePostAction(
		run,
		"TestHook",
		nil, // runCfg is nil
		"TestPlugin",
		func(r *PersistedRun, cfg interface{}) bool {
			invokeCalled = true
			return false
		},
	)

	assert.False(t, invokeCalled, "invoke function should not be called when runCfg is nil")
	assert.False(t, pluginSyncOK, "pluginSyncOK should be false when runCfg is nil")
	assert.False(t, retryRequested, "retryRequested should be false when runCfg is nil")
	assert.True(t, persisted, "persisted should be true when persistence succeeds")
	assert.Equal(t, 1, store.callCount, "store should be called once")
	assert.Equal(t, apiv2beta1.PluginState_PLUGIN_FAILED, run.PluginsOutput["TestPlugin"].State)
	assert.Equal(t, "TestPlugin TestHook sync failed: config unavailable", run.PluginsOutput["TestPlugin"].StateMessage)
}

func TestExecutePostAction_NilRunCfg_FailedPersistence(t *testing.T) {
	handler := &fakeHandler{name: "TestPlugin"}
	store := &fakeRunPluginOutputStoreWithError{shouldFail: true}
	dispatcher := &RunPluginDispatcherImpl{
		handlers:       []RunPluginHandler{handler},
		kubeClients:    &fakeKubeClientProvider{},
		runOutputStore: store,
	}

	run := &PersistedRun{
		RunID:     "run-456",
		Namespace: "test-ns",
		State:     "SUCCEEDED",
		PluginsOutput: map[string]*apiv2beta1.PluginOutput{
			"TestPlugin": {
				State:        apiv2beta1.PluginState_PLUGIN_SUCCEEDED,
				StateMessage: "initial state",
			},
		},
	}

	invokeCalled := false
	pluginSyncOK, retryRequested, persisted := dispatcher.executePostAction(
		run,
		"TestHook",
		nil, // runCfg is nil
		"TestPlugin",
		func(r *PersistedRun, cfg interface{}) bool {
			invokeCalled = true
			return false
		},
	)

	assert.False(t, invokeCalled, "invoke function should not be called when runCfg is nil")
	assert.False(t, pluginSyncOK, "pluginSyncOK should be false when runCfg is nil")
	assert.False(t, retryRequested, "retryRequested should be false when runCfg is nil")
	assert.False(t, persisted, "persisted should be false when persistence fails")
	assert.Equal(t, 1, store.callCount, "store should be called once")
	assert.Equal(t, apiv2beta1.PluginState_PLUGIN_FAILED, run.PluginsOutput["TestPlugin"].State)
	assert.Equal(t, "TestPlugin TestHook sync failed: config unavailable", run.PluginsOutput["TestPlugin"].StateMessage)
}

func TestExecutePostAction_WithRunCfg_SuccessfulSync_SuccessfulPersistence(t *testing.T) {
	handler := &fakeHandler{name: "TestPlugin"}
	store := &fakeRunPluginOutputStoreWithError{shouldFail: false}
	dispatcher := &RunPluginDispatcherImpl{
		handlers:       []RunPluginHandler{handler},
		kubeClients:    &fakeKubeClientProvider{},
		runOutputStore: store,
	}

	run := &PersistedRun{
		RunID:     "run-789",
		Namespace: "test-ns",
		State:     "SUCCEEDED",
		PluginsOutput: map[string]*apiv2beta1.PluginOutput{
			"TestPlugin": {
				State:        apiv2beta1.PluginState_PLUGIN_SUCCEEDED,
				StateMessage: "sync completed",
			},
		},
	}

	cfg := &PluginConfig{Endpoint: "http://test-endpoint"}
	invokeCalled := false
	pluginSyncOK, retryRequested, persisted := dispatcher.executePostAction(
		run,
		"TestHook",
		cfg,
		"TestPlugin",
		func(r *PersistedRun, receivedCfg interface{}) bool {
			invokeCalled = true
			assert.Equal(t, cfg, receivedCfg, "config should be passed to invoke function")
			return false
		},
	)

	assert.True(t, invokeCalled, "invoke function should be called when runCfg is not nil")
	assert.True(t, pluginSyncOK, "pluginSyncOK should be true when plugin state is PLUGIN_SUCCEEDED")
	assert.False(t, retryRequested, "retryRequested should be false when invoke returns false")
	assert.True(t, persisted, "persisted should be true when persistence succeeds")
	assert.Equal(t, 1, store.callCount, "store should be called once")
}

func TestExecutePostAction_WithRunCfg_FailedSync_SuccessfulPersistence(t *testing.T) {
	handler := &fakeHandler{name: "TestPlugin"}
	store := &fakeRunPluginOutputStoreWithError{shouldFail: false}
	dispatcher := &RunPluginDispatcherImpl{
		handlers:       []RunPluginHandler{handler},
		kubeClients:    &fakeKubeClientProvider{},
		runOutputStore: store,
	}

	run := &PersistedRun{
		RunID:     "run-abc",
		Namespace: "test-ns",
		State:     "FAILED",
		PluginsOutput: map[string]*apiv2beta1.PluginOutput{
			"TestPlugin": {
				State:        apiv2beta1.PluginState_PLUGIN_FAILED,
				StateMessage: "sync failed",
			},
		},
	}

	cfg := &PluginConfig{Endpoint: "http://test-endpoint"}
	invokeCalled := false
	pluginSyncOK, retryRequested, persisted := dispatcher.executePostAction(
		run,
		"TestHook",
		cfg,
		"TestPlugin",
		func(r *PersistedRun, receivedCfg interface{}) bool {
			invokeCalled = true
			return true // Request retry
		},
	)

	assert.True(t, invokeCalled, "invoke function should be called when runCfg is not nil")
	assert.False(t, pluginSyncOK, "pluginSyncOK should be false when plugin state is PLUGIN_FAILED")
	assert.True(t, retryRequested, "retryRequested should be true when invoke returns true")
	assert.True(t, persisted, "persisted should be true when persistence succeeds")
	assert.Equal(t, 1, store.callCount, "store should be called once")
}

func TestExecutePostAction_WithRunCfg_FailedPersistence(t *testing.T) {
	handler := &fakeHandler{name: "TestPlugin"}
	store := &fakeRunPluginOutputStoreWithError{shouldFail: true}
	dispatcher := &RunPluginDispatcherImpl{
		handlers:       []RunPluginHandler{handler},
		kubeClients:    &fakeKubeClientProvider{},
		runOutputStore: store,
	}

	run := &PersistedRun{
		RunID:     "run-def",
		Namespace: "test-ns",
		State:     "SUCCEEDED",
		PluginsOutput: map[string]*apiv2beta1.PluginOutput{
			"TestPlugin": {
				State:        apiv2beta1.PluginState_PLUGIN_SUCCEEDED,
				StateMessage: "sync completed",
			},
		},
	}

	cfg := &PluginConfig{Endpoint: "http://test-endpoint"}
	invokeCalled := false
	pluginSyncOK, retryRequested, persisted := dispatcher.executePostAction(
		run,
		"TestHook",
		cfg,
		"TestPlugin",
		func(r *PersistedRun, receivedCfg interface{}) bool {
			invokeCalled = true
			return false
		},
	)

	assert.True(t, invokeCalled, "invoke function should be called when runCfg is not nil")
	assert.True(t, pluginSyncOK, "pluginSyncOK should be true when plugin state is PLUGIN_SUCCEEDED")
	assert.False(t, retryRequested, "retryRequested should be false when invoke returns false")
	assert.False(t, persisted, "persisted should be false when persistence fails")
	assert.Equal(t, 1, store.callCount, "store should be called once")
}

func TestExecutePostAction_NilPluginOutput_WithRunCfg(t *testing.T) {
	handler := &fakeHandler{name: "TestPlugin"}
	store := &fakeRunPluginOutputStoreWithError{shouldFail: false}
	dispatcher := &RunPluginDispatcherImpl{
		handlers:       []RunPluginHandler{handler},
		kubeClients:    &fakeKubeClientProvider{},
		runOutputStore: store,
	}

	run := &PersistedRun{
		RunID:         "run-ghi",
		Namespace:     "test-ns",
		State:         "SUCCEEDED",
		PluginsOutput: map[string]*apiv2beta1.PluginOutput{}, // No plugin output for TestPlugin
	}

	cfg := &PluginConfig{Endpoint: "http://test-endpoint"}
	invokeCalled := false
	pluginSyncOK, retryRequested, persisted := dispatcher.executePostAction(
		run,
		"TestHook",
		cfg,
		"TestPlugin",
		func(r *PersistedRun, receivedCfg interface{}) bool {
			invokeCalled = true
			return false
		},
	)

	assert.True(t, invokeCalled, "invoke function should be called when runCfg is not nil")
	assert.False(t, pluginSyncOK, "pluginSyncOK should be false when plugin output is nil")
	assert.False(t, retryRequested, "retryRequested should be false when invoke returns false")
	assert.True(t, persisted, "persisted should be true when persistence succeeds")
	assert.Equal(t, 1, store.callCount, "store should be called once")
}

func TestOnBeforeRunCreation_HandlerFailure_ContinuesExecution(t *testing.T) {
	handler := &fakeHandler{
		name:         "FakePlugin",
		pluginOutput: &apiv2beta1.PluginOutput{},
		startErr:     fmt.Errorf("plugin startup failed"),
	}
	dispatcher, _ := newFakeDispatcher([]RunPluginHandler{handler})

	err := dispatcher.OnBeforeRunCreation(context.Background(), pendingRun, newFakeExecutionSpec())

	require.NoError(t, err)
}

func TestOnRunEnd_HandlerFailure_ReturnsTrueWithoutParentRun(t *testing.T) {
	handler := &fakeHandler{
		name:   "FakePlugin",
		endErr: fmt.Errorf("plugin end failed"),
	}
	dispatcher, _ := newFakeDispatcher([]RunPluginHandler{handler})

	ok := dispatcher.OnRunEnd(context.Background(), persistedRun)

	assert.True(t, ok)
}

func TestOnRunEnd_HandlerFailure_ReturnsErrorWithParentRun(t *testing.T) {
	handler := &fakeHandler{
		name:         "fakeplugin",
		pluginConfig: &PluginConfig{Endpoint: "http://test"},
		endErr:       fmt.Errorf("plugin end failed"),
		endBool:      true, // Request retry for transient failure
	}
	dispatcher, _ := newFakeDispatcher([]RunPluginHandler{handler})

	// Create a run with plugin output that has a parent run ID and failed state.
	// When hasParentRun=true and the handler requests retry (retryable=true),
	// OnRunEnd should return false to signal retry needed.
	runWithParent := &PersistedRun{
		RunID:     "run-456",
		Namespace: "test-ns",
		State:     "SUCCEEDED",
		PluginsOutput: map[string]*apiv2beta1.PluginOutput{
			"fakeplugin": {
				Entries: map[string]*apiv2beta1.MetadataValue{
					EntryRootRunID: {
						Value: structpb.NewStringValue("parent-run-123"),
					},
				},
				State:        apiv2beta1.PluginState_PLUGIN_FAILED,
				StateMessage: "sync failed",
			},
		},
	}

	ok := dispatcher.OnRunEnd(context.Background(), runWithParent)

	assert.False(t, ok)
}

func TestOnRunEnd_PermanentFailure_DoesNotRequestRetryWithParentRun(t *testing.T) {
	handler := &fakeHandler{
		name:         "fakeplugin",
		pluginConfig: &PluginConfig{Endpoint: "http://test"},
		endErr:       fmt.Errorf("plugin end failed"),
		endBool:      false, // Do not request retry for permanent failure
	}
	dispatcher, _ := newFakeDispatcher([]RunPluginHandler{handler})

	// Create a run with plugin output that has a parent run ID and failed state.
	// When hasParentRun=true but the handler indicates permanent failure (retryable=false),
	// OnRunEnd should return true (proceed with finalization, don't retry).
	runWithParent := &PersistedRun{
		RunID:     "run-789",
		Namespace: "test-ns",
		State:     "FAILED",
		PluginsOutput: map[string]*apiv2beta1.PluginOutput{
			"fakeplugin": {
				Entries: map[string]*apiv2beta1.MetadataValue{
					EntryRootRunID: {
						Value: structpb.NewStringValue("parent-run-456"),
					},
				},
				State:        apiv2beta1.PluginState_PLUGIN_FAILED,
				StateMessage: "config unavailable",
			},
		},
	}

	ok := dispatcher.OnRunEnd(context.Background(), runWithParent)

	assert.True(t, ok, "permanent failure should not request retry")
}

// ---- RetrieveMultiUserModeConfigOverrides tests ----

func TestRetrieveMultiUserModeConfigOverrides_MultiUserModeDisabled_ReturnsNil(t *testing.T) {
	handler := &fakeHandler{name: "TestPlugin"}
	dispatcher, err := newFakeDispatcher([]RunPluginHandler{handler})
	require.NoError(t, err)

	// Ensure multi-user mode is disabled (default behavior)
	// Note: viper defaults to false for MultiUserMode, so no explicit set needed

	cfgMap, err := dispatcher.RetrieveMultiUserModeConfigOverrides(context.Background(), "run-123", "test-ns")

	require.NoError(t, err)
	assert.Nil(t, cfgMap, "should return nil when multi-user mode is disabled")
}

func TestRetrieveMultiUserModeConfigOverrides_MultiUserModeEnabled_ConfigMapNotFound_ReturnsNil(t *testing.T) {
	handler := &fakeHandler{name: "TestPlugin"}
	dispatcher, err := newFakeDispatcher([]RunPluginHandler{handler})
	require.NoError(t, err)

	// Enable multi-user mode
	viper.Set("MULTIUSER", "true")
	t.Cleanup(func() { viper.Set("MULTIUSER", nil) })

	cfgMap, err := dispatcher.RetrieveMultiUserModeConfigOverrides(context.Background(), "run-123", "test-ns")

	require.NoError(t, err)
	assert.Nil(t, cfgMap, "should return nil when ConfigMap is not found")
}

func TestRetrieveMultiUserModeConfigOverrides_MultiUserModeEnabled_ConfigMapExists_ReturnsConfig(t *testing.T) {
	handler := &fakeHandler{name: "TestPlugin"}
	clientSet := fakeclientset.NewClientset()
	namespace := "test-ns"

	// Create a ConfigMap with plugin configs
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      LauncherConfigMapName,
			Namespace: namespace,
		},
		Data: map[string]string{
			"plugins.mlflow": `{"endpoint": "https://mlflow.example.com"}`,
			"plugins.custom": `{"endpoint": "https://custom.example.com"}`,
		},
	}
	_, err := clientSet.CoreV1().ConfigMaps(namespace).Create(context.Background(), cm, metav1.CreateOptions{})
	require.NoError(t, err)

	// Create dispatcher with a client provider that returns our fake clientSet
	customClientProvider := &fakeKubeClientProviderWithClientSet{clientSet: clientSet}
	dispatcher := &RunPluginDispatcherImpl{
		handlers:       []RunPluginHandler{handler},
		kubeClients:    customClientProvider,
		runOutputStore: &fakeRunPluginOutputStore{},
	}

	// Enable multi-user mode
	viper.Set("MULTIUSER", "true")
	t.Cleanup(func() { viper.Set("MULTIUSER", nil) })

	cfgMap, err := dispatcher.RetrieveMultiUserModeConfigOverrides(context.Background(), "run-123", namespace)

	require.NoError(t, err)
	require.NotNil(t, cfgMap)
	assert.Equal(t, 2, len(cfgMap), "should return both plugin configs")
	assert.Contains(t, cfgMap, "mlflow")
	assert.Contains(t, cfgMap, "custom")
	assert.Equal(t, `{"endpoint": "https://mlflow.example.com"}`, cfgMap["mlflow"])
	assert.Equal(t, `{"endpoint": "https://custom.example.com"}`, cfgMap["custom"])
}

func TestRetrieveMultiUserModeConfigOverrides_MultiUserModeEnabled_EmptyNamespace_ReturnsError(t *testing.T) {
	handler := &fakeHandler{name: "TestPlugin"}
	dispatcher, err := newFakeDispatcher([]RunPluginHandler{handler})
	require.NoError(t, err)

	// Enable multi-user mode
	viper.Set("MULTIUSER", "true")
	t.Cleanup(func() { viper.Set("MULTIUSER", nil) })

	cfgMap, err := dispatcher.RetrieveMultiUserModeConfigOverrides(context.Background(), "run-123", "")

	require.Error(t, err)
	assert.Nil(t, cfgMap)
	assert.Contains(t, err.Error(), "failed to retrieve launcher namespace plugin configs for run")
	assert.Contains(t, err.Error(), "namespace must be specified when reading Plugin config")
}

// fakeKubeClientProviderWithClientSet is a test helper that returns a specific clientSet
type fakeKubeClientProviderWithClientSet struct {
	clientSet kubernetes.Interface
}

func (f *fakeKubeClientProviderWithClientSet) GetClientSet() kubernetes.Interface {
	return f.clientSet
}
