package storage

import (
	"context"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/golang/glog"
	api "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/filter"
	"github.com/kubeflow/pipelines/backend/src/apiserver/list"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	"github.com/kubeflow/pipelines/backend/src/crd/kubernetes/v2beta1"
)

func TestListK8sPipelines(t *testing.T) {
	podNamespace := viper.Get("POD_NAMESPACE")
	viper.Set("POD_NAMESPACE", "Test")
	defer viper.Set("POD_NAMESPACE", podNamespace)

	store := NewPipelineStoreKubernetes(getClient())

	fc := &model.FilterContext{}
	options := list.EmptyOptions()

	_, size, _, err := store.ListPipelines(fc, options)
	require.Nil(t, err, "Failed to list all pipelines: %v")
	require.Equalf(t, size, 1, "List size is not zero")

	pipeline := &model.Pipeline{
		Name:        "test-pipeline",
		Description: model.LargeText("Test Pipeline Description"),
		Namespace:   "Test",
	}

	_, err = store.CreatePipeline(pipeline)
	require.Nil(t, err, "Failed to create Pipeline: %v", err)

	_, size, _, err = store.ListPipelines(fc, options)
	require.Nil(t, err, "Failed to list all pipelines: %v", err)
	require.Equalf(t, size, 2, "List size should not be zero")
}

func TestListK8sPipelines_WithFilter(t *testing.T) {
	podNamespace := viper.Get("POD_NAMESPACE")
	viper.Set("POD_NAMESPACE", "Test")
	defer viper.Set("POD_NAMESPACE", podNamespace)

	store := NewPipelineStoreKubernetes(getClient())

	pipeline := &model.Pipeline{
		Name:        "test-pipeline",
		Description: model.LargeText("Test Pipeline Description"),
		Namespace:   "Test",
	}
	_, err := store.CreatePipeline(pipeline)
	require.Nil(t, err, "Failed to create Pipeline: %v")

	filterProto := &api.Filter{
		Predicates: []*api.Predicate{
			{
				Key:       "name",
				Operation: api.Predicate_IS_SUBSTRING,
				Value:     &api.Predicate_StringValue{StringValue: "test"},
			},
		},
	}
	newFilter, _ := filter.New(filterProto)
	options, err1 := list.NewOptions(&model.Pipeline{}, 10, "id", newFilter)
	require.Nil(t, err1, "Failed to create list options: %v")

	pipelines, _, _, err2 := store.ListPipelines(&model.FilterContext{}, options)
	require.Nil(t, err2, "Failed to list pipelines: %v")
	require.Equalf(t, len(pipelines), 2, "List size should return 2")
}

func TestListK8sPipelines_Pagination(t *testing.T) {
	podNamespace := viper.Get("POD_NAMESPACE")
	viper.Set("POD_NAMESPACE", "Test")
	defer viper.Set("POD_NAMESPACE", podNamespace)

	store := NewPipelineStoreKubernetes(getClient())

	pipeline1 := &model.Pipeline{
		Name:        "test-pipeline-1",
		Description: model.LargeText("Test Pipeline 1 Description"),
		Namespace:   "Test",
	}
	pipeline2 := &model.Pipeline{
		Name:        "test-pipeline-2",
		Description: model.LargeText("Test Pipeline 2 Description"),
		Namespace:   "Test",
	}

	_, err := store.CreatePipeline(pipeline1)
	require.Nil(t, err, "Failed to create Pipeline: %v")
	_, err = store.CreatePipeline(pipeline2)
	require.Nil(t, err, "Failed to create Pipeline: %v")

	options, err1 := list.NewOptions(&model.Pipeline{}, 1, "", nil)
	require.Nil(t, err1, "Failed to create list options: %v")

	_, pageSize, npt, err2 := store.ListPipelines(&model.FilterContext{}, options)
	require.Nil(t, err2, "Failed to list pipelines: %v")
	require.NotNil(t, npt)
	require.Equalf(t, pageSize, 3, "List size should not be zero")

	options, err1 = list.NewOptionsFromToken(npt, 1)
	require.Nil(t, err1, "Failed to create list options: %v")
	pipelines, _, _, err3 := store.ListPipelines(&model.FilterContext{}, options)
	require.Nil(t, err3, "Failed to list pipelines: %v")
	require.Equalf(t, pipelines[0].Name, "test-pipeline-1", "Pagination failed")
}

func TestListK8sPipelines_Pagination_Descend(t *testing.T) {
	podNamespace := viper.Get("POD_NAMESPACE")
	viper.Set("POD_NAMESPACE", "Test")
	defer viper.Set("POD_NAMESPACE", podNamespace)

	store := NewPipelineStoreKubernetes(getClient())

	pipeline1 := &model.Pipeline{
		Name:        "test-pipeline-1",
		Description: model.LargeText("Test Pipeline 1 Description"),
		Namespace:   "Test",
	}
	pipeline2 := &model.Pipeline{
		Name:        "test-pipeline-2",
		Description: model.LargeText("Test Pipeline 2 Description"),
		Namespace:   "Test",
	}

	_, err := store.CreatePipeline(pipeline1)
	require.Nil(t, err, "Failed to create Pipeline: %v")
	_, err = store.CreatePipeline(pipeline2)
	require.Nil(t, err, "Failed to create Pipeline: %v")

	options, err1 := list.NewOptions(&model.Pipeline{}, 1, "name desc", nil)
	require.Nil(t, err1, "Failed to create list options: %v")

	_, pageSize, npt, err2 := store.ListPipelines(&model.FilterContext{}, options)
	require.Nil(t, err2, "Failed to list pipelines: %v")
	require.NotNil(t, npt)
	require.Equalf(t, pageSize, 3, "List size should not be zero")

	options, err1 = list.NewOptionsFromToken(npt, 1)
	require.NoError(t, err1)
	pipelines, _, _, err3 := store.ListPipelines(&model.FilterContext{}, options)
	require.Nil(t, err3, "Failed to list pipelines: %v")
	require.Equalf(t, pipelines[0].Name, "test-pipeline-2", "Pagination failed")
}

func TestListK8sPipelinesV1_Pagination_NameAsc(t *testing.T) {
	podNamespace := viper.Get("POD_NAMESPACE")
	viper.Set("POD_NAMESPACE", "Test")
	defer viper.Set("POD_NAMESPACE", podNamespace)

	store := NewPipelineStoreKubernetes(getClient())

	pipeline1 := &model.Pipeline{
		Name:        "test-pipeline-1",
		Description: model.LargeText("Test Pipeline 1 Description"),
		Namespace:   "Test",
	}
	pipeline2 := &model.Pipeline{
		Name:        "test-pipeline-2",
		Description: model.LargeText("Test Pipeline 2 Description"),
		Namespace:   "Test",
	}

	_, err := store.CreatePipeline(pipeline1)
	require.Nil(t, err, "Failed to create Pipeline: %v")
	_, err = store.CreatePipeline(pipeline2)
	require.Nil(t, err, "Failed to create Pipeline: %v")

	options, err1 := list.NewOptions(&model.Pipeline{}, 1, "name", nil)
	require.Nil(t, err1, "Failed to create list options: %v")

	_, pageSize, npt, err2 := store.ListPipelines(&model.FilterContext{}, options)
	require.Nil(t, err2, "Failed to list pipelines: %v")
	require.NotNil(t, npt)
	require.Equalf(t, pageSize, 3, "List size should not be zero")

	options, err1 = list.NewOptionsFromToken(npt, 1)
	require.NoError(t, err1)
	pipelines, _, _, err3 := store.ListPipelines(&model.FilterContext{}, options)
	require.Nil(t, err3, "Failed to list pipelines: %v")
	require.Equalf(t, pipelines[0].Name, "test-pipeline-2", "Pagination failed")
}

