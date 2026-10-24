package poc

import (
	"fmt"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	apiv2 "github.com/kubeflow/pipelines/backend/api/v2/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/v2/component"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
)

func reconcileLauncherCheckpoint(task *model.Task, attemptID string, result *TaskExecutionResult) (*TaskExecutionResult, error) {
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
	if attemptID == "" || report.AttemptID != attemptID {
		return result, nil
	}
	if len(result.OutputParameters) == 0 && len(task.OutputParameters) > 0 {
		parameters, err := model.JSONSliceToProtoSlice(task.OutputParameters, func() *apiv2.PipelineTask_InputOutputs_IOParameter {
			return &apiv2.PipelineTask_InputOutputs_IOParameter{}
		})
		if err != nil {
			return nil, err
		}
		result.OutputParameters = map[string]*structpb.Value{}
		for _, parameter := range parameters {
			result.OutputParameters[parameter.GetParameterKey()] = parameter.GetValue()
		}
	}
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
		if class == "output-contract" {
			result.Outcome = TaskOutcomeFailure
		}
		result.Failure = &TaskFailure{Class: class, Message: message, Fatal: class == "publication"}
	}
	return result, nil
}
