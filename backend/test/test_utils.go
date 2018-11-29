// Copyright 2018 Google LLC
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

	"flag"

	"testing"

	"net/http"

	"github.com/cenkalti/backoff"
	experimentparams "github.com/kubeflow/pipelines/backend/api/go_http_client/experiment_client/experiment_service"
	jobparams "github.com/kubeflow/pipelines/backend/api/go_http_client/job_client/job_service"
	pipelineparams "github.com/kubeflow/pipelines/backend/api/go_http_client/pipeline_client/pipeline_service"
	runparams "github.com/kubeflow/pipelines/backend/api/go_http_client/run_client/run_service"
	"github.com/kubeflow/pipelines/backend/src/common/client/api_server"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

var namespace = flag.String("namespace", "kubeflow", "The namespace ml pipeline deployed to")
var initializeTimeout = flag.Duration("initializeTimeout", 2*time.Minute, "Duration to wait for test initialization")

func waitForReady(namespace string, initializeTimeout time.Duration) error {
	var operation = func() error {
		response, err := http.Get(fmt.Sprintf("http://ml-pipeline.%s.svc.cluster.local:8888/apis/v1beta1/healthz", namespace))
		if err == nil {
			return nil
		}
		// we wait only on 503 service unavailable. Stop retry otherwise.
		if response.StatusCode != 503 {
			return backoff.Permanent(errors.Wrapf(err, "Waiting for ml pipeline failed with non retriable error."))
		}
		return err
	}

	b := backoff.NewExponentialBackOff()
	b.MaxElapsedTime = initializeTimeout
	err := backoff.Retry(operation, b)
	return errors.Wrapf(err, "Waiting for ml pipeline failed after all attempts.")
}

func getClientConfig(namespace string) clientcmd.ClientConfig {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	loadingRules.DefaultClientConfig = &clientcmd.DefaultClientConfig
	overrides := clientcmd.ConfigOverrides{Context: clientcmdapi.Context{Namespace: namespace}}
	return clientcmd.NewInteractiveDeferredLoadingClientConfig(loadingRules,
		&overrides, os.Stdin)
}

func deleteAllPipelines(client *api_server.PipelineClient, t *testing.T) {
	pipelines, _, err := client.List(&pipelineparams.ListPipelinesParams{})
	assert.Nil(t, err)
	for _, p := range pipelines {
		assert.Nil(t, client.Delete(&pipelineparams.DeletePipelineParams{ID: p.ID}))
	}
}

func deleteAllExperiments(client *api_server.ExperimentClient, t *testing.T) {
	experiments, _, err := client.List(&experimentparams.ListExperimentParams{})
	assert.Nil(t, err)
	for _, e := range experiments {
		assert.Nil(t, client.Delete(&experimentparams.DeleteExperimentParams{ID: e.ID}))
	}
}

func deleteAllRuns(client *api_server.RunClient, t *testing.T) {
	runs, _, err := client.List(&runparams.ListRunsParams{})
	assert.Nil(t, err)
	for _, r := range runs {
		assert.Nil(t, client.Delete(&runparams.DeleteRunParams{ID: r.ID}))
	}
}

func deleteAllJobs(client *api_server.JobClient, t *testing.T) {
	jobs, _, err := client.List(&jobparams.ListJobsParams{})
	assert.Nil(t, err)
	for _, j := range jobs {
		assert.Nil(t, client.Delete(&jobparams.DeleteJobParams{ID: j.ID}))
	}
}