func TestListK8sPipelines_Pagination_LessThanPageSize(t *testing.T) {
	podNamespace := viper.Get("POD_NAMESPACE")
	viper.Set("POD_NAMESPACE", "Test")
	defer viper.Set("POD_NAMESPACE", podNamespace)

	store := NewPipelineStoreKubernetes(getClient())

	options, err1 := list.NewOptions(&model.Pipeline{}, 10, "", nil)
	require.Nil(t, err1, "Failed to create list options: %v")

	pipelines, pageSize, _, err := store.ListPipelines(&model.FilterContext{}, options)
	require.Nil(t, err, "Failed to list pipelines: %v")
	require.Equalf(t, pageSize, 1, "Page size should be 1")
	require.Equalf(t, len(pipelines), 1, "List size should be 1")
}

func TestGetK8sPipeline(t *testing.T) {
	// This is important for getting a K8s pipeline
	podNamespace := viper.Get("POD_NAMESPACE")
	viper.Set("POD_NAMESPACE", "Test")
	defer viper.Set("POD_NAMESPACE", podNamespace)

	store := NewPipelineStoreKubernetes(getClient())

	p, err := store.GetPipeline(DefaultFakePipelineIdTwo)
	require.Nil(t, err, "Failed to get Pipeline: %v", err)
	require.Equal(t, p.UUID, DefaultFakePipelineIdTwo)
}

func TestGetK8sPipeline_NotFoundError(t *testing.T) {
	// This is important for getting a K8s pipeline
	podNamespace := viper.Get("POD_NAMESPACE")
	viper.Set("POD_NAMESPACE", "Test")
	defer viper.Set("POD_NAMESPACE", podNamespace)

	store := NewPipelineStoreKubernetes(getClient())

	_, err := store.GetPipeline(DefaultFakePipelineIdFive)
	require.NotNil(t, err)
}

func TestGetK8sPipelineByNameAndNamespace_SingleUserDefaultsToPodNamespace(t *testing.T) {
	podNamespace := viper.Get("POD_NAMESPACE")
	viper.Set("POD_NAMESPACE", "Test")
	defer viper.Set("POD_NAMESPACE", podNamespace)

	// Set the mode explicitly rather than relying on the ambient global, so this
	// case establishes single-user on its own.
	multiUser := viper.Get("MULTIUSER")
	viper.Set("MULTIUSER", "false")
	defer viper.Set("MULTIUSER", multiUser)

	store := NewPipelineStoreKubernetes(getClient())

	// Single-user mode has one namespace, so the fallback stays correct.
	pipeline, err := store.GetPipelineByNameAndNamespace("test-pipeline-3", "")
	require.NoError(t, err)
	assert.Equal(t, "test-pipeline-3", pipeline.Name)
}

func TestGetK8sPipelineByNameAndNamespace_MultiUserRequiresNamespace(t *testing.T) {
	podNamespace := viper.Get("POD_NAMESPACE")
	viper.Set("POD_NAMESPACE", "Test")
	defer viper.Set("POD_NAMESPACE", podNamespace)

	multiUser := viper.Get("MULTIUSER")
	viper.Set("MULTIUSER", "true")
	defer viper.Set("MULTIUSER", multiUser)

	store := NewPipelineStoreKubernetes(getClient())

	// "test-pipeline-3" lives in the pod namespace, which must not be reachable without naming it.
	_, err := store.GetPipelineByNameAndNamespace("test-pipeline-3", "")
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, err.(*util.UserError).ExternalStatusCode())

	pipeline, err := store.GetPipelineByNameAndNamespace("test-pipeline-3", "Test")
	require.NoError(t, err)
	assert.Equal(t, "test-pipeline-3", pipeline.Name)
}

func TestCreateK8sPipeline(t *testing.T) {
	podNamespace := viper.Get("POD_NAMESPACE")
	viper.Set("POD_NAMESPACE", "Test")
	defer viper.Set("POD_NAMESPACE", podNamespace)

	store := NewPipelineStoreKubernetes(getClient())

	pipeline := &model.Pipeline{
		Name:        "test-pipeline",
		Description: model.LargeText("Test Pipeline Description"),
		Namespace:   "Test",
	}

	pipeline, err := store.CreatePipeline(pipeline)
	if err != nil {
		t.Fatalf("Failed to create Pipeline: %v", err)
	}

	require.Equalf(t, pipeline.Name, "test-pipeline", "Pipeline name is not the same")
}

func TestDeleteK8sPipeline(t *testing.T) {
	podNamespace := viper.Get("POD_NAMESPACE")
	viper.Set("POD_NAMESPACE", "Test")
	defer viper.Set("POD_NAMESPACE", podNamespace)

	store := NewPipelineStoreKubernetes(getClient())

	err := store.DeletePipeline(DefaultFakePipelineId)
	require.Nil(t, err, "Failed to delete Pipeline: %v", err)

	// Check if Deletion worked by querying the same UUID
	_, err1 := store.GetPipeline(DefaultFakePipelineId)
	require.NotNil(t, err1)
}

func TestCreateK8sPipelineVersion(t *testing.T) {
	podNamespace := viper.Get("POD_NAMESPACE")
	viper.Set("POD_NAMESPACE", "Test")
	defer viper.Set("POD_NAMESPACE", podNamespace)

	store := NewPipelineStoreKubernetes(getClient())

	pipelineVersion := &model.PipelineVersion{
		Name:         "test-pipeline-version",
		PipelineId:   DefaultFakePipelineIdTwo,
		Description:  model.LargeText("Test Pipeline Version Description"),
		PipelineSpec: model.LargeText(getBasicPipelineSpecYAML()),
	}

	_, err := store.CreatePipelineVersion(pipelineVersion)
	require.Nil(t, err, "Failed to create PipelineVersion: %v", err)
}

func TestDeleteK8sPipelineVersion(t *testing.T) {
	podNamespace := viper.Get("POD_NAMESPACE")
	viper.Set("POD_NAMESPACE", "Test")
	defer viper.Set("POD_NAMESPACE", podNamespace)

	store := NewPipelineStoreKubernetes(getClient())

	err := store.DeletePipelineVersion(DefaultFakePipelineId)
	require.Nil(t, err, "Failed to delete PipelineVersion: %v", err)

	// Check if pipeline version was deleted
	pv, err1 := store.GetPipelineVersion(DefaultFakePipelineId)
	require.NotNil(t, err1)
	require.Nil(t, pv, "Failed to get PipelineVersion: %v", pv)
	require.Equal(t, err1.(*util.UserError).ExternalStatusCode(), codes.NotFound)
}

func TestGetK8sPipelineVersion(t *testing.T) {
	podNamespace := viper.Get("POD_NAMESPACE")
	viper.Set("POD_NAMESPACE", "Test")
	defer viper.Set("POD_NAMESPACE", podNamespace)

	store := NewPipelineStoreKubernetes(getClient())

	pipelineVersion := &model.PipelineVersion{
		UUID:        DefaultFakePipelineIdTwo,
		Name:        "Test Pipeline Version",
		Description: model.LargeText("Test Pipeline Version Description"),
	}

	p, err := store.GetPipelineVersion(DefaultFakePipelineIdTwo)
	require.Nil(t, err, "Failed to get Pipeline: %v", err)
	require.Equal(t, p.UUID, pipelineVersion.UUID)
}

func TestGetDefaultK8sPipelineVersion(t *testing.T) {
	podNamespace := viper.Get("POD_NAMESPACE")
	viper.Set("POD_NAMESPACE", "Test")
	defer viper.Set("POD_NAMESPACE", podNamespace)

	store := NewPipelineStoreKubernetes(getClient())

	pipelineVersion, err := store.GetDefaultPipelineVersion(DefaultFakePipelineIdTwo)
	require.Nil(t, err, "Failed to get latest pipeline version: %v", err)
	require.Equal(t, "test-pipeline-version-3", pipelineVersion.Name)
}

