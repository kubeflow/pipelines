// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package component

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	pipelinehttp "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_client"
	pipelineparams "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_client/pipeline_service"
	pipelinemodel "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_model"
	runhttp "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/run_client"
	runparams "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/run_client/run_service"
	runmodel "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/run_model"
	"github.com/kubeflow/pipelines/backend/src/common/client/api_server"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata"
	pb "github.com/kubeflow/pipelines/third_party/ml-metadata/go/ml_metadata"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
)

const (
	// LabelParentRunID marks a child run with its triggering parent run ID.
	// Stored in the child run Description until the Run API supports labels.
	LabelParentRunID = "pipelines.kubeflow.org/parent-run-id"
	// LabelParentTaskName marks a child run with the parent trigger task name.
	LabelParentTaskName = "pipelines.kubeflow.org/parent-task-name"
	// MLMDChildRunIDKey is the MLMD custom property used by the UI for Open Run.
	MLMDChildRunIDKey = "child_run_id"

	triggerOutputRunID = "run_id"
	triggerOutputState = "state"
)

// ParentRunLabels returns the parent linkage labels for a triggered child run.
func ParentRunLabels(parentRunID, parentTaskName string) map[string]string {
	return map[string]string{
		LabelParentRunID:    parentRunID,
		LabelParentTaskName: parentTaskName,
	}
}

// FormatParentLabelsDescription encodes parent labels into the Run Description
// field (v2beta1 Run has no Labels field yet).
func FormatParentLabelsDescription(parentRunID, parentTaskName string) string {
	return fmt.Sprintf("%s=%s %s=%s", LabelParentRunID, parentRunID, LabelParentTaskName, parentTaskName)
}

type TriggerPipelineLauncherOptions struct {
	PipelineName string
	RunID        string
	ParentDagID  int64
}

func (o *TriggerPipelineLauncherOptions) validate() error {
	if o == nil {
		return fmt.Errorf("empty trigger pipeline launcher options")
	}
	if o.PipelineName == "" {
		return fmt.Errorf("trigger pipeline launcher options: pipeline name is empty")
	}
	if o.RunID == "" {
		return fmt.Errorf("trigger pipeline launcher options: Run ID is empty")
	}
	if o.ParentDagID == 0 {
		return fmt.Errorf("trigger pipeline launcher options: Parent DAG ID is not provided")
	}
	return nil
}

type triggerAPIClients struct {
	pipeline *pipelinehttp.Pipeline
	run      *runhttp.Run
	auth     runtime.ClientAuthInfoWriter
}

type TriggerPipelineLauncher struct {
	component         *pipelinespec.ComponentSpec
	trigger           *pipelinespec.PipelineDeploymentConfig_TriggerPipelineSpec
	task              *pipelinespec.PipelineTaskSpec
	launcherV2Options LauncherV2Options
	triggerOpts       TriggerPipelineLauncherOptions

	metadataClient *metadata.Client
	api            triggerAPIClients
}

func newTriggerAPIClients(namespace string, tlsCfg *tls.Config) triggerAPIClients {
	httpRuntime := api_server.NewKubeflowInClusterHTTPRuntime(namespace, false, tlsCfg)
	return triggerAPIClients{
		pipeline: pipelinehttp.New(httpRuntime, strfmt.Default),
		run:      runhttp.New(httpRuntime, strfmt.Default),
		auth:     api_server.SATokenVolumeProjectionAuth,
	}
}

