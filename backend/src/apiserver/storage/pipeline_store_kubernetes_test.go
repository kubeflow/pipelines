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

	_, size, _, err := store.ListPipelines(fc, options)
	require.Nil(t, err, "Failed to list all pipelines: %v")
	require.Equalf(t, size, 1, "List size is not zero")

	pipeline := &model.Pipeline{
		Name:        "Test Pipeline",
		Description: "Test Pipeline Description",
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
		Name:        "Test Pipeline",
		Description: "Test Pipeline Description",
		Namespace:   "Test",
	}
	_, err := store.CreatePipeline(pipeline)
	require.Nil(t, err, "Failed to create Pipeline: %v")

	filterProto := &api.Filter{
		Predicates: []*api.Predicate{
			{
				Key:   "name",
				Op:    api.Predicate_IS_SUBSTRING,
				Value: &api.Predicate_StringValue{StringValue: "Test"},
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
		Name:        "Test Pipeline 1",
		Description: "Test Pipeline 1 Description",
		Namespace:   "Test",
	}
	pipeline2 := &model.Pipeline{
		Name:        "Test Pipeline 2",
		Description: "Test Pipeline 2 Description",
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
	require.Equalf(t, pipelines[0].Name, "Test Pipeline 3", "Pagination failed")
}

func TestListK8sPipelines_Pagination_Descend(t *testing.T) {
	podNamespace := viper.Get("POD_NAMESPACE")
	viper.Set("POD_NAMESPACE", "Test")
	defer viper.Set("POD_NAMESPACE", podNamespace)

	store := NewPipelineStoreKubernetes(getClient())

	pipeline1 := &model.Pipeline{
		Name:        "Test Pipeline 1",
		Description: "Test Pipeline 1 Description",
		Namespace:   "Test",
	}
	pipeline2 := &model.Pipeline{
		Name:        "Test Pipeline 2",
		Description: "Test Pipeline 2 Description",
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
	pipelines, _, _, err3 := store.ListPipelines(&model.FilterContext{}, options)
	require.Nil(t, err3, "Failed to list pipelines: %v")
	require.Equalf(t, pipelines[0].Name, "Test Pipeline 3", "Pagination failed")
}

func TestListK8sPipelinesV1_Pagination_NameAsc(t *testing.T) {
	podNamespace := viper.Get("POD_NAMESPACE")
	viper.Set("POD_NAMESPACE", "Test")
	defer viper.Set("POD_NAMESPACE", podNamespace)

	store := NewPipelineStoreKubernetes(getClient())

	pipeline1 := &model.Pipeline{
		Name:        "Test Pipeline 1",
		Description: "Test Pipeline 1 Description",
		Namespace:   "Test",
	}
	pipeline2 := &model.Pipeline{
		Name:        "Test Pipeline 2",
		Description: "Test Pipeline 2 Description",
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
	pipelines, _, _, err3 := store.ListPipelines(&model.FilterContext{}, options)
	require.Nil(t, err3, "Failed to list pipelines: %v")
	require.Equalf(t, pipelines[0].Name, "Test Pipeline 1", "Pagination failed")
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

func TestCreateK8sPipeline(t *testing.T) {
	podNamespace := viper.Get("POD_NAMESPACE")
	viper.Set("POD_NAMESPACE", "Test")
	defer viper.Set("POD_NAMESPACE", podNamespace)

	store := NewPipelineStoreKubernetes(getClient())

	pipeline := &model.Pipeline{
		Name:        "Test Pipeline",
		Description: "Test Pipeline Description",
		Namespace:   "Test",
	}

	pipeline, err := store.CreatePipeline(pipeline)
	if err != nil {
		t.Fatalf("Failed to create Pipeline: %v", err)
	}

	require.Equalf(t, pipeline.Name, "Test Pipeline", "Pipeline name is not the same")
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
		Name:         "Test Pipeline Version",
		PipelineId:   DefaultFakePipelineIdTwo,
		Description:  "Test Pipeline Version Description",
		PipelineSpec: getBasicPipelineSpecYAML(),
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
		Description: "Test Pipeline Version Description",
	}

	p, err := store.GetPipelineVersion(DefaultFakePipelineIdTwo)
	require.Nil(t, err, "Failed to get Pipeline: %v", err)
	require.Equal(t, p.UUID, pipelineVersion.UUID)
}

func TestGetLatestK8sPipelineVersion(t *testing.T) {
	podNamespace := viper.Get("POD_NAMESPACE")
	viper.Set("POD_NAMESPACE", "Test")
	defer viper.Set("POD_NAMESPACE", podNamespace)

	store := NewPipelineStoreKubernetes(getClient())

	pipelineVersion, err := store.GetLatestPipelineVersion(DefaultFakePipelineIdTwo)
	require.Nil(t, err, "Failed to get latest pipeline version: %v", err)
	require.Equal(t, "Test Pipeline Version 3", pipelineVersion.Name)
}

func TestGetK8sPipelineVersion_NotFoundError(t *testing.T) {
	podNamespace := viper.Get("POD_NAMESPACE")
	viper.Set("POD_NAMESPACE", "Test")
	defer viper.Set("POD_NAMESPACE", podNamespace)

	store := NewPipelineStoreKubernetes(getClient())

	_, err := store.GetLatestPipelineVersion(DefaultFakePipelineIdFive)
	require.NotNil(t, err)
	assert.Equal(t, err.(*util.UserError).ExternalStatusCode(), codes.NotFound)
}

func TestListK8sPipelineVersions_Pagination(t *testing.T) {
	podNamespace := viper.Get("POD_NAMESPACE")
	viper.Set("POD_NAMESPACE", "Test")
	defer viper.Set("POD_NAMESPACE", podNamespace)

	store := NewPipelineStoreKubernetes(getClient())

	pipelineVersion1 := &model.PipelineVersion{
		Name:         "Test Pipeline Version 1",
		PipelineId:   DefaultFakePipelineIdTwo,
		PipelineSpec: getBasicPipelineSpecYAML(),
	}

	pipelineVersion2 := &model.PipelineVersion{
		Name:         "Test Pipeline Version 2",
		PipelineId:   DefaultFakePipelineIdTwo,
		PipelineSpec: getBasicPipelineSpecYAML(),
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
	require.Equalf(t, pipelineVersions[0].Name, "Test Pipeline Version 3", "Pagination did not work as expected")
}

func TestListK8sPipelineVersions_Pagination_Descend(t *testing.T) {
	podNamespace := viper.Get("POD_NAMESPACE")
	viper.Set("POD_NAMESPACE", "Test")
	defer viper.Set("POD_NAMESPACE", podNamespace)

	store := NewPipelineStoreKubernetes(getClient())

	pipelineVersion1 := &model.PipelineVersion{
		Name:         "Test Pipeline Version 1",
		PipelineId:   DefaultFakePipelineIdTwo,
		PipelineSpec: getBasicPipelineSpecYAML(),
	}

	pipelineVersion2 := &model.PipelineVersion{
		Name:         "Test Pipeline Version 2",
		PipelineId:   DefaultFakePipelineIdTwo,
		PipelineSpec: getBasicPipelineSpecYAML(),
	}

	_, err := store.CreatePipelineVersion(pipelineVersion1)
	require.Nil(t, err, "Failed to create PipelineVersion: %v", err)
	_, err = store.CreatePipelineVersion(pipelineVersion2)
	require.Nil(t, err, "Failed to create PipelineVersion: %v", err)

	options, err := list.NewOptions(&model.PipelineVersion{}, 1, "name desc", nil)

	pipelineVersions, _, _, err1 := store.ListPipelineVersions(DefaultFakePipelineIdTwo, options)
	require.Nil(t, err1, "Failed to list pipeline versions: %v", err)
	require.Equalf(t, len(pipelineVersions), 1, "List size should not be zero")
	require.Equalf(t, pipelineVersions[0].Name, "Test Pipeline Version 3", "Pagination did not work as expected")
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

	pipelineVersion, err := store.GetPipelineVersionByName("Test Pipeline Version 3")
	require.Nil(t, err, "Failed to get Pipeline: %v", err)
	require.Equalf(t, pipelineVersion.Name, "Test Pipeline Version 3", pipelineVersion.Name)
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
				Value: &api.Predicate_StringValue{StringValue: "Test"},
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
		Name: "Test Pipeline",
	}
	k8sPipelineVersion := &model.PipelineVersion{
		Name:         "Test Pipeline Version",
		PipelineSpec: getBasicPipelineSpecYAML(),
	}

	_, _, err := store.CreatePipelineAndPipelineVersion(k8sPipeline, k8sPipelineVersion)
	require.Nil(t, err, "Failed to create Pipeline: %v", err)
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

func getClient() (client.Client, client.Client) {
	scheme := runtime.NewScheme()
	err := v2beta1.AddToScheme(scheme)
	if err != nil {
		glog.Fatalf("Failed to add to scheme: %v", err)
	}

	pipeline3 := &v2beta1.Pipeline{
		ObjectMeta: metav1.ObjectMeta{
			UID:       DefaultFakePipelineIdTwo,
			Name:      "Test Pipeline 3",
			Namespace: "Test",
		},
		Spec: v2beta1.PipelineSpec{
			Description: "Test Pipeline 3 Description",
		},
	}

	pipelineVersion := &v2beta1.PipelineVersion{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "Test Pipeline Version",
			Namespace: "Test",
		},
		Spec: v2beta1.PipelineVersionSpec{
			PipelineSpec: getBasicPipelineSpec(),
		},
	}

	pipelineVersion1 := &v2beta1.PipelineVersion{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "Test Pipeline Version 1",
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
			Name:      "Test Pipeline Version 2",
			Namespace: "Test",
		},
		Spec: v2beta1.PipelineVersionSpec{
			PipelineSpec: getBasicPipelineSpec(),
		},
	}

	pipelineVersion3 := &v2beta1.PipelineVersion{
		ObjectMeta: metav1.ObjectMeta{
			UID:       DefaultFakePipelineIdTwo,
			Name:      "Test Pipeline Version 3",
			Namespace: "Test",
			Labels: map[string]string{
				"pipelines.kubeflow.org/pipeline-id": DefaultFakePipelineIdTwo,
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					UID: DefaultFakePipelineIdTwo,
				},
			},
		},
		Spec: v2beta1.PipelineVersionSpec{
			Description:  "Test Pipeline Version 1 Description",
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
