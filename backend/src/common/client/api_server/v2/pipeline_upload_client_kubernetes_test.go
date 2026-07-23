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

package api_server_v2 //nolint:staticcheck // ST1003: package name matches existing convention in this directory

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/go-openapi/runtime"
	params "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_upload_client/pipeline_upload_service"
	k8sapi "github.com/kubeflow/pipelines/backend/src/crd/kubernetes/v2beta1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const testNamespace = "test-ns"

func basicPipelineSpecYAML() string {
	return `pipelineInfo:
  name: test-pipeline
  displayName: Test Pipeline
root:
  dag:
    tasks: {}
schemaVersion: "2.1.0"
sdkVersion: kfp-2.13.0`
}

func newFakeClient(objects ...ctrlclient.Object) ctrlclient.Client {
	testScheme := k8sruntime.NewScheme()
	if err := k8sapi.AddToScheme(testScheme); err != nil {
		panic(err)
	}

	builder := fake.NewClientBuilder().WithScheme(testScheme)
	if len(objects) > 0 {
		builder = builder.WithObjects(objects...)
	}

	return builder.Build()
}

func newUploadClient(client ctrlclient.Client) *PipelineUploadClientKubernetes {
	return &PipelineUploadClientKubernetes{
		ctrlClient: client,
		namespace:  testNamespace,
	}
}

func pipelineSpecReader(name string) runtime.NamedReadCloser {
	return runtime.NamedReader(name, io.NopCloser(bytes.NewReader([]byte(basicPipelineSpecYAML()))))
}

func stringPtr(s string) *string {
	return &s
}

// newExistingPipeline returns a Pipeline CRD pre-populated for UploadPipelineVersion tests.
func newExistingPipeline() *k8sapi.Pipeline {
	return &k8sapi.Pipeline{
		ObjectMeta: metav1.ObjectMeta{
			UID:       "pipeline-uid-123",
			Name:      "existing-pipeline",
			Namespace: testNamespace,
		},
		Spec: k8sapi.PipelineSpec{
			DisplayName: "Existing Pipeline",
			Description: "An existing pipeline for testing",
		},
	}
}

// writeSpecFile writes the basic pipeline spec YAML to a temp file and returns its path.
func writeSpecFile(t *testing.T) string {
	t.Helper()
	specPath := t.TempDir() + "/pipeline.yaml"
	require.NoError(t, os.WriteFile(specPath, []byte(basicPipelineSpecYAML()), 0644))
	return specPath
}

// --- Upload (pipeline + initial version) tests ---

func TestUpload_HappyPath(t *testing.T) {
	fakeClient := newFakeClient()
	uploadClient := newUploadClient(fakeClient)

	uploadParams := params.NewUploadPipelineParams()
	uploadParams.Name = stringPtr("my-pipeline")
	uploadParams.DisplayName = stringPtr("My Pipeline")
	uploadParams.Description = stringPtr("A test pipeline")
	uploadParams.Uploadfile = pipelineSpecReader("my-pipeline.yaml")

	result, err := uploadClient.Upload(uploadParams)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, "my-pipeline", result.Name)
	assert.Equal(t, "My Pipeline", result.DisplayName)
	assert.Equal(t, "A test pipeline", result.Description)
	assert.Equal(t, testNamespace, result.Namespace)

	// Verify both CRDs were created.
	pipelineList := &k8sapi.PipelineList{}
	require.NoError(t, fakeClient.List(t.Context(), pipelineList, &ctrlclient.ListOptions{Namespace: testNamespace}))
	require.Len(t, pipelineList.Items, 1)
	assert.Equal(t, "My Pipeline", pipelineList.Items[0].Spec.DisplayName)

	versionList := &k8sapi.PipelineVersionList{}
	require.NoError(t, fakeClient.List(t.Context(), versionList, &ctrlclient.ListOptions{Namespace: testNamespace}))
	require.Len(t, versionList.Items, 1)
	assert.Equal(t, "My Pipeline", versionList.Items[0].Spec.DisplayName)
	assert.Equal(t, "A test pipeline", versionList.Items[0].Spec.Description)
}

