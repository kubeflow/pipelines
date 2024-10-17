// Copyright 2023 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package metadata contains types to record/retrieve metadata stored in MLMD
// for individual pipeline steps.

package metadata

import (
	"context"

	"github.com/kubeflow/pipelines/backend/src/v2/objectstore"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	pb "github.com/kubeflow/pipelines/third_party/ml-metadata/go/ml_metadata"
	"google.golang.org/protobuf/types/known/structpb"
)

type FakeClient struct {
}

func NewFakeClient() *FakeClient {
	return &FakeClient{}
}

func (c *FakeClient) GetPipeline(ctx context.Context, pipelineName, runID, namespace, runResource, pipelineRoot string, storeSessionInfo string) (*Pipeline, error) {
	return nil, nil
}

func (c *FakeClient) GetDAG(ctx context.Context, executionID int64) (*DAG, error) {
	return nil, nil
}

func (c *FakeClient) PublishExecution(ctx context.Context, execution *Execution, outputParameters map[string]*structpb.Value, outputArtifacts []*OutputArtifact, state pb.Execution_State) error {
	return nil
}

func (c *FakeClient) CreateExecution(ctx context.Context, pipeline *Pipeline, config *ExecutionConfig) (*Execution, error) {
	return nil, nil
}
func (c *FakeClient) PrePublishExecution(ctx context.Context, execution *Execution, config *ExecutionConfig) (*Execution, error) {
	return nil, nil
}

func (c *FakeClient) GetExecutions(ctx context.Context, ids []int64) ([]*pb.Execution, error) {
	return nil, nil
}

func (c *FakeClient) GetExecution(ctx context.Context, id int64) (*Execution, error) {
	return nil, nil
}

func (c *FakeClient) GetPipelineFromExecution(ctx context.Context, id int64) (*Pipeline, error) {
	return nil, nil
}

func (c *FakeClient) GetExecutionsInDAG(ctx context.Context, dag *DAG, pipeline *Pipeline, filter bool) (executionsMap map[string]*Execution, err error) {
	return nil, nil
}
func (c *FakeClient) UpdateDAGExecutionsState(ctx context.Context, dag *DAG, pipeline *Pipeline) (err error) {
	return nil
}
func (c *FakeClient) PutDAGExecutionState(ctx context.Context, executionID int64, state pb.Execution_State) (err error) {
	return nil
}
func (c *FakeClient) GetEventsByArtifactIDs(ctx context.Context, artifactIds []int64) ([]*pb.Event, error) {
	return nil, nil
}

func (c *FakeClient) GetArtifactName(ctx context.Context, artifactId int64) (string, error) {
	return "", nil
}
func (c *FakeClient) GetArtifacts(ctx context.Context, ids []int64) ([]*pb.Artifact, error) {
	return nil, nil
}

func (c *FakeClient) GetOutputArtifactsByExecutionId(ctx context.Context, executionId int64) (map[string]*OutputArtifact, error) {
	return nil, nil
}

func (c *FakeClient) RecordArtifact(ctx context.Context, outputName, schema string, runtimeArtifact *pipelinespec.RuntimeArtifact, state pb.Artifact_State, bucketConfig *objectstore.Config) (*OutputArtifact, error) {
	return nil, nil
}

func (c *FakeClient) GetOrInsertArtifactType(ctx context.Context, schema string) (typeID int64, err error) {
	return 0, nil
}

func (c *FakeClient) FindMatchedArtifact(ctx context.Context, artifactToMatch *pb.Artifact, pipelineContextId int64) (matchedArtifact *pb.Artifact, err error) {
	return nil, nil
}