const defaultVersionPipelineID = "b0a1c2d3-0000-4000-8000-00000000000a"

// newPinnedPipelineVersion keeps objectName independent of versionName, as a CR authored outside the REST API may.
func newPinnedPipelineVersion(objectName, versionName string, created metav1.Time) *v2beta1.PipelineVersion {
	return &v2beta1.PipelineVersion{
		ObjectMeta: metav1.ObjectMeta{
			UID:               types.UID("uid-" + objectName),
			Name:              objectName,
			Namespace:         "Test",
			CreationTimestamp: created,
			Labels:            map[string]string{"pipelines.kubeflow.org/pipeline-id": defaultVersionPipelineID},
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: v2beta1.GroupVersion.String(),
				Kind:       "Pipeline",
				Name:       "pinned-pipeline",
				UID:        defaultVersionPipelineID,
			}},
		},
		Spec: v2beta1.PipelineVersionSpec{
			VersionName:  versionName,
			PipelineName: "pinned-pipeline",
			PipelineSpec: getBasicPipelineSpec(),
		},
	}
}

// newDefaultVersionFixture builds a pipeline pinned to defaultVersionName, owning an older "pinned"
// version and a newer "rolling" one, plus any extra versions.
func newDefaultVersionFixture(
	t *testing.T, defaultVersionName string, extraVersions ...client.Object,
) (client.Client, string) {
	t.Helper()

	scheme := runtime.NewScheme()
	require.NoError(t, v2beta1.AddToScheme(scheme))

	objects := []client.Object{
		&v2beta1.Pipeline{
			ObjectMeta: metav1.ObjectMeta{
				UID: defaultVersionPipelineID, Name: "pinned-pipeline", Namespace: "Test",
			},
			Spec: v2beta1.PipelineSpec{DefaultVersionName: defaultVersionName},
		},
		newPinnedPipelineVersion("gitops-authored-a", "pinned", metav1.Unix(1700000000, 0)),
		newPinnedPipelineVersion("gitops-authored-b", "rolling", metav1.Unix(1800000000, 0)),
	}

	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(append(objects, extraVersions...)...).
		Build()

	return k8sClient, defaultVersionPipelineID
}

func TestGetDefaultK8sPipelineVersion_PinnedDefaultWinsOverNewer(t *testing.T) {
	podNamespace := viper.Get("POD_NAMESPACE")
	viper.Set("POD_NAMESPACE", "Test")
	defer viper.Set("POD_NAMESPACE", podNamespace)

	k8sClient, pipelineID := newDefaultVersionFixture(t, "pinned")
	store := NewPipelineStoreKubernetes(k8sClient, k8sClient)

	version, err := store.GetDefaultPipelineVersion(pipelineID)
	require.NoError(t, err)
	assert.Equal(t, "pinned", version.Name)
}

func TestGetDefaultK8sPipelineVersion_NoDefaultUsesNewest(t *testing.T) {
	podNamespace := viper.Get("POD_NAMESPACE")
	viper.Set("POD_NAMESPACE", "Test")
	defer viper.Set("POD_NAMESPACE", podNamespace)

	k8sClient, pipelineID := newDefaultVersionFixture(t, "")
	store := NewPipelineStoreKubernetes(k8sClient, k8sClient)

	version, err := store.GetDefaultPipelineVersion(pipelineID)
	require.NoError(t, err)
	assert.Equal(t, "rolling", version.Name)
}

func TestGetDefaultK8sPipelineVersion_DanglingDefaultErrors(t *testing.T) {
	podNamespace := viper.Get("POD_NAMESPACE")
	viper.Set("POD_NAMESPACE", "Test")
	defer viper.Set("POD_NAMESPACE", podNamespace)

	k8sClient, pipelineID := newDefaultVersionFixture(t, "deleted-version")
	store := NewPipelineStoreKubernetes(k8sClient, k8sClient)

	_, err := store.GetDefaultPipelineVersion(pipelineID)
	require.ErrorIs(t, err, errDefaultVersionUnresolved)

	var userError *util.UserError
	require.ErrorAs(t, err, &userError)
	assert.Equal(t, codes.FailedPrecondition, userError.ExternalStatusCode())
	assert.Equal(t,
		`no pipeline version is named "deleted-version"; set spec.defaultVersionName to an existing version`,
		userError.ExternalMessage())
}

func TestGetDefaultK8sPipelineVersion_PinnedDefaultFallsBackToObjectName(t *testing.T) {
	podNamespace := viper.Get("POD_NAMESPACE")
	viper.Set("POD_NAMESPACE", "Test")
	defer viper.Set("POD_NAMESPACE", podNamespace)

	legacy := newPinnedPipelineVersion("legacy-version", "", metav1.Unix(1600000000, 0))
	k8sClient, pipelineID := newDefaultVersionFixture(t, "legacy-version", legacy)
	store := NewPipelineStoreKubernetes(k8sClient, k8sClient)

	version, err := store.GetDefaultPipelineVersion(pipelineID)
	require.NoError(t, err)
	assert.Equal(t, "legacy-version", version.Name)
}

func TestGetDefaultK8sPipelineVersion_AmbiguousDefaultErrors(t *testing.T) {
	podNamespace := viper.Get("POD_NAMESPACE")
	viper.Set("POD_NAMESPACE", "Test")
	defer viper.Set("POD_NAMESPACE", podNamespace)

	duplicate := newPinnedPipelineVersion("gitops-authored-c", "pinned", metav1.Unix(1900000000, 0))
	k8sClient, pipelineID := newDefaultVersionFixture(t, "pinned", duplicate)
	store := NewPipelineStoreKubernetes(k8sClient, k8sClient)

	_, err := store.GetDefaultPipelineVersion(pipelineID)
	require.ErrorIs(t, err, errDefaultVersionUnresolved)

	var userError *util.UserError
	require.ErrorAs(t, err, &userError)
	assert.Equal(t, codes.FailedPrecondition, userError.ExternalStatusCode())
	assert.Equal(t,
		`2 pipeline versions are named "pinned"; spec.defaultVersionName must match exactly one`,
		userError.ExternalMessage())
}

func TestGetDefaultK8sPipelineVersion_PinDoesNotMatchObjectName(t *testing.T) {
	podNamespace := viper.Get("POD_NAMESPACE")
	viper.Set("POD_NAMESPACE", "Test")
	defer viper.Set("POD_NAMESPACE", podNamespace)

	// gitops-authored-a is an object name; its version is named "pinned".
	k8sClient, pipelineID := newDefaultVersionFixture(t, "gitops-authored-a")
	store := NewPipelineStoreKubernetes(k8sClient, k8sClient)

	_, err := store.GetDefaultPipelineVersion(pipelineID)
	require.ErrorIs(t, err, errDefaultVersionUnresolved)
}

func TestGetDefaultK8sPipelineVersion_VersionNameBeatsAnotherObjectName(t *testing.T) {
	podNamespace := viper.Get("POD_NAMESPACE")
	viper.Set("POD_NAMESPACE", "Test")
	defer viper.Set("POD_NAMESPACE", podNamespace)

	decoy := newPinnedPipelineVersion("collision", "not-the-pin", metav1.Unix(1610000000, 0))
	target := newPinnedPipelineVersion("collision-owner", "collision", metav1.Unix(1620000000, 0))
	k8sClient, pipelineID := newDefaultVersionFixture(t, "collision", decoy, target)
	store := NewPipelineStoreKubernetes(k8sClient, k8sClient)

	version, err := store.GetDefaultPipelineVersion(pipelineID)
	require.NoError(t, err)
	assert.Equal(t, "collision", version.Name)
}