func TestUpload_NameDefaultsToFileName(t *testing.T) {
	fakeClient := newFakeClient()
	uploadClient := newUploadClient(fakeClient)

	uploadParams := params.NewUploadPipelineParams()
	// Name, DisplayName, and Description are all nil — name should default to the file name.
	uploadParams.Uploadfile = pipelineSpecReader("hello-world.yaml")

	result, err := uploadClient.Upload(uploadParams)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, "hello-world.yaml", result.Name)
	assert.Equal(t, "hello-world.yaml", result.DisplayName)
}

func TestUpload_NamespaceMismatchReturnsError(t *testing.T) {
	fakeClient := newFakeClient()
	uploadClient := newUploadClient(fakeClient)

	uploadParams := params.NewUploadPipelineParams()
	uploadParams.Name = stringPtr("my-pipeline")
	uploadParams.Namespace = stringPtr("wrong-namespace")
	uploadParams.Uploadfile = pipelineSpecReader("my-pipeline.yaml")

	result, err := uploadClient.Upload(uploadParams)
	assert.Nil(t, result)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Namespace cannot be set")
}

func TestUpload_InvalidTagsReturnsError(t *testing.T) {
	fakeClient := newFakeClient()
	uploadClient := newUploadClient(fakeClient)

	uploadParams := params.NewUploadPipelineParams()
	uploadParams.Name = stringPtr("my-pipeline")
	uploadParams.Tags = stringPtr("not valid json")
	uploadParams.Uploadfile = pipelineSpecReader("my-pipeline.yaml")

	result, err := uploadClient.Upload(uploadParams)
	assert.Nil(t, result)
	require.Error(t, err)
}

func TestUpload_TagsPersisted(t *testing.T) {
	fakeClient := newFakeClient()
	uploadClient := newUploadClient(fakeClient)

	uploadParams := params.NewUploadPipelineParams()
	uploadParams.Name = stringPtr("tagged-pipeline")
	uploadParams.Tags = stringPtr(`{"env": "staging", "team": "ml"}`)
	uploadParams.Uploadfile = pipelineSpecReader("tagged-pipeline.yaml")

	result, err := uploadClient.Upload(uploadParams)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, map[string]string{"env": "staging", "team": "ml"}, result.Tags)
}

func TestUpload_CodeSourceURL(t *testing.T) {
	tests := []struct {
		name          string
		codeSourceURL *string
		wantStoredURL string
	}{
		{
			name:          "CodeSourceURL is set",
			codeSourceURL: stringPtr("https://github.com/example/repo"),
			wantStoredURL: "https://github.com/example/repo",
		},
		{
			name:          "CodeSourceURL is nil",
			codeSourceURL: nil,
			wantStoredURL: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClient := newFakeClient()
			uploadClient := newUploadClient(fakeClient)

			uploadParams := params.NewUploadPipelineParams()
			uploadParams.Name = stringPtr("test-pipeline")
			uploadParams.Uploadfile = pipelineSpecReader("test-pipeline.yaml")
			uploadParams.CodeSourceURL = tt.codeSourceURL

			result, err := uploadClient.Upload(uploadParams)
			require.NoError(t, err)
			require.NotNil(t, result)

			versionList := &k8sapi.PipelineVersionList{}
			err = fakeClient.List(t.Context(), versionList, &ctrlclient.ListOptions{
				Namespace: testNamespace,
			})
			require.NoError(t, err)
			require.Len(t, versionList.Items, 1)

			assert.Equal(t, tt.wantStoredURL, versionList.Items[0].Spec.CodeSourceURL)
		})
	}
}

// --- UploadPipelineVersion tests ---

