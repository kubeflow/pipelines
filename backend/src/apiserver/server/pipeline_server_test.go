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
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	apiv2 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/timestamppb"
	authorizationv1 "k8s.io/api/authorization/v1"
)

func TestMain(m *testing.M) {
	// Disable URL validation for tests (allows mock server URLs)
	viper.Set(common.PipelineURLValidationEnabled, "false")
	os.Exit(m.Run())
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
	largePipelineURL := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/test_data/sdk_compiled_pipelines/valid/xgboost_sample_pipeline.yaml", repo, branch)
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
				Parameters:     "[]",
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
				Parameters:     "[]",
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

func TestRecoverClearTagsIntent(t *testing.T) {
	tests := []struct {
		name    string
		ctx     context.Context
		tags    map[string]string
		wantNil bool
		wantLen int
	}{
		{
			name:    "non-nil tags pass through unchanged",
			ctx:     context.Background(),
			tags:    map[string]string{"k": "v"},
			wantNil: false,
			wantLen: 1,
		},
		{
			name:    "nil tags without metadata stay nil",
			ctx:     context.Background(),
			tags:    nil,
			wantNil: true,
		},
		{
			name: "nil tags with clear header become empty map",
			ctx: metadata.NewIncomingContext(
				context.Background(),
				metadata.Pairs(common.ClearTagsMetadataKey, "true"),
			),
			tags:    nil,
			wantNil: false,
			wantLen: 0,
		},
		{
			name: "nil tags with wrong header value stay nil",
			ctx: metadata.NewIncomingContext(
				context.Background(),
				metadata.Pairs(common.ClearTagsMetadataKey, "false"),
			),
			tags:    nil,
			wantNil: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := recoverClearTagsIntent(tt.ctx, tt.tags)
			if tt.wantNil {
				assert.Nil(t, result)
			} else {
				assert.NotNil(t, result)
				assert.Len(t, result, tt.wantLen)
			}
		})
	}
}

func TestExtractTagFiltersFromFilterSpec(t *testing.T) {
	// Helper to build a URL-encoded filter JSON with given predicates.
	makeFilter := func(predicates ...string) string {
		return url.QueryEscape(fmt.Sprintf(`{"predicates":[%s]}`, strings.Join(predicates, ",")))
	}
	tagPredicate := func(key, value string) string {
		return fmt.Sprintf(`{"key":"%s","operation":"EQUALS","string_value":"%s"}`, key, value)
	}

	tests := []struct {
		name       string
		filterSpec string
		wantFilter string // expected remaining filter (empty means no remaining)
		wantTags   map[string]string
		wantErr    bool
		errMsg     string
	}{
		{
			name:       "empty filter spec",
			filterSpec: "",
			wantFilter: "",
			wantTags:   nil,
		},
		{
			name:       "no tag predicates",
			filterSpec: makeFilter(`{"key":"name","operation":"EQUALS","string_value":"my-pipeline"}`),
			wantTags:   nil,
		},
		{
			name:       "single tag predicate",
			filterSpec: makeFilter(tagPredicate("tags.env", "prod")),
			wantFilter: "",
			wantTags:   map[string]string{"env": "prod"},
		},
		{
			name:       "multiple tag predicates",
			filterSpec: makeFilter(tagPredicate("tags.env", "prod"), tagPredicate("tags.team", "ml")),
			wantFilter: "",
			wantTags:   map[string]string{"env": "prod", "team": "ml"},
		},
		{
			name:       "mixed tag and non-tag predicates",
			filterSpec: makeFilter(`{"key":"name","operation":"EQUALS","string_value":"test"}`, tagPredicate("tags.env", "prod")),
			wantTags:   map[string]string{"env": "prod"},
		},
		{
			name:       "tag predicate with non-EQUALS operation",
			filterSpec: makeFilter(`{"key":"tags.env","operation":"NOT_EQUALS","string_value":"prod"}`),
			wantErr:    true,
			errMsg:     "only EQUALS operation is supported",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			remainingFilter, tagFilters, err := extractTagFiltersFromFilterSpec(tt.filterSpec)
			if tt.wantErr {
				assert.NotNil(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				return
			}
			assert.Nil(t, err)
			if tt.wantTags == nil {
				assert.Nil(t, tagFilters)
			} else {
				assert.Equal(t, tt.wantTags, tagFilters)
			}
			if tt.wantFilter != "" {
				assert.NotEmpty(t, remainingFilter)
			}
			// For the "no tag predicates" case, the original filter should be returned unchanged
			if tt.wantTags == nil && tt.filterSpec != "" {
				assert.Equal(t, tt.filterSpec, remainingFilter)
			}
		})
	}
}

