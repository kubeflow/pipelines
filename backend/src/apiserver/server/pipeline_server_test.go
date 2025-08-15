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
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
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
	"google.golang.org/protobuf/types/known/timestamppb"
)

func createPipelineServerV1(resourceManager *resource.ResourceManager, httpClient *http.Client) *PipelineServerV1 {
	return &PipelineServerV1{
		BasePipelineServer: &BasePipelineServer{
			resourceManager: resourceManager, httpClient: httpClient, options: &PipelineServerOptions{CollectMetrics: false},
		},
	}
}

func createPipelineServer(resourceManager *resource.ResourceManager, httpClient *http.Client) *PipelineServer {
	return &PipelineServer{
		BasePipelineServer: &BasePipelineServer{
			resourceManager: resourceManager, httpClient: httpClient, options: &PipelineServerOptions{CollectMetrics: false},
		},
	}
}

func setupLargePipelineURL() string {
	// Set up the environment variables for the pipeline URL.
	// The URL points to a sample pipeline YAML file in the Kubeflow Pipelines repository.
	// The branch and repo can be overridden by environment variables for testing purposes.
	branch := os.Getenv("GIT_BRANCH")
	repo := os.Getenv("GIT_REPO")
	if repo == "" {
		repo = "kubeflow/pipelines"
	}
	if branch == "" {
		branch = "master"
	}
	largePipelineURL := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/sdk/python/test_data/pipelines/xgboost_sample_pipeline.yaml", repo, branch)
	return largePipelineURL
}
func TestBuildPipelineName_QueryStringNotEmpty(t *testing.T) {
	pipelineName := buildPipelineName("pipeline one", "", "file one")
	assert.Equal(t, "pipeline one", pipelineName)
}

func TestBuildPipelineName(t *testing.T) {
	pipelineName := buildPipelineName("", "", "file one")
	assert.Equal(t, "file one", pipelineName)
}

func TestBuildPipelineName_empty(t *testing.T) {
	newName := buildPipelineName("", "", "")
	assert.Empty(t, newName)
}

func TestBuildPipelineName_display_name(t *testing.T) {
	newName := buildPipelineName("", "My display name", "filename")
	assert.Equal(t, "My display name", newName)
}

func TestCreatePipelineV1_YAML(t *testing.T) {
	httpServer := getMockServer(t)
	// Close the server when test finishes
	defer httpServer.Close()

	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})

	pipelineServer := createPipelineServerV1(resourceManager, httpServer.Client())
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
	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})

	pipelineServer := createPipelineServerV1(resourceManager, httpServer.Client())
	pipeline, err := pipelineServer.CreatePipelineV1(context.Background(), &api.CreatePipelineRequest{
		Pipeline: &api.Pipeline{
			Url:         &api.Url{PipelineUrl: setupLargePipelineURL()},
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
	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})

	pipelineServer := createPipelineServerV1(resourceManager, httpServer.Client())
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
	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})

	pipelineServer := createPipelineServerV1(resourceManager, httpServer.Client())
	createdPipeline, err := pipelineServer.CreatePipelineV1(
		context.Background(), &api.CreatePipelineRequest{
			Pipeline: &api.Pipeline{
				Url:  &api.Url{PipelineUrl: httpServer.URL + "/invalid-workflow.yaml"},
				Name: "argument-parameters",
			},
		},
	)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "pipeline spec is invalid")
	assert.Nil(t, createdPipeline)
}

func TestCreatePipelineV1_InvalidURL(t *testing.T) {
	httpServer := getBadMockServer()
	// Close the server when test finishes
	defer httpServer.Close()

	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})

	pipelineServer := createPipelineServerV1(resourceManager, httpServer.Client())
	createdPipeline, err := pipelineServer.CreatePipelineV1(
		context.Background(), &api.CreatePipelineRequest{
			Pipeline: &api.Pipeline{
				Url:  &api.Url{PipelineUrl: httpServer.URL + "/invalid-workflow.yaml"},
				Name: "argument-parameters",
			},
		},
	)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "error fetching pipeline spec")
	assert.Nil(t, createdPipeline)
}