func newDefaultVersionFixtureWithPin(
	t *testing.T, pin string, extra ...client.Object,
) client.Client {
	t.Helper()

	k8sClient, _ := newDefaultVersionFixture(t, pin, extra...)

	return k8sClient
}

// The pipeline-id label is user-mutable, so ownerReferences decide which versions are candidates.
func TestGetDefaultK8sPipelineVersion_IgnoresLabelWithoutOwnership(t *testing.T) {
	podNamespace := viper.Get("POD_NAMESPACE")
	viper.Set("POD_NAMESPACE", "Test")
	defer viper.Set("POD_NAMESPACE", podNamespace)

	// Carries this pipeline's label but is owned by another pipeline.
	foreign := newPinnedPipelineVersion("foreign", "borrowed", metav1.Unix(1900000000, 0))
	foreign.OwnerReferences[0].UID = "f0f0f0f0-0000-4000-8000-00000000000f"

	k8sClient, pipelineID := newDefaultVersionFixture(t, "", foreign)
	store := NewPipelineStoreKubernetes(k8sClient, k8sClient)

	// It is the newest, but must not be selected as the default.
	version, err := store.GetDefaultPipelineVersion(pipelineID)
	require.NoError(t, err)
	assert.Equal(t, "rolling", version.Name)

	// Nor may it be pinned.
	_, err = NewPipelineStoreKubernetes(
		newDefaultVersionFixtureWithPin(t, "borrowed", foreign), k8sClient,
	).GetDefaultPipelineVersion(pipelineID)
	require.ErrorIs(t, err, errDefaultVersionUnresolved)
}

// Candidates are scoped to the pipeline's namespace; in multi-user mode the list is cluster-wide.
func TestGetDefaultK8sPipelineVersion_IgnoresOtherNamespace(t *testing.T) {
	podNamespace := viper.Get("POD_NAMESPACE")
	viper.Set("POD_NAMESPACE", "Test")
	defer viper.Set("POD_NAMESPACE", podNamespace)

	multiUser := viper.Get("MULTIUSER")
	viper.Set("MULTIUSER", "true")
	defer viper.Set("MULTIUSER", multiUser)

	// Correct label and owner UID, but a different namespace.
	otherNs := newPinnedPipelineVersion("other-ns", "tenant-b", metav1.Unix(1900000000, 0))
	otherNs.Namespace = "other"

	k8sClient, pipelineID := newDefaultVersionFixture(t, "", otherNs)
	store := NewPipelineStoreKubernetes(k8sClient, k8sClient)

	version, err := store.GetDefaultPipelineVersion(pipelineID)
	require.NoError(t, err)
	assert.Equal(t, "rolling", version.Name)
}

// The inverse of IgnoresLabelWithoutOwnership: an owned version must stay visible even if the
// mutable label is missing or stale.
func TestGetDefaultK8sPipelineVersion_FindsOwnedVersionWithStaleLabel(t *testing.T) {
	podNamespace := viper.Get("POD_NAMESPACE")
	viper.Set("POD_NAMESPACE", "Test")
	defer viper.Set("POD_NAMESPACE", podNamespace)

	// Correct ownerReference, but the label points at a different pipeline.
	mislabeled := newPinnedPipelineVersion("mislabeled", "recovered", metav1.Unix(1900000000, 0))
	mislabeled.Labels["pipelines.kubeflow.org/pipeline-id"] = "f0f0f0f0-0000-4000-8000-00000000000f"

	k8sClient, pipelineID := newDefaultVersionFixture(t, "", mislabeled)

	// It is the newest owned version, so it is the default.
	version, err := NewPipelineStoreKubernetes(k8sClient, k8sClient).GetDefaultPipelineVersion(pipelineID)
	require.NoError(t, err)
	assert.Equal(t, "recovered", version.Name)

	// And it can be pinned by name.
	version, err = NewPipelineStoreKubernetes(
		newDefaultVersionFixtureWithPin(t, "recovered", mislabeled), k8sClient,
	).GetDefaultPipelineVersion(pipelineID)
	require.NoError(t, err)
	assert.Equal(t, "recovered", version.Name)
}

func TestGetDefaultK8sPipelineVersion_NoVersions(t *testing.T) {
	podNamespace := viper.Get("POD_NAMESPACE")
	viper.Set("POD_NAMESPACE", "Test")
	defer viper.Set("POD_NAMESPACE", podNamespace)

	tests := []struct {
		pin      string
		wantCode codes.Code
	}{
		{pin: "", wantCode: codes.NotFound},
		{pin: "pinned", wantCode: codes.FailedPrecondition},
	}

	for _, test := range tests {
		t.Run("pin="+test.pin, func(t *testing.T) {
			scheme := runtime.NewScheme()
			require.NoError(t, v2beta1.AddToScheme(scheme))

			k8sClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(&v2beta1.Pipeline{
				ObjectMeta: metav1.ObjectMeta{
					UID: defaultVersionPipelineID, Name: "pinned-pipeline", Namespace: "Test",
				},
				Spec: v2beta1.PipelineSpec{DefaultVersionName: test.pin},
			}).Build()

			_, err := NewPipelineStoreKubernetes(k8sClient, k8sClient).GetDefaultPipelineVersion(defaultVersionPipelineID)

			var userError *util.UserError
			require.ErrorAs(t, err, &userError)
			assert.Equal(t, test.wantCode, userError.ExternalStatusCode())
		})
	}
}

func TestGetK8sPipelineVersion_NotFoundError(t *testing.T) {
	podNamespace := viper.Get("POD_NAMESPACE")
	viper.Set("POD_NAMESPACE", "Test")
	defer viper.Set("POD_NAMESPACE", podNamespace)

	store := NewPipelineStoreKubernetes(getClient())

	_, err := store.GetDefaultPipelineVersion(DefaultFakePipelineIdFive)
	require.NotNil(t, err)
	assert.Equal(t, err.(*util.UserError).ExternalStatusCode(), codes.NotFound)
}

func TestListK8sPipelineVersions_Pagination(t *testing.T) {
	podNamespace := viper.Get("POD_NAMESPACE")
	viper.Set("POD_NAMESPACE", "Test")
	defer viper.Set("POD_NAMESPACE", podNamespace)

	store := NewPipelineStoreKubernetes(getClient())

	pipelineVersion1 := &model.PipelineVersion{
		Name:         "test-pipeline-version-1",
		PipelineId:   DefaultFakePipelineIdTwo,
		PipelineSpec: model.LargeText(getBasicPipelineSpecYAML()),
	}

	pipelineVersion2 := &model.PipelineVersion{
		Name:         "test-pipeline-version-2",
		PipelineId:   DefaultFakePipelineIdTwo,
		PipelineSpec: model.LargeText(getBasicPipelineSpecYAML()),
	}

	_, err := store.CreatePipelineVersion(pipelineVersion1)
	require.Nil(t, err, "Failed to create PipelineVersion: %v", err)
	_, err = store.CreatePipelineVersion(pipelineVersion2)
	require.Nil(t, err, "Failed to create PipelineVersion: %v", err)

	options, err := list.NewOptions(&model.PipelineVersion{}, 1, "", nil)
	require.Nil(t, err, "Failed to create list options")

	pipelineVersions, _, npt, err := store.ListPipelineVersions(DefaultFakePipelineIdTwo, options)
	require.Nil(t, err, "Failed to list pipeline versions: %v", err)
	require.Equalf(t, len(pipelineVersions), 1, "List size should not be zero")
	require.NotNil(t, npt, "Npt should not be nil")

	options, err = list.NewOptionsFromToken(npt, 1)
	require.Nil(t, err, "Failed to create list options")
	pipelineVersions, _, _, err = store.ListPipelineVersions(DefaultFakePipelineIdTwo, options)
	require.Nil(t, err, "Failed to list pipeline versions: %v", err)
	require.Equalf(t, len(pipelineVersions), 1, "List size should not be zero")
	require.Equalf(t, pipelineVersions[0].Name, "test-pipeline-version-1", "Pagination did not work as expected")
}

