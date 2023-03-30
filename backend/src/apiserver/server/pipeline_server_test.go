// Copyright 2018 The Kubeflow Authors
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

package server

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	api "github.com/kubeflow/pipelines/backend/api/v1beta1/go_client"
	apiv2 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
)

func TestBuildPipelineName_QueryStringNotEmpty(t *testing.T) {
	pipelineName, err := buildPipelineName("pipeline%20one", "file one")
	assert.Nil(t, err)
	assert.Equal(t, "pipeline one", pipelineName)
}

func TestBuildPipelineName(t *testing.T) {
	pipelineName, err := buildPipelineName("", "file one")
	assert.Nil(t, err)
	assert.Equal(t, "file one", pipelineName)
}

func TestBuildPipelineName_InvalidQueryString(t *testing.T) {
	_, err := buildPipelineName("pipeline!$%one", "file one")
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "invalid format")
}

func TestBuildPipelineName_NameTooLong(t *testing.T) {
	_, err := buildPipelineName("",
		"this is a loooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooog name")
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "is too long")
}

func TestCreatePipelineV1_YAML(t *testing.T) {
	httpServer := getMockServer(t)
	// Close the server when test finishes
	defer httpServer.Close()

	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager, "default")

	pipelineServer := PipelineServer{resourceManager: resourceManager, httpClient: httpServer.Client(), options: &PipelineServerOptions{CollectMetrics: false}}
	pipeline, err := pipelineServer.CreatePipelineV1(context.Background(), &api.CreatePipelineRequest{
		Pipeline: &api.Pipeline{
			Url:         &api.Url{PipelineUrl: httpServer.URL + "/arguments-parameters.yaml"},
			Name:        "argument-parameters",
			Description: "pipeline description",
		},
	})

	assert.Nil(t, err)
	assert.NotNil(t, pipeline)
	assert.Equal(t, "argument-parameters", pipeline.Name)
	newPipeline, err := resourceManager.GetPipeline(pipeline.Id)
	assert.Nil(t, err)
	newPipelineVersion, err := resourceManager.GetLatestPipelineVersion(pipeline.Id)
	assert.Nil(t, err)
	assert.NotNil(t, newPipeline)
	assert.Equal(t, "pipeline description", newPipeline.Description)
	assert.Equal(t, newPipeline.UUID, newPipelineVersion.PipelineId)
}

func TestCreatePipelineV1_LargeFile(t *testing.T) {
	httpServer := getMockServer(t)
	// Close the server when test finishes
	defer httpServer.Close()

	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager, "default")

	pipelineServer := PipelineServer{resourceManager: resourceManager, httpClient: httpServer.Client(), options: &PipelineServerOptions{CollectMetrics: false}}
	pipeline, err := pipelineServer.CreatePipelineV1(context.Background(), &api.CreatePipelineRequest{
		Pipeline: &api.Pipeline{
			Url:         &api.Url{PipelineUrl: "https://raw.githubusercontent.com/kubeflow/pipelines/master/sdk/python/test_data/pipelines/xgboost_sample_pipeline.yaml"},
			Name:        "xgboost-url",
			Description: "pipeline description",
		},
	})

	assert.Nil(t, err)
	assert.NotNil(t, pipeline)
	assert.Equal(t, "xgboost-url", pipeline.Name)
	newPipeline, err := resourceManager.GetPipeline(pipeline.Id)
	assert.Nil(t, err)
	newPipelineVersion, err := resourceManager.GetLatestPipelineVersion(pipeline.Id)
	assert.Nil(t, err)
	assert.NotNil(t, newPipeline)
	assert.Equal(t, "pipeline description", newPipeline.Description)
	assert.Equal(t, newPipeline.UUID, newPipelineVersion.PipelineId)
}

