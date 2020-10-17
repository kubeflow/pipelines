package server

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	api "github.com/kubeflow/pipelines/backend/api/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
)

func TestCreatePipeline_YAML(t *testing.T) {
	httpServer := getMockServer(t)
	// Close the server when test finishes
	defer httpServer.Close()

	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager)

	pipelineServer := PipelineServer{resourceManager: resourceManager, httpClient: httpServer.Client(), options: &PipelineServerOptions{CollectMetrics: false}}
	pipeline, err := pipelineServer.CreatePipeline(context.Background(), &api.CreatePipelineRequest{
		Pipeline: &api.Pipeline{
			Url:         &api.Url{PipelineUrl: httpServer.URL + "/arguments-parameters.yaml"},
			Name:        "argument-parameters",
			Description: "pipeline description",
		}})

	assert.Nil(t, err)
	assert.NotNil(t, pipeline)
	assert.Equal(t, "argument-parameters", pipeline.Name)
	newPipeline, err := resourceManager.GetPipeline(pipeline.Id)
	assert.Nil(t, err)
	assert.NotNil(t, newPipeline)
	var params []api.Parameter
	err = json.Unmarshal([]byte(newPipeline.Parameters), &params)
	assert.Nil(t, err)
	assert.Equal(t, []api.Parameter{{Name: "param1", Value: "hello"}, {Name: "param2"}}, params)
	assert.Equal(t, "pipeline description", newPipeline.Description)
}

func TestCreatePipeline_Tarball(t *testing.T) {
	httpServer := getMockServer(t)
	// Close the server when test finishes
	defer httpServer.Close()

	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager)

	pipelineServer := PipelineServer{resourceManager: resourceManager, httpClient: httpServer.Client(), options: &PipelineServerOptions{CollectMetrics: false}}
	pipeline, err := pipelineServer.CreatePipeline(context.Background(), &api.CreatePipelineRequest{
		Pipeline: &api.Pipeline{
			Url:         &api.Url{PipelineUrl: httpServer.URL + "/arguments_tarball/arguments.tar.gz"},
			Name:        "argument-parameters",
			Description: "pipeline description",
		}})

	assert.Nil(t, err)
	assert.NotNil(t, pipeline)
	assert.Equal(t, "argument-parameters", pipeline.Name)
	newPipeline, err := resourceManager.GetPipeline(pipeline.Id)
	assert.Nil(t, err)
	assert.NotNil(t, newPipeline)
	var params []api.Parameter
	err = json.Unmarshal([]byte(newPipeline.Parameters), &params)
	assert.Nil(t, err)
	assert.Equal(t, []api.Parameter{{Name: "param1", Value: "hello"}, {Name: "param2"}}, params)
	assert.Equal(t, "pipeline description", newPipeline.Description)
}

func TestCreatePipeline_InvalidYAML(t *testing.T) {
	httpServer := getMockServer(t)
	// Close the server when test finishes
	defer httpServer.Close()

	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager)

	pipelineServer := PipelineServer{resourceManager: resourceManager, httpClient: httpServer.Client(), options: &PipelineServerOptions{CollectMetrics: false}}
	_, err := pipelineServer.CreatePipeline(context.Background(), &api.CreatePipelineRequest{
		Pipeline: &api.Pipeline{
			Url:  &api.Url{PipelineUrl: httpServer.URL + "/invalid-workflow.yaml"},
			Name: "argument-parameters",
		}})

	assert.NotNil(t, err)
	assert.Equal(t, codes.InvalidArgument, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "Unexpected resource type")
}

func TestCreatePipeline_InvalidURL(t *testing.T) {
	httpServer := getBadMockServer()
	// Close the server when test finishes
	defer httpServer.Close()

	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager)

	pipelineServer := PipelineServer{resourceManager: resourceManager, httpClient: httpServer.Client(), options: &PipelineServerOptions{CollectMetrics: false}}
	_, err := pipelineServer.CreatePipeline(context.Background(), &api.CreatePipelineRequest{
		Pipeline: &api.Pipeline{
			Url:  &api.Url{PipelineUrl: httpServer.URL + "/invalid-workflow.yaml"},
			Name: "argument-parameters",
		}})
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
}