func TestUploadPipelineVersion_HappyPath(t *testing.T) {
	fakeClient := newFakeClient(newExistingPipeline())
	uploadClient := newUploadClient(fakeClient)
	specPath := writeSpecFile(t)

	versionParams := params.NewUploadPipelineVersionParams()
	versionParams.Pipelineid = stringPtr("pipeline-uid-123")
	versionParams.Name = stringPtr("version-one")
	versionParams.DisplayName = stringPtr("Version One")
	versionParams.Description = stringPtr("First version")

	result, err := uploadClient.UploadPipelineVersion(specPath, versionParams)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, "version-one", result.Name)
	assert.Equal(t, "Version One", result.DisplayName)
	assert.Equal(t, "First version", result.Description)
	assert.Equal(t, "pipeline-uid-123", result.PipelineID)
	assert.NotNil(t, result.PipelineSpec, "PipelineSpec should be populated in the response")

	// Verify the CRD was created with an owner reference to the parent pipeline.
	versionList := &k8sapi.PipelineVersionList{}
	require.NoError(t, fakeClient.List(t.Context(), versionList, &ctrlclient.ListOptions{Namespace: testNamespace}))
	require.Len(t, versionList.Items, 1)

	ownerRefs := versionList.Items[0].OwnerReferences
	require.Len(t, ownerRefs, 1, "PipelineVersion should have an owner reference to its Pipeline")
	assert.Equal(t, "existing-pipeline", ownerRefs[0].Name)
}

func TestUploadPipelineVersion_MissingPipelineIDReturnsError(t *testing.T) {
	fakeClient := newFakeClient()
	uploadClient := newUploadClient(fakeClient)
	specPath := writeSpecFile(t)

	versionParams := params.NewUploadPipelineVersionParams()
	// Pipelineid is nil.
	versionParams.Name = stringPtr("version-one")

	result, err := uploadClient.UploadPipelineVersion(specPath, versionParams)
	assert.Nil(t, result)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "pipelineid is required")
}

func TestUploadPipelineVersion_PipelineNotFoundReturnsError(t *testing.T) {
	fakeClient := newFakeClient() // No pre-existing pipeline.
	uploadClient := newUploadClient(fakeClient)
	specPath := writeSpecFile(t)

	versionParams := params.NewUploadPipelineVersionParams()
	versionParams.Pipelineid = stringPtr("nonexistent-uid")
	versionParams.Name = stringPtr("version-one")

	result, err := uploadClient.UploadPipelineVersion(specPath, versionParams)
	assert.Nil(t, result)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "pipeline not found")
}

func TestUploadPipelineVersion_InvalidTagsReturnsError(t *testing.T) {
	fakeClient := newFakeClient(newExistingPipeline())
	uploadClient := newUploadClient(fakeClient)
	specPath := writeSpecFile(t)

	versionParams := params.NewUploadPipelineVersionParams()
	versionParams.Pipelineid = stringPtr("pipeline-uid-123")
	versionParams.Name = stringPtr("version-one")
	versionParams.Tags = stringPtr("{bad json")

	result, err := uploadClient.UploadPipelineVersion(specPath, versionParams)
	assert.Nil(t, result)
	require.Error(t, err)
}

func TestUploadPipelineVersion_TagsPersisted(t *testing.T) {
	fakeClient := newFakeClient(newExistingPipeline())
	uploadClient := newUploadClient(fakeClient)
	specPath := writeSpecFile(t)

	versionParams := params.NewUploadPipelineVersionParams()
	versionParams.Pipelineid = stringPtr("pipeline-uid-123")
	versionParams.Name = stringPtr("tagged-version")
	versionParams.Tags = stringPtr(`{"release": "canary"}`)

	result, err := uploadClient.UploadPipelineVersion(specPath, versionParams)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, map[string]string{"release": "canary"}, result.Tags)
}

func TestUploadPipelineVersion_FileNotFoundReturnsError(t *testing.T) {
	fakeClient := newFakeClient(newExistingPipeline())
	uploadClient := newUploadClient(fakeClient)

	versionParams := params.NewUploadPipelineVersionParams()
	versionParams.Pipelineid = stringPtr("pipeline-uid-123")
	versionParams.Name = stringPtr("version-one")

	result, err := uploadClient.UploadPipelineVersion("/nonexistent/path.yaml", versionParams)
	assert.Nil(t, result)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Failed to open file")
}