func TestListK8sPipelineVersions_Pagination_Descend(t *testing.T) {
	podNamespace := viper.Get("POD_NAMESPACE")
	viper.Set("POD_NAMESPACE", "Test")
	defer viper.Set("POD_NAMESPACE", podNamespace)

	store := NewPipelineStoreKubernetes(getClient())

	pipelineVersion1 := &model.PipelineVersion{
		Name:         "test-pipeline-version-1",
		PipelineId:   DefaultFakePipelineIdTwo,
		PipelineSpec: model.LargeText(getBasicPipelineSpecYAML()),
	}

	pipelineVersion2 := &model.PipelineVersion{
		Name:         "test-pipeline-version-2",
		PipelineId:   DefaultFakePipelineIdTwo,
		PipelineSpec: model.LargeText(getBasicPipelineSpecYAML()),
	}

	_, err := store.CreatePipelineVersion(pipelineVersion1)
	require.Nil(t, err, "Failed to create PipelineVersion: %v", err)
	_, err = store.CreatePipelineVersion(pipelineVersion2)
	require.Nil(t, err, "Failed to create PipelineVersion: %v", err)

	options, err := list.NewOptions(&model.PipelineVersion{}, 1, "name desc", nil)

	pipelineVersions, _, _, err1 := store.ListPipelineVersions(DefaultFakePipelineIdTwo, options)
	require.Nil(t, err1, "Failed to list pipeline versions: %v", err)
	require.Equalf(t, len(pipelineVersions), 1, "List size should not be zero")
	require.Equalf(t, pipelineVersions[0].Name, "test-pipeline-version-3", "Pagination did not work as expected")
}

func TestListK8sPipelineVersions_Pagination_LessThanPageSize(t *testing.T) {
	podNamespace := viper.Get("POD_NAMESPACE")
	viper.Set("POD_NAMESPACE", "Test")
	defer viper.Set("POD_NAMESPACE", podNamespace)

	store := NewPipelineStoreKubernetes(getClient())

	options, err1 := list.NewOptions(&model.Pipeline{}, 10, "", nil)
	require.Nil(t, err1, "Failed to create list options: %v")

	pipelines, pageSize, _, err := store.ListPipelineVersions(DefaultFakePipelineIdTwo, options)
	require.Nil(t, err, "Failed to list pipeline Versions: %v")
	require.Equalf(t, pageSize, 1, "Page size should be 1")
	require.Equalf(t, len(pipelines), 1, "List size should be 1")
}

func TestGetK8sPipelineVersionByName(t *testing.T) {
	podNamespace := viper.Get("POD_NAMESPACE")
	viper.Set("POD_NAMESPACE", "Test")
	defer viper.Set("POD_NAMESPACE", podNamespace)

	store := NewPipelineStoreKubernetes(getClient())

	// Legacy-style CR (bare metadata.name) — should be found via bare-name fallback
	pipelineVersion, err := store.GetPipelineVersionByName(DefaultFakePipelineIdTwo, "test-pipeline-version-3")
	require.Nil(t, err, "Failed to get Pipeline: %v", err)
	require.Equalf(t, pipelineVersion.Name, "test-pipeline-version-3", pipelineVersion.Name)
}

func TestListK8sPipelineVersions_WithFilter(t *testing.T) {
	podNamespace := viper.Get("POD_NAMESPACE")
	viper.Set("POD_NAMESPACE", "Test")
	defer viper.Set("POD_NAMESPACE", podNamespace)

	store := NewPipelineStoreKubernetes(getClient())

	filterProto := &api.Filter{
		Predicates: []*api.Predicate{
			{
				Key:       "name",
				Operation: api.Predicate_IS_SUBSTRING,
				Value:     &api.Predicate_StringValue{StringValue: "test"},
			},
		},
	}

	newFilter, err := filter.New(filterProto)
	options, err1 := list.NewOptions(&model.PipelineVersion{}, 1, "", newFilter)
	require.Nil(t, err1, "Failed to list pipeline versions: %v", err)

	pipelineVersions, _, _, err2 := store.ListPipelineVersions(DefaultFakePipelineIdTwo, options)
	require.Nil(t, err2, "Failed to list pipeline versions: %v", err)
	require.Equalf(t, len(pipelineVersions), 1, "List size should not be zero")
}

func TestCreatePipelineAndPipelineVersion(t *testing.T) {
	podNamespace := viper.Get("POD_NAMESPACE")
	viper.Set("POD_NAMESPACE", "Test")
	defer viper.Set("POD_NAMESPACE", podNamespace)

	store := NewPipelineStoreKubernetes(getClient())

	k8sPipeline := &model.Pipeline{
		Name: "test-pipeline",
	}
	k8sPipelineVersion := &model.PipelineVersion{
		Name:         "test-pipeline-version",
		PipelineSpec: model.LargeText(getBasicPipelineSpecYAML()),
	}

	_, _, err := store.CreatePipelineAndPipelineVersion(k8sPipeline, k8sPipelineVersion)
	require.Nil(t, err, "Failed to create Pipeline: %v", err)
}

func TestCreateK8sPipeline_InvalidName(t *testing.T) {
	podNamespace := viper.Get("POD_NAMESPACE")
	viper.Set("POD_NAMESPACE", "Test")
	defer viper.Set("POD_NAMESPACE", podNamespace)

	store := NewPipelineStoreKubernetes(getClient())

	pipeline := &model.Pipeline{
		Name:        "My-Pipeline",
		Description: model.LargeText("Invalid name with uppercase"),
		Namespace:   "Test",
	}

	_, err := store.CreatePipeline(pipeline)
	require.NotNil(t, err, "Expected error for invalid pipeline name")
	assert.Contains(t, err.Error(), "Invalid pipeline name")
	assert.Contains(t, err.Error(), "display_name")
}

func TestCreateK8sPipelineVersion_InvalidName(t *testing.T) {
	podNamespace := viper.Get("POD_NAMESPACE")
	viper.Set("POD_NAMESPACE", "Test")
	defer viper.Set("POD_NAMESPACE", podNamespace)

	store := NewPipelineStoreKubernetes(getClient())

	k8sPipeline := &model.Pipeline{
		Name: "test-pipeline",
	}
	k8sPipelineVersion := &model.PipelineVersion{
		Name:         "My-Pipeline-Version",
		PipelineSpec: model.LargeText(getBasicPipelineSpecYAML()),
	}

	_, _, err := store.CreatePipelineAndPipelineVersion(k8sPipeline, k8sPipelineVersion)
	require.NotNil(t, err, "Expected error for invalid pipeline version name")
	assert.Contains(t, err.Error(), "Invalid pipeline version name")
	assert.Contains(t, err.Error(), "display_name")
}

func TestCreateK8sPipelineVersion_InvalidName_Standalone(t *testing.T) {
	podNamespace := viper.Get("POD_NAMESPACE")
	viper.Set("POD_NAMESPACE", "Test")
	defer viper.Set("POD_NAMESPACE", podNamespace)

	store := NewPipelineStoreKubernetes(getClient())

	k8sPipeline := &model.Pipeline{
		Name: "test-pipeline",
	}
	k8sPipelineVersion := &model.PipelineVersion{
		Name:         "test-pipeline",
		PipelineSpec: model.LargeText(getBasicPipelineSpecYAML()),
	}
	pipeline, _, err := store.CreatePipelineAndPipelineVersion(k8sPipeline, k8sPipelineVersion)
	require.NoError(t, err)

	invalidVersion := &model.PipelineVersion{
		Name:         "Invalid-Version-Name",
		PipelineId:   pipeline.UUID,
		PipelineSpec: model.LargeText(getBasicPipelineSpecYAML()),
	}
	_, err = store.CreatePipelineVersion(invalidVersion)
	require.NotNil(t, err, "Expected error for invalid pipeline version name")
	assert.Contains(t, err.Error(), "Invalid pipeline version name")
	assert.Contains(t, err.Error(), "display_name")
	assert.Equal(t, codes.InvalidArgument, err.(*util.UserError).ExternalStatusCode())
}

