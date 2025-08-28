package test

import (
	experiment_params "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/experiment_client/experiment_service"
	api_server "github.com/kubeflow/pipelines/backend/src/common/client/api_server/v2"
	"github.com/kubeflow/pipelines/backend/test/v2/api/logger"
)

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