func TestCreatePipelineVersion_YAML(t *testing.T) {
	httpServer := getMockServer(t)
	// Close the server when test finishes
	defer httpServer.Close()

	clientManager := resource.NewFakeClientManagerOrFatal(
		util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager)

	pipelineServer := PipelineServer{
		resourceManager: resourceManager, httpClient: httpServer.Client(), options: &PipelineServerOptions{CollectMetrics: false}}
	pipelineVersion, err := pipelineServer.CreatePipelineVersion(
		context.Background(), &api.CreatePipelineVersionRequest{
			Version: &api.PipelineVersion{
				PackageUrl: &api.Url{
					PipelineUrl: httpServer.URL + "/arguments-parameters.yaml"},
				Name: "argument-parameters",
				ResourceReferences: []*api.ResourceReference{
					&api.ResourceReference{
						Key: &api.ResourceKey{
							Id:   "pipeline",
							Type: api.ResourceType_PIPELINE,
						},
						Relationship: api.Relationship_OWNER,
					}}}})

	assert.Nil(t, err)
	assert.NotNil(t, pipelineVersion)
	assert.Equal(t, "argument-parameters", pipelineVersion.Name)
	newPipelineVersion, err := resourceManager.GetPipelineVersion(
		pipelineVersion.Id)
	assert.Nil(t, err)
	assert.NotNil(t, newPipelineVersion)
	var params []api.Parameter
	err = json.Unmarshal([]byte(newPipelineVersion.Parameters), &params)
	assert.Nil(t, err)
	assert.Equal(t, []api.Parameter{
		{Name: "param1", Value: "hello"}, {Name: "param2"}}, params)
}

func TestCreatePipelineVersion_InvalidYAML(t *testing.T) {
	httpServer := getMockServer(t)
	// Close the server when test finishes
	defer httpServer.Close()

	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager)

	pipelineServer := PipelineServer{resourceManager: resourceManager, httpClient: httpServer.Client(), options: &PipelineServerOptions{CollectMetrics: false}}
	_, err := pipelineServer.CreatePipelineVersion(
		context.Background(), &api.CreatePipelineVersionRequest{
			Version: &api.PipelineVersion{
				PackageUrl: &api.Url{
					PipelineUrl: httpServer.URL + "/invalid-workflow.yaml"},
				Name: "argument-parameters",
				ResourceReferences: []*api.ResourceReference{
					&api.ResourceReference{
						Key: &api.ResourceKey{
							Id:   "pipeline",
							Type: api.ResourceType_PIPELINE,
						},
						Relationship: api.Relationship_OWNER,
					}}}})

	assert.NotNil(t, err)
	assert.Equal(t, codes.InvalidArgument, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "Unexpected resource type")
}

func TestCreatePipelineVersion_Tarball(t *testing.T) {
	httpServer := getMockServer(t)
	// Close the server when test finishes
	defer httpServer.Close()

	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager)

	pipelineServer := PipelineServer{resourceManager: resourceManager, httpClient: httpServer.Client(), options: &PipelineServerOptions{CollectMetrics: false}}
	pipelineVersion, err := pipelineServer.CreatePipelineVersion(
		context.Background(), &api.CreatePipelineVersionRequest{
			Version: &api.PipelineVersion{
				PackageUrl: &api.Url{
					PipelineUrl: httpServer.URL +
						"/arguments_tarball/arguments.tar.gz"},
				Name: "argument-parameters",
				ResourceReferences: []*api.ResourceReference{
					&api.ResourceReference{
						Key: &api.ResourceKey{
							Id:   "pipeline",
							Type: api.ResourceType_PIPELINE,
						},
						Relationship: api.Relationship_OWNER,
					}}}})

	assert.Nil(t, err)
	assert.NotNil(t, pipelineVersion)
	assert.Equal(t, "argument-parameters", pipelineVersion.Name)
	newPipelineVersion, err := resourceManager.GetPipelineVersion(pipelineVersion.Id)
	assert.Nil(t, err)
	assert.NotNil(t, newPipelineVersion)
	var params []api.Parameter
	err = json.Unmarshal([]byte(newPipelineVersion.Parameters), &params)
	assert.Nil(t, err)
	assert.Equal(t, []api.Parameter{{Name: "param1", Value: "hello"}, {Name: "param2"}}, params)
}