func TestCreatePipelineV1_Tarball(t *testing.T) {
	httpServer := getMockServer(t)
	// Close the server when test finishes
	defer httpServer.Close()

	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager, "default")

	pipelineServer := PipelineServer{resourceManager: resourceManager, httpClient: httpServer.Client(), options: &PipelineServerOptions{CollectMetrics: false}}
	pipeline, err := pipelineServer.CreatePipelineV1(context.Background(), &api.CreatePipelineRequest{
		Pipeline: &api.Pipeline{
			Url:         &api.Url{PipelineUrl: httpServer.URL + "/arguments_tarball/arguments.tar.gz"},
			Name:        "argument-parameters",
			Description: "pipeline description",
		},
	})

	assert.Nil(t, err)
	assert.NotNil(t, pipeline)
	assert.Equal(t, "argument-parameters", pipeline.Name)
	newPipeline, err := resourceManager.GetPipeline(pipeline.Id)
	assert.Nil(t, err)
	newPipelineVersion, err := resourceManager.GetLatestPipelineVersion(pipeline.Id)
	assert.Nil(t, err)
	assert.NotNil(t, newPipeline)
	assert.NotNil(t, newPipelineVersion)
	assert.Equal(t, "pipeline description", newPipeline.Description)
	assert.Equal(t, newPipeline.UUID, newPipelineVersion.PipelineId)
}

func TestCreatePipelineV1_InvalidYAML(t *testing.T) {
	httpServer := getMockServer(t)
	// Close the server when test finishes
	defer httpServer.Close()

	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager, "default")

	pipelineServer := PipelineServer{resourceManager: resourceManager, httpClient: httpServer.Client(), options: &PipelineServerOptions{CollectMetrics: false}}
	createdPipeline, err := pipelineServer.CreatePipelineV1(
		context.Background(), &api.CreatePipelineRequest{
			Pipeline: &api.Pipeline{
				Url:  &api.Url{PipelineUrl: httpServer.URL + "/invalid-workflow.yaml"},
				Name: "argument-parameters",
			},
		},
	)
	assert.Nil(t, err)
	assert.NotEqual(t, "", createdPipeline.GetId())
	assert.Less(t, 0, int(createdPipeline.GetCreatedAt().Seconds))
	assert.Equal(t, "", createdPipeline.GetDefaultVersion().GetId())
	assert.Equal(t, 0, int(createdPipeline.GetDefaultVersion().GetCreatedAt().Seconds))
	newPipelineVersion, err := resourceManager.GetLatestPipelineVersion(createdPipeline.Id)
	assert.NotNil(t, err)
	assert.Nil(t, newPipelineVersion)
}

func TestCreatePipelineV1_InvalidURL(t *testing.T) {
	httpServer := getBadMockServer()
	// Close the server when test finishes
	defer httpServer.Close()

	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager, "default")

	pipelineServer := PipelineServer{resourceManager: resourceManager, httpClient: httpServer.Client(), options: &PipelineServerOptions{CollectMetrics: false}}
	createdPipeline, err := pipelineServer.CreatePipelineV1(
		context.Background(), &api.CreatePipelineRequest{
			Pipeline: &api.Pipeline{
				Url:  &api.Url{PipelineUrl: httpServer.URL + "/invalid-workflow.yaml"},
				Name: "argument-parameters",
			},
		},
	)
	assert.Nil(t, err)
	assert.NotEqual(t, "", createdPipeline.GetId())
	assert.Less(t, 0, int(createdPipeline.GetCreatedAt().Seconds))
	assert.Equal(t, "", createdPipeline.GetDefaultVersion().GetId())
	assert.Equal(t, 0, int(createdPipeline.GetDefaultVersion().GetCreatedAt().Seconds))
	newPipelineVersion, err := resourceManager.GetLatestPipelineVersion(createdPipeline.Id)
	assert.NotNil(t, err)
	assert.Nil(t, newPipelineVersion)
}

