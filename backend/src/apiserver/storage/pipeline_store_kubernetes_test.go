package storage

import (
	"testing"

	"github.com/golang/glog"
	api "github.com/kubeflow/pipelines/backend/api/v1beta1/go_client"
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

	"github.com/kubeflow/pipelines/backend/src/crd/kubernetes/v2beta1"
)

func TestListK8sPipelines(t *testing.T) {
	podNamespace := viper.Get("POD_NAMESPACE")
	viper.Set("POD_NAMESPACE", "Test")
	defer viper.Set("POD_NAMESPACE", podNamespace)

	store := NewPipelineStoreKubernetes(getClient())

	fc := &model.FilterContext{}
	options := list.EmptyOptions()

	_, size, _, err := store.ListPipelines(fc, options, nil)
	require.Nil(t, err, "Failed to list all pipelines: %v")
	require.Equalf(t, size, 1, "List size is not zero")

	pipeline := &model.Pipeline{
		Name:        "test-pipeline",
		Description: model.LargeText("Test Pipeline Description"),
		Namespace:   "Test",
	}

	_, err = store.CreatePipeline(pipeline)
	require.Nil(t, err, "Failed to create Pipeline: %v", err)

	_, size, _, err = store.ListPipelines(fc, options, nil)
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
				Key:   "name",
				Op:    api.Predicate_IS_SUBSTRING,
				Value: &api.Predicate_StringValue{StringValue: "test"},
			},
		},
	}
	newFilter, _ := filter.New(filterProto)
	options, err1 := list.NewOptions(&model.Pipeline{}, 10, "id", newFilter)
	require.Nil(t, err1, "Failed to create list options: %v")

	pipelines, _, _, err2 := store.ListPipelines(&model.FilterContext{}, options, nil)
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

	_, pageSize, npt, err2 := store.ListPipelines(&model.FilterContext{}, options, nil)
	require.Nil(t, err2, "Failed to list pipelines: %v")
	require.NotNil(t, npt)
	require.Equalf(t, pageSize, 3, "List size should not be zero")

	options, err1 = list.NewOptionsFromToken(npt, 1)
	require.Nil(t, err1, "Failed to create list options: %v")
	pipelines, _, _, err3 := store.ListPipelines(&model.FilterContext{}, options, nil)
	require.Nil(t, err3, "Failed to list pipelines: %v")
	require.Equalf(t, pipelines[0].Name, "test-pipeline-3", "Pagination failed")
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

	_, pageSize, npt, err2 := store.ListPipelines(&model.FilterContext{}, options, nil)
	require.Nil(t, err2, "Failed to list pipelines: %v")
	require.NotNil(t, npt)
	require.Equalf(t, pageSize, 3, "List size should not be zero")

	options, err1 = list.NewOptionsFromToken(npt, 1)
	pipelines, _, _, err3 := store.ListPipelines(&model.FilterContext{}, options, nil)
	require.Nil(t, err3, "Failed to list pipelines: %v")
	require.Equalf(t, pipelines[0].Name, "test-pipeline-3", "Pagination failed")
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

	_, pageSize, npt, err2 := store.ListPipelines(&model.FilterContext{}, options, nil)
	require.Nil(t, err2, "Failed to list pipelines: %v")
	require.NotNil(t, npt)
	require.Equalf(t, pageSize, 3, "List size should not be zero")

	options, err1 = list.NewOptionsFromToken(npt, 1)
	pipelines, _, _, err3 := store.ListPipelines(&model.FilterContext{}, options, nil)
	require.Nil(t, err3, "Failed to list pipelines: %v")
	require.Equalf(t, pipelines[0].Name, "test-pipeline-1", "Pagination failed")
}

func TestListK8sPipelines_Pagination_LessThanPageSize(t *testing.T) {
	podNamespace := viper.Get("POD_NAMESPACE")
	viper.Set("POD_NAMESPACE", "Test")
	defer viper.Set("POD_NAMESPACE", podNamespace)

	store := NewPipelineStoreKubernetes(getClient())

	options, err1 := list.NewOptions(&model.Pipeline{}, 10, "", nil)
	require.Nil(t, err1, "Failed to create list options: %v")

	pipelines, pageSize, _, err := store.ListPipelines(&model.FilterContext{}, options, nil)
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

	pipelineVersions, _, npt, err := store.ListPipelineVersions(DefaultFakePipelineIdTwo, options, nil)
	require.Nil(t, err, "Failed to list pipeline versions: %v", err)
	require.Equalf(t, len(pipelineVersions), 1, "List size should not be zero")
	require.NotNil(t, npt, "Npt should not be nil")

	options, err = list.NewOptionsFromToken(npt, 1)
	require.Nil(t, err, "Failed to create list options")
	pipelineVersions, _, _, err = store.ListPipelineVersions(DefaultFakePipelineIdTwo, options, nil)
	require.Nil(t, err, "Failed to list pipeline versions: %v", err)
	require.Equalf(t, len(pipelineVersions), 1, "List size should not be zero")
	require.Equalf(t, pipelineVersions[0].Name, "test-pipeline-version-3", "Pagination did not work as expected")
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

	pipelineVersions, _, _, err1 := store.ListPipelineVersions(DefaultFakePipelineIdTwo, options, nil)
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

	pipelines, pageSize, _, err := store.ListPipelineVersions(DefaultFakePipelineIdTwo, options, nil)
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
				Key:   "name",
				Op:    api.Predicate_IS_SUBSTRING,
				Value: &api.Predicate_StringValue{StringValue: "test"},
			},
		},
	}

	newFilter, err := filter.New(filterProto)
	options, err1 := list.NewOptions(&model.PipelineVersion{}, 1, "", newFilter)
	require.Nil(t, err1, "Failed to list pipeline versions: %v", err)

	pipelineVersions, _, _, err2 := store.ListPipelineVersions(DefaultFakePipelineIdTwo, options, nil)
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

	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(pipelineVersion, pipelineVersion1, pipelineVersion2, pipelineVersion3).
		WithObjects(pipeline3, pipelineVersion3).
		Build()

	return k8sClient, k8sClient
}
