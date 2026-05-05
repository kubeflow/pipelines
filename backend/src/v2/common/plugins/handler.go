package plugins

import (
	"context"
)

// TaskStartResult stores the handler-specific start results keyed by handler name.
type TaskStartResult struct {
	Results          map[string]TaskHandlerStartResult
	CustomProperties map[string]string
}

// TaskHandlerStartResult is implemented by each TaskPluginHandler to carry
// handler-specific state from OnTaskStart through to OnTaskEnd and
// RetrieveUserContainerEnvVars. Handlers type-assert to their own concrete
// implementation.
type TaskHandlerStartResult interface{}

// TaskPluginHandler defines the generic task-level plugin lifecycle hooks
type TaskPluginHandler interface {
	// Name returns the name of the plugin
	Name() string
	// OnTaskStart initializes task-level plugin execution for the specified task and returns execution results or an error.
	OnTaskStart(ctx context.Context, taskInfo *TaskInfo) (TaskHandlerStartResult, error)
	// OnTaskEnd updates task-level plugin execution for the specified task with metrics and parameters and completes plugin execution.
	OnTaskEnd(ctx context.Context, startResult TaskHandlerStartResult, taskInfo *TaskInfo) error
	// RetrieveUserContainerEnvVars returns the user-specified environment variables to be set in the task user container.
	RetrieveUserContainerEnvVars(startResult TaskHandlerStartResult) (injectVars map[string]string, err error)
	// RetrieveCustomProperties returns key-value pairs to persist as MLMD
	// execution custom properties. The driver relays these generically without
	// knowing which plugin produced them.
	RetrieveCustomProperties(startResult TaskHandlerStartResult) map[string]string
	// ApplyCustomProperties applies properties represented by key-value pairs to the handler configuration.
	ApplyCustomProperties(customProperties map[string]string) error
}
