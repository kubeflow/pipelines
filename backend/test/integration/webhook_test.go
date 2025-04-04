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

package integration

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/crd/kubernetes/v2beta1"
	"github.com/stretchr/testify/require"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
)

func getSimplePipelineVersion() (*unstructured.Unstructured, *unstructured.Unstructured) {
	p := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "pipelines.kubeflow.org/v2beta1",
			"kind":       "Pipeline",
			"metadata": map[string]interface{}{
				"name": "simple-pipeline",
			},
		},
	}

	pv := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "pipelines.kubeflow.org/v2beta1",
			"kind":       "PipelineVersion",
			"metadata": map[string]interface{}{
				"name": "simple-pipeline-v1",
			},
			"spec": map[string]interface{}{
				"pipelineName": p.GetName(),
				"pipelineSpec": map[string]interface{}{
					"components": map[string]interface{}{
						"comp-hello": map[string]interface{}{
							"executorLabel": "exec-hello",
						},
					},
					"deploymentSpec": map[string]interface{}{
						"executors": map[string]interface{}{
							"exec-hello": map[string]interface{}{
								"container": map[string]interface{}{
									"args":    []interface{}{"hello"},
									"command": []interface{}{"echo"},
									"image":   "python3.12",
								},
							},
						},
					},
					"pipelineInfo": map[string]interface{}{
						"description": "An simple pipeline",
						"name":        "my-pipeline",
					},
					"root": map[string]interface{}{
						"dag": map[string]interface{}{
							"tasks": map[string]interface{}{
								"hello": map[string]interface{}{
									"componentRef": map[string]interface{}{
										"name": "comp-hello",
									},
									"taskInfo": map[string]interface{}{
										"name": "hello",
									},
								},
							},
						},
					},
					"schemaVersion": "2.1.0",
					"sdkVersion":    "kfp-2.12.1",
				},
			},
		},
	}

	return p, pv
}

// createPipelineVersion will create a PipelineVersion and retry for up to 5 seconds if it fails to be created.
func createPipelineVersion(
	t *testing.T, client dynamic.ResourceInterface, pv *unstructured.Unstructured,
) *unstructured.Unstructured {
	t.Helper()

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Second*5))
	defer cancel()

	var prevErr error

	for {
		newPv, err := client.Create(ctx, pv, metav1.CreateOptions{})
		if err == nil {
			return newPv
		}

		// Workaround because it seems that the error isn't wrapped, so errors.Is doesn't work.
		if strings.Contains(err.Error(), "context deadline exceeded") {
			errToAssert := err
			if prevErr != nil {
				errToAssert = prevErr
			}

			require.NoError(t, errToAssert, "Timed out creating the PipelineVersion")
		}

		prevErr = err
		time.Sleep(500 * time.Millisecond)
	}
}

func TestPipelineVersionWebhookIntegration(t *testing.T) {
	if os.Getenv("WEBHOOK_INTEGRATION") != "true" {
		t.Skip("The WEBHOOK_INTEGRATION environment variable is not set to true")
	}

	config, err := util.GetKubernetesConfig()
	require.NoError(t, err, "Expected no error to get the Kubernetes configuration")

	client, err := dynamic.NewForConfig(config)
	require.NoError(t, err, "Expected no error when creating the dynamic client")

	pClient := client.Resource(v2beta1.GroupVersion.WithResource("pipelines")).Namespace("kubeflow")
	pvClient := client.Resource(v2beta1.GroupVersion.WithResource("pipelineversions")).Namespace("kubeflow")

	p, pv := getSimplePipelineVersion()

	defer func(p *unstructured.Unstructured, pv *unstructured.Unstructured) {
		err = pvClient.Delete(context.TODO(), pv.GetName(), metav1.DeleteOptions{})
		if err != nil && !k8serrors.IsNotFound(err) {
			t.Logf("Failed to delete the pipeline version: %v", err)
		}

		err := pClient.Delete(context.TODO(), p.GetName(), metav1.DeleteOptions{})
		if err != nil && !k8serrors.IsNotFound(err) {
			t.Logf("Failed to delete the pipeline: %v", err)
		}
	}(p, pv)

	p, err = pClient.Create(context.TODO(), p, metav1.CreateOptions{})
	require.NoError(t, err, "Expected no error when creating the pipeline")

	pv = createPipelineVersion(t, pvClient, pv)

	t.Log("Checking that the mutating webhook set the pipeline-id label")
	require.Equal(t, pv.GetLabels()["pipelines.kubeflow.org/pipeline-id"], string(p.GetUID()))

	t.Log("Checking that the mutating webhook set the pipeline label")
	require.Equal(t, pv.GetLabels()["pipelines.kubeflow.org/pipeline"], p.GetName())

	t.Log("Checking that the mutating webhook added the owners reference of the Pipeline object")
	ownerRefFound := false

	for _, ownerRef := range pv.GetOwnerReferences() {
		if ownerRef.APIVersion == v2beta1.GroupVersion.String() && ownerRef.Kind == "Pipeline" {
			ownerRefFound = true
			require.Equal(t, ownerRef.UID, p.GetUID())

			break
		}
	}

	require.True(t, ownerRefFound, "Expected to find an owner ref of the Pipeline on the PipelineVersion")

	// This test is to ensure the validating webhook is available. Most of the test coverage occurs in unit tests.
	t.Log("Checking that the validating webhook ensures the PipelineVersion is immutable")
	err = unstructured.SetNestedField(pv.Object, "new-name", "spec", "pipelineSpec", "pipelineInfo", "name")
	require.NoError(t, err, "Failed to set a field on the PipelineVersion using unstructured.SetNestedField")

	_, err = pvClient.Update(context.TODO(), pv, metav1.UpdateOptions{})
	require.ErrorContains(t, err, "Pipeline spec is immutable")
}
