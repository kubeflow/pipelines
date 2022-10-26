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
	"fmt"
	"os"
	"time"

	"testing"

	"net/http"

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
	"github.com/kubeflow/pipelines/backend/src/common/client/api_server"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

func WaitForReady(namespace string, initializeTimeout time.Duration) error {
	var operation = func() error {
		response, err := http.Get(fmt.Sprintf("http://ml-pipeline.%s.svc.cluster.local:8888/apis/v1beta1/healthz", namespace))
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
	for _, p := range pipelines {
		assert.Nil(t, client.Delete(&pipelineparams.DeletePipelineParams{ID: p.ID}))
	}
}

func DeleteAllExperiments(client *api_server.ExperimentClient, namespace string, t *testing.T) {
	experiments, _, _, err := ListAllExperiment(client, namespace)
	assert.Nil(t, err)
	for _, e := range experiments {
		assert.Nil(t, client.Delete(&experimentparams.DeleteExperimentParams{ID: e.ID}))
	}
}

func DeleteAllRuns(client *api_server.RunClient, namespace string, t *testing.T) {
	runs, _, _, err := ListAllRuns(client, namespace)
	assert.Nil(t, err)
	for _, r := range runs {
		assert.Nil(t, client.Delete(&runparams.DeleteRunParams{ID: r.ID}))
	}
}

func DeleteAllJobs(client *api_server.JobClient, namespace string, t *testing.T) {
	jobs, _, _, err := ListAllJobs(client, namespace)
	assert.Nil(t, err)
	for _, j := range jobs {
		assert.Nil(t, client.Delete(&jobparams.DeleteJobParams{ID: j.ID}))
	}
}

func GetExperimentIDFromAPIResourceReferences(resourceRefs []*run_model.V1beta1ResourceReference) string {
	experimentID := ""
	for _, resourceRef := range resourceRefs {
		if resourceRef.Key.Type == run_model.V1beta1ResourceTypeEXPERIMENT {
			experimentID = resourceRef.Key.ID
			break
		}
	}
	return experimentID
}

func ListPipelines(client *api_server.PipelineClient) (
	[]*pipeline_model.V1beta1Pipeline, int, string, error) {
	parameters := &pipelineparams.ListPipelinesParams{}

	return client.List(parameters)
}

func ListAllExperiment(client *api_server.ExperimentClient, namespace string) ([]*experiment_model.V1beta1Experiment, int, string, error) {
	return ListExperiment(client, &experimentparams.ListExperimentParams{}, namespace)
}

func ListExperiment(client *api_server.ExperimentClient, parameters *experimentparams.ListExperimentParams, namespace string) ([]*experiment_model.V1beta1Experiment, int, string, error) {
	if namespace != "" {
		parameters.SetResourceReferenceKeyType(util.StringPointer(api.ResourceType_name[int32(api.ResourceType_NAMESPACE)]))
		parameters.SetResourceReferenceKeyID(&namespace)
	}

	return client.List(parameters)
}

func ListAllRuns(client *api_server.RunClient, namespace string) ([]*run_model.V1beta1Run, int, string, error) {
	parameters := &runparams.ListRunsParams{}
	return ListRuns(client, parameters, namespace)
}

func ListRuns(client *api_server.RunClient, parameters *runparams.ListRunsParams, namespace string) ([]*run_model.V1beta1Run, int, string, error) {
	if namespace != "" {
		parameters.SetResourceReferenceKeyType(util.StringPointer(api.ResourceType_name[int32(api.ResourceType_NAMESPACE)]))
		parameters.SetResourceReferenceKeyID(&namespace)
	}

	return client.List(parameters)
}

func ListAllJobs(client *api_server.JobClient, namespace string) ([]*job_model.V1beta1Job, int, string, error) {
	return ListJobs(client, &jobparams.ListJobsParams{}, namespace)
}

func ListJobs(client *api_server.JobClient, parameters *jobparams.ListJobsParams, namespace string) ([]*job_model.V1beta1Job, int, string, error) {
	if namespace != "" {
		parameters.SetResourceReferenceKeyType(util.StringPointer(api.ResourceType_name[int32(api.ResourceType_NAMESPACE)]))
		parameters.SetResourceReferenceKeyID(&namespace)
	}

	return client.List(parameters)
}

func GetExperiment(name string, description string, namespace string) *experiment_model.V1beta1Experiment {
	experiment := &experiment_model.V1beta1Experiment{
		Name:        name,
		Description: description}

	if namespace != "" {
		experiment.ResourceReferences = []*experiment_model.V1beta1ResourceReference{
			{
				Key: &experiment_model.V1beta1ResourceKey{
					Type: experiment_model.V1beta1ResourceTypeNAMESPACE,
					ID:   namespace,
				},
				Relationship: experiment_model.V1beta1RelationshipOWNER,
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