func TestCreatePipelineV1_ExistingPipeline(t *testing.T) {
	httpServer := getMockServer(t)
	// Close the server when test finishes
	defer httpServer.Close()

	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager, "default")

	pipelineServer := PipelineServer{resourceManager: resourceManager, httpClient: httpServer.Client(), options: &PipelineServerOptions{CollectMetrics: false}}
	existingPipeline, _ := pipelineServer.CreatePipelineV1(
		context.Background(), &api.CreatePipelineRequest{
			Pipeline: &api.Pipeline{
				Name: "argument-parameters",
				ResourceReferences: []*api.ResourceReference{
					{
						Key: &api.ResourceKey{
							Id:   "test-namespace",
							Type: api.ResourceType_NAMESPACE,
						},
						Relationship: api.Relationship_OWNER,
					},
				},
			},
		},
	)
	createdPipeline, err := pipelineServer.CreatePipelineV1(
		context.Background(), &api.CreatePipelineRequest{
			Pipeline: &api.Pipeline{
				Url:  &api.Url{PipelineUrl: httpServer.URL + "/arguments-parameters.yaml"},
				Name: "argument-parameters",
				ResourceReferences: []*api.ResourceReference{
					{
						Key: &api.ResourceKey{
							Id:   "test-namespace",
							Type: api.ResourceType_NAMESPACE,
						},
						Relationship: api.Relationship_OWNER,
					},
				},
			},
		},
	)
	assert.Nil(t, err)
	assert.NotEqual(t, "", createdPipeline.GetId())
	assert.Equal(t, existingPipeline.GetId(), createdPipeline.GetId())
	assert.Less(t, 0, int(createdPipeline.GetCreatedAt().Seconds))

	newPipelineVersion, err := resourceManager.GetLatestPipelineVersion(createdPipeline.Id)
	assert.Nil(t, err)
	assert.NotNil(t, newPipelineVersion)
	assert.Equal(t, newPipelineVersion.PipelineId, existingPipeline.GetId())
	assert.Equal(t, newPipelineVersion.PipelineId, createdPipeline.GetId())

	assert.Equal(t, "", existingPipeline.GetDefaultVersion().GetId())
	assert.Equal(t, 0, int(existingPipeline.GetDefaultVersion().GetCreatedAt().Seconds))

	assert.Equal(t, newPipelineVersion.UUID, createdPipeline.GetDefaultVersion().GetId())
	assert.Less(t, 0, int(createdPipeline.GetDefaultVersion().GetCreatedAt().Seconds))
}

func TestCreatePipelineVersionV1_YAML(t *testing.T) {
	httpServer := getMockServer(t)
	// Close the server when test finishes
	defer httpServer.Close()

	clientManager := resource.NewFakeClientManagerOrFatal(
		util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager, "default")

	pipelineServer := PipelineServer{
		resourceManager: resourceManager, httpClient: httpServer.Client(), options: &PipelineServerOptions{CollectMetrics: false},
	}
	pipelineVersion, err := pipelineServer.CreatePipelineVersionV1(
		context.Background(), &api.CreatePipelineVersionRequest{
			Version: &api.PipelineVersion{
				PackageUrl: &api.Url{
					PipelineUrl: httpServer.URL + "/arguments-parameters.yaml",
				},
				Name: "argument-parameters",
				ResourceReferences: []*api.ResourceReference{
					{
						Key: &api.ResourceKey{
							Id:   "pipeline",
							Type: api.ResourceType_PIPELINE,
						},
						Relationship: api.Relationship_OWNER,
					},
				},
			},
		})

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
		{Name: "param1", Value: "hello"}, {Name: "param2"},
	}, params)
}

func TestCreatePipelineVersion_InvalidYAML(t *testing.T) {
	httpServer := getMockServer(t)
	// Close the server when test finishes
	defer httpServer.Close()

	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager, "default")

	pipelineServer := PipelineServer{resourceManager: resourceManager, httpClient: httpServer.Client(), options: &PipelineServerOptions{CollectMetrics: false}}
	_, err := pipelineServer.CreatePipelineVersionV1(
		context.Background(), &api.CreatePipelineVersionRequest{
			Version: &api.PipelineVersion{
				PackageUrl: &api.Url{
					PipelineUrl: httpServer.URL + "/invalid-workflow.yaml",
				},
				Name: "argument-parameters",
				ResourceReferences: []*api.ResourceReference{
					{
						Key: &api.ResourceKey{
							Id:   "pipeline",
							Type: api.ResourceType_PIPELINE,
						},
						Relationship: api.Relationship_OWNER,
					},
				},
			},
		})

	assert.NotNil(t, err)
	assert.Equal(t, codes.InvalidArgument, err.(*util.UserError).ExternalStatusCode())
}

