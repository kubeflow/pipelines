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

package test_utils

import (
	"fmt"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/test/logger"
	"math/rand"
	"slices"
	"strconv"
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"

	run_params "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/run_client/run_service"
	api_server "github.com/kubeflow/pipelines/backend/src/common/client/api_server/v2"

	"github.com/go-openapi/strfmt"
	"github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/run_model"
)

func DeletePipelineRun(client *api_server.RunClient, runID string) {
	_, err := client.Get(&run_params.RunServiceGetRunParams{RunID: runID})
	if err != nil {
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

func TerminatePipelineRun(client *api_server.RunClient, runID string) {
	_, err := client.Get(&run_params.RunServiceGetRunParams{RunID: runID})
	if err != nil {
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
	maxTimeToWait := time.Duration(30)
	pollTime := time.Duration(1)
	if timeout != nil {
		maxTimeToWait = *timeout
		pollTime = time.Duration(5)
	}
	waitTime := time.After(maxTimeToWait * time.Second)
	currentPipelineRunState := GetPipelineRun(runClient, pipelineRunID).State
	if currentPipelineRunState != nil {
		for !slices.Contains(expectedStates, *currentPipelineRunState) {
			logger.Log("Waiting for pipeline run with id=%s to be in one of %s", *pipelineRunID, expectedStates)
			time.Sleep(pollTime * time.Second)
			select {
			case <-waitTime:
				ginkgo.Fail("Timed out waiting for pipeline run with id runId=" + *pipelineRunID + " to be in expected state")
			default:
				logger.Log("Pipeline run with id=%s is in %s state", *pipelineRunID, *currentPipelineRunState)
				currentPipelineRunState = GetPipelineRun(runClient, pipelineRunID).State
			}
		}
	} else {
		logger.Log("Pipeline run with id=%s is now NIL", *pipelineRunID)
		ginkgo.Fail("Something went wrong, Pipeline run with id runId=" + *pipelineRunID + " is now in NIL state, please check Pod logs")
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
						pipelineInputMap[name] = rand.Int()
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

func ToRunDetailsFromPipelineSpec(pipelineSpec interface{}, runID string) *run_model.V2beta1RunDetails {
	logger.Log("Converting Pipeline Spec to run details")
	parsedRunDetails := &run_model.V2beta1RunDetails{}
	pipelineSpecMap := pipelineSpec.(map[string]interface{})
	specs, ok := pipelineSpecMap["pipeline_specs"]
	var specsMap map[string]interface{}
	if !ok {
		specsMap = pipelineSpecMap
	} else {
		specsMap = specs.(map[string]interface{})
	}
	root := pipelineSpecMap["root"].(map[string]interface{})
	components := pipelineSpecMap["components"].(map[string]interface{})
	tasks := root["dag"].(map[string]interface{})["tasks"].(map[string]interface{})
	executors := specsMap["deploymentSpec"].(map[string]interface{})["executors"].(map[string]interface{})
	parsedRunDetails.TaskDetails = append(parsedRunDetails.TaskDetails, GetTaskDetailsForComponent(runID, tasks, components)...)
	parsedRunDetails.TaskDetails = append(parsedRunDetails.TaskDetails, createTaskDetail(runID, "root", tasks, "", components))
	parsedRunDetails.TaskDetails = append(parsedRunDetails.TaskDetails, createTaskDetail(runID, "root-driver", tasks, "", components))
	for range executors {
		parsedRunDetails.TaskDetails = append(parsedRunDetails.TaskDetails, createTaskDetail(runID, "executor", tasks, "", make(map[string]interface{})))
	}
	return parsedRunDetails
}

func GetTaskDetailsForComponent(runID string, tasks map[string]interface{}, components map[string]interface{}) []*run_model.V2beta1PipelineTaskDetail {
	var parsedTaskDetails []*run_model.V2beta1PipelineTaskDetail
	for _, task := range tasks {
		taskMap := task.(map[string]interface{})
		taskName := taskMap["taskInfo"].(map[string]interface{})["name"].(string)
		parsedTaskDetails = append(parsedTaskDetails, createTaskDetail(runID, taskName, taskMap, "", components))
		parsedTaskDetails = append(parsedTaskDetails, createTaskDetail(runID, taskName+"-driver", taskMap, "", components))
		// Process nested tasks if this is a DAG component
		componentName := taskMap["componentRef"].(map[string]interface{})["name"].(string)
		if component, compExists := components[componentName].(map[string]interface{}); compExists && component["dag"] != nil {
			nestedTasks := GetTaskDetailsForComponent(runID, component["dag"].(map[string]interface{})["tasks"].(map[string]interface{}), make(map[string]interface{}))
			parsedTaskDetails = append(parsedTaskDetails, nestedTasks...)
		}
	}
	return parsedTaskDetails
}

// createTaskDetail creates a V2beta1PipelineTaskDetail from a TaskYAML
func createTaskDetail(runID string, taskName string, task map[string]interface{}, parentTaskID string, components map[string]interface{}) *run_model.V2beta1PipelineTaskDetail {
	now := time.Now().UTC()

	taskDetail := &run_model.V2beta1PipelineTaskDetail{
		RunID:        runID,
		DisplayName:  taskName,
		ParentTaskID: parentTaskID,
		CreateTime:   strfmt.DateTime(now),
		StartTime:    strfmt.DateTime(now),
		StateHistory: []*run_model.V2beta1RuntimeStatus{},
		Inputs:       make(map[string]run_model.V2beta1ArtifactList),
		Outputs:      make(map[string]run_model.V2beta1ArtifactList),
		ChildTasks:   []*run_model.PipelineTaskDetailChildTask{},
	}

	// Set executor detail if component exists
	componentRef, componentRefExists := task["componentRef"]
	if componentRefExists {
		componentRefMap := componentRef.(map[string]interface{})
		componentRefName := componentRefMap["name"]
		for _, component := range components {
			componentMap := component.(map[string]interface{})
			componentName := componentMap["name"]
			if componentName == componentRefName {
				taskDetail.ExecutorDetail = &run_model.V2beta1PipelineTaskExecutorDetail{
					MainJob:                   componentName.(string),
					PreCachingCheckJob:        "",
					FailedMainJobs:            []string{},
					FailedPreCachingCheckJobs: []string{},
				}
			}
		}
	}

	// Process inputs
	taskInputs, taskInputExists := task["inputs"]
	if taskInputExists {
		taskInputsMap := taskInputs.(map[string]interface{})
		taskInputParameters := taskInputsMap["parameters"].(map[string]interface{})
		if len(taskInputParameters) > 0 {
			for paramName, paramValue := range taskInputParameters {
				// Convert parameter value to artifact list
				artifactList := run_model.V2beta1ArtifactList{
					ArtifactIds: []string{},
				}

				// Create a simple artifact representation
				if _, ok := paramValue.(map[string]interface{}); ok {
					artifactList.ArtifactIds = append(artifactList.ArtifactIds, fmt.Sprintf("%s-%s", taskName, paramName))
				}

				taskDetail.Inputs[paramName] = artifactList
			}
		}
	}

	// Process parameter iterator for loop tasks
	if task["parameterIterator"] != nil {
		// Add loop-specific metadata
		taskDetail.ExecutorDetail = &run_model.V2beta1PipelineTaskExecutorDetail{
			MainJob:                   "loop-executor",
			PreCachingCheckJob:        "",
			FailedMainJobs:            []string{},
			FailedPreCachingCheckJobs: []string{},
		}
	}

	return taskDetail
}
