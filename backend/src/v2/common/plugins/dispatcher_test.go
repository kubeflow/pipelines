package plugins

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var _ TaskPluginHandler = (*fakeHandler)(nil)

type fakeStartResult struct {
	RunID string
}

type fakeHandler struct {
	name        string
	startResult TaskHandlerStartResult
	startErr    error
	endErr      error
	envVars     map[string]string
	envErr      error
	customProps map[string]string
}

func (f *fakeHandler) Name() string { return f.name }
func (f *fakeHandler) OnTaskStart(_ context.Context, _ *TaskInfo) (TaskHandlerStartResult, error) {
	return f.startResult, f.startErr
}
func (f *fakeHandler) OnTaskEnd(_ context.Context, _ TaskHandlerStartResult, _ *TaskInfo) error {
	return f.endErr
}
func (f *fakeHandler) RetrieveUserContainerEnvVars(_ TaskHandlerStartResult) (map[string]string, error) {
	return f.envVars, f.envErr
}
func (f *fakeHandler) RetrieveCustomProperties(_ TaskHandlerStartResult) map[string]string {
	return f.customProps
}

func (f *fakeHandler) ApplyCustomProperties(customProperties map[string]string) error { return nil }

var taskInfoStart = &TaskInfo{
	Name: "test-task",
}

var taskInfoEnd = &TaskInfo{
	Name:          "test-task",
	RunEndTime:    int64(1714400000000),
	RunStatus:     "COMPLETED",
	ScalarMetrics: map[string]float64{},
	Parameters:    map[string]string{},
	TaskStartResult: &TaskStartResult{
		Results: map[string]TaskHandlerStartResult{
			"FakePlugin": &fakeStartResult{RunID: "fake-run-1"},
		},
	},
}

func TestNewTaskPluginDispatcherImpl_SingleHandler_Success(t *testing.T) {
	handler := &fakeHandler{name: "FakePlugin"}
	dispatcher, err := NewTaskPluginDispatcherImpl([]TaskPluginHandler{handler})

	require.NoError(t, err)
	require.NotNil(t, dispatcher)
	require.NotNil(t, dispatcher.handlers)
	require.Len(t, dispatcher.handlers, 1)
}

func TestNewTaskPluginDispatcherImpl_NilHandlers_Failure(t *testing.T) {
	dispatcher, err := NewTaskPluginDispatcherImpl(nil)

	require.Nil(t, dispatcher)
	require.Error(t, err)
	assert.Equal(t, "NewTaskPluginDispatcherImpl requires non-nil slice containing minimum one handler", err.Error())
}

func TestNewTaskPluginDispatcherImpl_EmptyHandlers_Failure(t *testing.T) {
	dispatcher, err := NewTaskPluginDispatcherImpl([]TaskPluginHandler{})

	require.Nil(t, dispatcher)
	require.Error(t, err)
	assert.Equal(t, "NewTaskPluginDispatcherImpl requires non-nil slice containing minimum one handler", err.Error())
}

func TestOnTaskStart_SingleHandler_Success(t *testing.T) {
	handler := &fakeHandler{
		name:        "FakePlugin",
		startResult: &fakeStartResult{RunID: "fake-run-1"},
	}
	dispatcher, _ := NewTaskPluginDispatcherImpl([]TaskPluginHandler{handler})

	result, err := dispatcher.OnTaskStart(context.Background(), taskInfoStart)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "fake-run-1", result.Results["FakePlugin"].(*fakeStartResult).RunID)
}

func TestOnTaskStart_MultipleHandlers_Success(t *testing.T) {
	handler1 := &fakeHandler{
		name:        "FakePluginA",
		startResult: &fakeStartResult{RunID: "run-a"},
	}
	handler2 := &fakeHandler{
		name:        "FakePluginB",
		startResult: &fakeStartResult{RunID: "run-b"},
	}
	dispatcher, _ := NewTaskPluginDispatcherImpl([]TaskPluginHandler{handler1, handler2})

	result, err := dispatcher.OnTaskStart(context.Background(), taskInfoStart)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "run-a", result.Results["FakePluginA"].(*fakeStartResult).RunID)
	assert.Equal(t, "run-b", result.Results["FakePluginB"].(*fakeStartResult).RunID)
}

func TestOnTaskStart_NilDispatcher_Failure(t *testing.T) {
	var dispatcher *TaskPluginDispatcherImpl

	result, err := dispatcher.OnTaskStart(context.Background(), taskInfoStart)

	require.Error(t, err)
	assert.Equal(t, "dispatcher and taskInfo must be non-nil", err.Error())
	require.Nil(t, result)
}

func TestOnTaskStart_NilTaskInfo_Failure(t *testing.T) {
	handler := &fakeHandler{name: "FakePlugin", startResult: &fakeStartResult{RunID: "run-1"}}
	dispatcher, _ := NewTaskPluginDispatcherImpl([]TaskPluginHandler{handler})

	result, err := dispatcher.OnTaskStart(context.Background(), nil)

	require.Error(t, err)
	assert.Equal(t, "dispatcher and taskInfo must be non-nil", err.Error())
	require.Nil(t, result)
}

func TestOnTaskStart_CustomPropertiesMerged(t *testing.T) {
	handler := &fakeHandler{
		name:        "FakePlugin",
		startResult: &fakeStartResult{RunID: "fake-run-1"},
		customProps: map[string]string{"plugins.fake.run_id": "fake-run-1"},
	}
	dispatcher, _ := NewTaskPluginDispatcherImpl([]TaskPluginHandler{handler})

	result, err := dispatcher.OnTaskStart(context.Background(), taskInfoStart)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "fake-run-1", result.CustomProperties["plugins.fake.run_id"])
}