func TestCreatePipelineVersion_Tarball(t *testing.T) {
	httpServer := getMockServer(t)
	// Close the server when test finishes
	defer httpServer.Close()

	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager, "default")

	pipelineServer := PipelineServer{resourceManager: resourceManager, httpClient: httpServer.Client(), options: &PipelineServerOptions{CollectMetrics: false}}
	pipelineVersion, err := pipelineServer.CreatePipelineVersionV1(
		context.Background(), &api.CreatePipelineVersionRequest{
			Version: &api.PipelineVersion{
				PackageUrl: &api.Url{
					PipelineUrl: httpServer.URL +
						"/arguments_tarball/arguments.tar.gz",
				},
				Name: "argument-parameters",
				ResourceReferences: []*api.ResourceReference{
					{
						Key: &api.ResourceKey{
							Id:   "pipeline",
							Type: api.ResourceType_PIPELINE,
						},
						Relationship: api.Relationship_OWNER,
					},
				},
			},
		})

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
	resourceManager := resource.NewResourceManager(clientManager, "default")

	pipelineServer := PipelineServer{resourceManager: resourceManager, httpClient: httpServer.Client(), options: &PipelineServerOptions{CollectMetrics: false}}
	_, err := pipelineServer.CreatePipelineVersionV1(context.Background(), &api.CreatePipelineVersionRequest{
		Version: &api.PipelineVersion{
			PackageUrl: &api.Url{
				PipelineUrl: httpServer.URL + "/invalid-workflow.yaml",
			},
			Name: "argument-parameters",
			ResourceReferences: []*api.ResourceReference{
				{
					Key: &api.ResourceKey{
						Id:   "pipeline",
						Type: api.ResourceType_PIPELINE,
					},
					Relationship: api.Relationship_OWNER,
				},
			},
		},
	})

	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
}

func TestListPipelineVersion_NoResourceKey(t *testing.T) {
	httpServer := getMockServer(t)
	// Close the server when test finishes
	defer httpServer.Close()

	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager, "default")

	pipelineServer := PipelineServer{resourceManager: resourceManager, httpClient: httpServer.Client(), options: &PipelineServerOptions{CollectMetrics: false}}

	_, err := pipelineServer.ListPipelineVersionsV1(context.Background(), &api.ListPipelineVersionsRequest{
		ResourceKey: nil,
		PageSize:    20,
	})
	assert.Contains(t, err.Error(), "missing pipeline id")
}

func TestListPipelinesPublic(t *testing.T) {
	httpServer := getMockServer(t)
	// Close the server when test finishes
	defer httpServer.Close()
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager, "default")

	pipelineServer := PipelineServer{resourceManager: resourceManager, httpClient: httpServer.Client(), options: &PipelineServerOptions{CollectMetrics: false}}
	_, err := pipelineServer.ListPipelinesV1(context.Background(),
		&api.ListPipelinesRequest{
			PageSize: 20,
			ResourceReferenceKey: &api.ResourceKey{
				Type: api.ResourceType_NAMESPACE,
				Id:   "",
			},
		})
	assert.EqualValues(t, nil, err, err)
}

func TestGetPipelineByName_OK(t *testing.T) {
	httpServer := getMockServer(t)
	// Close the server when test finishes
	defer httpServer.Close()
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager, "default")
	pipelineServer := PipelineServer{resourceManager: resourceManager, httpClient: httpServer.Client(), options: &PipelineServerOptions{CollectMetrics: false}}
	pipeline, err := pipelineServer.CreatePipelineV1(context.Background(), &api.CreatePipelineRequest{
		Pipeline: &api.Pipeline{
			Url:  &api.Url{PipelineUrl: httpServer.URL + "/arguments-parameters.yaml"},
			Name: "argument-parameters",
			ResourceReferences: []*api.ResourceReference{
				{
					Key: &api.ResourceKey{
						Id:   "ns1",
						Type: api.ResourceType_NAMESPACE,
					},
				},
			},
		},
	})
	assert.Nil(t, err)
	assert.NotNil(t, pipeline)
	newPipeline, err := pipelineServer.GetPipelineByNameV1(context.Background(),
		&api.GetPipelineByNameRequest{
			Name:      pipeline.Name,
			Namespace: "ns1",
		})
	assert.Nil(t, err)
	assert.NotNil(t, newPipeline)
	assert.Equal(t, "argument-parameters", pipeline.Name)
}