func TestCreatePipelineVersion_InvalidURL(t *testing.T) {
	// Use a bad mock server
	httpServer := getBadMockServer()
	// Close the server when test finishes
	defer httpServer.Close()

	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager)

	pipelineServer := PipelineServer{resourceManager: resourceManager, httpClient: httpServer.Client(), options: &PipelineServerOptions{CollectMetrics: false}}
	_, err := pipelineServer.CreatePipelineVersion(context.Background(), &api.CreatePipelineVersionRequest{
		Version: &api.PipelineVersion{
			PackageUrl: &api.Url{
				PipelineUrl: httpServer.URL + "/invalid-workflow.yaml"},
			Name: "argument-parameters",
			ResourceReferences: []*api.ResourceReference{
				&api.ResourceReference{
					Key: &api.ResourceKey{
						Id:   "pipeline",
						Type: api.ResourceType_PIPELINE,
					},
					Relationship: api.Relationship_OWNER,
				}}}})

	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
}

func TestListPipelineVersion_NoResourceKey(t *testing.T) {
	httpServer := getMockServer(t)
	// Close the server when test finishes
	defer httpServer.Close()

	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager)

	pipelineServer := PipelineServer{resourceManager: resourceManager, httpClient: httpServer.Client(), options: &PipelineServerOptions{CollectMetrics: false}}

	_, err := pipelineServer.ListPipelineVersions(context.Background(), &api.ListPipelineVersionsRequest{
		ResourceKey: nil,
		PageSize:    20,
	})
	assert.Equal(t, "Invalid input error: ResourceKey must be set in the input", err.Error())
}

func TestCreatePipelineVersionDontUpdateDefault(t *testing.T) {
	viper.Set(common.UpdatePipelineVersionByDefault, "false")
	defer viper.Set(common.UpdatePipelineVersionByDefault, "true")
	httpServer := getMockServer(t)
	// Close the server when test finishes
	defer httpServer.Close()
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager)

	pipelineServer := PipelineServer{resourceManager: resourceManager, httpClient: httpServer.Client(), options: &PipelineServerOptions{CollectMetrics: false}}
	pipeline, err := pipelineServer.CreatePipeline(context.Background(), &api.CreatePipelineRequest{
		Pipeline: &api.Pipeline{
			Url:         &api.Url{PipelineUrl: httpServer.URL + "/arguments_tarball/arguments.tar.gz"},
			Name:        "argument-parameters",
			Description: "pipeline description",
		}})

	assert.Nil(t, err)
	assert.NotNil(t, pipeline)

	clientManager.UpdateUUID(util.NewFakeUUIDGeneratorOrFatal("123e4567-e89b-12d3-a456-526655440001", nil))
	resourceManager = resource.NewResourceManager(clientManager)

	pipelineServer = PipelineServer{resourceManager: resourceManager, httpClient: httpServer.Client(), options: &PipelineServerOptions{CollectMetrics: false}}
	pipelineVersion, err := pipelineServer.CreatePipelineVersion(
		context.Background(), &api.CreatePipelineVersionRequest{
			Version: &api.PipelineVersion{
				PackageUrl: &api.Url{
					PipelineUrl: httpServer.URL + "/arguments-parameters.yaml"},
				Name: "argument-parameters-update",
				ResourceReferences: []*api.ResourceReference{
					&api.ResourceReference{
						Key: &api.ResourceKey{
							Type: api.ResourceType_PIPELINE,
							Id:   pipeline.Id,
						},
						Relationship: api.Relationship_OWNER,
					}}}})
	assert.Nil(t, err)

	pipelines, err := pipelineServer.GetPipeline(context.Background(), &api.GetPipelineRequest{Id: pipeline.Id})
	assert.NotNil(t, pipelineVersion.Id)
	assert.NotEqual(t, pipelines.DefaultVersion.Id, pipelineVersion.Id)
}

func getMockServer(t *testing.T) *httptest.Server {
	httpServer := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		// Send response to be tested
		file, err := os.Open("test" + req.URL.String())
		assert.Nil(t, err)
		bytes, err := ioutil.ReadAll(file)
		assert.Nil(t, err)

		rw.WriteHeader(http.StatusOK)
		rw.Write(bytes)
	}))
	return httpServer
}

func getBadMockServer() *httptest.Server {
	httpServer := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.WriteHeader(404)
	}))
	return httpServer
}