func NewTriggerPipelineLauncher(
	ctx context.Context,
	componentSpecJSON, triggerSpecJSON, taskSpecJSON string,
	launcherV2Opts *LauncherV2Options,
	triggerOpts *TriggerPipelineLauncherOptions,
) (l *TriggerPipelineLauncher, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to create trigger pipeline launcher: %w", err)
		}
	}()
	component := &pipelinespec.ComponentSpec{}
	if err = protojson.Unmarshal([]byte(componentSpecJSON), component); err != nil {
		return nil, fmt.Errorf("failed to unmarshal component spec: %w", err)
	}
	trigger := &pipelinespec.PipelineDeploymentConfig_TriggerPipelineSpec{}
	if err = protojson.Unmarshal([]byte(triggerSpecJSON), trigger); err != nil {
		return nil, fmt.Errorf("failed to unmarshal trigger pipeline spec: %w", err)
	}
	task := &pipelinespec.PipelineTaskSpec{}
	if err = protojson.Unmarshal([]byte(taskSpecJSON), task); err != nil {
		return nil, fmt.Errorf("failed to unmarshal task spec: %w", err)
	}
	if err = launcherV2Opts.validate(); err != nil {
		return nil, err
	}
	if err = triggerOpts.validate(); err != nil {
		return nil, err
	}
	tlsCfg, err := util.GetTLSConfig(launcherV2Opts.CaCertPath)
	if err != nil {
		return nil, err
	}
	metadataClient, err := metadata.NewClient(launcherV2Opts.MLMDServerAddress, launcherV2Opts.MLMDServerPort, tlsCfg)
	if err != nil {
		return nil, err
	}
	_ = ctx
	return &TriggerPipelineLauncher{
		component:         component,
		trigger:           trigger,
		task:              task,
		launcherV2Options: *launcherV2Opts,
		triggerOpts:       *triggerOpts,
		metadataClient:    metadataClient,
		api:               newTriggerAPIClients(launcherV2Opts.Namespace, tlsCfg),
	}, nil
}

func (l *TriggerPipelineLauncher) Execute(ctx context.Context) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to execute trigger pipeline component: %w", err)
		}
	}()

	pipeline, err := l.metadataClient.GetPipeline(ctx, l.triggerOpts.PipelineName, l.triggerOpts.RunID, "", "", "", "")
	if err != nil {
		return err
	}
	ecfg := &metadata.ExecutionConfig{
		TaskName:      l.task.GetTaskInfo().GetName(),
		PodName:       l.launcherV2Options.PodName,
		PodUID:        l.launcherV2Options.PodUID,
		Namespace:     l.launcherV2Options.Namespace,
		ExecutionType: metadata.ContainerExecutionTypeName,
		ParentDagID:   l.triggerOpts.ParentDagID,
	}
	createdExecution, err := l.metadataClient.CreateExecution(ctx, pipeline, ecfg)
	if err != nil {
		return err
	}

	params, err := l.resolveInputParameters(ctx, pipeline)
	if err != nil {
		return err
	}

	childPipeline, err := l.getPipelineByName(l.trigger.GetPipelineName())
	if err != nil {
		return fmt.Errorf("failed to resolve pipeline %q: %w", l.trigger.GetPipelineName(), err)
	}

	parentRun, err := l.getRun(l.triggerOpts.RunID)
	if err != nil {
		return fmt.Errorf("failed to get parent run %q: %w", l.triggerOpts.RunID, err)
	}

	taskName := l.task.GetTaskInfo().GetName()
	displayName := fmt.Sprintf("%s-from-%s", l.trigger.GetPipelineName(), taskName)
	if len(displayName) > 100 {
		displayName = displayName[:100]
	}

	runtimeParams := make(map[string]interface{}, len(params))
	for k, v := range params {
		runtimeParams[k] = structpbValueToInterface(v)
	}

	childRun, err := l.createRun(&runmodel.V2beta1Run{
		ExperimentID: parentRun.ExperimentID,
		DisplayName:  displayName,
		Description:  FormatParentLabelsDescription(l.triggerOpts.RunID, taskName),
		PipelineVersionReference: &runmodel.V2beta1PipelineVersionReference{
			PipelineID:        childPipeline.PipelineID,
			PipelineVersionID: l.trigger.GetPipelineVersionId(),
		},
		RuntimeConfig: &runmodel.V2beta1RuntimeConfig{
			Parameters: runtimeParams,
		},
		ServiceAccount: parentRun.ServiceAccount,
	})
	if err != nil {
		return fmt.Errorf("failed to create child run: %w", err)
	}
	glog.Infof("Created child run %s for pipeline %s", childRun.RunID, l.trigger.GetPipelineName())

	state := ""
	if childRun.State != nil {
		state = string(*childRun.State)
	}

	if l.trigger.GetWaitForCompletion() {
		poke := time.Duration(l.trigger.GetPokeIntervalSeconds()) * time.Second
		if poke <= 0 {
			poke = 30 * time.Second
		}
		childRun, err = l.waitForTerminal(ctx, childRun.RunID, poke)
		if err != nil {
			return err
		}
		if childRun.State != nil {
			state = string(*childRun.State)
		}
		if childRun.State == nil || *childRun.State != runmodel.V2beta1RuntimeStateSUCCEEDED {
			_ = l.publish(ctx, createdExecution, childRun.RunID, state, pb.Execution_FAILED)
			return fmt.Errorf("child run %s finished with state %s (expected SUCCEEDED)", childRun.RunID, state)
		}
	}

	return l.publish(ctx, createdExecution, childRun.RunID, state, pb.Execution_COMPLETE)
}

