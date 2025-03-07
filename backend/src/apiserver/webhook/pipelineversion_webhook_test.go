/*
Copyright 2025.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package webhook

import (
	"context"
	"encoding/json"
	"testing"

	k8sapi "github.com/kubeflow/pipelines/backend/src/crd/kubernetes/v2beta1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sfake "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func setupPipelineWebhookTest(t *testing.T) (*PipelineVersionsWebhook, string) {
	scheme := runtime.NewScheme()
	require.NoError(t, k8sapi.AddToScheme(scheme))

	fakeClient := k8sfake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(&k8sapi.Pipeline{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pipeline",
				Namespace: "default",
			},
		}).Build()

	pipelineWebhook := &PipelineVersionsWebhook{Client: fakeClient}

	validPipelineSpec := map[string]interface{}{
		"pipelineInfo": map[string]interface{}{
			"name":        "test-pipeline-v1",
			"description": "A simple test pipeline",
		},
		"root": map[string]interface{}{
			"dag": map[string]interface{}{
				"tasks": map[string]interface{}{},
			},
		},
		"schemaVersion": "2.1.0",
		"sdkVersion":    "kfp-2.11.0",
	}

	validPipelineSpecJSON, err := json.Marshal(validPipelineSpec)
	require.NoError(t, err, "Failed to marshal pipeline spec")

	return pipelineWebhook, string(validPipelineSpecJSON)
}

func TestPipelineVersionWebhook_ValidateCreate(t *testing.T) {
	pipelineWebhook, validPipelineSpecJSON := setupPipelineWebhookTest(t)

	pipelineVersion := &k8sapi.PipelineVersion{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pipeline-v1",
			Namespace: "default",
		},
		Spec: k8sapi.PipelineVersionSpec{
			PipelineName: "test-pipeline",
			PipelineSpec: k8sapi.PipelineIRSpec{
				Value: json.RawMessage(validPipelineSpecJSON),
			},
		},
	}
	_, err := pipelineWebhook.ValidateCreate(context.TODO(), pipelineVersion)
	assert.NoError(t, err, "Expected no error for a valid PipelineVersion")
}

func TestPipelineVersionWebhook_ValidateCreate_InvalidObjectType(t *testing.T) {
	pipelineWebhook, _ := setupPipelineWebhookTest(t)

	_, err := pipelineWebhook.ValidateCreate(context.TODO(), &k8sapi.Pipeline{})
	assert.Error(t, err, "Expected error when passing an object that is not a PipelineVersion")
	assert.Contains(t, err.Error(), "Expected a PipelineVersion object")
}

func TestPipelineVersionWebhook_ValidateCreate_InvalidPipelineSpec(t *testing.T) {
	pipelineWebhook, _ := setupPipelineWebhookTest(t)

	invalidPipelineSpec := map[string]interface{}{
		"pipelineInfo": map[string]interface{}{
			"name":        "test-pipeline-v1",
			"description": "A simple test pipeline",
		},
	}

	invalidPipelineSpecJSON, err := json.Marshal(invalidPipelineSpec)
	require.NoError(t, err, "Failed to marshal pipeline spec")

	invalidPipelineVersion := &k8sapi.PipelineVersion{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pipeline-v1",
			Namespace: "default",
		},
		Spec: k8sapi.PipelineVersionSpec{
			PipelineName: "test-pipeline",
			PipelineSpec: k8sapi.PipelineIRSpec{
				Value: json.RawMessage(invalidPipelineSpecJSON),
			},
		},
	}

	_, err = pipelineWebhook.ValidateCreate(context.TODO(), invalidPipelineVersion)
	assert.Error(t, err, "Expected error for invalid PipelineSpec")
	assert.Contains(t, err.Error(), "The pipeline spec is invalid")
}

func TestPipelineVersionWebhook_ValidateCreate_PipelineNameMismatch(t *testing.T) {
	pipelineWebhook, validPipelineSpecJSON := setupPipelineWebhookTest(t)

	invalidPipelineVersion := &k8sapi.PipelineVersion{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "wrong-name",
			Namespace: "default",
		},
		Spec: k8sapi.PipelineVersionSpec{
			PipelineName: "test-pipeline",
			PipelineSpec: k8sapi.PipelineIRSpec{
				Value: json.RawMessage(validPipelineSpecJSON),
			},
		},
	}

	_, err := pipelineWebhook.ValidateCreate(context.TODO(), invalidPipelineVersion)
	assert.Error(t, err, "Expected error for mismatched pipeline name")
	assert.Contains(t, err.Error(), "The object name must match spec.pipelineSpec.pipelineInformation.name")
}