func TestGetPipelineByName_Shared_OK(t *testing.T) {
	httpServer := getMockServer(t)
	// Close the server when test finishes
	defer httpServer.Close()
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager, "default")
	pipelineServer := PipelineServer{resourceManager: resourceManager, httpClient: httpServer.Client(), options: &PipelineServerOptions{CollectMetrics: false}}
	pipeline, err := pipelineServer.CreatePipelineV1(context.Background(), &api.CreatePipelineRequest{
		Pipeline: &api.Pipeline{
			Url:  &api.Url{PipelineUrl: httpServer.URL + "/arguments-parameters.yaml"},
			Name: "argument-parameters",
		},
	},
	)
	namespace := getNamespaceFromResourceReferenceV1(pipeline.GetResourceReferences())

	assert.Nil(t, err)
	assert.NotNil(t, pipeline)
	newPipeline, err := pipelineServer.GetPipelineByNameV1(context.Background(),
		&api.GetPipelineByNameRequest{
			Name:      pipeline.Name,
			Namespace: namespace,
		})
	assert.Nil(t, err)
	assert.NotNil(t, newPipeline)
	assert.Equal(t, "argument-parameters", pipeline.Name)
}

func TestGetPipelineByName_NotFound(t *testing.T) {
	httpServer := getMockServer(t)
	// Close the server when test finishes
	defer httpServer.Close()
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager, "default")
	pipelineServer := PipelineServer{resourceManager: resourceManager, httpClient: httpServer.Client(), options: &PipelineServerOptions{CollectMetrics: false}}
	_, err := pipelineServer.GetPipelineByNameV1(context.Background(),
		&api.GetPipelineByNameRequest{
			Name: "foo",
		})
	assert.EqualValues(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode(), err)
}

func TestGetPipelineByName_WrongNameSpace(t *testing.T) {
	httpServer := getMockServer(t)
	// Close the server when test finishes
	defer httpServer.Close()
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager, "default")
	pipelineServer := PipelineServer{resourceManager: resourceManager, httpClient: httpServer.Client(), options: &PipelineServerOptions{CollectMetrics: false}}
	pipeline, err := pipelineServer.CreatePipelineV1(context.Background(), &api.CreatePipelineRequest{
		Pipeline: &api.Pipeline{
			Url:         &api.Url{PipelineUrl: httpServer.URL + "/arguments-parameters.yaml"},
			Name:        "argument-parameters",
			Description: "pipeline description",
			ResourceReferences: []*api.ResourceReference{
				{
					Key: &api.ResourceKey{
						Id:   "ns1",
						Type: api.ResourceType_NAMESPACE,
					},
				},
			},
		},
	})

	assert.Nil(t, err)
	assert.NotNil(t, pipeline)
	newPipeline, err := pipelineServer.GetPipelineByNameV1(context.Background(),
		&api.GetPipelineByNameRequest{
			Name:      pipeline.Name,
			Namespace: "wrong_namespace",
		})
	assert.Nil(t, err)
	assert.Equal(t, pipeline, newPipeline)
}

