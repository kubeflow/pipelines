package poc

import (
	"fmt"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/v2/component"
	"google.golang.org/protobuf/encoding/protojson"
)

func (c *Coordinator) reconcileLauncherResult(attempt *model.RuntimeTask, result *TaskExecutionResult) (*TaskExecutionResult, error) {
	task, err := c.taskStore.GetTask(attempt.TaskUUID)
	if err != nil {
		return nil, err
	}
	properties, _ := task.StatusMetadata["customProperties"].(map[string]interface{})
	raw, exists := properties[component.RuntimeLauncherReportKey]
	if !exists {
		return result, nil
	}
	encoded, ok := raw.(string)
	if !ok {
		return nil, fmt.Errorf("task %s has a malformed launcher checkpoint", task.UUID)
	}
	report, err := component.ParseRuntimeLauncherReport(encoded)
	if err != nil {
		return nil, fmt.Errorf("task %s has an invalid launcher checkpoint: %w", task.UUID, err)
	}
	if attempt.AttemptUUID == nil || report.AttemptID != *attempt.AttemptUUID {
		return result, nil
	}
	result.ComputationCompleted = report.ComputationCompleted
	result.PublicationCompleted = report.PublicationCompleted
	if report.ExecutorOutputJSON != "" {
		output := &pipelinespec.ExecutorOutput{}
		if err := protojson.Unmarshal([]byte(report.ExecutorOutputJSON), output); err != nil {
			return nil, fmt.Errorf("task %s has an invalid checkpointed executor output: %w", task.UUID, err)
		}
		result.OutputParameters = output.GetParameterValues()
	}
	if report.PublicationCompleted {
		result.Outcome, result.Failure = TaskOutcomeSuccess, nil
	} else if report.ComputationCompleted {
		class := report.FailureClass
		if class != "output-contract" {
			class = "publication"
		}
		message := report.Message
		if message == "" {
			message = "workload stopped after successful computation but before publication completed"
		}
		result.Outcome = TaskOutcomeError
		result.Failure = &TaskFailure{Class: class, Message: message}
	}
	return result, nil
}