func TestCreatePipelineV1_MissingUrl(t *testing.T) {
	httpServer := getMockServer(t)
	// Close the server when test finishes
	defer httpServer.Close()

	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})

	pipelineServer := createPipelineServerV1(resourceManager, httpServer.Client())
	createdPipeline, err := pipelineServer.CreatePipelineV1(
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
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "invalid pipeline spec URL")
	assert.Nil(t, createdPipeline)
}

func TestCreatePipelineV1_ExistingPipeline(t *testing.T) {
	httpServer := getMockServer(t)
	// Close the server when test finishes
	defer httpServer.Close()

	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})

	pipelineServer := createPipelineServerV1(resourceManager, httpServer.Client())
	pipelineServer.CreatePipelineV1(
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
	createdPipeline, err := pipelineServer.CreatePipelineV1(
		context.Background(), &api.CreatePipelineRequest{
			Pipeline: &api.Pipeline{
				Url:  &api.Url{PipelineUrl: httpServer.URL + "/xgboost_sample_pipeline.yaml"},
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
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Failed to create a new pipeline. The name argument-parameters already exists")
	assert.Nil(t, createdPipeline)
}

func TestCreatePipelineVersionV1_NameTooLong(t *testing.T) {
	httpServer := getMockServer(t)
	defer httpServer.Close()
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})
	pipelineServer := createPipelineServerV1(resourceManager, httpServer.Client())

	longName := strings.Repeat("x", 192) // max valid length is 191
	_, err := pipelineServer.CreatePipelineVersionV1(
		context.Background(),
		&api.CreatePipelineVersionRequest{
			Version: &api.PipelineVersion{
				Name:       longName,
				PackageUrl: &api.Url{PipelineUrl: httpServer.URL + "/arguments-parameters.yaml"},
				ResourceReferences: []*api.ResourceReference{{
					Key:          &api.ResourceKey{Type: api.ResourceType_PIPELINE, Id: "pipeline"},
					Relationship: api.Relationship_OWNER,
				}},
			},
		},
	)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "PipelineVersion.Name length cannot exceed")
}

func TestCreatePipelineVersionV1_YAML(t *testing.T) {
	httpServer := getMockServer(t)
	// Close the server when test finishes
	defer httpServer.Close()

	clientManager := resource.NewFakeClientManagerOrFatal(
		util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})

	pipelineServer := createPipelineServerV1(resourceManager, httpServer.Client())
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
	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})

	pipelineServer := createPipelineServerV1(resourceManager, httpServer.Client())
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
	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})

	pipelineServer := createPipelineServerV1(resourceManager, httpServer.Client())
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
	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})

	pipelineServer := createPipelineServerV1(resourceManager, httpServer.Client())
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
	assert.NotNil(t, err)
	assert.Equal(t, codes.InvalidArgument, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "Request returned 404 Not Found")
}

