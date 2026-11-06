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

package driver

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/v2/common/plugins"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

func TestPinnedRuntimeFixture(t *testing.T) {
	if os.Getenv("KFP_RUNTIME_FIXTURE") != "1" {
		t.Skip("local fixture capture only")
	}
	original := buildK8sClient
	defer func() { buildK8sClient = original }()
	buildK8sClient = func() (kubernetes.Interface, error) {
		return fake.NewClientset(&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: "kfp-launcher", Namespace: "migration-fixture"},
			Data:       map[string]string{"defaultPipelineRoot": "s3://fixture/pipelines"},
		}), nil
	}
	client, err := metadata.NewClient("127.0.0.1", "18081", nil)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	root, err := RootDAG(ctx, Options{
		PipelineName: "pinned-runtime-fixture", RunID: "22222222-2222-4222-8222-222222222222",
		Component:     &pipelinespec.ComponentSpec{},
		RuntimeConfig: &pipelinespec.PipelineJob_RuntimeConfig{GcsOutputDirectory: "s3://fixture/pipelines"},
		Namespace:     "migration-fixture", IterationIndex: -1,
	}, client)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("root execution ID: %d", root.ID)
	skipped, err := DAG(ctx, Options{
		PipelineName: "pinned-runtime-fixture", RunID: "22222222-2222-4222-8222-222222222222",
		Component: &pipelinespec.ComponentSpec{Implementation: &pipelinespec.ComponentSpec_Dag{Dag: &pipelinespec.DagSpec{}}},
		Task: &pipelinespec.PipelineTaskSpec{TaskInfo: &pipelinespec.PipelineTaskInfo{Name: "skipped"},
			TriggerPolicy: &pipelinespec.PipelineTaskSpec_TriggerPolicy{Condition: "false"}},
		TaskName: "skipped", DAGExecutionID: root.ID, IterationIndex: -1,
		PluginDispatcher: plugins.NoOpDispatcher{},
	}, client)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("skipped execution ID: %d", skipped.ID)
}
