// Copyright 2025 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package proto_tests

import (
	"os"
	"path/filepath"
	"time"

	"github.com/golang/glog"
	specPB "github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	pb "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/server"
	"google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const mockPipelineID = "9b187b86-7c0a-42ae-a0bc-2a746b6eb7a3"
const mockPipelineVersionID = "e15dc3ec-b45e-4cc7-bb07-e76b5dbce99a"
const pipelineSpecYamlPath = "pipelinespec.yaml"

func mockPipelineSpec() *structpb.Struct {
	yamlContent, err := os.ReadFile(filepath.Join("testdata", pipelineSpecYamlPath))
	if err != nil {
		glog.Fatal(err)
	}
	spec, err := server.YamlStringToPipelineSpecStruct(string(yamlContent))
	if err != nil {
		glog.Fatal(err)
	}
	return spec
}

func fixedTimestamp() *timestamppb.Timestamp {
	return timestamppb.New(time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC))
}

var completedRun = &pb.Run{
	RunId:          "completed-run-123",
	DisplayName:    "Production Pipeline Run",
	ExperimentId:   "exp-456",
	StorageState:   pb.Run_AVAILABLE,
	Description:    "Production pipeline execution for data processing",
	ServiceAccount: "sa1",
	CreatedAt:      fixedTimestamp(),
	ScheduledAt:    fixedTimestamp(),
	FinishedAt:     fixedTimestamp(),
	RecurringRunId: "recurring-schedule-001",
	State:          pb.RuntimeState_SUCCEEDED,
	PipelineSource: &pb.Run_PipelineVersionReference{
		PipelineVersionReference: &pb.PipelineVersionReference{
			PipelineId:        mockPipelineID,
			PipelineVersionId: mockPipelineVersionID,
		},
	},
	RuntimeConfig: &pb.RuntimeConfig{
		Parameters: map[string]*structpb.Value{
			"batch_size":    structpb.NewNumberValue(1000),
			"learning_rate": structpb.NewStringValue("foo"),
		},
	},
}

var completedRunWithPipelineSpec = &pb.Run{
	RunId:          "completed-run-123",
	DisplayName:    "Production Pipeline Run",
	ExperimentId:   "exp-456",
	StorageState:   pb.Run_AVAILABLE,
	Description:    "Production pipeline execution for data processing",
	ServiceAccount: "sa1",
	CreatedAt:      fixedTimestamp(),
	ScheduledAt:    fixedTimestamp(),
	FinishedAt:     fixedTimestamp(),
	RecurringRunId: "recurring-schedule-001",
	State:          pb.RuntimeState_SUCCEEDED,
	PipelineSource: &pb.Run_PipelineSpec{
		PipelineSpec: mockPipelineSpec(),
	},
	RuntimeConfig: &pb.RuntimeConfig{
		Parameters: map[string]*structpb.Value{
			"batch_size":    structpb.NewNumberValue(1000),
			"learning_rate": structpb.NewStringValue("foo"),
		},
	},
}

var failedRun = &pb.Run{
	RunId:          "failed-run-456",
	DisplayName:    "Data Processing Pipeline",
	ExperimentId:   "exp-789",
	StorageState:   pb.Run_AVAILABLE,
	Description:    "Failed attempt to process customer data",
	ServiceAccount: "sa2",
	CreatedAt:      fixedTimestamp(),
	ScheduledAt:    fixedTimestamp(),
	FinishedAt:     fixedTimestamp(),
	State:          pb.RuntimeState_FAILED,
	Error: &status.Status{
		Code:    1,
		Message: "This was a Failed Run.",
	},
}

var pipeline = &pb.Pipeline{
	PipelineId:  mockPipelineID,
	DisplayName: "Production Data Processing Pipeline",
	Name:        "pipeline1",
	Description: "Pipeline for processing and analyzing production data",
	CreatedAt:   fixedTimestamp(),
	Namespace:   "namespace1",
	Error: &status.Status{
		Code:    0,
		Message: "This a successful pipeline.",
	},
}

var pipelineVersion = &pb.PipelineVersion{
	PipelineVersionId: mockPipelineVersionID,
	PipelineId:        mockPipelineID,
	DisplayName:       "v1.0.0 Production Data Processing Pipeline",
	Name:              "pipelineversion1",
	Description:       "First stable version of the production data processing pipeline",
	CreatedAt:         fixedTimestamp(),
	PipelineSpec:      mockPipelineSpec(),
	PackageUrl:        &pb.Url{PipelineUrl: "gs://my-bucket/pipelines/pipeline1-v1.0.0.yaml"},
	CodeSourceUrl:     "https://github.com/org/repo/pipeline1/tree/v1.0.0",
	Error: &status.Status{
		Code:    0,
		Message: "This is a successful pipeline version.",
	},
}

var experiment = &pb.Experiment{
	ExperimentId:     "exp-456",
	DisplayName:      "Production Data Processing Experiment",
	Description:      "Experiment for testing production data processing pipeline",
	CreatedAt:        fixedTimestamp(),
	Namespace:        "namespace1",
	StorageState:     pb.Experiment_AVAILABLE,
	LastRunCreatedAt: fixedTimestamp(),
}

var visualization = &pb.Visualization{
	Type:      pb.Visualization_ROC_CURVE,
	Source:    "gs://my-bucket/data/visualization.csv",
	Arguments: "{\"param1\": \"value1\", \"param2\": \"value2\"}",
	Html:      "<div>Generated Visualization</div>",
	Error:     "",
}

