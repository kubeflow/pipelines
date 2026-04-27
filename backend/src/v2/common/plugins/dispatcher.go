package plugins

import (
	"context"
	"fmt"
	"time"

	"github.com/golang/glog"
)

// TaskPluginDispatcher orchestrates plugin lifecycle hooks
type TaskPluginDispatcher interface {
	// OnTaskStart is called when a task starts.
	// The dispatcher reads taskInfo and pluginConfig and returns a TaskStartResult.
	// Returns an error only for validation failures that should block task execution.
	OnTaskStart(ctx context.Context, taskInfo *TaskInfo) (*TaskStartResult, error)

	// OnTaskEnd is called when a task reaches a terminal state. Returns true if all plugin syncs succeeded.
	OnTaskEnd(ctx context.Context, taskInfo *TaskInfo) error

	// RetrieveUserContainerEnvVars returns the user-specified environment variables to be set in the task user container.
	RetrieveUserContainerEnvVars(taskInfo *TaskInfo) (envVars map[string]string, err error)

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
func (NoOpDispatcher) RetrieveUserContainerEnvVars(taskInfo *TaskInfo) (envVars map[string]string, err error) {
	return nil, nil
}
func (NoOpDispatcher) ApplyCustomProperties(properties map[string]string) {
}

var _ TaskPluginDispatcher = NoOpDispatcher{}

var _ TaskPluginDispatcher = (*TaskPluginDispatcherImpl)(nil)

type TaskPluginDispatcherImpl struct {
	handlers []TaskPluginHandler
}

func NewTaskPluginDispatcherImpl(handlers []TaskPluginHandler) (*TaskPluginDispatcherImpl, error) {
	if handlers == nil || len(handlers) == 0 {
		return nil, fmt.Errorf("NewTaskPluginDispatcherImpl requires non-nil slice containing minimum one handler")
	}
	return &TaskPluginDispatcherImpl{
		handlers: handlers,
	}, nil
}

func (t *TaskPluginDispatcherImpl) OnTaskStart(ctx context.Context, taskInfo *TaskInfo) (*TaskStartResult, error) {
	if t == nil || taskInfo == nil {
		return nil, fmt.Errorf("dispatcher and taskInfo must be non-nil")
	}

	handlerResults := map[string]TaskHandlerStartResult{}
	customProperties := map[string]string{}
	for _, handler := range t.handlers {
		result, err := handler.OnTaskStart(ctx, taskInfo)
		if err != nil {
			glog.Errorf("failed to launch task-level %s handler: %v", handler.Name(), err)
			continue
		}
		handlerResults[handler.Name()] = result
		for k, v := range handler.RetrieveCustomProperties(result) {
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
		var result TaskHandlerStartResult
		taskStartResult := taskInfo.TaskStartResult
		if taskStartResult != nil {
			var ok bool
			result, ok = taskInfo.TaskStartResult.Results[handler.Name()]
			if !ok {
				glog.Infof("Task-level %s handler TaskStartResult not stored on dispatcher", handler.Name())
			}
		}
		err := handler.OnTaskEnd(ctx, result, taskInfo)
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

func (t *TaskPluginDispatcherImpl) RetrieveUserContainerEnvVars(taskInfo *TaskInfo) (injectVars map[string]string, err error) {
	if t == nil {
		return nil, fmt.Errorf("dispatcher must be non-nil")
	}

	injectVars = make(map[string]string)
	for _, handler := range t.handlers {
		taskStartResult := taskInfo.TaskStartResult
		if taskStartResult != nil {
			result, ok := taskInfo.TaskStartResult.Results[handler.Name()]
			if !ok {
				glog.Infof("Task-level %s handler TaskStartResult not stored on dispatcher", handler.Name())
			}
			vars, err := handler.RetrieveUserContainerEnvVars(result)
			if err != nil {
				glog.Errorf("failed to retrieve user container env vars for handler %s: %v", handler.Name(), err)
				continue
			}

			for k, v := range vars {
				if _, ok = injectVars[k]; !ok {
					injectVars[k] = v
				} else {
					glog.Errorf("Key %s already present in container env vars. This key-value pair will not be added.", k)
				}
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