func TestListPipelineVersion_NoResourceKey(t *testing.T) {
	httpServer := getMockServer(t)
	// Close the server when test finishes
	defer httpServer.Close()

	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})

	pipelineServer := createPipelineServerV1(resourceManager, httpServer.Client())

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
	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})

	pipelineServer := createPipelineServerV1(resourceManager, httpServer.Client())
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
	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})
	pipelineServer := createPipelineServerV1(resourceManager, httpServer.Client())
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
	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})
	pipelineServer := createPipelineServerV1(resourceManager, httpServer.Client())
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
	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})
	pipelineServer := createPipelineServerV1(resourceManager, httpServer.Client())
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
	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})
	pipelineServer := createPipelineServerV1(resourceManager, httpServer.Client())
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
	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})

	pipelineServer := createPipelineServerV1(resourceManager, httpServer.Client())
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
	resourceManager = resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})

	pipelineServer = createPipelineServerV1(resourceManager, httpServer.Client())
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

	pipeline2, _ := pipelineServer.GetPipelineV1(context.Background(), &api.GetPipelineRequest{Id: pipeline.Id})
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
		bytes, err := io.ReadAll(file)
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
	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})
	pipelineServer := createPipelineServer(resourceManager, httpServer.Client())

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
				Name:        "Pipeline #1",
				DisplayName: "Pipeline #1",
				Namespace:   "namespace1",
			},
			&apiv2.Pipeline{
				Name:        "Pipeline #1",
				DisplayName: "Pipeline #1",
				Namespace:   "",
			},
			false,
			"",
		},
		{
			"Valid - empty namespace",
			DefaultFakeIdTwo,
			&apiv2.Pipeline{
				Name:        "Pipeline 2",
				DisplayName: "Pipeline 2",
			},
			&apiv2.Pipeline{
				Name:        "Pipeline 2",
				DisplayName: "Pipeline 2",
				Namespace:   "",
			},
			false,
			"",
		},
		{
			"Invalid - duplicate name",
			DefaultFakeIdThree,
			&apiv2.Pipeline{
				DisplayName: "Pipeline 2",
			},
			nil,
			true,
			"The name Pipeline 2 already exist. Please specify a new name",
		},
		{
			"Invalid - missing name",
			DefaultFakeIdFour,
			&apiv2.Pipeline{
				Namespace: "namespace1",
			},
			nil,
			true,
			"name is required",
		},
		{
			name: "Invalid - name too long",
			id:   DefaultFakeIdOne,
			arg: &apiv2.Pipeline{
				Name:        strings.Repeat("a", 129),
				DisplayName: strings.Repeat("a", 129),
				Namespace:   "",
			},
			want:    nil,
			wantErr: true,
			errMsg:  "Pipeline.Name length cannot exceed 128",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clientManager.UpdateUUID(util.NewFakeUUIDGeneratorOrFatal(tt.id, nil))
			resourceManager = resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})
			pipelineServer = createPipelineServer(resourceManager, httpServer.Client())
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