func (l *TriggerPipelineLauncher) getPipelineByName(name string) (*pipelinemodel.V2beta1Pipeline, error) {
	ctx, cancel := context.WithTimeout(context.Background(), api_server.APIServerDefaultTimeout)
	defer cancel()
	params := pipelineparams.NewPipelineServiceGetPipelineByNameParams().
		WithContext(ctx).
		WithName(name).
		WithNamespace(&l.launcherV2Options.Namespace)
	resp, err := l.api.pipeline.PipelineService.PipelineServiceGetPipelineByName(params, l.api.auth)
	if err != nil {
		return nil, err
	}
	return resp.Payload, nil
}

func (l *TriggerPipelineLauncher) getRun(runID string) (*runmodel.V2beta1Run, error) {
	ctx, cancel := context.WithTimeout(context.Background(), api_server.APIServerDefaultTimeout)
	defer cancel()
	params := runparams.NewRunServiceGetRunParams().WithContext(ctx).WithRunID(runID)
	resp, err := l.api.run.RunService.RunServiceGetRun(params, l.api.auth)
	if err != nil {
		return nil, err
	}
	return resp.Payload, nil
}

func (l *TriggerPipelineLauncher) createRun(run *runmodel.V2beta1Run) (*runmodel.V2beta1Run, error) {
	ctx, cancel := context.WithTimeout(context.Background(), api_server.APIServerDefaultTimeout)
	defer cancel()
	params := runparams.NewRunServiceCreateRunParams().WithContext(ctx).WithRun(run)
	resp, err := l.api.run.RunService.RunServiceCreateRun(params, l.api.auth)
	if err != nil {
		return nil, err
	}
	return resp.Payload, nil
}

func (l *TriggerPipelineLauncher) publish(
	ctx context.Context,
	execution *metadata.Execution,
	childRunID, state string,
	mlmdState pb.Execution_State,
) error {
	if execution.GetExecution().CustomProperties == nil {
		execution.GetExecution().CustomProperties = map[string]*pb.Value{}
	}
	execution.GetExecution().CustomProperties[MLMDChildRunIDKey] = metadata.StringValue(childRunID)
	outputs := map[string]*structpb.Value{
		triggerOutputRunID: structpb.NewStringValue(childRunID),
		triggerOutputState: structpb.NewStringValue(state),
	}
	if err := l.metadataClient.PublishExecution(ctx, execution, outputs, nil, mlmdState); err != nil {
		return fmt.Errorf("failed to publish trigger pipeline execution: %w", err)
	}
	return nil
}

func (l *TriggerPipelineLauncher) waitForTerminal(ctx context.Context, runID string, poke time.Duration) (*runmodel.V2beta1Run, error) {
	for {
		run, err := l.getRun(runID)
		if err != nil {
			return nil, fmt.Errorf("failed to get child run %s while waiting: %w", runID, err)
		}
		if run.State != nil && isTerminalRuntimeState(*run.State) {
			return run, nil
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(poke):
		}
	}
}

func isTerminalRuntimeState(state runmodel.V2beta1RuntimeState) bool {
	switch state {
	case runmodel.V2beta1RuntimeStateSUCCEEDED,
		runmodel.V2beta1RuntimeStateFAILED,
		runmodel.V2beta1RuntimeStateSKIPPED,
		runmodel.V2beta1RuntimeStateCANCELED:
		return true
	default:
		return false
	}
}

