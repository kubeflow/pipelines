/*
Copyright 2026.

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

package storage

import (
	"context"
	"fmt"
	"testing"

	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/backend/src/apiserver/list"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	"github.com/kubeflow/pipelines/backend/src/crd/kubernetes/v2beta1"
)

func TestBackwardCompat_GetPipelineVersion_LegacyCR(t *testing.T) {
	podNamespace := viper.Get("POD_NAMESPACE")
	viper.Set("POD_NAMESPACE", "Test")
	defer viper.Set("POD_NAMESPACE", podNamespace)

	store := NewPipelineStoreKubernetes(getClient())

	// The pre-loaded "test-pipeline-version-3" CR is legacy-style (bare metadata.name, no spec.VersionName)
	pv, err := store.GetPipelineVersion(DefaultFakePipelineIdTwo)
	require.NoError(t, err)
	assert.Equal(t, "test-pipeline-version-3", pv.Name)
}

func TestBackwardCompat_GetPipelineVersionByName_LegacyCR_WrongPipeline(t *testing.T) {
	podNamespace := viper.Get("POD_NAMESPACE")
	viper.Set("POD_NAMESPACE", "Test")
	defer viper.Set("POD_NAMESPACE", podNamespace)

	scheme := runtime.NewScheme()
	require.NoError(t, v2beta1.AddToScheme(scheme))

	// Pipeline that owns the version
	ownerPipeline := &v2beta1.Pipeline{
		ObjectMeta: metav1.ObjectMeta{
			UID:       DefaultFakePipelineIdTwo,
			Name:      "owner-pipeline",
			Namespace: "Test",
		},
	}
	// A different pipeline that does NOT own the version
	otherPipeline := &v2beta1.Pipeline{
		ObjectMeta: metav1.ObjectMeta{
			UID:       DefaultFakePipelineIdThree,
			Name:      "other-pipeline",
			Namespace: "Test",
		},
	}
	// Legacy-style CR belonging to ownerPipeline
	oldVersion := &v2beta1.PipelineVersion{
		ObjectMeta: metav1.ObjectMeta{
			UID:       DefaultFakePipelineIdFour,
			Name:      "shared-version-name",
			Namespace: "Test",
			Labels: map[string]string{
				"pipelines.kubeflow.org/pipeline-id": DefaultFakePipelineIdTwo,
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: v2beta1.GroupVersion.String(),
					Kind:       "Pipeline",
					UID:        DefaultFakePipelineIdTwo,
					Name:       "owner-pipeline",
				},
			},
		},
		Spec: v2beta1.PipelineVersionSpec{
			PipelineName: "owner-pipeline",
			PipelineSpec: getBasicPipelineSpec(),
		},
	}

	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(ownerPipeline, otherPipeline, oldVersion).
		Build()

	store := NewPipelineStoreKubernetes(k8sClient, k8sClient)

	// Looking up with the wrong pipeline should fail with NotFound
	_, err := store.GetPipelineVersionByName(DefaultFakePipelineIdThree, "other-pipeline", "shared-version-name")
	require.NotNil(t, err)
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())

	// Looking up with the correct pipeline should succeed
	pv, err := store.GetPipelineVersionByName(DefaultFakePipelineIdTwo, "owner-pipeline", "shared-version-name")
	require.NoError(t, err)
	assert.Equal(t, "shared-version-name", pv.Name)
}

func TestBackwardCompat_GetPipelineVersionByName_OwnerRefLabelMismatch(t *testing.T) {
	podNamespace := viper.Get("POD_NAMESPACE")
	viper.Set("POD_NAMESPACE", "Test")
	defer viper.Set("POD_NAMESPACE", podNamespace)

	scheme := runtime.NewScheme()
	require.NoError(t, v2beta1.AddToScheme(scheme))

	pipelineA := &v2beta1.Pipeline{
		ObjectMeta: metav1.ObjectMeta{
			UID:       DefaultFakePipelineIdTwo,
			Name:      "pipeline-a",
			Namespace: "Test",
		},
	}
	pipelineB := &v2beta1.Pipeline{
		ObjectMeta: metav1.ObjectMeta{
			UID:       DefaultFakePipelineIdThree,
			Name:      "pipeline-b",
			Namespace: "Test",
		},
	}
	// CR with OwnerReference pointing to pipeline A but label edited to pipeline B
	mismatchVersion := &v2beta1.PipelineVersion{
		ObjectMeta: metav1.ObjectMeta{
			UID:       DefaultFakePipelineIdFour,
			Name:      "mismatch-version",
			Namespace: "Test",
			Labels: map[string]string{
				"pipelines.kubeflow.org/pipeline-id": DefaultFakePipelineIdThree,
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: v2beta1.GroupVersion.String(),
					Kind:       "Pipeline",
					UID:        DefaultFakePipelineIdTwo,
					Name:       "pipeline-a",
				},
			},
		},
		Spec: v2beta1.PipelineVersionSpec{
			PipelineName: "pipeline-a",
			PipelineSpec: getBasicPipelineSpec(),
		},
	}

	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(pipelineA, pipelineB, mismatchVersion).
		Build()

	store := NewPipelineStoreKubernetes(k8sClient, k8sClient)

	// OwnerRef points to A, label points to B. Querying with B should reject
	// because ownership is determined by OwnerReferences.
	_, err := store.GetPipelineVersionByName(DefaultFakePipelineIdThree, "pipeline-b", "mismatch-version")
	require.NotNil(t, err)
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())

	// Querying with A (the true owner) should succeed
	pv, err := store.GetPipelineVersionByName(DefaultFakePipelineIdTwo, "pipeline-a", "mismatch-version")
	require.NoError(t, err)
	assert.Equal(t, "mismatch-version", pv.Name)
}

func TestBackwardCompat_ListPipelineVersions_MixedCRs(t *testing.T) {
	podNamespace := viper.Get("POD_NAMESPACE")
	viper.Set("POD_NAMESPACE", "Test")
	defer viper.Set("POD_NAMESPACE", podNamespace)

	store := NewPipelineStoreKubernetes(getClient())

	// The pre-loaded "test-pipeline-version-3" is legacy-style (bare name, under DefaultFakePipelineIdTwo).
	// Create a new-style version under the same pipeline.
	_, err := store.CreatePipelineVersion(&model.PipelineVersion{
		Name:         "new-style-version",
		PipelineId:   DefaultFakePipelineIdTwo,
		PipelineSpec: model.LargeText(getBasicPipelineSpecYAML()),
	})
	require.NoError(t, err)

	options := list.EmptyOptions()
	versions, total, _, err := store.ListPipelineVersions(DefaultFakePipelineIdTwo, options, nil)
	require.NoError(t, err)
	require.Equal(t, 2, total)

	// Both old and new versions should have correct bare names
	names := make([]string, len(versions))
	for i, v := range versions {
		names[i] = v.Name
	}
	assert.Contains(t, names, "test-pipeline-version-3")
	assert.Contains(t, names, "new-style-version")
}

func TestBackwardCompat_GetLatestPipelineVersion_MixedCRs(t *testing.T) {
	podNamespace := viper.Get("POD_NAMESPACE")
	viper.Set("POD_NAMESPACE", "Test")
	defer viper.Set("POD_NAMESPACE", podNamespace)

	store := NewPipelineStoreKubernetes(getClient())

	// Create a new-style version (will have a later creation timestamp)
	_, err := store.CreatePipelineVersion(&model.PipelineVersion{
		Name:         "new-latest-version",
		PipelineId:   DefaultFakePipelineIdTwo,
		PipelineSpec: model.LargeText(getBasicPipelineSpecYAML()),
	})
	require.NoError(t, err)

	latest, err := store.GetLatestPipelineVersion(DefaultFakePipelineIdTwo)
	require.NoError(t, err)
	assert.Equal(t, "new-latest-version", latest.Name)
}

func TestBackwardCompat_DeletePipelineVersion_LegacyCR(t *testing.T) {
	podNamespace := viper.Get("POD_NAMESPACE")
	viper.Set("POD_NAMESPACE", "Test")
	defer viper.Set("POD_NAMESPACE", podNamespace)

	store := NewPipelineStoreKubernetes(getClient())

	// Delete legacy-style CR by UUID
	err := store.DeletePipelineVersion(DefaultFakePipelineIdTwo)
	require.NoError(t, err)

	// Verify it's gone
	_, err = store.GetPipelineVersion(DefaultFakePipelineIdTwo)
	require.NotNil(t, err)
	assert.Equal(t, err.(*util.UserError).ExternalStatusCode(), codes.NotFound)
}

func TestCreatePipelineVersion_DuplicateUnderSamePipeline(t *testing.T) {
	podNamespace := viper.Get("POD_NAMESPACE")
	viper.Set("POD_NAMESPACE", "Test")
	defer viper.Set("POD_NAMESPACE", podNamespace)

	store := NewPipelineStoreKubernetes(getClient())

	pipeline, err := store.CreatePipeline(&model.Pipeline{
		Name:      "dup-test-pipeline",
		Namespace: "Test",
	})
	require.NoError(t, err)

	_, err = store.CreatePipelineVersion(&model.PipelineVersion{
		Name:         "v1",
		PipelineId:   pipeline.UUID,
		PipelineSpec: model.LargeText(getBasicPipelineSpecYAML()),
	})
	require.NoError(t, err)

	// Second creation with same name under same pipeline should fail
	_, err = store.CreatePipelineVersion(&model.PipelineVersion{
		Name:         "v1",
		PipelineId:   pipeline.UUID,
		PipelineSpec: model.LargeText(getBasicPipelineSpecYAML()),
	})
	require.NotNil(t, err)
	assert.Contains(t, err.Error(), "already exist")
}

func TestCreatePipelineAndPipelineVersion_SameVersionNameDifferentPipelines(t *testing.T) {
	podNamespace := viper.Get("POD_NAMESPACE")
	viper.Set("POD_NAMESPACE", "Test")
	defer viper.Set("POD_NAMESPACE", podNamespace)

	store := NewPipelineStoreKubernetes(getClient())

	_, _, err := store.CreatePipelineAndPipelineVersion(
		&model.Pipeline{Name: "atomic-pipeline-a"},
		&model.PipelineVersion{
			Name:         "initial",
			PipelineSpec: model.LargeText(getBasicPipelineSpecYAML()),
		},
	)
	require.NoError(t, err)

	_, _, err = store.CreatePipelineAndPipelineVersion(
		&model.Pipeline{Name: "atomic-pipeline-b"},
		&model.PipelineVersion{
			Name:         "initial",
			PipelineSpec: model.LargeText(getBasicPipelineSpecYAML()),
		},
	)
	require.NoError(t, err)
}

func TestCreatePipelineVersion_HyphenCollision(t *testing.T) {
	podNamespace := viper.Get("POD_NAMESPACE")
	viper.Set("POD_NAMESPACE", "Test")
	defer viper.Set("POD_NAMESPACE", podNamespace)

	scheme := runtime.NewScheme()
	require.NoError(t, v2beta1.AddToScheme(scheme))

	pipelineFoo := &v2beta1.Pipeline{
		ObjectMeta: metav1.ObjectMeta{
			UID:       DefaultFakePipelineIdThree,
			Name:      "foo",
			Namespace: "Test",
		},
	}
	pipelineFooBar := &v2beta1.Pipeline{
		ObjectMeta: metav1.ObjectMeta{
			UID:       DefaultFakePipelineIdFour,
			Name:      "foo-bar",
			Namespace: "Test",
		},
	}
	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(pipelineFoo, pipelineFooBar).
		Build()

	store := NewPipelineStoreKubernetes(k8sClient, k8sClient)

	// Pipeline "foo" + version "bar-baz" → composite "foo-bar-baz"
	_, err := store.CreatePipelineVersion(&model.PipelineVersion{
		Name:         "bar-baz",
		PipelineId:   DefaultFakePipelineIdThree,
		PipelineSpec: model.LargeText(getBasicPipelineSpecYAML()),
	})
	require.NoError(t, err)

	// Pipeline "foo-bar" + version "baz" → also composite "foo-bar-baz"
	// Known limitation: this collides because both produce the same composite K8s name
	_, err = store.CreatePipelineVersion(&model.PipelineVersion{
		Name:         "baz",
		PipelineId:   DefaultFakePipelineIdFour,
		PipelineSpec: model.LargeText(getBasicPipelineSpecYAML()),
	})
	require.NotNil(t, err, "Expected collision due to identical composite K8s names")
	assert.Contains(t, err.Error(), "already exist")
	assert.Contains(t, err.Error(), "foo-bar-baz")
}

func TestCreatePipelineVersion_DuplicateLegacyBareName(t *testing.T) {
	podNamespace := viper.Get("POD_NAMESPACE")
	viper.Set("POD_NAMESPACE", "Test")
	defer viper.Set("POD_NAMESPACE", podNamespace)

	scheme := runtime.NewScheme()
	require.NoError(t, v2beta1.AddToScheme(scheme))

	pipeline := &v2beta1.Pipeline{
		ObjectMeta: metav1.ObjectMeta{
			UID:       DefaultFakePipelineIdThree,
			Name:      "my-pipeline",
			Namespace: "Test",
		},
	}

	// Pre-seed a legacy bare-name CR "v1" owned by the same pipeline
	legacyVersion := &v2beta1.PipelineVersion{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "v1",
			Namespace: "Test",
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: v2beta1.GroupVersion.String(),
					Kind:       "Pipeline",
					UID:        DefaultFakePipelineIdThree,
					Name:       "my-pipeline",
				},
			},
		},
		Spec: v2beta1.PipelineVersionSpec{
			PipelineSpec: getBasicPipelineSpec(),
			PipelineName: "my-pipeline",
			VersionName:  "v1",
		},
	}

	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(pipeline, legacyVersion).
		Build()

	store := NewPipelineStoreKubernetes(k8sClient, k8sClient)

	// Creating a new version "v1" under the same pipeline should fail,
	// even though the composite K8s name "my-pipeline-v1" differs from legacy "v1"
	_, err := store.CreatePipelineVersion(&model.PipelineVersion{
		Name:         "v1",
		PipelineId:   DefaultFakePipelineIdThree,
		PipelineSpec: model.LargeText(getBasicPipelineSpecYAML()),
	})
	require.NotNil(t, err, "Expected duplicate rejection against legacy bare-name CR")
	assert.Contains(t, err.Error(), "already exist")
}

func TestCreatePipelineVersion_TransientErrorDuringLookup(t *testing.T) {
	podNamespace := viper.Get("POD_NAMESPACE")
	viper.Set("POD_NAMESPACE", "Test")
	defer viper.Set("POD_NAMESPACE", podNamespace)

	scheme := runtime.NewScheme()
	require.NoError(t, v2beta1.AddToScheme(scheme))

	pipeline := &v2beta1.Pipeline{
		ObjectMeta: metav1.ObjectMeta{
			UID:       DefaultFakePipelineIdThree,
			Name:      "my-pipeline",
			Namespace: "Test",
		},
	}

	legacyVersion := &v2beta1.PipelineVersion{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "v1",
			Namespace: "Test",
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: v2beta1.GroupVersion.String(),
					Kind:       "Pipeline",
					UID:        DefaultFakePipelineIdThree,
					Name:       "my-pipeline",
				},
			},
		},
		Spec: v2beta1.PipelineVersionSpec{
			PipelineSpec: getBasicPipelineSpec(),
			PipelineName: "my-pipeline",
			VersionName:  "v1",
		},
	}

	// Inject a transient error on the second List call (the one inside
	// GetPipelineVersionByName → getK8sPipeline). The first List call
	// (from CreatePipelineVersion → GetPipeline → getK8sPipeline) succeeds.
	listCallCount := 0
	transientError := fmt.Errorf("simulated transient API server error")
	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(pipeline, legacyVersion).
		WithInterceptorFuncs(interceptor.Funcs{
			List: func(ctx context.Context, client client.WithWatch, list client.ObjectList, opts ...client.ListOption) error {
				listCallCount++
				if listCallCount == 2 {
					return transientError
				}
				return client.List(ctx, list, opts...)
			},
		}).
		Build()

	store := NewPipelineStoreKubernetes(k8sClient, k8sClient)

	_, err := store.CreatePipelineVersion(&model.PipelineVersion{
		Name:         "v1",
		PipelineId:   DefaultFakePipelineIdThree,
		PipelineSpec: model.LargeText(getBasicPipelineSpecYAML()),
	})
	require.NotNil(t, err, "Expected error to propagate from transient lookup failure")
	assert.Contains(t, err.Error(), "simulated transient")
}

func TestGetPipelineVersionByName_InvalidPipelineId(t *testing.T) {
	podNamespace := viper.Get("POD_NAMESPACE")
	viper.Set("POD_NAMESPACE", "Test")
	defer viper.Set("POD_NAMESPACE", podNamespace)

	store := NewPipelineStoreKubernetes(getClient())

	_, err := store.GetPipelineVersionByName("nonexistent-pipeline-id", "", "v1.0")
	require.NotNil(t, err)
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())
}

func TestGetPipelineVersionByName_HyphenCollisionFallthrough(t *testing.T) {
	podNamespace := viper.Get("POD_NAMESPACE")
	viper.Set("POD_NAMESPACE", "Test")
	defer viper.Set("POD_NAMESPACE", podNamespace)

	scheme := runtime.NewScheme()
	require.NoError(t, v2beta1.AddToScheme(scheme))

	// Pipeline "foo" and pipeline "foo-bar"
	pipelineFoo := &v2beta1.Pipeline{
		ObjectMeta: metav1.ObjectMeta{
			UID:       DefaultFakePipelineIdThree,
			Name:      "foo",
			Namespace: "Test",
		},
	}
	pipelineFooBar := &v2beta1.Pipeline{
		ObjectMeta: metav1.ObjectMeta{
			UID:       DefaultFakePipelineIdFour,
			Name:      "foo-bar",
			Namespace: "Test",
		},
	}

	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(pipelineFoo, pipelineFooBar).
		Build()

	store := NewPipelineStoreKubernetes(k8sClient, k8sClient)

	// Create "foo" + "bar-baz" → composite "foo-bar-baz", owned by pipelineFoo
	_, err := store.CreatePipelineVersion(&model.PipelineVersion{
		Name:         "bar-baz",
		PipelineId:   DefaultFakePipelineIdThree,
		PipelineSpec: model.LargeText(getBasicPipelineSpecYAML()),
	})
	require.NoError(t, err)

	// Lookup "foo-bar" + "baz" → composite is also "foo-bar-baz", but owned by
	// pipelineFoo, not pipelineFooBar. The code should detect the ownership
	// mismatch and fall through to bare-name lookup, which also fails → NotFound.
	_, err = store.GetPipelineVersionByName(DefaultFakePipelineIdFour, "foo-bar", "baz")
	require.NotNil(t, err)
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())
}

func TestGetPipelineVersionByName_HyphenCollisionFallbackSuccess(t *testing.T) {
	podNamespace := viper.Get("POD_NAMESPACE")
	viper.Set("POD_NAMESPACE", "Test")
	defer viper.Set("POD_NAMESPACE", podNamespace)

	scheme := runtime.NewScheme()
	require.NoError(t, v2beta1.AddToScheme(scheme))

	pipelineFoo := &v2beta1.Pipeline{
		ObjectMeta: metav1.ObjectMeta{
			UID:       DefaultFakePipelineIdThree,
			Name:      "foo",
			Namespace: "Test",
		},
	}
	pipelineFooBar := &v2beta1.Pipeline{
		ObjectMeta: metav1.ObjectMeta{
			UID:       DefaultFakePipelineIdFour,
			Name:      "foo-bar",
			Namespace: "Test",
		},
	}

	// Legacy bare-name CR owned by pipeline "foo-bar"
	legacyVersion := &v2beta1.PipelineVersion{
		ObjectMeta: metav1.ObjectMeta{
			UID:       "legacy-version-uid",
			Name:      "baz",
			Namespace: "Test",
			Labels: map[string]string{
				"pipelines.kubeflow.org/pipeline-id": DefaultFakePipelineIdFour,
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: v2beta1.GroupVersion.String(),
					Kind:       "Pipeline",
					UID:        DefaultFakePipelineIdFour,
					Name:       "foo-bar",
				},
			},
		},
		Spec: v2beta1.PipelineVersionSpec{
			PipelineName: "foo-bar",
			PipelineSpec: getBasicPipelineSpec(),
		},
	}

	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(pipelineFoo, pipelineFooBar, legacyVersion).
		Build()

	store := NewPipelineStoreKubernetes(k8sClient, k8sClient)

	// Pipeline "foo" + version "bar-baz" → composite "foo-bar-baz"
	_, err := store.CreatePipelineVersion(&model.PipelineVersion{
		Name:         "bar-baz",
		PipelineId:   DefaultFakePipelineIdThree,
		PipelineSpec: model.LargeText(getBasicPipelineSpecYAML()),
	})
	require.NoError(t, err)

	// Lookup "foo-bar" + "baz": composite "foo-bar-baz" exists but is owned by
	// pipeline "foo". The code detects the ownership mismatch, falls back to
	// bare-name "baz", and finds the legacy CR owned by pipeline "foo-bar".
	pv, err := store.GetPipelineVersionByName(DefaultFakePipelineIdFour, "foo-bar", "baz")
	require.NoError(t, err)
	assert.Equal(t, "baz", pv.Name)
	assert.Equal(t, DefaultFakePipelineIdFour, pv.PipelineId)
	assert.Equal(t, "legacy-version-uid", pv.UUID)
}

func TestCreatePipelineAndPipelineVersion_InvalidVersionName(t *testing.T) {
	podNamespace := viper.Get("POD_NAMESPACE")
	viper.Set("POD_NAMESPACE", "Test")
	defer viper.Set("POD_NAMESPACE", podNamespace)

	store := NewPipelineStoreKubernetes(getClient())

	// Invalid version name should fail before creating the pipeline (no orphan)
	_, _, err := store.CreatePipelineAndPipelineVersion(
		&model.Pipeline{Name: "should-not-be-created"},
		&model.PipelineVersion{
			Name:         "INVALID-UPPERCASE",
			PipelineSpec: model.LargeText(getBasicPipelineSpecYAML()),
		},
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid pipeline version name")

	// Verify the pipeline was NOT created
	_, err = store.GetPipelineByNameAndNamespace("should-not-be-created", "")
	require.NotNil(t, err)
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())
}

func TestGetPipelineVersionByName_DifferentNamespace(t *testing.T) {
	podNamespace := viper.Get("POD_NAMESPACE")
	viper.Set("POD_NAMESPACE", "DefaultNS")
	defer viper.Set("POD_NAMESPACE", podNamespace)

	multiUser := viper.Get("MULTIUSER")
	viper.Set("MULTIUSER", "true")
	defer viper.Set("MULTIUSER", multiUser)

	scheme := runtime.NewScheme()
	require.NoError(t, v2beta1.AddToScheme(scheme))

	pipeline := &v2beta1.Pipeline{
		ObjectMeta: metav1.ObjectMeta{
			UID:       DefaultFakePipelineIdThree,
			Name:      "user-pipeline",
			Namespace: "UserNS",
		},
	}

	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(pipeline).
		Build()

	store := NewPipelineStoreKubernetes(k8sClient, k8sClient)

	_, err := store.CreatePipelineVersion(&model.PipelineVersion{
		Name:         "v1",
		PipelineId:   DefaultFakePipelineIdThree,
		PipelineSpec: model.LargeText(getBasicPipelineSpecYAML()),
	})
	require.NoError(t, err)

	// POD_NAMESPACE is "DefaultNS" but the version lives in "UserNS"
	version, err := store.GetPipelineVersionByName(DefaultFakePipelineIdThree, "user-pipeline", "v1")
	require.NoError(t, err)
	assert.Equal(t, "v1", version.Name)
	assert.Equal(t, DefaultFakePipelineIdThree, version.PipelineId)
}

func getClientWithTwoPipelines() (client.Client, client.Client) {
	scheme := runtime.NewScheme()
	err := v2beta1.AddToScheme(scheme)
	if err != nil {
		glog.Fatalf("Failed to add to scheme: %v", err)
	}

	pipelineAlpha := &v2beta1.Pipeline{
		ObjectMeta: metav1.ObjectMeta{
			UID:       DefaultFakePipelineIdThree,
			Name:      "pipeline-alpha",
			Namespace: "Test",
		},
		Spec: v2beta1.PipelineSpec{
			Description: "Pipeline Alpha",
		},
	}

	pipelineBeta := &v2beta1.Pipeline{
		ObjectMeta: metav1.ObjectMeta{
			UID:       DefaultFakePipelineIdFour,
			Name:      "pipeline-beta",
			Namespace: "Test",
		},
		Spec: v2beta1.PipelineSpec{
			Description: "Pipeline Beta",
		},
	}

	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(pipelineAlpha, pipelineBeta).
		Build()

	return k8sClient, k8sClient
}