func TestPipelineServer_CreatePipelineAndVersion_v2(t *testing.T) {
	httpServer := getMockServer(t)
	defer httpServer.Close()
	tests := []struct {
		name    string
		request *apiv2.CreatePipelineAndVersionRequest
		want    *apiv2.Pipeline
		wantPv  *model.PipelineVersion
		wantErr bool
		errMsg  string
	}{
		{
			name: "Invalid - name too long",
			request: &apiv2.CreatePipelineAndVersionRequest{
				Pipeline: &apiv2.Pipeline{
					DisplayName: strings.Repeat("a", 129),
					Description: "pipeline description",
					Namespace:   "",
				},
				PipelineVersion: &apiv2.PipelineVersion{
					PackageUrl: &apiv2.Url{
						PipelineUrl: httpServer.URL + "/arguments-parameters.yaml",
					},
				},
			},
			want:    nil,
			wantPv:  nil,
			wantErr: true,
			errMsg:  "Pipeline.Name length cannot exceed 128",
		},
		{
			"Valid - yaml",
			&apiv2.CreatePipelineAndVersionRequest{
				Pipeline: &apiv2.Pipeline{
					DisplayName: "User's pipeline 1",
					Description: "Pipeline built by a user",
					Namespace:   "",
				},
				PipelineVersion: &apiv2.PipelineVersion{
					PackageUrl: &apiv2.Url{
						PipelineUrl: httpServer.URL + "/arguments-parameters.yaml",
					},
				},
			},
			&apiv2.Pipeline{
				PipelineId:  DefaultFakeUUID,
				CreatedAt:   &timestamppb.Timestamp{Seconds: 1},
				Name:        "User's pipeline 1",
				DisplayName: "User's pipeline 1",
				Description: "Pipeline built by a user",
				Namespace:   "",
			},
			&model.PipelineVersion{
				UUID:           DefaultFakeUUID,
				CreatedAtInSec: 2,
				PipelineId:     DefaultFakeUUID,
				Name:           "User's pipeline 1",
				DisplayName:    "User's pipeline 1",
				Description:    "Pipeline built by a user",
				Parameters:     "[{\"name\":\"param1\",\"value\":\"hello\"},{\"name\":\"param2\"}]",
				Status:         model.PipelineVersionReady,
			},
			false,
			"",
		},
		{
			"Valid - large yaml",
			&apiv2.CreatePipelineAndVersionRequest{
				Pipeline: &apiv2.Pipeline{
					DisplayName: "User's pipeline 1",
					Description: "Pipeline built by a user",
					Namespace:   "",
				},
				PipelineVersion: &apiv2.PipelineVersion{
					PackageUrl: &apiv2.Url{
						PipelineUrl: setupLargePipelineURL(),
					},
				},
			},
			&apiv2.Pipeline{
				PipelineId:  DefaultFakeUUID,
				CreatedAt:   &timestamppb.Timestamp{Seconds: 1},
				Name:        "User's pipeline 1",
				DisplayName: "User's pipeline 1",
				Description: "Pipeline built by a user",
				Namespace:   "",
			},
			&model.PipelineVersion{
				UUID:           DefaultFakeUUID,
				CreatedAtInSec: 2,
				PipelineId:     DefaultFakeUUID,
				Name:           "User's pipeline 1",
				DisplayName:    "User's pipeline 1",
				Parameters:     "[]",
				Description:    "Pipeline built by a user",
				Status:         model.PipelineVersionReady,
			},
			false,
			"",
		},
		{
			"Valid - tarball",
			&apiv2.CreatePipelineAndVersionRequest{
				Pipeline: &apiv2.Pipeline{
					DisplayName: "User's pipeline 1",
					Description: "Pipeline built by a user",
					Namespace:   "",
				},
				PipelineVersion: &apiv2.PipelineVersion{
					PackageUrl: &apiv2.Url{
						PipelineUrl: httpServer.URL + "/arguments_tarball/arguments.tar.gz",
					},
				},
			},
			&apiv2.Pipeline{
				PipelineId:  DefaultFakeUUID,
				CreatedAt:   &timestamppb.Timestamp{Seconds: 1},
				Name:        "User's pipeline 1",
				DisplayName: "User's pipeline 1",
				Description: "Pipeline built by a user",
				Namespace:   "",
			},
			&model.PipelineVersion{
				UUID:           DefaultFakeUUID,
				CreatedAtInSec: 2,
				PipelineId:     DefaultFakeUUID,
				Name:           "User's pipeline 1",
				DisplayName:    "User's pipeline 1",
				Parameters:     "[{\"name\":\"param1\",\"value\":\"hello\"},{\"name\":\"param2\"}]",
				Description:    "Pipeline built by a user",
				Status:         model.PipelineVersionReady,
			},
			false,
			"",
		},
		{
			"Invalid - wrong yaml",
			&apiv2.CreatePipelineAndVersionRequest{
				Pipeline: &apiv2.Pipeline{
					DisplayName: "User's pipeline 1",
					Description: "Pipeline built by a user",
					Namespace:   "",
				},
				PipelineVersion: &apiv2.PipelineVersion{
					PackageUrl: &apiv2.Url{
						PipelineUrl: httpServer.URL + "/invalid-workflow.yaml",
					},
				},
			},
			nil,
			nil,
			true,
			"pipeline spec is invalid",
		},
	}
	for _, tt := range tests {
		clientManager := resource.NewFakeClientManagerOrFatal(
			util.NewFakeTimeForEpoch())
		resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})
		pipelineServer := createPipelineServer(resourceManager, httpServer.Client())
		t.Run(tt.name, func(t *testing.T) {
			got, err := pipelineServer.CreatePipelineAndVersion(context.Background(), tt.request)
			if tt.wantErr {
				assert.NotNil(t, err)
				assert.Nil(t, got)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.Nil(t, err)
				assert.Equal(t, tt.want, got)
				pv, err := resourceManager.GetLatestPipelineVersion(got.GetPipelineId())
				assert.Nil(t, err)
				assert.NotEmpty(t, pv.PipelineSpec)
				assert.NotEmpty(t, pv.PipelineSpecURI)
				tt.wantPv.PipelineSpecURI = pv.PipelineSpecURI
				tt.wantPv.PipelineSpec = pv.PipelineSpec
				assert.Equal(t, tt.wantPv, pv)
			}
		})
	}
}
