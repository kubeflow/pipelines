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

package api_server_v2

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	params "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_upload_client/pipeline_upload_service"
	k8sapi "github.com/kubeflow/pipelines/backend/src/crd/kubernetes/v2beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const testPipelineSpecYAML = `# PIPELINE DEFINITION
# Name: echo
components:
  comp-echo:
    executorLabel: exec-echo
deploymentSpec:
  executors:
    exec-echo:
      container:
        args:
        - hello world
        command:
        - echo
        image: registry.access.redhat.com/ubi9/python-311:latest
pipelineInfo:
  name: echo
root:
  dag:
    tasks:
      echo:
        cachingOptions:
          enableCache: true
        componentRef:
          name: comp-echo
        taskInfo:
          name: echo
schemaVersion: 2.1.0
sdkVersion: kfp-2.14.6
`

func TestUploadPipelineVersionKubernetesClient_PersistsTags(t *testing.T) {
	pipelineID := "pipeline-id"
	namespace := "kubeflow"
	pipeline := &k8sapi.Pipeline{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pipeline",
			Namespace: namespace,
			UID:       types.UID(pipelineID),
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(pipeline).
		Build()
	client := &PipelineUploadClientKubernetes{ctrlClient: fakeClient, namespace: namespace}

	specPath := writePipelineSpecFile(t)
	expectedTags := map[string]string{"project": "kfp", "owner": "test"}
	rawTags := `{"project":"kfp","owner":"test"}`
	versionName := "tagged-version"
	params := params.NewUploadPipelineVersionParams()
	params.Pipelineid = &pipelineID
	params.SetName(&versionName)
	params.SetTags(&rawTags)

	response, err := client.UploadPipelineVersion(specPath, params)
	if err != nil {
		t.Fatalf("UploadPipelineVersion() error = %v", err)
	}
	if !reflect.DeepEqual(response.Tags, expectedTags) {
		t.Fatalf("UploadPipelineVersion() tags = %#v, want %#v", response.Tags, expectedTags)
	}

	var versions k8sapi.PipelineVersionList
	if err := fakeClient.List(context.Background(), &versions, &ctrlclient.ListOptions{Namespace: namespace}); err != nil {
		t.Fatalf("failed to list pipeline versions: %v", err)
	}
	if len(versions.Items) != 1 {
		t.Fatalf("expected 1 pipeline version, got %d", len(versions.Items))
	}
	if !reflect.DeepEqual(versions.Items[0].Spec.Tags, expectedTags) {
		t.Fatalf("stored pipeline version tags = %#v, want %#v", versions.Items[0].Spec.Tags, expectedTags)
	}
}

func TestUploadPipelineVersionKubernetesClient_InvalidTagsReturnsError(t *testing.T) {
	pipelineID := "pipeline-id"
	namespace := "kubeflow"
	pipeline := &k8sapi.Pipeline{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pipeline",
			Namespace: namespace,
			UID:       types.UID(pipelineID),
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(pipeline).
		Build()
	client := &PipelineUploadClientKubernetes{ctrlClient: fakeClient, namespace: namespace}

	specPath := writePipelineSpecFile(t)
	rawTags := "not-valid-json"
	versionName := "tagged-version"
	params := params.NewUploadPipelineVersionParams()
	params.Pipelineid = &pipelineID
	params.SetName(&versionName)
	params.SetTags(&rawTags)

	_, err := client.UploadPipelineVersion(specPath, params)
	if err == nil {
		t.Fatalf("expected UploadPipelineVersion() to fail for invalid tags JSON")
	}

	var versions k8sapi.PipelineVersionList
	if err := fakeClient.List(context.Background(), &versions, &ctrlclient.ListOptions{Namespace: namespace}); err != nil {
		t.Fatalf("failed to list pipeline versions: %v", err)
	}
	if len(versions.Items) != 0 {
		t.Fatalf("expected no pipeline versions on invalid tags, got %d", len(versions.Items))
	}
}

func TestUploadPipelineVersionKubernetesClient_InvalidTagValueLengthReturnsError(t *testing.T) {
	pipelineID := "pipeline-id"
	namespace := "kubeflow"
	pipeline := &k8sapi.Pipeline{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pipeline",
			Namespace: namespace,
			UID:       types.UID(pipelineID),
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(pipeline).
		Build()
	client := &PipelineUploadClientKubernetes{ctrlClient: fakeClient, namespace: namespace}

	specPath := writePipelineSpecFile(t)
	rawTags := `{"team":"this-value-is-way-too-long-for-tags-and-exceeds-the-sixty-three-char-limit"}`
	versionName := "tagged-version"
	params := params.NewUploadPipelineVersionParams()
	params.Pipelineid = &pipelineID
	params.SetName(&versionName)
	params.SetTags(&rawTags)

	_, err := client.UploadPipelineVersion(specPath, params)
	if err == nil {
		t.Fatalf("expected UploadPipelineVersion() to fail for tag value longer than 63 characters")
	}

	var versions k8sapi.PipelineVersionList
	if err := fakeClient.List(context.Background(), &versions, &ctrlclient.ListOptions{Namespace: namespace}); err != nil {
		t.Fatalf("failed to list pipeline versions: %v", err)
	}
	if len(versions.Items) != 0 {
		t.Fatalf("expected no pipeline versions on invalid tag value length, got %d", len(versions.Items))
	}
}

func writePipelineSpecFile(t *testing.T) string {
	t.Helper()

	tmpDir := t.TempDir()
	specPath := filepath.Join(tmpDir, "pipeline.yaml")
	if err := os.WriteFile(specPath, []byte(testPipelineSpecYAML), 0600); err != nil {
		t.Fatalf("failed to write temporary pipeline spec file: %v", err)
	}
	return specPath
}