// getBasicPipelineSpec returns a basic PipelineSpec for testing purposes
func getBasicPipelineSpec() v2beta1.IRSpec {
	return v2beta1.IRSpec{
		Value: map[string]interface{}{
			"pipelineInfo": map[string]interface{}{
				"name":        "test-pipeline",
				"displayName": "Test Pipeline",
			},
			"root": map[string]interface{}{
				"dag": map[string]interface{}{
					"tasks": map[string]interface{}{},
				},
			},
			"schemaVersion": "2.1.0",
			"sdkVersion":    "kfp-2.13.0",
		},
	}
}

// getBasicPipelineSpecYAML returns a basic PipelineSpec as YAML string for model.PipelineVersion objects
func getBasicPipelineSpecYAML() string {
	return `pipelineInfo:
  name: test-pipeline
  displayName: Test Pipeline
root:
  dag:
    tasks: {}
schemaVersion: "2.1.0"
sdkVersion: kfp-2.13.0`
}

func TestGetPipelineVersionByName_CompositeNameLookup(t *testing.T) {
	podNamespace := viper.Get("POD_NAMESPACE")
	viper.Set("POD_NAMESPACE", "Test")
	defer viper.Set("POD_NAMESPACE", podNamespace)

	store := NewPipelineStoreKubernetes(getClientWithTwoPipelines())

	_, err := store.CreatePipelineVersion(&model.PipelineVersion{
		Name:         "v1.0",
		PipelineId:   DefaultFakePipelineIdThree,
		PipelineSpec: model.LargeText(getBasicPipelineSpecYAML()),
	})
	require.NoError(t, err)

	_, err = store.CreatePipelineVersion(&model.PipelineVersion{
		Name:         "v1.0",
		PipelineId:   DefaultFakePipelineIdFour,
		PipelineSpec: model.LargeText(getBasicPipelineSpecYAML()),
	})
	require.NoError(t, err)

	// Look up each by pipeline ID + bare version name
	versionA, err := store.GetPipelineVersionByName(DefaultFakePipelineIdThree, "v1.0")
	require.NoError(t, err)
	assert.Equal(t, "v1.0", versionA.Name)
	assert.Equal(t, DefaultFakePipelineIdThree, versionA.PipelineId)

	versionB, err := store.GetPipelineVersionByName(DefaultFakePipelineIdFour, "v1.0")
	require.NoError(t, err)
	assert.Equal(t, "v1.0", versionB.Name)
	assert.Equal(t, DefaultFakePipelineIdFour, versionB.PipelineId)
}

func TestGetPipelineVersionByName_NotFound(t *testing.T) {
	podNamespace := viper.Get("POD_NAMESPACE")
	viper.Set("POD_NAMESPACE", "Test")
	defer viper.Set("POD_NAMESPACE", podNamespace)

	store := NewPipelineStoreKubernetes(getClient())

	_, err := store.GetPipelineVersionByName(DefaultFakePipelineIdTwo, "nonexistent")
	require.NotNil(t, err)
	assert.Equal(t, err.(*util.UserError).ExternalStatusCode(), codes.NotFound)
}

func TestIsNewerPipelineVersion(t *testing.T) {
	earlier := metav1.Unix(1700000000, 0)
	later := metav1.Unix(1700000001, 0)

	newerVersion := func(uid string, created metav1.Time) *v2beta1.PipelineVersion {
		return &v2beta1.PipelineVersion{
			ObjectMeta: metav1.ObjectMeta{UID: types.UID(uid), CreationTimestamp: created},
		}
	}

	tests := []struct {
		name     string
		a        *v2beta1.PipelineVersion
		b        *v2beta1.PipelineVersion
		expected bool
	}{
		{"later timestamp wins", newerVersion("aaa", later), newerVersion("zzz", earlier), true},
		{"earlier timestamp loses", newerVersion("zzz", earlier), newerVersion("aaa", later), false},
		{"tie broken by higher uid", newerVersion("bbb", earlier), newerVersion("aaa", earlier), true},
		{"tie broken against lower uid", newerVersion("aaa", earlier), newerVersion("bbb", earlier), false},
		{"identical is not newer", newerVersion("aaa", earlier), newerVersion("aaa", earlier), false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expected, isNewerPipelineVersion(test.a, test.b))
		})
	}
}

// Versions created within the same second tie on CreationTimestamp.
func TestGetDefaultK8sPipelineVersion_SameCreationSecondIsDeterministic(t *testing.T) {
	podNamespace := viper.Get("POD_NAMESPACE")
	viper.Set("POD_NAMESPACE", "Test")
	defer viper.Set("POD_NAMESPACE", podNamespace)

	scheme := runtime.NewScheme()
	require.NoError(t, v2beta1.AddToScheme(scheme))

	const pipelineID = "e1b2c3d4-0000-4000-8000-000000000001"
	sameSecond := metav1.Unix(1700000000, 0)

	pipeline := &v2beta1.Pipeline{
		ObjectMeta: metav1.ObjectMeta{UID: pipelineID, Name: "tie-pipeline", Namespace: "Test"},
	}

	version := func(name, uid string) *v2beta1.PipelineVersion {
		return &v2beta1.PipelineVersion{
			ObjectMeta: metav1.ObjectMeta{
				UID:               types.UID(uid),
				Name:              name,
				Namespace:         "Test",
				CreationTimestamp: sameSecond,
				Labels:            map[string]string{"pipelines.kubeflow.org/pipeline-id": pipelineID},
				OwnerReferences: []metav1.OwnerReference{{
					APIVersion: v2beta1.GroupVersion.String(),
					Kind:       "Pipeline",
					Name:       "tie-pipeline",
					UID:        pipelineID,
				}},
			},
			Spec: v2beta1.PipelineVersionSpec{
				VersionName:  name,
				PipelineName: "tie-pipeline",
				PipelineSpec: getBasicPipelineSpec(),
			},
		}
	}

	lowUID := version("tie-version-low", "00000000-0000-4000-8000-000000000001")
	highUID := version("tie-version-high", "ffffffff-0000-4000-8000-000000000002")

	// Seed both orderings; the same version must win regardless of list order.
	for _, ordering := range [][]client.Object{
		{lowUID, highUID},
		{highUID, lowUID},
	} {
		k8sClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(append([]client.Object{pipeline}, ordering...)...).
			Build()

		store := NewPipelineStoreKubernetes(k8sClient, k8sClient)

		latest, err := store.GetDefaultPipelineVersion(pipelineID)
		require.NoError(t, err)
		assert.Equal(t, "tie-version-high", latest.Name)
	}
}

