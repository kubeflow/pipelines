// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package component

import (
	"context"
	"testing"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	api "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/v2/apiclient/kfpapi"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestOutputPropagationDriverRetryGenerationSurvivesRefresh(t *testing.T) {
	outputDefinitions := &pipelinespec.ComponentOutputsSpec{Parameters: map[string]*pipelinespec.ComponentOutputsSpec_ParameterSpec{
		"result": {ParameterType: pipelinespec.ParameterType_STRING},
	}}
	dag := func(child string) *pipelinespec.ComponentSpec {
		return &pipelinespec.ComponentSpec{
			OutputDefinitions: outputDefinitions,
			Implementation: &pipelinespec.ComponentSpec_Dag{Dag: &pipelinespec.DagSpec{
				Tasks: map[string]*pipelinespec.PipelineTaskSpec{
					child: {ComponentRef: &pipelinespec.ComponentRef{Name: child}},
				},
				Outputs: &pipelinespec.DagOutputsSpec{Parameters: map[string]*pipelinespec.DagOutputsSpec_DagOutputParameterSpec{
					"result": {Kind: &pipelinespec.DagOutputsSpec_DagOutputParameterSpec_ValueFromParameter{
						ValueFromParameter: &pipelinespec.DagOutputsSpec_ParameterSelectorSpec{ProducerSubtask: child, OutputParameterKey: "result"},
					}},
				}},
			}},
		}
	}
	pipelineSpec, err := pipelineSpecToStruct(t, &pipelinespec.PipelineSpec{
		Root: dag("middle"),
		Components: map[string]*pipelinespec.ComponentSpec{
			"middle": dag("leaf"),
			"leaf":   {OutputDefinitions: outputDefinitions},
		},
	})
	require.NoError(t, err)
	scope, err := util.ScopePathFromDotNotation(pipelineSpec, "root.middle.leaf")
	require.NoError(t, err)
	metadata := func(generation, checkpoint string) *api.PipelineTask_StatusMetadata {
		return &api.PipelineTask_StatusMetadata{CustomProperties: map[string]*structpb.Value{
			util.DriverRetryGenerationKey: structpb.NewStringValue(generation),
			"_kfp_driver_checkpoint":      structpb.NewStringValue(checkpoint),
		}}
	}
	run := &api.Run{RunId: "run"}
	root := &api.PipelineTask{
		TaskId: "root", RunId: run.RunId, Name: "ROOT", ScopePath: "root",
		Type: api.PipelineTask_ROOT, State: api.PipelineTask_RUNNING,
		StatusMetadata: metadata("9", "root-checkpoint"),
	}
	middle := &api.PipelineTask{
		TaskId: "middle", RunId: run.RunId, Name: "middle", ScopePath: "root.middle", ParentTaskId: util.StringPointer(root.TaskId),
		Type: api.PipelineTask_DAG, State: api.PipelineTask_RUNNING,
		StatusMetadata: metadata("8", "middle-checkpoint"),
	}
	leaf := &api.PipelineTask{
		TaskId: "leaf", RunId: run.RunId, Name: "leaf", ScopePath: "root.middle.leaf", ParentTaskId: util.StringPointer(middle.TaskId),
		Type: api.PipelineTask_RUNTIME, State: api.PipelineTask_SUCCEEDED,
		StatusMetadata: metadata("7", "leaf-checkpoint"),
		Outputs: &api.PipelineTask_InputOutputs{Parameters: []*api.PipelineTask_InputOutputs_IOParameter{{
			ParameterKey: "result", Value: structpb.NewStringValue("output"), Type: api.IOType_OUTPUT,
			Producer: &api.IOProducer{TaskName: "leaf"},
		}}},
	}
	origin := proto.Clone(leaf).(*api.PipelineTask)
	// Even the current task may already belong to a newer generation by the
	// time this old invocation refreshes its outputs.
	leaf.StatusMetadata = metadata("10", "newer-leaf-checkpoint")
	client := kfpapi.NewMockAPI()
	client.AddRun(run)
	for _, task := range []*api.PipelineTask{root, middle, leaf} {
		_, err := client.CreateTask(context.Background(), &api.CreateTaskRequest{RunId: run.RunId, Task: task})
		require.NoError(t, err)
	}
	queued := NewBatchUpdater()
	err = propagateOutputsUpDAG(context.Background(), OutputPropagationOptions{
		Run: run, Task: origin, ParentTask: middle, ScopePath: scope, PipelineSpec: pipelineSpec,
	}, client, queued)
	require.NoError(t, err)
	require.Len(t, queued.taskUpdates, 2)
	for _, parent := range []*api.PipelineTask{root, middle} {
		update := queued.taskUpdates[parent.TaskId]
		require.NotNil(t, update)
		require.Equal(t, "7", update.GetStatusMetadata().GetCustomProperties()[util.DriverRetryGenerationKey].GetStringValue())
		require.Equal(t, parent.GetStatusMetadata().GetCustomProperties()["_kfp_driver_checkpoint"].GetStringValue(),
			update.GetStatusMetadata().GetCustomProperties()["_kfp_driver_checkpoint"].GetStringValue())
		require.Len(t, update.GetOutputs().GetParameters(), 1)
	}
}
