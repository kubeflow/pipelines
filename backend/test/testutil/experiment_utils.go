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

package testutil

import (
	experiment_params "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/experiment_client/experiment_service"
	"github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/experiment_model"
	api_server "github.com/kubeflow/pipelines/backend/src/common/client/api_server/v2"
	"github.com/kubeflow/pipelines/backend/test/config"
	"github.com/kubeflow/pipelines/backend/test/logger"

	"github.com/onsi/gomega"
)

func CreateExperimentWithParams(experimentClient *api_server.ExperimentClient, experimentParams *experiment_model.V2beta1Experiment) *experiment_model.V2beta1Experiment {
	logger.Log("Create an experiment with name %s", experimentParams.DisplayName)
	createdExperiment, experimentErr := experimentClient.Create(&experiment_params.ExperimentServiceCreateExperimentParams{
		Experiment: experimentParams,
	})
	gomega.Expect(experimentErr).NotTo(gomega.HaveOccurred(), "Failed to create experiment with name '%s'", experimentParams.DisplayName)
	return createdExperiment
}

func CreateExperiment(experimentClient *api_server.ExperimentClient, experimentName string, namespace ...string) *experiment_model.V2beta1Experiment {
	logger.Log("Create an experiment with name %s", experimentName)
	createExperimentParams := experiment_params.NewExperimentServiceCreateExperimentParams()
	namespaceToUse := *config.Namespace
	if len(namespace) > 0 {
		namespaceToUse = namespace[0]
	}
	createExperimentParams.Experiment = &experiment_model.V2beta1Experiment{
		DisplayName: experimentName,
		Namespace:   namespaceToUse,
	}
	createdExperiment, experimentErr := experimentClient.Create(createExperimentParams)
	gomega.Expect(experimentErr).NotTo(gomega.HaveOccurred(), "Failed to create experiment")
	return createdExperiment
}

func ListExperiments(experimentClient *api_server.ExperimentClient, params *experiment_params.ExperimentServiceListExperimentsParams) []*experiment_model.V2beta1Experiment {
	experiments, _, _, err := experimentClient.List(params)
	gomega.Expect(err).NotTo(gomega.HaveOccurred(), "Failed to list experiments")
	return experiments
}

func DeleteExperiment(experimentClient *api_server.ExperimentClient, experimentID string) {
	_, err := experimentClient.Get(&experiment_params.ExperimentServiceGetExperimentParams{ExperimentID: experimentID})
	if err == nil {
		logger.Log("Delete experiment %s", experimentID)
		experimentDeleteParams := experiment_params.ExperimentServiceDeleteExperimentParams{
			ExperimentID: experimentID,
		}
		err = experimentClient.Delete(&experimentDeleteParams)
		if err != nil {
			logger.Log("Failed to delete experiment %s", experimentID)
		}
	} else {
		logger.Log("Skipping Deletion of the experiment %s, as it does not exist", experimentID)
	}
}