func getClient() (client.Client, client.Client) {
	scheme := runtime.NewScheme()
	err := v2beta1.AddToScheme(scheme)
	if err != nil {
		glog.Fatalf("Failed to add to scheme: %v", err)
	}

	pipeline3 := &v2beta1.Pipeline{
		ObjectMeta: metav1.ObjectMeta{
			UID:       DefaultFakePipelineIdTwo,
			Name:      "test-pipeline-3",
			Namespace: "Test",
		},
		Spec: v2beta1.PipelineSpec{
			Description: "Test Pipeline 3 Description",
		},
	}

	pipelineVersion := &v2beta1.PipelineVersion{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pipeline-version",
			Namespace: "Test",
		},
		Spec: v2beta1.PipelineVersionSpec{
			PipelineSpec: getBasicPipelineSpec(),
		},
	}

	pipelineVersion1 := &v2beta1.PipelineVersion{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pipeline-version-1",
			Namespace: "Test",
			Labels: map[string]string{
				"pipelines.kubeflow.org/pipeline-id": DefaultFakePipelineId,
			},
		},
		Spec: v2beta1.PipelineVersionSpec{
			PipelineSpec: getBasicPipelineSpec(),
		},
	}

	pipelineVersion2 := &v2beta1.PipelineVersion{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pipeline-version-2",
			Namespace: "Test",
		},
		Spec: v2beta1.PipelineVersionSpec{
			PipelineSpec: getBasicPipelineSpec(),
		},
	}

	pipelineVersion3 := &v2beta1.PipelineVersion{
		ObjectMeta: metav1.ObjectMeta{
			UID:       DefaultFakePipelineIdTwo,
			Name:      "test-pipeline-version-3",
			Namespace: "Test",
			Labels: map[string]string{
				"pipelines.kubeflow.org/pipeline-id": DefaultFakePipelineIdTwo,
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: v2beta1.GroupVersion.String(),
					Kind:       "Pipeline",
					UID:        DefaultFakePipelineIdTwo,
					Name:       "test-pipeline-3",
				},
			},
		},
		Spec: v2beta1.PipelineVersionSpec{
			Description:  "Test Pipeline Version 1 Description",
			PipelineName: "test-pipeline-3",
			PipelineSpec: getBasicPipelineSpec(),
		},
	}

	// The API server assigns a UID on create; the fake client does not.
	generatedUIDs := 0
	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(pipelineVersion, pipelineVersion1, pipelineVersion2, pipelineVersion3).
		WithObjects(pipeline3, pipelineVersion3).
		WithInterceptorFuncs(interceptor.Funcs{
			Create: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.CreateOption) error {
				if obj.GetUID() == "" {
					generatedUIDs++
					obj.SetUID(types.UID(fmt.Sprintf("generated-uid-%d", generatedUIDs)))
				}
				return c.Create(ctx, obj, opts...)
			},
		}).
		Build()

	return k8sClient, k8sClient
}

// newVersionFixture builds an unpinned pipeline owning the given versions.
func newVersionFixture(t *testing.T, versions ...client.Object) (client.Client, string) {
	t.Helper()

	scheme := runtime.NewScheme()
	require.NoError(t, v2beta1.AddToScheme(scheme))

	objects := []client.Object{&v2beta1.Pipeline{
		ObjectMeta: metav1.ObjectMeta{
			UID: defaultVersionPipelineID, Name: "pinned-pipeline", Namespace: "Test",
		},
	}}

	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(append(objects, versions...)...).
		Build()

	return k8sClient, defaultVersionPipelineID
}

// newUnparseableVersion returns an owned version whose manifest ToModel cannot parse, so any code
// path that converts it fails loudly.
func newUnparseableVersion(objectName string) *v2beta1.PipelineVersion {
	version := newPinnedPipelineVersion(objectName, objectName, metav1.Unix(1700000000, 0))
	version.Spec.PipelineSpec = v2beta1.IRSpec{Value: map[string]interface{}{"not": "a pipeline spec"}}

	return version
}

// The existence check must not parse manifests: callers only need to know a version exists.
func TestGetAnyK8sPipelineVersionId_DoesNotConvertManifests(t *testing.T) {
	podNamespace := viper.Get("POD_NAMESPACE")
	viper.Set("POD_NAMESPACE", "Test")
	defer viper.Set("POD_NAMESPACE", podNamespace)

	k8sClient, pipelineID := newVersionFixture(t,
		newUnparseableVersion("a"), newUnparseableVersion("b"),
		newUnparseableVersion("c"), newUnparseableVersion("d"),
	)
	store := NewPipelineStoreKubernetes(k8sClient, k8sClient)

	pipelineVersionID, err := store.GetAnyPipelineVersionID(pipelineID)
	require.NoError(t, err)
	assert.NotEmpty(t, pipelineVersionID)
}

// Documents why the existence check does not list: ListPipelineVersions converts every candidate
// before paginating, so a page size of 1 does not bound the work. If the list path is ever made to
// paginate first, this test becomes obsolete rather than wrong.
func TestListK8sPipelineVersions_ConvertsEveryManifest(t *testing.T) {
	podNamespace := viper.Get("POD_NAMESPACE")
	viper.Set("POD_NAMESPACE", "Test")
	defer viper.Set("POD_NAMESPACE", podNamespace)

	k8sClient, pipelineID := newVersionFixture(t, newUnparseableVersion("a"), newUnparseableVersion("b"))
	store := NewPipelineStoreKubernetes(k8sClient, k8sClient)

	opts, err := list.NewOptions(&model.PipelineVersion{}, 1, "id", nil)
	require.NoError(t, err)
	_, _, _, err = store.ListPipelineVersions(pipelineID, opts, nil)
	require.Error(t, err)
}

func TestGetAnyK8sPipelineVersionId_NoVersions(t *testing.T) {
	podNamespace := viper.Get("POD_NAMESPACE")
	viper.Set("POD_NAMESPACE", "Test")
	defer viper.Set("POD_NAMESPACE", podNamespace)

	k8sClient, pipelineID := newVersionFixture(t)
	store := NewPipelineStoreKubernetes(k8sClient, k8sClient)

	pipelineVersionID, err := store.GetAnyPipelineVersionID(pipelineID)
	require.NoError(t, err)
	assert.Empty(t, pipelineVersionID)
}

// A foreign CR carrying this pipeline's mutable label must not block its deletion.
func TestGetAnyK8sPipelineVersionId_IgnoresLabelWithoutOwnership(t *testing.T) {
	podNamespace := viper.Get("POD_NAMESPACE")
	viper.Set("POD_NAMESPACE", "Test")
	defer viper.Set("POD_NAMESPACE", podNamespace)

	foreign := newPinnedPipelineVersion("foreign", "borrowed", metav1.Unix(1900000000, 0))
	foreign.OwnerReferences[0].UID = "f0f0f0f0-0000-4000-8000-00000000000f"

	k8sClient, pipelineID := newVersionFixture(t, foreign)
	store := NewPipelineStoreKubernetes(k8sClient, k8sClient)

	pipelineVersionID, err := store.GetAnyPipelineVersionID(pipelineID)
	require.NoError(t, err)
	assert.Empty(t, pipelineVersionID)
}

// The inverse: an owned version with a stale label still blocks deletion.
func TestGetAnyK8sPipelineVersionId_FindsOwnedVersionWithStaleLabel(t *testing.T) {
	podNamespace := viper.Get("POD_NAMESPACE")
	viper.Set("POD_NAMESPACE", "Test")
	defer viper.Set("POD_NAMESPACE", podNamespace)

	stale := newPinnedPipelineVersion("stale-label", "owned", metav1.Unix(1900000000, 0))
	stale.Labels = map[string]string{"pipelines.kubeflow.org/pipeline-id": "stale"}

	k8sClient, pipelineID := newVersionFixture(t, stale)
	store := NewPipelineStoreKubernetes(k8sClient, k8sClient)

	pipelineVersionID, err := store.GetAnyPipelineVersionID(pipelineID)
	require.NoError(t, err)
	assert.Equal(t, string(stale.UID), pipelineVersionID)
}

func paginationTestPipelines(n int, sameTimestamp bool) []client.Object {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	objs := make([]client.Object, 0, n)
	for i := 1; i <= n; i++ {
		created := base
		if !sameTimestamp {
			created = base.Add(time.Duration(i) * time.Minute)
		}
		objs = append(objs, &v2beta1.Pipeline{
			ObjectMeta: metav1.ObjectMeta{
				Name:              fmt.Sprintf("p%d", i),
				Namespace:         "Test",
				UID:               types.UID(fmt.Sprintf("uid-%d", i)),
				CreationTimestamp: metav1.NewTime(created),
			},
		})
	}
	return objs
}

