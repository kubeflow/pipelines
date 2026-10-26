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
	"os"
	"testing"
	"time"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestPinnedRuntimeImporterFixture(t *testing.T) {
	if os.Getenv("KFP_RUNTIME_FIXTURE") != "1" {
		t.Skip("local fixture capture only")
	}
	client, err := metadata.NewClient("127.0.0.1", "18081", nil)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	root, err := client.GetExecutionByTypeAndName(ctx, string(metadata.DagExecutionTypeName), "run/22222222-2222-4222-8222-222222222222")
	if err != nil {
		t.Fatal(err)
	}
	if root == nil {
		t.Fatal("root execution missing")
	}
	schema := &pipelinespec.ArtifactTypeSchema{Kind: &pipelinespec.ArtifactTypeSchema_SchemaTitle{SchemaTitle: "system.Dataset"}}
	launcher := &ImportLauncher{
		component: &pipelinespec.ComponentSpec{OutputDefinitions: &pipelinespec.ComponentOutputsSpec{Artifacts: map[string]*pipelinespec.ComponentOutputsSpec_ArtifactSpec{
			"dataset": {ArtifactType: schema},
		}}},
		importer: &pipelinespec.PipelineDeploymentConfig_ImporterSpec{
			ArtifactUri: &pipelinespec.ValueOrRuntimeParameter{Value: &pipelinespec.ValueOrRuntimeParameter_Constant{Constant: structpb.NewStringValue("s3://fixture/runtime-dataset")}},
			TypeSchema:  schema, Reimport: true,
		},
		task: &pipelinespec.PipelineTaskSpec{TaskInfo: &pipelinespec.PipelineTaskInfo{Name: "importer"}},
		launcherV2Options: LauncherV2Options{
			Namespace: "migration-fixture", PodName: "runtime-importer-pod", PodUID: "runtime-importer-uid",
		},
		importerLauncherOptions: ImporterLauncherOptions{
			PipelineName: "pinned-runtime-fixture", RunID: "22222222-2222-4222-8222-222222222222", ParentDagID: root.GetID(),
		},
		metadataClient: client,
	}
	if err := launcher.Execute(ctx); err != nil {
		t.Fatal(err)
	}
}
