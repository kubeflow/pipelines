// Copyright 2018-2023 The Kubeflow Authors
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
	experiment_params "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/experiment_client/experiment_service"
	"github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/experiment_model"
	pipeline_params "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_client/pipeline_service"
	"github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_model"
	recurring_run_params "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/recurring_run_client/recurring_run_service"
	"github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/recurring_run_model"
	run_params "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/run_client/run_service"
	"github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/run_model"
	api_server "github.com/kubeflow/pipelines/backend/src/common/client/api_server/v2"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

func WaitForReady(initializeTimeout time.Duration) error {
	operation := func() error {
		response, err := http.Get("http://localhost:8888/apis/v2beta1/healthz")
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

func GetDefaultPipelineRunnerServiceAccount(isKubeflowMode bool) string {
	if isKubeflowMode {
		return "default-editor"
	} else {
		return "pipeline-runner"
	}
}

func ListAllExperiment(client *api_server.ExperimentClient, namespace string) ([]*experiment_model.V2beta1Experiment, int, string, error) {
	return ListExperiment(client, &experiment_params.ExperimentServiceListExperimentsParams{}, namespace)
}

func ListExperiment(client *api_server.ExperimentClient, parameters *experiment_params.ExperimentServiceListExperimentsParams, namespace string) ([]*experiment_model.V2beta1Experiment, int, string, error) {
	if namespace != "" {
		parameters.Namespace = &namespace
	}
	return client.List(parameters)
}

func DeleteAllExperiments(client *api_server.ExperimentClient, namespace string, t *testing.T) {
	experiments, _, _, err := ListAllExperiment(client, namespace)
	assert.Nil(t, err)
	for _, e := range experiments {
		if e.DisplayName != "Default" {
			assert.Nil(t, client.Delete(&experiment_params.ExperimentServiceDeleteExperimentParams{ExperimentID: e.ExperimentID}))
		}
	}
}

func MakeExperiment(name string, description string, namespace string) *experiment_model.V2beta1Experiment {
	experiment := &experiment_model.V2beta1Experiment{
		DisplayName: name,
		Description: description,
	}

	if namespace != "" {
		experiment.Namespace = namespace
	}

	return experiment
}

func ListRuns(client *api_server.RunClient, parameters *run_params.RunServiceListRunsParams, namespace string) ([]*run_model.V2beta1Run, int, string, error) {
	if namespace != "" {
		parameters.Namespace = &namespace
	}
	return client.List(parameters)
}

func ListAllRuns(client *api_server.RunClient, namespace string) ([]*run_model.V2beta1Run, int, string, error) {
	parameters := &run_params.RunServiceListRunsParams{}
	return ListRuns(client, parameters, namespace)
}

func DeleteAllRuns(client *api_server.RunClient, namespace string, t *testing.T) {
	runs, _, _, err := ListAllRuns(client, namespace)
	assert.Nil(t, err)
	for _, r := range runs {
		assert.Nil(t, client.Delete(&run_params.RunServiceDeleteRunParams{RunID: r.RunID}))
	}
}

func ListRecurringRuns(client *api_server.RecurringRunClient, parameters *recurring_run_params.RecurringRunServiceListRecurringRunsParams, namespace string) ([]*recurring_run_model.V2beta1RecurringRun, int, string, error) {
	if namespace != "" {
		parameters.Namespace = &namespace
	}
	return client.List(parameters)
}

func ListAllRecurringRuns(client *api_server.RecurringRunClient, namespace string) ([]*recurring_run_model.V2beta1RecurringRun, int, string, error) {
	return ListRecurringRuns(client, &recurring_run_params.RecurringRunServiceListRecurringRunsParams{}, namespace)
}

func DeleteAllRecurringRuns(client *api_server.RecurringRunClient, namespace string, t *testing.T) {
	recurringRuns, _, _, err := ListAllRecurringRuns(client, namespace)
	assert.Nil(t, err)
	for _, r := range recurringRuns {
		assert.Nil(t, client.Delete(&recurring_run_params.RecurringRunServiceDeleteRecurringRunParams{RecurringRunID: r.RecurringRunID}))
	}
}

func ListPipelineVersions(client *api_server.PipelineClient, pipelineId string) (
	[]*pipeline_model.V2beta1PipelineVersion, int, string, error,
) {
	parameters := &pipeline_params.PipelineServiceListPipelineVersionsParams{PipelineID: pipelineId}
	return client.ListPipelineVersions(parameters)
}

func ListPipelines(client *api_server.PipelineClient) (
	[]*pipeline_model.V2beta1Pipeline, int, string, error,
) {
	parameters := &pipeline_params.PipelineServiceListPipelinesParams{}
	return client.List(parameters)
}

func DeleteAllPipelineVersions(client *api_server.PipelineClient, t *testing.T, pipelineId string) {
	pipelineVersions, _, _, err := ListPipelineVersions(client, pipelineId)
	assert.Nil(t, err)
	for _, pv := range pipelineVersions {
		assert.Nil(t, client.DeletePipelineVersion(&pipeline_params.PipelineServiceDeletePipelineVersionParams{PipelineID: pipelineId, PipelineVersionID: pv.PipelineVersionID}))
	}
}

func DeleteAllPipelines(client *api_server.PipelineClient, t *testing.T) {
	pipelines, _, _, err := ListPipelines(client)
	assert.Nil(t, err)
	deletedPipelines := make(map[string]bool)
	for _, p := range pipelines {
		deletedPipelines[p.PipelineID] = false
	}
	for pId, isRemoved := range deletedPipelines {
		if !isRemoved {
			DeleteAllPipelineVersions(client, t, pId)
			deletedPipelines[pId] = true
		}
		assert.Nil(t, client.Delete(&pipeline_params.PipelineServiceDeletePipelineParams{PipelineID: pId}))
	}
	for _, isRemoved := range deletedPipelines {
		assert.True(t, isRemoved)
	}
}