func paginationTestScheme(t *testing.T) *runtime.Scheme {
	scheme := runtime.NewScheme()
	require.NoError(t, v2beta1.AddToScheme(scheme))
	return scheme
}

// listAllPages pages until the token is empty, giving up after maxCalls.
func listAllPages(t *testing.T, store *PipelineStoreKubernetes, sortBy string, pageSize, maxCalls int) (names []string, calls int) {
	options, err := list.NewOptions(&model.Pipeline{}, pageSize, sortBy, nil)
	require.NoError(t, err)
	for calls < maxCalls {
		calls++
		page, _, npt, err := store.ListPipelines(&model.FilterContext{}, options)
		require.NoError(t, err)
		for _, p := range page {
			names = append(names, p.Name)
		}
		if npt == "" {
			return names, calls
		}
		options, err = list.NewOptionsFromToken(npt, pageSize)
		require.NoError(t, err)
	}
	return names, calls
}

// The pipeline a page token points at is deleted before the next page is requested.
func TestListK8sPipelines_Pagination_AnchorDeleted(t *testing.T) {
	k8sClient := fake.NewClientBuilder().WithScheme(paginationTestScheme(t)).WithObjects(paginationTestPipelines(5, false)...).Build()
	store := NewPipelineStoreKubernetes(k8sClient, k8sClient)

	options, err := list.NewOptions(&model.Pipeline{}, 2, "name asc", nil)
	require.NoError(t, err)
	page1, _, npt, err := store.ListPipelines(&model.FilterContext{}, options)
	require.NoError(t, err)
	require.Equal(t, []string{"p1", "p2"}, []string{page1[0].Name, page1[1].Name})

	// p3 is the first row of page 2, so it is what the token points at.
	require.NoError(t, k8sClient.Delete(context.Background(), &v2beta1.Pipeline{
		ObjectMeta: metav1.ObjectMeta{Name: "p3", Namespace: "Test"},
	}))

	options, err = list.NewOptionsFromToken(npt, 2)
	require.NoError(t, err)
	page2, _, _, err := store.ListPipelines(&model.FilterContext{}, options)
	require.NoError(t, err)
	got := []string{}
	for _, p := range page2 {
		got = append(got, p.Name)
	}
	require.Equal(t, []string{"p4", "p5"}, got, "page 2 after its anchor was deleted")
}

// Pipelines share a creation timestamp and the cache returns them in a
// different order on each List, as a Go map iteration does.
func TestListK8sPipelines_Pagination_TiedSortValues(t *testing.T) {
	calls := 0
	k8sClient := fake.NewClientBuilder().WithScheme(paginationTestScheme(t)).WithObjects(paginationTestPipelines(6, true)...).
		WithInterceptorFuncs(interceptor.Funcs{
			List: func(ctx context.Context, c client.WithWatch, l client.ObjectList, opts ...client.ListOption) error {
				if err := c.List(ctx, l, opts...); err != nil {
					return err
				}
				calls++
				if pl, ok := l.(*v2beta1.PipelineList); ok && calls%2 == 0 {
					slices.Reverse(pl.Items)
				}
				return nil
			},
		}).Build()
	store := NewPipelineStoreKubernetes(k8sClient, k8sClient)

	names, n := listAllPages(t, store, "", 2, 10)
	t.Logf("calls=%d names=%v", n, names)
	sorted := slices.Clone(names)
	slices.Sort(sorted)
	require.Equal(t, []string{"p1", "p2", "p3", "p4", "p5", "p6"}, sorted, "every pipeline exactly once")
}

func paginationTestPipelineVersions(n int, sameTimestamp bool) []client.Object {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	objs := make([]client.Object, 0, n)
	for i := 1; i <= n; i++ {
		created := base
		if !sameTimestamp {
			created = base.Add(time.Duration(i) * time.Minute)
		}
		objs = append(objs, &v2beta1.PipelineVersion{
			ObjectMeta: metav1.ObjectMeta{
				Name:              fmt.Sprintf("v%d", i),
				Namespace:         "Test",
				UID:               types.UID(fmt.Sprintf("vuid-%d", i)),
				CreationTimestamp: metav1.NewTime(created),
				Labels:            map[string]string{"pipelines.kubeflow.org/pipeline-id": "uid-1"},
			},
			Spec: v2beta1.PipelineVersionSpec{PipelineSpec: getBasicPipelineSpec()},
		})
	}
	return objs
}

func TestListK8sPipelineVersions_Pagination_AnchorDeleted(t *testing.T) {
	podNamespace := viper.Get("POD_NAMESPACE")
	viper.Set("POD_NAMESPACE", "Test")
	defer viper.Set("POD_NAMESPACE", podNamespace)
	k8sClient := fake.NewClientBuilder().WithScheme(paginationTestScheme(t)).WithObjects(paginationTestPipelineVersions(5, false)...).Build()
	store := NewPipelineStoreKubernetes(k8sClient, k8sClient)

	options, err := list.NewOptions(&model.PipelineVersion{}, 2, "name asc", nil)
	require.NoError(t, err)
	page1, _, npt, err := store.ListPipelineVersions("uid-1", options)
	require.NoError(t, err)
	require.Equal(t, []string{"v1", "v2"}, []string{page1[0].Name, page1[1].Name})

	require.NoError(t, k8sClient.Delete(context.Background(), &v2beta1.PipelineVersion{
		ObjectMeta: metav1.ObjectMeta{Name: "v3", Namespace: "Test"},
	}))

	options, err = list.NewOptionsFromToken(npt, 2)
	require.NoError(t, err)
	page2, _, _, err := store.ListPipelineVersions("uid-1", options)
	require.NoError(t, err)
	got := []string{}
	for _, v := range page2 {
		got = append(got, v.Name)
	}
	require.Equal(t, []string{"v4", "v5"}, got, "page 2 after its anchor was deleted")
}

func TestListK8sPipelineVersions_Pagination_TiedSortValues(t *testing.T) {
	podNamespace := viper.Get("POD_NAMESPACE")
	viper.Set("POD_NAMESPACE", "Test")
	defer viper.Set("POD_NAMESPACE", podNamespace)
	calls := 0
	k8sClient := fake.NewClientBuilder().WithScheme(paginationTestScheme(t)).WithObjects(paginationTestPipelineVersions(6, true)...).
		WithInterceptorFuncs(interceptor.Funcs{
			List: func(ctx context.Context, c client.WithWatch, l client.ObjectList, opts ...client.ListOption) error {
				if err := c.List(ctx, l, opts...); err != nil {
					return err
				}
				calls++
				if vl, ok := l.(*v2beta1.PipelineVersionList); ok && calls%2 == 0 {
					slices.Reverse(vl.Items)
				}
				return nil
			},
		}).Build()
	store := NewPipelineStoreKubernetes(k8sClient, k8sClient)

	options, err := list.NewOptions(&model.PipelineVersion{}, 2, "", nil)
	require.NoError(t, err)
	names := []string{}
	n := 0
	for n < 10 {
		n++
		page, _, npt, err := store.ListPipelineVersions("uid-1", options)
		require.NoError(t, err)
		for _, v := range page {
			names = append(names, v.Name)
		}
		if npt == "" {
			break
		}
		options, err = list.NewOptionsFromToken(npt, 2)
		require.NoError(t, err)
	}
	t.Logf("calls=%d names=%v", n, names)
	sorted := slices.Clone(names)
	slices.Sort(sorted)
	require.Equal(t, []string{"v1", "v2", "v3", "v4", "v5", "v6"}, sorted, "every version exactly once")
}
