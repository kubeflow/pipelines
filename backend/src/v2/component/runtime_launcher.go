// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package component

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	apiV2 "github.com/kubeflow/pipelines/backend/api/v2/go_client"
	"github.com/kubeflow/pipelines/backend/src/v2/apiclient"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
)

const (
	RuntimeLauncherReportKey         = "kfp_runtime_launcher"
	RuntimeLauncherTerminationPath   = "/dev/termination-log"
	RuntimePublicationMaxAttempts    = 3
	RuntimePhaseComputed             = "computed"
	RuntimePhasePublishing           = "publishing"
	RuntimePhasePublished            = "published"
	RuntimePhasePublicationFailed    = "publication-failed"
	RuntimePhaseOutputContractFailed = "output-contract"
)

// RuntimeLauncherReport separates completed computation from bounded publication.
// The task API and local checkpoint retain the payload; the Pod termination
// message contains only the small summary needed by the workload executor.
type RuntimeLauncherReport struct {
	Version              int    `json:"version"`
	AttemptID            string `json:"attemptId"`
	ComputationCompleted bool   `json:"computationCompleted"`
	PublicationCompleted bool   `json:"publicationCompleted"`
	Phase                string `json:"phase"`
	PublicationAttempts  int    `json:"publicationAttempts"`
	FailureClass         string `json:"failureClass,omitempty"`
	Message              string `json:"message,omitempty"`
	ExecutorInputJSON    string `json:"executorInput,omitempty"`
	ExecutorOutputJSON   string `json:"executorOutput,omitempty"`
}

// ParseRuntimeLauncherReport validates the versioned task/Pod publication report.
func ParseRuntimeLauncherReport(raw string) (*RuntimeLauncherReport, error) {
	report := &RuntimeLauncherReport{}
	if err := json.Unmarshal([]byte(raw), report); err != nil {
		return nil, err
	}
	if report.Version != 1 || report.AttemptID == "" ||
		report.PublicationAttempts < 0 || report.PublicationAttempts > RuntimePublicationMaxAttempts {
		return nil, fmt.Errorf("invalid runtime launcher report identity, version or publication budget")
	}
	switch report.Phase {
	case RuntimePhaseComputed, RuntimePhasePublishing, RuntimePhasePublished, RuntimePhasePublicationFailed, RuntimePhaseOutputContractFailed:
	default:
		return nil, fmt.Errorf("unknown runtime launcher phase %q", report.Phase)
	}
	if !report.ComputationCompleted || (report.PublicationCompleted && report.Phase != RuntimePhasePublished) {
		return nil, fmt.Errorf("runtime launcher report has inconsistent computation/publication completion")
	}
	return report, nil
}

// RuntimeLauncherReportForTask reads the attempt-scoped computation checkpoint.
func RuntimeLauncherReportForTask(task *apiV2.PipelineTask) (*RuntimeLauncherReport, error) {
	raw := task.GetStatusMetadata().GetCustomProperties()[RuntimeLauncherReportKey].GetStringValue()
	if raw == "" {
		return nil, nil
	}
	return ParseRuntimeLauncherReport(raw)
}

