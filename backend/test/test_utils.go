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

package test

import (
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/cenkalti/backoff"
	api "github.com/kubeflow/pipelines/backend/api/v1beta1/go_client"
	experimentparams "github.com/kubeflow/pipelines/backend/api/v1beta1/go_http_client/experiment_client/experiment_service"
	"github.com/kubeflow/pipelines/backend/api/v1beta1/go_http_client/experiment_model"
	jobparams "github.com/kubeflow/pipelines/backend/api/v1beta1/go_http_client/job_client/job_service"
	"github.com/kubeflow/pipelines/backend/api/v1beta1/go_http_client/job_model"
	pipelineparams "github.com/kubeflow/pipelines/backend/api/v1beta1/go_http_client/pipeline_client/pipeline_service"
	"github.com/kubeflow/pipelines/backend/api/v1beta1/go_http_client/pipeline_model"
	runparams "github.com/kubeflow/pipelines/backend/api/v1beta1/go_http_client/run_client/run_service"
	"github.com/kubeflow/pipelines/backend/api/v1beta1/go_http_client/run_model"
	api_server "github.com/kubeflow/pipelines/backend/src/common/client/api_server/v1"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

func WaitForReady(initializeTimeout time.Duration) error {
	operation := func() error {
		response, err := http.Get("http://localhost:8888/apis/v1beta1/healthz")
		if err != nil {
			return err
		}

		// If we get a 503 service unavailable, it's a non-retriable error.
		if response.StatusCode == 503 {
			return backoff.Permanent(errors.Wrapf(
				err, "Waiting for ml pipeline API server failed with non retriable error."))
		}

		return nil
	}

	b := backoff.NewExponentialBackOff()
	b.MaxElapsedTime = initializeTimeout
	err := backoff.Retry(operation, b)
	return errors.Wrapf(err, "Waiting for ml pipeline API server failed after all attempts.")
}

func GetClientConfig(namespace string) clientcmd.ClientConfig {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	loadingRules.DefaultClientConfig = &clientcmd.DefaultClientConfig
	overrides := clientcmd.ConfigOverrides{Context: clientcmdapi.Context{Namespace: namespace}}
	return clientcmd.NewInteractiveDeferredLoadingClientConfig(loadingRules,
		&overrides, os.Stdin)
}

func DeleteAllPipelines(client *api_server.PipelineClient, t *testing.T) {
	pipelines, _, _, err := ListPipelines(client)
	assert.Nil(t, err)
	deletedPipelines := make(map[string]bool)
	for _, p := range pipelines {
		deletedPipelines[p.ID] = false
	}
	for pId, isRemoved := range deletedPipelines {
		if !isRemoved {
			DeleteAllPipelineVersions(client, t, pId)
			deletedPipelines[pId] = true
		}
		assert.Nil(t, client.Delete(&pipelineparams.PipelineServiceDeletePipelineV1Params{ID: pId}))
	}
	for _, isRemoved := range deletedPipelines {
		assert.True(t, isRemoved)
	}
}

func DeleteAllPipelineVersions(client *api_server.PipelineClient, t *testing.T, pipelineId string) {
	pipelineVersions, _, _, err := ListPipelineVersions(client, pipelineId)
	assert.Nil(t, err)
	for _, pv := range pipelineVersions {
		assert.Nil(t, client.DeletePipelineVersion(&pipelineparams.PipelineServiceDeletePipelineVersionV1Params{VersionID: pv.ID}))
	}
}

func DeleteAllExperiments(client *api_server.ExperimentClient, namespace string, t *testing.T) {
	experiments, _, _, err := ListAllExperiment(client, namespace)
	assert.Nil(t, err)
	for _, e := range experiments {
		if e.Name != "Default" {
			assert.Nil(t, client.Delete(&experimentparams.ExperimentServiceDeleteExperimentV1Params{ID: e.ID}))
		}
	}
}

func DeleteAllRuns(client *api_server.RunClient, namespace string, t *testing.T) {
	runs, _, _, err := ListAllRuns(client, namespace)
	assert.Nil(t, err)
	for _, r := range runs {
		assert.Nil(t, client.Delete(&runparams.RunServiceDeleteRunV1Params{ID: r.ID}))
	}
}

func DeleteAllJobs(client *api_server.JobClient, namespace string, t *testing.T) {
	jobs, _, _, err := ListAllJobs(client, namespace)
	assert.Nil(t, err)
	for _, j := range jobs {
		assert.Nil(t, client.Delete(&jobparams.JobServiceDeleteJobParams{ID: j.ID}))
	}
}

func GetExperimentIDFromV1beta1ResourceReferences(resourceRefs []*run_model.APIResourceReference) string {
	experimentID := ""
	for _, resourceRef := range resourceRefs {
		if resourceRef.Key.Type == run_model.APIResourceTypeEXPERIMENT {
			experimentID = resourceRef.Key.ID
			break
		}
	}
	return experimentID
}

func ListPipelineVersions(client *api_server.PipelineClient, pipelineId string) (
	[]*pipeline_model.APIPipelineVersion, int, string, error,
) {
	parameters := &pipelineparams.PipelineServiceListPipelineVersionsV1Params{}
	parameters.WithResourceKeyType(util.StringPointer(api.ResourceType_name[int32(api.ResourceType_PIPELINE)]))
	parameters.SetResourceKeyID(&pipelineId)
	return client.ListPipelineVersions(parameters)
}

func ListPipelines(client *api_server.PipelineClient) (
	[]*pipeline_model.APIPipeline, int, string, error,
) {
	parameters := &pipelineparams.PipelineServiceListPipelinesV1Params{}
	return client.List(parameters)
}

func ListAllExperiment(client *api_server.ExperimentClient, namespace string) ([]*experiment_model.APIExperiment, int, string, error) {
	return ListExperiment(client, &experimentparams.ExperimentServiceListExperimentsV1Params{}, namespace)
}

func ListExperiment(client *api_server.ExperimentClient, parameters *experimentparams.ExperimentServiceListExperimentsV1Params, namespace string) ([]*experiment_model.APIExperiment, int, string, error) {
	if namespace != "" {
		parameters.SetResourceReferenceKeyType(util.StringPointer(api.ResourceType_name[int32(api.ResourceType_NAMESPACE)]))
		parameters.SetResourceReferenceKeyID(&namespace)
	}
	return client.List(parameters)
}

func ListAllRuns(client *api_server.RunClient, namespace string) ([]*run_model.APIRun, int, string, error) {
	parameters := &runparams.RunServiceListRunsV1Params{}
	return ListRuns(client, parameters, namespace)
}

func ListRuns(client *api_server.RunClient, parameters *runparams.RunServiceListRunsV1Params, namespace string) ([]*run_model.APIRun, int, string, error) {
	if namespace != "" {
		parameters.SetResourceReferenceKeyType(util.StringPointer(api.ResourceType_name[int32(api.ResourceType_NAMESPACE)]))
		parameters.SetResourceReferenceKeyID(&namespace)
	}

	return client.List(parameters)
}

func ListAllJobs(client *api_server.JobClient, namespace string) ([]*job_model.APIJob, int, string, error) {
	return ListJobs(client, &jobparams.JobServiceListJobsParams{}, namespace)
}

func ListJobs(client *api_server.JobClient, parameters *jobparams.JobServiceListJobsParams, namespace string) ([]*job_model.APIJob, int, string, error) {
	if namespace != "" {
		parameters.SetResourceReferenceKeyType(util.StringPointer(api.ResourceType_name[int32(api.ResourceType_NAMESPACE)]))
		parameters.SetResourceReferenceKeyID(&namespace)
	}

	return client.List(parameters)
}

func GetExperiment(name string, description string, namespace string) *experiment_model.APIExperiment {
	experiment := &experiment_model.APIExperiment{
		Name:        name,
		Description: description,
	}

	if namespace != "" {
		experiment.ResourceReferences = []*experiment_model.APIResourceReference{
			{
				Key: &experiment_model.APIResourceKey{
					Type: experiment_model.APIResourceTypeNAMESPACE,
					ID:   namespace,
				},
				Relationship: experiment_model.APIRelationshipOWNER,
			},
		}
	}

	return experiment
}

func GetDefaultPipelineRunnerServiceAccount(isKubeflowMode bool) string {
	if isKubeflowMode {
		return "default-editor"
	} else {
		return "pipeline-runner"
	}
}

// Checks if the resRefs contain the targets. Ignores resource's Name. Ignores duplicate or nil entries in targets.
func VerifyExperimentResourceReferences(resRefs []*experiment_model.APIResourceReference, targets []*experiment_model.APIResourceReference) bool {
	matches := 0
	for _, target := range targets {
		if target.Key == nil {
			matches++
			continue
		}
		for _, resRef := range resRefs {
			if resRef.Key.ID == target.Key.ID && resRef.Key.Type == target.Key.Type && resRef.Relationship == target.Relationship {
				matches++
				break
			}
		}
	}
	return matches == len(targets)
}

// Checks if the resRefs contain the targets. Ignores resource's Name. Ignores duplicate or nil entries in targets.
func VerifyJobResourceReferences(resRefs []*job_model.APIResourceReference, targets []*job_model.APIResourceReference) bool {
	matches := 0
	for _, target := range targets {
		for _, resRef := range resRefs {
			if target.Key == nil {
				matches++
				break
			}
			if resRef.Key != nil {
				if resRef.Key.ID == target.Key.ID && resRef.Key.Type == target.Key.Type && resRef.Relationship == target.Relationship {
					matches++
					break
				}
			}
		}
	}
	return matches == len(targets)
}

// Checks if the resRefs contain the targets. Ignores resource's Name. Ignores duplicate or nil entries in targets.
func VerifyPipelineResourceReferences(resRefs []*pipeline_model.APIResourceReference, targets []*pipeline_model.APIResourceReference) bool {
	matches := 0
	for _, target := range targets {
		for _, resRef := range resRefs {
			if target.Key == nil {
				matches++
				break
			}
			if resRef.Key != nil {
				if resRef.Key.ID == target.Key.ID && resRef.Key.Type == target.Key.Type && resRef.Relationship == target.Relationship {
					matches++
					break
				}
			}
		}
	}
	return matches == len(targets)
}

// Checks if the resRefs contain the targets. Ignores resource's Name. Ignores duplicate or nil entries in targets.
func VerifyRunResourceReferences(resRefs []*run_model.APIResourceReference, targets []*run_model.APIResourceReference) bool {
	matches := 0
	for _, target := range targets {
		for _, resRef := range resRefs {
			if target.Key == nil {
				matches++
				break
			}
			if resRef.Key != nil {
				if resRef.Key.ID == target.Key.ID && resRef.Key.Type == target.Key.Type && resRef.Relationship == target.Relationship {
					matches++
					break
				}
			}
		}
	}
	return matches == len(targets)
}
