package plugins

import (
	"context"

	"github.com/kubeflow/pipelines/backend/src/v2/metadata"
	k8score "k8s.io/api/core/v1"
)

// TaskPluginDispatcher orchestrates plugin lifecycle hooks
type TaskPluginDispatcher interface {
	// OnTaskStart is called when a task starts.
	// The dispatcher reads taskInfo and pluginConfig and returns a TaskStartResult.
	// Returns an error only for validation failures that should block task execution.
	OnTaskStart(ctx context.Context, taskName string) (*TaskStartResult, error)

	// OnTaskEnd is called when a task reaches a terminal state. Returns true if all plugin syncs succeeded.
	OnTaskEnd(ctx context.Context, taskExecution *metadata.Execution, outputArtifacts []*metadata.OutputArtifact) error

	// RetrieveUserContainerEnvVars returns the user-specified environment variables to be set in the task user container.
	RetrieveUserContainerEnvVars() (envVars []k8score.EnvVar)
}

// NoOpDispatcher is a TaskPluginDispatcher that does nothing.
type NoOpDispatcher struct{}

func (NoOpDispatcher) OnTaskStart(ctx context.Context, taskName string) (*TaskStartResult, error) {
	return nil, nil
}
func (NoOpDispatcher) OnTaskEnd(ctx context.Context, taskExecution *metadata.Execution, outputArtifacts []*metadata.OutputArtifact) error {
	return nil
}
func (NoOpDispatcher) RetrieveUserContainerEnvVars() (envVars []k8score.EnvVar) {
	return nil
}

var _ TaskPluginDispatcher = NoOpDispatcher{}