func TestCreatePipelineVersionAndCheckLatestVersion(t *testing.T) {
	viper.Set(common.UpdatePipelineVersionByDefault, "false")
	defer viper.Set(common.UpdatePipelineVersionByDefault, "true")
	httpServer := getMockServer(t)
	// Close the server when test finishes
	defer httpServer.Close()
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager, "default")

	pipelineServer := PipelineServer{resourceManager: resourceManager, httpClient: httpServer.Client(), options: &PipelineServerOptions{CollectMetrics: false}}
	pipeline, err := pipelineServer.CreatePipelineV1(context.Background(), &api.CreatePipelineRequest{
		Pipeline: &api.Pipeline{
			Url:         &api.Url{PipelineUrl: httpServer.URL + "/arguments_tarball/arguments.tar.gz"},
			Name:        "argument-parameters",
			Description: "pipeline description",
		},
	})

	assert.Nil(t, err)
	assert.NotNil(t, pipeline)
	assert.NotNil(t, pipeline.DefaultVersion.Id)

	clientManager.UpdateUUID(util.NewFakeUUIDGeneratorOrFatal("123e4567-e89b-12d3-a456-526655440001", nil))
	resourceManager = resource.NewResourceManager(clientManager, "default")

	pipelineServer = PipelineServer{resourceManager: resourceManager, httpClient: httpServer.Client(), options: &PipelineServerOptions{CollectMetrics: false}}
	pipelineVersion, err := pipelineServer.CreatePipelineVersionV1(
		context.Background(), &api.CreatePipelineVersionRequest{
			Version: &api.PipelineVersion{
				PackageUrl: &api.Url{
					PipelineUrl: httpServer.URL + "/arguments-parameters.yaml",
				},
				Name: "argument-parameters-update",
				ResourceReferences: []*api.ResourceReference{
					{
						Key: &api.ResourceKey{
							Type: api.ResourceType_PIPELINE,
							Id:   pipeline.Id,
						},
						Relationship: api.Relationship_OWNER,
					},
				},
			},
		})
	assert.Nil(t, err)

	pipeline2, err := pipelineServer.GetPipelineV1(context.Background(), &api.GetPipelineRequest{Id: pipeline.Id})
	assert.Nil(t, nil)
	assert.NotNil(t, pipelineVersion.Id)
	assert.Equal(t, pipeline2.DefaultVersion.Id, pipelineVersion.Id)
	assert.NotEqual(t, pipeline2.DefaultVersion.Id, pipeline.DefaultVersion.Id)
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

func TestPipelineServer_CreatePipeline(t *testing.T) {
	httpServer := getMockServer(t)
	defer httpServer.Close()
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager, "default")
	pipelineServer := PipelineServer{resourceManager: resourceManager, httpClient: httpServer.Client(), options: &PipelineServerOptions{CollectMetrics: false}}

	type args struct {
		pipeline *model.Pipeline
	}
	tests := []struct {
		name    string
		id      string
		arg     *apiv2.Pipeline
		want    *apiv2.Pipeline
		wantErr bool
		errMsg  string
	}{
		{
			"Valid - single user",
			DefaultFakeIdOne,
			&apiv2.Pipeline{
				DisplayName: "pipeline 1",
				Namespace:   "namespace1",
			},
			&apiv2.Pipeline{
				DisplayName: "pipeline 1",
				Namespace:   "default",
			},
			false,
			"",
		},
		{
			"Valid - empty namespace",
			DefaultFakeIdTwo,
			&apiv2.Pipeline{
				DisplayName: "pipeline 2",
			},
			&apiv2.Pipeline{
				DisplayName: "pipeline 2",
				Namespace:   "default",
			},
			false,
			"",
		},
		{
			"Invalid - duplicate name",
			DefaultFakeIdThree,
			&apiv2.Pipeline{
				DisplayName: "pipeline 2",
			},
			nil,
			true,
			"The name pipeline 2 already exist. Please specify a new name",
		},
		{
			"Invalid - missing name",
			DefaultFakeIdFour,
			&apiv2.Pipeline{
				Namespace: "namespace1",
			},
			nil,
			true,
			"Failed create to a pipeline due to empty name. Please specify a valid name",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clientManager.UpdateUUID(util.NewFakeUUIDGeneratorOrFatal(tt.id, nil))
			resourceManager = resource.NewResourceManager(clientManager, "default")
			pipelineServer = PipelineServer{resourceManager: resourceManager, httpClient: httpServer.Client(), options: &PipelineServerOptions{CollectMetrics: false}}
			got, err := pipelineServer.CreatePipeline(context.Background(), &apiv2.CreatePipelineRequest{Pipeline: tt.arg})
			if tt.wantErr {
				assert.NotNil(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.Nil(t, err)
				assert.NotEmpty(t, got.GetPipelineId())
				assert.NotEmpty(t, got.GetCreatedAt())
				tt.want.CreatedAt = got.GetCreatedAt()
				tt.want.PipelineId = got.GetPipelineId()
			}
			assert.Equal(t, tt.want, got)
		})
	}
}