var recurringRun = &pb.RecurringRun{
	RecurringRunId: "recurring-run-789",
	DisplayName:    "Daily Data Processing",
	Description:    "Scheduled pipeline for daily data processing tasks",
	ServiceAccount: "sa3",
	CreatedAt:      fixedTimestamp(),
	UpdatedAt:      fixedTimestamp(),
	Status:         pb.RecurringRun_ENABLED,
	PipelineSource: &pb.RecurringRun_PipelineVersionReference{
		PipelineVersionReference: &pb.PipelineVersionReference{
			PipelineId:        mockPipelineID,
			PipelineVersionId: mockPipelineVersionID,
		},
	},
	RuntimeConfig: &pb.RuntimeConfig{
		Parameters: map[string]*structpb.Value{
			"processing_date": structpb.NewStringValue("${system.date}"),
			"batch_size":      structpb.NewNumberValue(500),
		},
	},
	Trigger: &pb.Trigger{
		Trigger: &pb.Trigger_PeriodicSchedule{
			PeriodicSchedule: &pb.PeriodicSchedule{
				StartTime:      fixedTimestamp(),
				EndTime:        nil,
				IntervalSecond: 86400,
			},
		},
	},
	Mode:      1,
	Namespace: "namespace1",
}

var pipelineSpec = &specPB.PipelineSpec{
	PipelineInfo: &specPB.PipelineInfo{
		Name:        "sample-pipeline",
		Description: "Sample pipeline for testing",
	},
	DeploymentSpec: &structpb.Struct{},
	Root: &specPB.ComponentSpec{
		InputDefinitions: &specPB.ComponentInputsSpec{
			Parameters: map[string]*specPB.ComponentInputsSpec_ParameterSpec{
				"input1": {
					ParameterType: specPB.ParameterType_STRING,
					DefaultValue:  structpb.NewStringValue("foo"),
				},
				"input2": {
					ParameterType: specPB.ParameterType_STRING,
				},
			},
		},
		OutputDefinitions: &specPB.ComponentOutputsSpec{
			Parameters: map[string]*specPB.ComponentOutputsSpec_ParameterSpec{
				"output1": {
					ParameterType: specPB.ParameterType_STRING,
				},
				"output2": {
					ParameterType: specPB.ParameterType_NUMBER_INTEGER,
				},
			},
		},
		Implementation: &specPB.ComponentSpec_ExecutorLabel{
			ExecutorLabel: "root-executor",
		},
	},
	Components: map[string]*specPB.ComponentSpec{
		"comp-1": {
			Implementation: &specPB.ComponentSpec_Dag{
				Dag: &specPB.DagSpec{
					Tasks: map[string]*specPB.PipelineTaskSpec{
						"task1": {
							TaskInfo: &specPB.PipelineTaskInfo{
								Name: "task1",
							},
							ComponentRef: &specPB.ComponentRef{
								Name: "comp-1",
							},
							Inputs: &specPB.TaskInputsSpec{
								Parameters: map[string]*specPB.TaskInputsSpec_InputParameterSpec{
									"param1": {
										Kind: &specPB.TaskInputsSpec_InputParameterSpec_ComponentInputParameter{
											ComponentInputParameter: "param1",
										},
									},
								},
							},
						},
					},
					Outputs: &specPB.DagOutputsSpec{
						Parameters: map[string]*specPB.DagOutputsSpec_DagOutputParameterSpec{
							"output1": {
								Kind: &specPB.DagOutputsSpec_DagOutputParameterSpec_ValueFromParameter{
									ValueFromParameter: &specPB.DagOutputsSpec_ParameterSelectorSpec{
										ProducerSubtask:    "foo",
										OutputParameterKey: "bar",
									},
								},
							},
						},
					},
				},
			},
			InputDefinitions: &specPB.ComponentInputsSpec{
				Parameters: map[string]*specPB.ComponentInputsSpec_ParameterSpec{
					"param1": {
						ParameterType: specPB.ParameterType_STRING,
					},
				},
			},
			OutputDefinitions: &specPB.ComponentOutputsSpec{
				Parameters: map[string]*specPB.ComponentOutputsSpec_ParameterSpec{
					"output1": {
						ParameterType: specPB.ParameterType_STRING,
					},
				},
			},
		},
	},
}

var platformSpec = &specPB.PlatformSpec{
	Platforms: map[string]*specPB.SinglePlatformSpec{
		"kubernetes": {
			Platform: "kubernetes",
			DeploymentSpec: &specPB.PlatformDeploymentConfig{
				Executors: map[string]*structpb.Struct{
					"root-executor": {
						Fields: map[string]*structpb.Value{
							"container": structpb.NewStructValue(&structpb.Struct{
								Fields: map[string]*structpb.Value{
									"image": structpb.NewStringValue("test-image"),
								},
							}),
						},
					},
				},
			},
			Config: &structpb.Struct{
				Fields: map[string]*structpb.Value{
					"project": structpb.NewStringValue("test-project"),
				},
			},
			PipelineConfig: &specPB.PipelineConfig{
				SemaphoreKey: "test-key",
				MutexName:    "test-mutex",
				ResourceTtl:  24,
			},
		},
	},
}