func (l *TriggerPipelineLauncher) resolveInputParameters(ctx context.Context, pipeline *metadata.Pipeline) (map[string]*structpb.Value, error) {
	out := make(map[string]*structpb.Value)
	inputs := l.task.GetInputs()
	if inputs == nil {
		return out, nil
	}
	dag, err := l.metadataClient.GetDAG(ctx, l.triggerOpts.ParentDagID)
	if err != nil {
		return nil, fmt.Errorf("failed to get parent DAG for input resolution: %w", err)
	}
	dagInputs, _, err := dag.Execution.GetParameters()
	if err != nil {
		return nil, err
	}
	executions, err := l.metadataClient.GetExecutionsInDAG(ctx, dag, pipeline, true)
	if err != nil {
		return nil, fmt.Errorf("failed to list DAG executions for input resolution: %w", err)
	}

	for name, paramSpec := range inputs.GetParameters() {
		v, err := resolveTriggerInputParameter(paramSpec, dagInputs, executions, l.triggerOpts.ParentDagID)
		if err != nil {
			return nil, fmt.Errorf("resolving input parameter %q: %w", name, err)
		}
		out[name] = v
	}
	return out, nil
}

func resolveTriggerInputParameter(
	paramSpec *pipelinespec.TaskInputsSpec_InputParameterSpec,
	dagInputs map[string]*structpb.Value,
	executions map[string]*metadata.Execution,
	parentDagID int64,
) (*structpb.Value, error) {
	switch paramSpec.Kind.(type) {
	case *pipelinespec.TaskInputsSpec_InputParameterSpec_ComponentInputParameter:
		key := paramSpec.GetComponentInputParameter()
		v, ok := dagInputs[key]
		if !ok {
			return nil, fmt.Errorf("parent DAG does not have input parameter %s", key)
		}
		return v, nil
	case *pipelinespec.TaskInputsSpec_InputParameterSpec_RuntimeValue:
		rv := paramSpec.GetRuntimeValue()
		if rv.GetConstant() == nil {
			return nil, fmt.Errorf("runtime value without constant is not supported")
		}
		return rv.GetConstant(), nil
	case *pipelinespec.TaskInputsSpec_InputParameterSpec_TaskOutputParameter:
		producer := paramSpec.GetTaskOutputParameter().GetProducerTask()
		outputKey := paramSpec.GetTaskOutputParameter().GetOutputParameterKey()
		lookup := metadata.GetTaskNameWithDagID(producer, parentDagID)
		exec, ok := executions[lookup]
		if !ok {
			exec, ok = executions[producer]
			if !ok {
				return nil, fmt.Errorf("producer task %q not found in DAG", producer)
			}
		}
		_, outputs, err := exec.GetParameters()
		if err != nil {
			return nil, err
		}
		v, ok := outputs[outputKey]
		if !ok {
			return nil, fmt.Errorf("producer task %q has no output parameter %q", producer, outputKey)
		}
		return v, nil
	default:
		return nil, fmt.Errorf("unsupported input parameter kind %T", paramSpec.Kind)
	}
}

func structpbValueToInterface(v *structpb.Value) interface{} {
	if v == nil {
		return nil
	}
	switch k := v.Kind.(type) {
	case *structpb.Value_StringValue:
		return k.StringValue
	case *structpb.Value_NumberValue:
		return k.NumberValue
	case *structpb.Value_BoolValue:
		return k.BoolValue
	case *structpb.Value_NullValue:
		return nil
	case *structpb.Value_ListValue:
		out := make([]interface{}, 0, len(k.ListValue.GetValues()))
		for _, item := range k.ListValue.GetValues() {
			out = append(out, structpbValueToInterface(item))
		}
		return out
	case *structpb.Value_StructValue:
		out := make(map[string]interface{}, len(k.StructValue.GetFields()))
		for key, val := range k.StructValue.GetFields() {
			out[key] = structpbValueToInterface(val)
		}
		return out
	default:
		return v.AsInterface()
	}
}
