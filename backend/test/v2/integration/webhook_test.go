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
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/storage"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/crd/kubernetes/v2beta1"

	"github.com/google/uuid"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
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
	var statusError *k8serrors.StatusError
	require.ErrorAs(t, err, &statusError)
	assert.EqualValues(t, 400, statusError.ErrStatus.Code)
	cause, found := k8serrors.StatusCause(err, "pipelines.kubeflow.org/PipelineVersionRejected")
	require.True(t, found, "the API server must preserve the validating webhook's rejection cause")
	assert.Equal(t, "Pipeline spec is immutable; only mutable fields (display_name, tags) can be updated", cause.Message)
}

func TestPipelineVersionAdmissionErrorsIntegration(t *testing.T) {
	if os.Getenv("WEBHOOK_INTEGRATION") != "true" {
		t.Skip("The WEBHOOK_INTEGRATION environment variable is not set to true")
	}
	config, err := util.GetKubernetesConfig()
	require.NoError(t, err)
	config.Timeout = 30 * time.Second
	scheme := runtime.NewScheme()
	require.NoError(t, v2beta1.AddToScheme(scheme))
	k8sClient, err := ctrlclient.NewWithWatch(config, ctrlclient.Options{Scheme: scheme})
	require.NoError(t, err)
	podNamespace := viper.Get("POD_NAMESPACE")
	viper.Set("POD_NAMESPACE", "kubeflow")
	t.Cleanup(func() { viper.Set("POD_NAMESPACE", podNamespace) })
	name := "admission-errors-" + uuid.NewString()
	key := ctrlclient.ObjectKey{Namespace: "kubeflow", Name: name + "-v1"}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		for _, obj := range []ctrlclient.Object{
			&v2beta1.PipelineVersion{ObjectMeta: metav1.ObjectMeta{Namespace: key.Namespace, Name: key.Name}},
			&v2beta1.Pipeline{ObjectMeta: metav1.ObjectMeta{Namespace: key.Namespace, Name: name}},
		} {
			if err := k8sClient.Delete(ctx, obj); err != nil && !k8serrors.IsNotFound(err) {
				t.Errorf("Failed to clean up %T %s: %v", obj, obj.GetName(), err)
			}
		}
	})
	// Delete the real parent after the store resolves it but before the version reaches admission.
	// The error then comes from the deployed mutating webhook through the Kubernetes API server.
	racingClient := interceptor.NewClient(k8sClient, interceptor.Funcs{
		Create: func(ctx context.Context, c ctrlclient.WithWatch, obj ctrlclient.Object, opts ...ctrlclient.CreateOption) error {
			if version, ok := obj.(*v2beta1.PipelineVersion); ok && version.Spec.PipelineName == name {
				parent := &v2beta1.Pipeline{ObjectMeta: metav1.ObjectMeta{Namespace: key.Namespace, Name: name}}
				if err := c.Delete(ctx, parent); err != nil {
					return err
				}
				// Wait for the webhook's cache to observe deletion. Dry-run requests cannot persist an orphan.
				if err := wait.PollUntilContextTimeout(ctx, 100*time.Millisecond, 10*time.Second, true, func(ctx context.Context) (bool, error) {
					err := c.Create(ctx, version.DeepCopy(), ctrlclient.DryRunAll)
					if err == nil {
						return false, nil
					}
					if k8serrors.IsBadRequest(err) && strings.Contains(err.Error(), "The spec.pipelineName doesn't map to an existing Pipeline object") {
						return true, nil
					}
					return false, err
				}); err != nil {
					return err
				}
			}
			return c.Create(ctx, obj, opts...)
		},
	})
	store := storage.NewPipelineStoreKubernetes(racingClient, k8sClient)
	parent, err := store.CreatePipeline(&model.Pipeline{Name: name, Namespace: key.Namespace})
	require.NoError(t, err)
	_, fixture := getSimplePipelineVersion()
	spec, found, err := unstructured.NestedMap(fixture.Object, "spec", "pipelineSpec")
	require.NoError(t, err)
	require.True(t, found)
	validSpec, err := json.Marshal(spec)
	require.NoError(t, err)
	_, err = store.CreatePipelineVersion(&model.PipelineVersion{
		Name: "v1", PipelineId: parent.UUID, PipelineSpec: model.LargeText(validSpec),
	})
	var userError *util.UserError
	require.ErrorAs(t, err, &userError)
	assert.Equal(t, codes.InvalidArgument, userError.ExternalStatusCode())
	const message = "The spec.pipelineName doesn't map to an existing Pipeline object"
	assert.Equal(t, message, userError.ExternalMessage())
	assert.NotContains(t, userError.GRPCStatus().Message(), "admission webhook")
	assert.Equal(t, 1, strings.Count(userError.GRPCStatus().Message(), message))
	err = k8sClient.Get(context.Background(), key, &v2beta1.PipelineVersion{})
	assert.True(t, k8serrors.IsNotFound(err), "a denied version must not have been persisted: %v", err)
}