func TestUploadPipelineVersion_CodeSourceURL(t *testing.T) {
	tests := []struct {
		name            string
		codeSourceURL   *string
		wantStoredURL   string
		wantReturnedURL string
	}{
		{
			name:            "CodeSourceURL is set",
			codeSourceURL:   stringPtr("https://github.com/example/repo/v2"),
			wantStoredURL:   "https://github.com/example/repo/v2",
			wantReturnedURL: "https://github.com/example/repo/v2",
		},
		{
			name:            "CodeSourceURL is nil",
			codeSourceURL:   nil,
			wantStoredURL:   "",
			wantReturnedURL: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClient := newFakeClient(newExistingPipeline())
			uploadClient := newUploadClient(fakeClient)
			specPath := writeSpecFile(t)

			versionParams := params.NewUploadPipelineVersionParams()
			versionParams.Pipelineid = stringPtr("pipeline-uid-123")
			versionParams.Name = stringPtr("version-one")
			versionParams.CodeSourceURL = tt.codeSourceURL

			result, err := uploadClient.UploadPipelineVersion(specPath, versionParams)
			require.NoError(t, err)
			require.NotNil(t, result)

			assert.Equal(t, tt.wantReturnedURL, result.CodeSourceURL,
				"CodeSourceURL in the returned pipeline version model should match")

			versionList := &k8sapi.PipelineVersionList{}
			err = fakeClient.List(t.Context(), versionList, &ctrlclient.ListOptions{
				Namespace: testNamespace,
			})
			require.NoError(t, err)
			require.Len(t, versionList.Items, 1)

			assert.Equal(t, tt.wantStoredURL, versionList.Items[0].Spec.CodeSourceURL,
				"CodeSourceURL on the created PipelineVersion CRD should match the upload parameter")
		})
	}
}

// --- UploadFile tests ---

func TestUploadFile_HappyPath(t *testing.T) {
	fakeClient := newFakeClient()
	uploadClient := newUploadClient(fakeClient)
	specPath := writeSpecFile(t)

	uploadParams := params.NewUploadPipelineParams()
	uploadParams.Name = stringPtr("file-upload-pipeline")

	result, err := uploadClient.UploadFile(specPath, uploadParams)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, "file-upload-pipeline", result.Name)
	assert.Equal(t, testNamespace, result.Namespace)
}

func TestUploadFile_FileNotFoundReturnsError(t *testing.T) {
	fakeClient := newFakeClient()
	uploadClient := newUploadClient(fakeClient)

	uploadParams := params.NewUploadPipelineParams()
	uploadParams.Name = stringPtr("my-pipeline")

	result, err := uploadClient.UploadFile("/nonexistent/path.yaml", uploadParams)
	assert.Nil(t, result)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Failed to open file")
}

// --- Helper function tests ---

func TestDeriveNameDisplayAndDescription(t *testing.T) {
	tests := []struct {
		testName            string
		providedName        *string
		providedDisplayName *string
		providedDescription *string
		defaultName         string
		wantName            string
		wantDisplayName     string
		wantDescription     string
	}{
		{
			testName:        "all nil uses default",
			defaultName:     "default.yaml",
			wantName:        "default.yaml",
			wantDisplayName: "default.yaml",
			wantDescription: "",
		},
		{
			testName:        "name provided",
			providedName:    stringPtr("custom-name"),
			defaultName:     "default.yaml",
			wantName:        "custom-name",
			wantDisplayName: "custom-name",
			wantDescription: "",
		},
		{
			testName:            "display name provided without name",
			providedDisplayName: stringPtr("Custom Display"),
			defaultName:         "default.yaml",
			wantName:            "Custom Display",
			wantDisplayName:     "Custom Display",
			wantDescription:     "",
		},
		{
			testName:            "all fields provided",
			providedName:        stringPtr("my-name"),
			providedDisplayName: stringPtr("My Display Name"),
			providedDescription: stringPtr("My description"),
			defaultName:         "default.yaml",
			wantName:            "my-name",
			wantDisplayName:     "My Display Name",
			wantDescription:     "My description",
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			name, displayName, description := deriveNameDisplayAndDescription(
				tt.providedName, tt.providedDisplayName, tt.providedDescription, tt.defaultName,
			)
			assert.Equal(t, tt.wantName, name)
			assert.Equal(t, tt.wantDisplayName, displayName)
			assert.Equal(t, tt.wantDescription, description)
		})
	}
}
