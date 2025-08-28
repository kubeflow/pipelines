package test_utils

import (
	experiment_params "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/experiment_client/experiment_service"
	"github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/experiment_model"
	api_server "github.com/kubeflow/pipelines/backend/src/common/client/api_server/v2"
	"github.com/kubeflow/pipelines/backend/test/config"
	"github.com/kubeflow/pipelines/backend/test/logger"
	"github.com/onsi/gomega"
)

func CreateExperiment(experimentClient *api_server.ExperimentClient, experimentName string) *experiment_model.V2beta1Experiment {
	logger.Log("Create an experiment with name %s", experimentName)
	createExperimentParams := experiment_params.NewExperimentServiceCreateExperimentParams()
	createExperimentParams.Experiment = &experiment_model.V2beta1Experiment{
		DisplayName: experimentName,
		Namespace:   *config.Namespace,
	}
	createdExperiment, experimentErr := experimentClient.Create(createExperimentParams)
	gomega.Expect(experimentErr).NotTo(gomega.HaveOccurred(), "Failed to create experiment")
	return createdExperiment
}

func DeleteExperiment(experimentClient *api_server.ExperimentClient, experimentID string) {
	_, err := experimentClient.Get(&experiment_params.ExperimentServiceGetExperimentParams{ExperimentID: experimentID})
	if err != nil {
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
