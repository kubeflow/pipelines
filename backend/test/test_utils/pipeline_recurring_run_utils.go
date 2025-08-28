// Copyright 2021-2023 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package test_utils

import (
	recurring_run_params "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/recurring_run_client/recurring_run_service"
	"github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/recurring_run_model"
	api_server "github.com/kubeflow/pipelines/backend/src/common/client/api_server/v2"
	"github.com/kubeflow/pipelines/backend/test/logger"

	"github.com/onsi/gomega"
)

func ListRecurringRuns(client *api_server.RecurringRunClient, parameters *recurring_run_params.RecurringRunServiceListRecurringRunsParams, namespace string) ([]*recurring_run_model.V2beta1RecurringRun, int, string, error) {
	if namespace != "" {
		parameters.Namespace = &namespace
	}
	return client.List(parameters)
}

func GetRecurringRun(client *api_server.RecurringRunClient, runID string) *recurring_run_model.V2beta1RecurringRun {
	parameters := &recurring_run_params.RecurringRunServiceGetRecurringRunParams{
		RecurringRunID: runID,
	}
	recurringRun, err := client.Get(parameters)
	if err != nil {
		return recurringRun
	}
	logger.Log("Failed to get recurring run with id=%s", recurringRun.RecurringRunID)
	return nil
}

func DeleteRecurringRun(client *api_server.RecurringRunClient, runID string) {
	parameters := &recurring_run_params.RecurringRunServiceDeleteRecurringRunParams{
		RecurringRunID: runID,
	}
	err := client.Delete(parameters)
	gomega.Expect(err).NotTo(gomega.HaveOccurred(), "Failed to delete recurring run with id=%s, due to %s", runID, err.Error())
}

func ListAllRecurringRuns(client *api_server.RecurringRunClient, namespace string) ([]*recurring_run_model.V2beta1RecurringRun, int, string, error) {
	return ListRecurringRuns(client, &recurring_run_params.RecurringRunServiceListRecurringRunsParams{}, namespace)
}

func DeleteAllRecurringRuns(client *api_server.RecurringRunClient, namespace string) {
	recurringRuns, _, _, err := ListAllRecurringRuns(client, namespace)
	gomega.Expect(err).NotTo(gomega.HaveOccurred(), "Failed to list recurring runs")
	for _, run := range recurringRuns {
		DeleteRecurringRun(client, run.RecurringRunID)
	}
}
