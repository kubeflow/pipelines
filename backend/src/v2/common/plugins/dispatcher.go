package plugins

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/golang/glog"
	corev1 "k8s.io/api/core/v1"
)

// TaskPluginDispatcher orchestrates plugin lifecycle hooks
type TaskPluginDispatcher interface {
	// OnTaskStart is called when a task starts.
	// The dispatcher reads taskInfo and pluginConfig and returns a TaskStartResult.
	// Individual handler errors are best-effort (logged but non-blocking); the
	// dispatcher continues with remaining handlers. Only handlers that started
	// successfully will have OnTaskEnd invoked.
	OnTaskStart(ctx context.Context, taskInfo *TaskInfo) (*TaskStartResult, error)

	// OnTaskEnd is called when a task reaches a terminal state. Returns true if all plugin syncs succeeded.
	OnTaskEnd(ctx context.Context, taskInfo *TaskInfo) error

	// RetrieveUserContainerEnvVars returns the user-specified environment variables to be set in the task user container.
	RetrieveUserContainerEnvVars(taskInfo *TaskInfo) (envVars []corev1.EnvVar, err error)

	// ApplyCustomProperties updates the custom properties for all registered task-level plugins.
	ApplyCustomProperties(properties map[string]string)
}

// NoOpDispatcher is a TaskPluginDispatcher that does nothing.
type NoOpDispatcher struct{}

func (NoOpDispatcher) OnTaskStart(ctx context.Context, taskInfo *TaskInfo) (*TaskStartResult, error) {
	return nil, nil
}
func (NoOpDispatcher) OnTaskEnd(ctx context.Context, taskInfo *TaskInfo) error {
	return nil
}
func (NoOpDispatcher) RetrieveUserContainerEnvVars(taskInfo *TaskInfo) (envVars []corev1.EnvVar, err error) {
	return nil, nil
}
func (NoOpDispatcher) ApplyCustomProperties(properties map[string]string) {
}

var _ TaskPluginDispatcher = NoOpDispatcher{}

var _ TaskPluginDispatcher = (*TaskPluginDispatcherImpl)(nil)

type TaskPluginDispatcherImpl struct {
	handlers        []TaskPluginHandler
	startedHandlers map[string]bool
}

func NewTaskPluginDispatcherImpl(handlers []TaskPluginHandler) (*TaskPluginDispatcherImpl, error) {
	if len(handlers) == 0 {
		return nil, fmt.Errorf("NewTaskPluginDispatcherImpl requires non-nil slice containing minimum one handler")
	}
	sorted := make([]TaskPluginHandler, len(handlers))
	copy(sorted, handlers)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Name() < sorted[j].Name()
	})
	return &TaskPluginDispatcherImpl{
		handlers: sorted,
	}, nil
}

func (t *TaskPluginDispatcherImpl) OnTaskStart(ctx context.Context, taskInfo *TaskInfo) (*TaskStartResult, error) {
	if t == nil || taskInfo == nil {
		return nil, fmt.Errorf("dispatcher and taskInfo must be non-nil")
	}

	t.startedHandlers = make(map[string]bool)
	handlerResults := map[string]TaskHandlerStartResult{}
	customProperties := map[string]string{}
	for _, handler := range t.handlers {
		result, err := handler.OnTaskStart(ctx, taskInfo)
		if err != nil {
			glog.Errorf("failed to launch task-level %s handler: %v", handler.Name(), err)
			continue
		}
		t.startedHandlers[handler.Name()] = true
		handlerResults[handler.Name()] = result
		for k, v := range handler.GenerateCustomProperties(result) {
			customProperties[k] = v
		}
	}
	return &TaskStartResult{
		Results:          handlerResults,
		CustomProperties: customProperties,
	}, nil
}

func (t *TaskPluginDispatcherImpl) OnTaskEnd(ctx context.Context, taskInfo *TaskInfo) error {
	if t == nil || taskInfo == nil {
		return fmt.Errorf("dispatcher and taskInfo must be non-nil")
	}

	taskInfo.UpdateTaskInfoWithRunEndTime(time.Now().UnixMilli())

	var taskEndFailures []string
	for _, handler := range t.handlers {
		if t.startedHandlers != nil && !t.startedHandlers[handler.Name()] {
			glog.Infof("Skipping OnTaskEnd for handler %s (OnTaskStart did not succeed)", handler.Name())
			continue
		}
		err := handler.OnTaskEnd(ctx, taskInfo)
		if err != nil {
			glog.Errorf("Failed to complete task-level %s handler: %v", handler.Name(), err)
			taskEndFailures = append(taskEndFailures, handler.Name())
		}
	}
	if len(taskEndFailures) > 0 {
		return fmt.Errorf("failed to complete the following task-level plugin(s): %v", taskEndFailures)
	}
	return nil
}

func (t *TaskPluginDispatcherImpl) RetrieveUserContainerEnvVars(taskInfo *TaskInfo) (injectVars []corev1.EnvVar, err error) {
	if t == nil {
		return nil, fmt.Errorf("dispatcher must be non-nil")
	}

	injectVarNameSet := map[string]bool{}
	for _, handler := range t.handlers {
		vars, err := handler.RetrieveUserContainerEnvVars()
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve user container env vars for handler %s: %v", handler.Name(), err)
		}

		for _, envVar := range vars {
			if !injectVarNameSet[envVar.Name] {
				injectVars = append(injectVars, envVar)
				injectVarNameSet[envVar.Name] = true
			} else {
				glog.Errorf("Key %s already present in container env vars. Duplicate env var will not be added.", envVar.Name)
			}
		}
	}
	return injectVars, nil
}

// ApplyCustomProperties updates the custom properties for all registered task-level plugins.
func (t *TaskPluginDispatcherImpl) ApplyCustomProperties(properties map[string]string) {
	for _, handler := range t.handlers {
		err := handler.ApplyCustomProperties(properties)
		if err != nil {
			glog.Errorf("failed to apply custom properties for handler %s: %v", handler.Name(), err)
		}
	}
}
