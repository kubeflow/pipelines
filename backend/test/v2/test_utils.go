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
	"context"
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"os"
	"testing"
	"time"

	experiment_params "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/experiment_client/experiment_service"
	"github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/experiment_model"
	pipeline_params "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_client/pipeline_service"
	"github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_model"
	recurring_run_params "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/recurring_run_client/recurring_run_service"
	"github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/recurring_run_model"
	run_params "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/run_client/run_service"
	"github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/run_model"
	api_server "github.com/kubeflow/pipelines/backend/src/common/client/api_server/v2"

	"github.com/cenkalti/backoff"
	"github.com/golang/glog"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

const (
	// defaultCleanupTimeout is the timeout for waiting for runs and recurring runs to be deleted in cleanup functions
	defaultCleanupTimeout = 60 * time.Second
	// pipelineVersionCleanupTimeout is the timeout for waiting for pipeline versions to be deleted
	pipelineVersionCleanupTimeout = 5 * time.Second
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
	for _, run := range runs {
		err := client.Delete(&run_params.RunServiceDeleteRunParams{RunID: run.RunID})
		// In some cases, the run may have been deleted by other tests.
		// We can ignore the not found error.
		if err != nil {
			assert.Contains(t, err.Error(), "not found")
		}
	}
	// Wait for runs to be deleted.
	// Increased timeout for local dev environment where cleanup can be slow
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(defaultCleanupTimeout))
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			remainingRuns, _, _, _ := ListAllRuns(client, namespace)
			var remainingRunNames []string
			for _, r := range remainingRuns {
				remainingRunNames = append(remainingRunNames, r.DisplayName)
			}
			require.FailNowf(t, "Runs were not deleted after "+defaultCleanupTimeout.String(), "Remaining runs: %v", remainingRunNames)
		default:
			remainingRuns, _, _, err := ListAllRuns(client, namespace)
			if err != nil {
				glog.Errorf("Error listing runs, will retry: %v", err)
				time.Sleep(1 * time.Second)
				continue
			}
			if len(remainingRuns) == 0 {
				return
			}
			time.Sleep(1 * time.Second)
		}
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
	for _, recurringRun := range recurringRuns {
		err := client.Delete(&recurring_run_params.RecurringRunServiceDeleteRecurringRunParams{RecurringRunID: recurringRun.RecurringRunID})
		// In some cases, the recurring run may have been deleted by other tests.
		// We can ignore the not found error.
		if err != nil {
			assert.Contains(t, err.Error(), "not found")
		}
	}
	// Wait for recurring runs to be deleted.
	// Increased timeout for local dev environment where cleanup can be slow
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(defaultCleanupTimeout))
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			remainingRuns, _, _, _ := ListAllRecurringRuns(client, namespace)
			var remainingRunNames []string
			for _, r := range remainingRuns {
				remainingRunNames = append(remainingRunNames, r.DisplayName)
			}
			require.FailNowf(t, "Recurring runs were not deleted after "+defaultCleanupTimeout.String(), "Remaining recurring runs: %v", remainingRunNames)
		default:
			remainingRuns, _, _, err := ListAllRecurringRuns(client, namespace)
			if err != nil {
				glog.Errorf("Error listing recurring runs, will retry: %v", err)
				time.Sleep(1 * time.Second)
				continue
			}
			if len(remainingRuns) == 0 {
				return
			}
			time.Sleep(1 * time.Second)
		}
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

	// Wait for pipeline versions to be deleted. In Kubernetes mode, there can be a slight delay before the cache is
	// updated.
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(pipelineVersionCleanupTimeout))
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			// This should be a ContextDeadlineExceeded error and causes the test to fail.
			require.Nil(t, ctx.Err(), "Pipeline versions have not been deleted after "+pipelineVersionCleanupTimeout.String())

			return
		default:
			pipelineVersions, _, _, err = ListPipelineVersions(client, pipelineId)
			if err != nil {
				glog.Errorf("Error listing pipeline versions, will retry: %v", err)
				time.Sleep(500 * time.Millisecond)
				continue
			}

			if len(pipelineVersions) == 0 {
				return
			}

			time.Sleep(500 * time.Millisecond)
		}
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

func GetPipelineUploadClient(
	uploadPipelinesWithKubernetes bool,
	isKubeflowMode bool,
	isDebugMode bool,
	namespace string,
	clientConfig clientcmd.ClientConfig,
	tlsCfg *tls.Config,
) (api_server.PipelineUploadInterface, error) {
	if uploadPipelinesWithKubernetes {
		return api_server.NewPipelineUploadClientKubernetes(clientConfig, namespace)
	}

	if isKubeflowMode {
		return api_server.NewKubeflowInClusterPipelineUploadClient(namespace, isDebugMode, tlsCfg)
	}

	return api_server.NewPipelineUploadClient(clientConfig, isDebugMode, tlsCfg)
}

// GetTLSConfig returns TLS config set with system CA certs as well as custom CA stored at input caCertPath if provided.
func GetTLSConfig(caCertPath string) (*tls.Config, error) {
	caCertPool, err := x509.SystemCertPool()
	if err != nil {
		return nil, err
	}
	if caCertPath != "" {
		caCert, err := os.ReadFile(caCertPath)
		if err != nil {
			return nil, err
		}
		if ok := caCertPool.AppendCertsFromPEM(caCert); !ok {
			return nil, err
		}
	}
	return &tls.Config{
		RootCAs: caCertPool,
	}, nil
}
