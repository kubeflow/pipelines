package plugins

import (
	"context"
	"fmt"
	"testing"

	workflowapi "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ RunPluginHandler = (*fakeHandler)(nil)

type fakeHandler struct {
	name         string
	pluginConfig *PluginConfig
	pluginOutput *apiv2beta1.PluginOutput
	startErr     error
	endErr       error
	envVars      map[string]string
}

func (f *fakeHandler) Name() string { return f.name }

func (f *fakeHandler) GetGlobalPluginConfig() (*PluginConfig, error) {
	return f.pluginConfig, nil
}

func (f *fakeHandler) OnBeforeRunCreation(_ context.Context, _ *PendingRun, _ *PluginConfig) (*apiv2beta1.PluginOutput, map[string]string, error) {
	return f.pluginOutput, f.envVars, f.startErr
}

func (f *fakeHandler) HandleRetry(_ context.Context, _ *PersistedRun, _ *PluginConfig) {}

func (f *fakeHandler) OnRunEnd(_ context.Context, _ *PersistedRun, _ *PluginConfig) error {
	return f.endErr
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

func TestOnRunEnd_SingleHandler_Success(t *testing.T) {
	handler := &fakeHandler{name: "FakePlugin"}
	dispatcher, _ := newFakeDispatcher([]RunPluginHandler{handler})

	result := dispatcher.OnRunEnd(context.Background(), persistedRun)

	assert.True(t, result)
}

func TestOnRunEnd_MultipleHandlers_Success(t *testing.T) {
	handler1 := &fakeHandler{name: "FakePluginA"}
	handler2 := &fakeHandler{name: "FakePluginB"}
	dispatcher, _ := newFakeDispatcher([]RunPluginHandler{handler1, handler2})

	result := dispatcher.OnRunEnd(context.Background(), persistedRun)

	assert.True(t, result)
}

func TestOnRunEnd_NilDispatcher_Failure(t *testing.T) {
	var dispatcher *RunPluginDispatcherImpl

	assert.Panics(t, func() {
		dispatcher.OnRunEnd(context.Background(), persistedRun)
	})
}

func TestOnRunEnd_NilRun_Failure(t *testing.T) {
	handler := &fakeHandler{name: "FakePlugin"}
	dispatcher, _ := newFakeDispatcher([]RunPluginHandler{handler})

	assert.Panics(t, func() {
		dispatcher.OnRunEnd(context.Background(), nil)
	})
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

	result := dispatcher.OnRunEnd(context.Background(), persistedRun)

	assert.True(t, result)
}