func TestOnTaskStart_MultipleHandlers_CustomPropertiesMerged(t *testing.T) {
	handler1 := &fakeHandler{
		name:        "FakePluginA",
		startResult: &fakeStartResult{RunID: "run-a"},
		customProps: map[string]string{"plugins.a.id": "run-a"},
	}
	handler2 := &fakeHandler{
		name:        "FakePluginB",
		startResult: &fakeStartResult{RunID: "run-b"},
		customProps: map[string]string{"plugins.b.id": "run-b"},
	}
	dispatcher, _ := NewTaskPluginDispatcherImpl([]TaskPluginHandler{handler1, handler2})

	result, err := dispatcher.OnTaskStart(context.Background(), taskInfoStart)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "run-a", result.CustomProperties["plugins.a.id"])
	assert.Equal(t, "run-b", result.CustomProperties["plugins.b.id"])
}

func TestOnTaskStart_HandlerFailure_NoCustomProperties(t *testing.T) {
	handler := &fakeHandler{
		name:        "FakePlugin",
		startErr:    fmt.Errorf("plugin startup failed"),
		customProps: map[string]string{"plugins.fake.id": "should-not-appear"},
	}
	dispatcher, _ := NewTaskPluginDispatcherImpl([]TaskPluginHandler{handler})

	result, err := dispatcher.OnTaskStart(context.Background(), taskInfoStart)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Empty(t, result.CustomProperties)
}

func TestOnTaskStart_SingleHandler_HandlerFailure(t *testing.T) {
	handler := &fakeHandler{
		name:     "FakePlugin",
		startErr: fmt.Errorf("plugin startup failed"),
	}
	dispatcher, _ := NewTaskPluginDispatcherImpl([]TaskPluginHandler{handler})

	result, err := dispatcher.OnTaskStart(context.Background(), taskInfoStart)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Empty(t, result.Results)
}

func TestOnTaskEnd_SingleHandler_Success(t *testing.T) {
	handler := &fakeHandler{name: "FakePlugin"}
	dispatcher, _ := NewTaskPluginDispatcherImpl([]TaskPluginHandler{handler})

	err := dispatcher.OnTaskEnd(context.Background(), taskInfoEnd)

	require.NoError(t, err)
}

func TestOnTaskEnd_NilDispatcher_Failure(t *testing.T) {
	var dispatcher *TaskPluginDispatcherImpl

	err := dispatcher.OnTaskEnd(context.Background(), taskInfoEnd)

	require.Error(t, err)
	assert.Equal(t, "dispatcher and taskInfo must be non-nil", err.Error())
}

func TestOnTaskEnd_NilTaskInfo_Failure(t *testing.T) {
	handler := &fakeHandler{name: "FakePlugin"}
	dispatcher, _ := NewTaskPluginDispatcherImpl([]TaskPluginHandler{handler})

	err := dispatcher.OnTaskEnd(context.Background(), nil)

	require.Error(t, err)
	assert.Equal(t, "dispatcher and taskInfo must be non-nil", err.Error())
}

func TestOnTaskEnd_SingleHandler_HandlerFailure(t *testing.T) {
	handler := &fakeHandler{
		name:   "FakePlugin",
		endErr: fmt.Errorf("plugin shutdown failed"),
	}
	dispatcher, _ := NewTaskPluginDispatcherImpl([]TaskPluginHandler{handler})

	err := dispatcher.OnTaskEnd(context.Background(), taskInfoEnd)

	require.Error(t, err)
	assert.Equal(t, "failed to complete the following task-level plugin(s): [FakePlugin]", err.Error())
}

func TestOnTaskEnd_NilTaskStartResult_NoHandlerCalled(t *testing.T) {
	handler := &fakeHandler{
		name:   "FakePlugin",
		endErr: fmt.Errorf("should not be called"),
	}
	dispatcher, _ := NewTaskPluginDispatcherImpl([]TaskPluginHandler{handler})
	info := &TaskInfo{
		Name:            "test-task",
		TaskStartResult: nil,
	}

	err := dispatcher.OnTaskEnd(context.Background(), info)

	require.NoError(t, err)
}

func TestRetrieveUserContainerEnvVars_NilDispatcher_Failure(t *testing.T) {
	var dispatcher *TaskPluginDispatcherImpl

	vars, err := dispatcher.RetrieveUserContainerEnvVars(taskInfoEnd)

	require.Error(t, err)
	assert.Equal(t, "dispatcher must be non-nil", err.Error())
	require.Nil(t, vars)
}

func TestRetrieveUserContainerEnvVars_Success(t *testing.T) {
	expectedVars := map[string]string{
		"PLUGIN_RUN_ID": "fake-run-1",
	}
	handler := &fakeHandler{
		name:    "FakePlugin",
		envVars: expectedVars,
	}
	dispatcher, _ := NewTaskPluginDispatcherImpl([]TaskPluginHandler{handler})

	vars, err := dispatcher.RetrieveUserContainerEnvVars(taskInfoEnd)

	require.NoError(t, err)
	assert.Equal(t, expectedVars, vars)
}

func TestRetrieveUserContainerEnvVars_HandlerError_Skipped(t *testing.T) {
	handler := &fakeHandler{
		name:   "FakePlugin",
		envErr: fmt.Errorf("env var retrieval failed"),
	}
	dispatcher, _ := NewTaskPluginDispatcherImpl([]TaskPluginHandler{handler})

	vars, err := dispatcher.RetrieveUserContainerEnvVars(taskInfoEnd)

	require.NoError(t, err)
	assert.Empty(t, vars)
}