func TestCanAccessPipeline_SharedPipeline_ReadAllowed(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")

	initEnvVars()
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})
	defer clientManager.Close()

	// Create a shared pipeline (empty namespace)
	pipeline, err := resourceManager.CreatePipeline(&model.Pipeline{
		Name:      "shared-pipeline",
		Namespace: "",
	})
	assert.Nil(t, err)

	pipelineServer := createPipelineServer(resourceManager, nil)

	md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + "user@google.com"})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	// GET on shared pipeline should be allowed
	err = pipelineServer.canAccessPipeline(ctx, pipeline.UUID, &authorizationv1.ResourceAttributes{Verb: common.RbacResourceVerbGet})
	assert.Nil(t, err)

	// LIST on shared pipeline should be allowed
	err = pipelineServer.canAccessPipeline(ctx, "", &authorizationv1.ResourceAttributes{Verb: common.RbacResourceVerbList})
	assert.Nil(t, err)
}

func TestCanAccessPipeline_SharedPipeline_WriteUnauthorized(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")

	initEnvVars()
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	clientManager.SubjectAccessReviewClientFake = client.NewFakeSubjectAccessReviewClientUnauthorized()
	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})
	defer clientManager.Close()

	// Create a shared pipeline (empty namespace)
	pipeline, err := resourceManager.CreatePipeline(&model.Pipeline{
		Name:      "shared-pipeline",
		Namespace: "",
	})
	assert.Nil(t, err)

	pipelineServer := createPipelineServer(resourceManager, nil)

	md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + "user@google.com"})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	// DELETE on shared pipeline should be rejected for unauthorized user
	err = pipelineServer.canAccessPipeline(ctx, pipeline.UUID, &authorizationv1.ResourceAttributes{Verb: common.RbacResourceVerbDelete})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Failed to access shared pipeline")

	// UPDATE on shared pipeline should be rejected for unauthorized user
	err = pipelineServer.canAccessPipeline(ctx, pipeline.UUID, &authorizationv1.ResourceAttributes{Verb: common.RbacResourceVerbUpdate})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Failed to access shared pipeline")

	// CREATE on shared pipeline should be rejected for unauthorized user
	err = pipelineServer.canAccessPipeline(ctx, "", &authorizationv1.ResourceAttributes{Verb: common.RbacResourceVerbCreate})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Failed to access shared pipeline")
}

func TestCanAccessPipeline_SharedPipeline_WriteAuthorized(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")

	initEnvVars()
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})
	defer clientManager.Close()

	// Create a shared pipeline (empty namespace)
	pipeline, err := resourceManager.CreatePipeline(&model.Pipeline{
		Name:      "shared-pipeline",
		Namespace: "",
	})
	assert.Nil(t, err)

	pipelineServer := createPipelineServer(resourceManager, nil)

	md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + "user@google.com"})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	// DELETE on shared pipeline should succeed for authorized user
	err = pipelineServer.canAccessPipeline(ctx, pipeline.UUID, &authorizationv1.ResourceAttributes{Verb: common.RbacResourceVerbDelete})
	assert.Nil(t, err)

	// UPDATE on shared pipeline should succeed for authorized user
	err = pipelineServer.canAccessPipeline(ctx, pipeline.UUID, &authorizationv1.ResourceAttributes{Verb: common.RbacResourceVerbUpdate})
	assert.Nil(t, err)
}

func TestCanAccessPipeline_SharedPipeline_ReadAllowed_EvenWhenUnauthorized(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")

	initEnvVars()
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	clientManager.SubjectAccessReviewClientFake = client.NewFakeSubjectAccessReviewClientUnauthorized()
	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})
	defer clientManager.Close()

	// Create a shared pipeline (empty namespace)
	pipeline, err := resourceManager.CreatePipeline(&model.Pipeline{
		Name:      "shared-pipeline",
		Namespace: "",
	})
	assert.Nil(t, err)

	pipelineServer := createPipelineServer(resourceManager, nil)

	md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + "user@google.com"})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	// GET on shared pipeline should still be allowed even for unauthorized user
	err = pipelineServer.canAccessPipeline(ctx, pipeline.UUID, &authorizationv1.ResourceAttributes{Verb: common.RbacResourceVerbGet})
	assert.Nil(t, err)

	// LIST on shared pipeline should still be allowed even for unauthorized user
	err = pipelineServer.canAccessPipeline(ctx, "", &authorizationv1.ResourceAttributes{Verb: common.RbacResourceVerbList})
	assert.Nil(t, err)
}

type errorRoundTripper struct {
	err error
}

func (rt errorRoundTripper) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, rt.err
}