func (l *LauncherV2) runtimeCheckpointPath() (string, error) {
	attempt := l.options.TaskAttemptID
	if attempt == "" || strings.ContainsAny(attempt, `/\`) || attempt == "." || attempt == ".." {
		return "", fmt.Errorf("runtime launcher requires a safe task attempt ID")
	}
	output := l.executorInput.GetOutputs().GetOutputFile()
	if output == "" {
		return "", fmt.Errorf("runtime launcher requires an executor output file for durable staging")
	}
	return filepath.Join(filepath.Dir(output), ".kfp-runtime-"+attempt+".json"), nil
}

func (l *LauncherV2) loadRuntimeReport() (*RuntimeLauncherReport, error) {
	path, err := l.runtimeCheckpointPath()
	if err != nil {
		return nil, err
	}
	raw, err := l.fileSystem.ReadFile(path)
	if err == nil {
		report, err := ParseRuntimeLauncherReport(string(raw))
		if err != nil {
			return nil, fmt.Errorf("invalid local runtime checkpoint; refusing to rerun computation: %w", err)
		}
		if report.AttemptID != l.options.TaskAttemptID {
			return nil, fmt.Errorf("local checkpoint belongs to a different attempt")
		}
		return report, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	report, err := RuntimeLauncherReportForTask(l.options.Task)
	if err != nil {
		return nil, err
	}
	if report != nil && report.AttemptID != l.options.TaskAttemptID {
		return nil, nil
	}
	return report, nil
}

func (l *LauncherV2) writeRuntimeCheckpoint(report *RuntimeLauncherReport) error {
	path, err := l.runtimeCheckpointPath()
	if err != nil {
		return err
	}
	raw, err := json.Marshal(report)
	if err != nil {
		return err
	}
	if err := l.fileSystem.MkdirAll(filepath.Dir(path), 0o777); err != nil {
		return err
	}
	if _, realFileSystem := l.fileSystem.(*OSFileSystem); !realFileSystem {
		return l.fileSystem.WriteFile(path, raw, 0o600)
	}
	// A launcher restart must see the old or new checkpoint, never a torn JSON file.
	file, err := os.OpenFile(path+".next", os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	if _, err := file.Write(raw); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Rename(path+".next", path)
}

func (l *LauncherV2) writeRuntimeTerminationReport(report *RuntimeLauncherReport) error {
	summary := *report
	summary.ExecutorInputJSON, summary.ExecutorOutputJSON = "", ""
	if len(summary.Message) > 512 {
		summary.Message = summary.Message[:512]
	}
	raw, err := json.Marshal(summary)
	if err != nil {
		return err
	}
	return l.fileSystem.WriteFile(RuntimeLauncherTerminationPath, raw, 0o600)
}

func (l *LauncherV2) persistRuntimeReport(ctx context.Context, report *RuntimeLauncherReport) error {
	terminationErr := l.writeRuntimeTerminationReport(report)
	checkpointErr := l.writeRuntimeCheckpoint(report)
	raw, err := json.Marshal(report)
	if err != nil {
		return err
	}
	task := l.options.Task
	task.State = apiV2.PipelineTask_RUNNING
	task.EndTime = nil
	if task.StatusMetadata == nil {
		task.StatusMetadata = &apiV2.PipelineTask_StatusMetadata{}
	}
	if task.StatusMetadata.CustomProperties == nil {
		task.StatusMetadata.CustomProperties = map[string]*structpb.Value{}
	}
	task.StatusMetadata.CustomProperties[RuntimeLauncherReportKey] = structpb.NewStringValue(string(raw))
	var lastErr error
	for attempt := 0; attempt < RuntimePublicationMaxAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		reportCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
		_, lastErr = l.clientManager.KFPAPIClient().UpdateTask(reportCtx, &apiV2.UpdateTaskRequest{
			RunId: l.options.Run.GetRunId(), TaskId: task.GetTaskId(), Task: task,
		})
		cancel()
		if lastErr == nil {
			return errors.Join(terminationErr, checkpointErr)
		}
	}
	return errors.Join(terminationErr, checkpointErr, fmt.Errorf("cannot persist computation/publication checkpoint: %w", lastErr))
}

func (l *LauncherV2) failRuntimePublication(ctx context.Context, report *RuntimeLauncherReport, class string, failure error) error {
	report.PublicationCompleted = false
	report.FailureClass, report.Message = class, failure.Error()
	report.Phase = RuntimePhasePublicationFailed
	if class == "output-contract" {
		report.Phase = RuntimePhaseOutputContractFailed
	}
	if err := l.persistRuntimeReport(ctx, report); err != nil {
		return errors.Join(failure, err)
	}
	return fmt.Errorf("%s failed after successful computation: %w", class, failure)
}

func (l *LauncherV2) executeRuntimeAttempt(ctx context.Context) (*pipelinespec.ExecutorOutput, error) {
	if taskID := os.Getenv(apiclient.KFPTaskIDEnvVar); taskID != "" && taskID != l.options.Task.GetTaskId() {
		return nil, fmt.Errorf("runtime task identity differs from launcher task")
	}
	report, err := l.loadRuntimeReport()
	if err != nil {
		return nil, err
	}
	if report == nil {
		report = &RuntimeLauncherReport{Version: 1, AttemptID: l.options.TaskAttemptID, Phase: RuntimePhaseComputed}
	}
	output, err := l.runtimeComputationOutput(ctx, report)
	if err != nil {
		return output, err
	}
	if report.Phase == RuntimePhasePublicationFailed || report.Phase == RuntimePhaseOutputContractFailed {
		_ = l.writeRuntimeTerminationReport(report)
		return output, fmt.Errorf("retained %s failure: %s", report.FailureClass, report.Message)
	}
	if report.PublicationCompleted {
		return output, l.persistRuntimeReport(ctx, report)
	}
	for report.PublicationAttempts < RuntimePublicationMaxAttempts {
		report.Phase = RuntimePhasePublishing
		report.PublicationAttempts++
		if err := l.persistRuntimeReport(ctx, report); err != nil {
			if ctx.Err() != nil {
				return output, ctx.Err()
			}
			return output, l.failRuntimePublication(ctx, report, "publication", err)
		}
		publicationCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		l.batchUpdater = NewBatchUpdater()
		err = l.uploadOutputArtifacts(publicationCtx, output)
		if err == nil {
			err = l.batchUpdater.Flush(publicationCtx, l.clientManager.KFPAPIClient())
		}
		cancel()
		if err == nil {
			report.Phase, report.PublicationCompleted = RuntimePhasePublished, true
			if err := l.persistRuntimeReport(ctx, report); err != nil {
				if ctx.Err() != nil {
					return output, ctx.Err()
				}
				return output, l.failRuntimePublication(ctx, report, "publication", err)
			}
			return output, nil
		}
		if ctx.Err() != nil {
			return output, ctx.Err()
		}
		if !isRuntimePublicationTransient(err) || report.PublicationAttempts == RuntimePublicationMaxAttempts {
			break
		}
		if refreshErr := l.objectStore.RefreshSession(); refreshErr != nil {
			err = errors.Join(err, refreshErr)
			break
		}
		timer := time.NewTimer(time.Duration(report.PublicationAttempts) * 100 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return output, ctx.Err()
		case <-timer.C:
		}
	}
	if err == nil {
		err = fmt.Errorf("publication budget exhausted after interrupted attempts")
	}
	return output, l.failRuntimePublication(ctx, report, "publication", err)
}

func isRuntimePublicationTransient(err error) bool {
	if isTransientArtifactUploadError(err) || errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	switch status.Code(err) {
	case codes.Unavailable, codes.DeadlineExceeded, codes.Aborted, codes.ResourceExhausted:
		return true
	default:
		return false
	}
}

func (l *LauncherV2) runtimeComputationOutput(ctx context.Context, report *RuntimeLauncherReport) (*pipelinespec.ExecutorOutput, error) {
	if report.ExecutorInputJSON != "" {
		input := &pipelinespec.ExecutorInput{}
		if err := protojson.Unmarshal([]byte(report.ExecutorInputJSON), input); err != nil {
			return nil, err
		}
		l.executorInput = input
	}
	output := &pipelinespec.ExecutorOutput{}
	if report.ComputationCompleted {
		l.computationCompleted = true
		if report.ExecutorOutputJSON != "" {
			if err := protojson.Unmarshal([]byte(report.ExecutorOutputJSON), output); err != nil {
				return nil, err
			}
		} else {
			var err error
			output, err = l.getExecutorOutputFile(l.executorInput.GetOutputs().GetOutputFile())
			if err != nil {
				return nil, l.failRuntimePublication(ctx, report, "output-contract", err)
			}
		}
	} else {
		defaults, err := AddDefaultParams(l.executorInput, l.options.ComponentSpec)
		if err != nil {
			return nil, err
		}
		command, args, err := compileCmdAndArgs(defaults, l.command, l.args)
		if err != nil {
			return nil, err
		}
		var checkpointFailure error
		l.onComputationCompleted = func(ctx context.Context) error {
			report.ComputationCompleted = true
			inputJSON, err := protojson.Marshal(l.executorInput)
			if err != nil {
				return err
			}
			report.ExecutorInputJSON = string(inputJSON)
			checkpointFailure = l.persistRuntimeReport(ctx, report)
			return checkpointFailure
		}
		output, err = l.execute(ctx, command, args)
		l.onComputationCompleted = nil
		if err != nil {
			if l.computationCompleted {
				if ctx.Err() != nil {
					return nil, ctx.Err()
				}
				class := "output-contract"
				if checkpointFailure != nil {
					class = "publication"
				}
				return nil, l.failRuntimePublication(ctx, report, class, err)
			}
			return nil, err
		}
		if !report.ComputationCompleted {
			report.ComputationCompleted = true
			if err := l.persistRuntimeReport(ctx, report); err != nil {
				return nil, err
			}
		}
	}
	if err := l.collectOutputParameters(output); err != nil {
		return output, l.failRuntimePublication(ctx, report, "output-contract", err)
	}
	outputJSON, err := protojson.Marshal(output)
	if err != nil {
		return output, l.failRuntimePublication(ctx, report, "output-contract", err)
	}
	report.ExecutorOutputJSON = string(outputJSON)
	if l.options.Task.Outputs == nil {
		l.options.Task.Outputs = &apiV2.PipelineTask_InputOutputs{}
	}
	parameters := []*apiV2.PipelineTask_InputOutputs_IOParameter{}
	for key, value := range output.GetParameterValues() {
		parameter := &apiV2.PipelineTask_InputOutputs_IOParameter{ParameterKey: key, Type: apiV2.IOType_OUTPUT,
			Value: value, Producer: &apiV2.IOProducer{TaskName: l.options.Task.GetName()}}
		if l.options.IterationIndex != nil {
			parameter.Type = apiV2.IOType_ITERATOR_OUTPUT
			parameter.Producer.Iteration = l.options.IterationIndex
		}
		parameters = append(parameters, parameter)
	}
	l.options.Task.Outputs.Parameters = parameters
	if err := l.persistRuntimeReport(ctx, report); err != nil {
		if ctx.Err() != nil {
			return output, ctx.Err()
		}
		return output, l.failRuntimePublication(ctx, report, "publication", err)
	}
	return output, nil
}
