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

package testutil

import (
	"math/rand"
	"slices"
	"strconv"
	"time"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	run_params "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/run_client/run_service"
	"github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/run_model"
	api_server "github.com/kubeflow/pipelines/backend/src/common/client/api_server/v2"
	"github.com/kubeflow/pipelines/backend/test/logger"

	"github.com/onsi/gomega"
)

func DeletePipelineRun(client *api_server.RunClient, runID string) {
	_, err := client.Get(&run_params.RunServiceGetRunParams{RunID: runID})
	if err == nil {
		logger.Log("Deleting run %s", runID)
		deleteRunParams := run_params.NewRunServiceDeleteRunParams()
		deleteRunParams.RunID = runID
		deleteErr := client.Delete(deleteRunParams)
		if deleteErr != nil {
			logger.Log("Failed to delete run %s", runID)
		}
	} else {
		logger.Log("Skipping Deletion of the run %s, as it does not exist", runID)
	}
}

func ArchivePipelineRun(client *api_server.RunClient, runID string) {
	_, err := client.Get(&run_params.RunServiceGetRunParams{RunID: runID})
	if err == nil {
		logger.Log("Terminate run %s", runID)
		archiveRunParams := run_params.NewRunServiceArchiveRunParams()
		archiveRunParams.RunID = runID
		archiveErr := client.Archive(archiveRunParams)
		if archiveErr != nil {
			logger.Log("Failed to archive run %s", runID)
		}
	} else {
		logger.Log("Skipping Archiving  of the run %s, because it does not exist", runID)
	}
}

func TerminatePipelineRun(client *api_server.RunClient, runID string) {
	_, err := client.Get(&run_params.RunServiceGetRunParams{RunID: runID})
	if err == nil {
		logger.Log("Terminate run %s", runID)
		terminateRunParams := run_params.NewRunServiceTerminateRunParams()
		terminateRunParams.RunID = runID
		terminateErr := client.Terminate(terminateRunParams)
		if terminateErr != nil {
			logger.Log("Failed to terminate run %s", runID)
		}
	} else {
		logger.Log("Skipping Termination of the run %s, because it does not exist", runID)
	}
}

func GetPipelineRun(runClient *api_server.RunClient, pipelineRunID *string) *run_model.V2beta1Run {
	logger.Log("Get a pipeline run with id=%s", *pipelineRunID)
	pipelineRun, runError := runClient.Get(&run_params.RunServiceGetRunParams{
		RunID: *pipelineRunID,
	})
	gomega.Expect(runError).NotTo(gomega.HaveOccurred(), "Failed to get run with id="+*pipelineRunID)
	return pipelineRun
}

func WaitForRunToBeInState(runClient *api_server.RunClient, pipelineRunID *string, expectedStates []run_model.V2beta1RuntimeState, timeout *time.Duration) {
	logger.Log("Waiting for pipeline run with id=%s to be in one of '%s'", *pipelineRunID, expectedStates)
	maxTimeToWait := time.Duration(300)
	pollTime := time.Duration(5)
	if timeout != nil {
		maxTimeToWait = *timeout
	}
	deadline := time.Now().Add(maxTimeToWait * time.Second)
	ticker := time.NewTicker(pollTime * time.Second)
	defer ticker.Stop()
	for {
		currentPipelineRunState := GetPipelineRun(runClient, pipelineRunID).State
		if currentPipelineRunState != nil {
			if slices.Contains(expectedStates, *currentPipelineRunState) {
				logger.Log("Pipeline run with id=%s reached expected state %s", *pipelineRunID, *currentPipelineRunState)
				return
			}

			if time.Now().After(deadline) {
				logger.Log("Pipeline run with id=%s is in %s state, did not reach one of '%s' ", *pipelineRunID, *currentPipelineRunState, expectedStates)
				return
			}
			logger.Log("Pipeline run with id=%s is in %s state, waiting...", *pipelineRunID, *currentPipelineRunState)
		} else {
			logger.Log("Pipeline run with id=%s is in nil state, rechecking...", *pipelineRunID)
		}
		<-ticker.C
	}

}

func GetPipelineRunTimeInputs(pipelineSpecFile string) map[string]interface{} {
	logger.Log("Get the pipeline run time inputs from pipeline spec file %s", pipelineSpecFile)
	pipelineSpec := ParseFileToSpecs(pipelineSpecFile, false, nil).PipelineSpec()
	pipelineInputMap := make(map[string]interface{})
	if pipelineSpec.Root.InputDefinitions != nil {
		if pipelineSpec.Root.InputDefinitions.Parameters != nil {
			for name, parameterSpec := range pipelineSpec.Root.InputDefinitions.Parameters {
				defaultValExists := false
				if parameterSpec.DefaultValue != nil {
					defaultValExists = true
				}
				if !defaultValExists || !parameterSpec.IsOptional {
					switch parameterSpec.ParameterType {
					case pipelinespec.ParameterType_NUMBER_INTEGER:
						pipelineInputMap[name] = rand.Intn(1000)
					case pipelinespec.ParameterType_STRING:
						pipelineInputMap[name] = GetRandomString(20)
					case pipelinespec.ParameterType_STRUCT:
						pipelineInputMap[name] = map[string]interface{}{
							"A": strconv.FormatFloat(rand.Float64(), 'g', -1, 64),
							"B": strconv.FormatFloat(rand.Float64(), 'g', -1, 64),
						}
					case pipelinespec.ParameterType_LIST:
						pipelineInputMap[name] = []string{GetRandomString(20)}
					case pipelinespec.ParameterType_BOOLEAN:
						pipelineInputMap[name] = true
					case pipelinespec.ParameterType_NUMBER_DOUBLE:
						pipelineInputMap[name] = rand.Float64()
					default:
						pipelineInputMap[name] = GetRandomString(20)
					}
				}

			}
		}
	}
	logger.Log("Returning pipeline run time inputs %v", pipelineInputMap)
	return pipelineInputMap
}
