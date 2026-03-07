// Package utils provides helpers shared across end-to-end tests.
package utils

import (
	"fmt"
	"maps"
	"sort"
	"strings"
	"time"

	runparams "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/run_client/run_service"
	"github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/run_model"
	apiserver "github.com/kubeflow/pipelines/backend/src/common/client/api_server/v2"
	"github.com/kubeflow/pipelines/backend/test/config"
	"github.com/kubeflow/pipelines/backend/test/logger"
	"github.com/kubeflow/pipelines/backend/test/testutil"
	apitests "github.com/kubeflow/pipelines/backend/test/v2/api"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

// CreatePipelineRun - Create a pipeline run
func CreatePipelineRun(runClient *apiserver.RunClient, testContext *apitests.TestContext, pipelineID *string, pipelineVersionID *string, experimentID *string, inputParams map[string]interface{}) *run_model.V2beta1Run {
	runName := fmt.Sprintf("E2e Test Run-%v", testContext.TestStartTimeUTC)
	runDescription := fmt.Sprintf("Run for %s", runName)
	logger.Log("Create a pipeline run for pipeline with id=%s and versionId=%s", *pipelineID, *pipelineVersionID)
	createRunRequest := &runparams.RunServiceCreateRunParams{
		ExperimentID: experimentID,
		Run:          CreatePipelineRunPayload(runName, runDescription, pipelineID, pipelineVersionID, experimentID, inputParams),
	}
	createdRun, createRunError := runClient.Create(createRunRequest)
	gomega.Expect(createRunError).NotTo(gomega.HaveOccurred(), "Failed to create run for pipeline with id="+*pipelineID)
	testContext.PipelineRun.CreatedRunIds = append(testContext.PipelineRun.CreatedRunIds, createdRun.RunID)
	logger.Log("Created Pipeline Run successfully with runId=%s", createdRun.RunID)
	return createdRun
}

// CreatePipelineRunPayload - Create a pipeline run payload
func CreatePipelineRunPayload(runName string, runDescription string, pipelineID *string, pipelineVersionID *string, experimentID *string, inputParams map[string]interface{}) *run_model.V2beta1Run {
	logger.Log("Create a pipeline run body")
	return &run_model.V2beta1Run{
		DisplayName:    runName,
		Description:    runDescription,
		ExperimentID:   testutil.ParsePointersToString(experimentID),
		ServiceAccount: testutil.GetDefaultPipelineRunnerServiceAccount(),
		PipelineVersionReference: &run_model.V2beta1PipelineVersionReference{
			PipelineID:        testutil.ParsePointersToString(pipelineID),
			PipelineVersionID: testutil.ParsePointersToString(pipelineVersionID),
		},
		RuntimeConfig: &run_model.V2beta1RuntimeConfig{
			Parameters: inputParams,
		},
	}
}

// CreatePipelineRunAndWaitForItToFinish - Create a pipeline run and wait for it complete
func CreatePipelineRunAndWaitForItToFinish(runClient *apiserver.RunClient, testContext *apitests.TestContext, pipelineID string, pipelineDisplayName string, pipelineVersionID *string, experimentID *string, runTimeParams map[string]interface{}, maxPipelineWaitTime int) string {
	logger.Log("Create run for pipeline with id: '%s' and name: '%s'", pipelineID, pipelineDisplayName)
	uploadedPipelineRun := CreatePipelineRun(runClient, testContext, &pipelineID, pipelineVersionID, experimentID, runTimeParams)
	logger.Log("Created Pipeline Run with id: %s for pipeline with id: %s", uploadedPipelineRun.RunID, pipelineID)
	timeout := time.Duration(maxPipelineWaitTime)
	testutil.WaitForRunToBeInState(runClient, &uploadedPipelineRun.RunID, []run_model.V2beta1RuntimeState{run_model.V2beta1RuntimeStateSUCCEEDED, run_model.V2beta1RuntimeStateSKIPPED, run_model.V2beta1RuntimeStateFAILED, run_model.V2beta1RuntimeStateCANCELED}, &timeout)
	return uploadedPipelineRun.RunID
}

// ValidateComponentStatuses - Validate that all the components of a pipeline run ran successfully
func ValidateComponentStatuses(runClient *apiserver.RunClient, k8Client *kubernetes.Clientset, testContext *apitests.TestContext, runID string, compiledWorkflow *v1alpha1.Workflow) {
	logger.Log("Fetching updated pipeline run details for run with id=%s", runID)
	updatedRun := testutil.GetPipelineRun(runClient, &runID)
	actualTaskDetails := updatedRun.RunDetails.TaskDetails
	logger.Log("Updated pipeline run details")
	expectedTaskDetails := GetTasksFromWorkflow(compiledWorkflow)
	if *updatedRun.State == run_model.V2beta1RuntimeStateRUNNING {
		logger.Log("Pipeline run did not finish")
		logger.Log("Checking workflow controller logs")
		podLog := testutil.ReadContainerLogs(k8Client, *config.Namespace, "workflow-controller", nil, &testContext.TestStartTimeUTC, config.PodLogLimit)
		logger.Log("Attaching Workflow Controller logs to the report")
		ginkgo.AddReportEntry("Workflow Controller Logs", podLog)
		ginkgo.Fail("Pipeline run did not complete, it stayed in RUNNING state")

	} else {
		if *updatedRun.State != run_model.V2beta1RuntimeStateSUCCEEDED {
			logger.Log("Looks like the run %s FAILED, so capture pod logs for the failed task", runID)
			logger.Log("Checking workflow controller logs")
			podLog := testutil.ReadContainerLogs(k8Client, *config.Namespace, "workflow-controller", nil, &testContext.TestStartTimeUTC, config.PodLogLimit)
			logger.Log("Attaching Workflow Controller logs to the report")
			ginkgo.AddReportEntry("Workflow Controller Logs", podLog)
			CapturePodLogsForUnsuccessfulTasks(k8Client, testContext, actualTaskDetails)
			ginkgo.Fail("Failing test because the pipeline run was not SUCCESSFUL")
		} else {
			logger.Log("Pipeline run succeeded, checking if the number of tasks are what is expected")
			gomega.Expect(len(actualTaskDetails)).To(gomega.BeNumerically(">=", len(expectedTaskDetails)), "Number of created DAG tasks should be >= number of expected tasks")
		}
	}

}

// CapturePodLogsForUnsuccessfulTasks - Capture pod logs of a failed component
func CapturePodLogsForUnsuccessfulTasks(k8Client *kubernetes.Clientset, testContext *apitests.TestContext, taskDetails []*run_model.V2beta1PipelineTaskDetail) {
	failedTasks := make(map[string]string)
	sort.Slice(taskDetails, func(i, j int) bool {
		return time.Time(taskDetails[i].EndTime).After(time.Time(taskDetails[j].EndTime)) // Sort Tasks by End Time in descending order
	})
	for _, task := range taskDetails {
		if task.State != nil {
			switch *task.State {
			case run_model.V2beta1RuntimeStateSUCCEEDED:
				{
					logger.Log("SUCCEEDED - Task %s for run %s has finished successfully", task.DisplayName, task.RunID)
				}
			case run_model.V2beta1RuntimeStateRUNNING:
				{
					logger.Log("RUNNING - Task %s for Run %s is running", task.DisplayName, task.RunID)

				}
			case run_model.V2beta1RuntimeStateSKIPPED:
				{
					logger.Log("SKIPPED - Task %s for Run %s skipped", task.DisplayName, task.RunID)
				}
			case run_model.V2beta1RuntimeStateCANCELED:
				{
					logger.Log("CANCELED - Task %s for Run %s canceled", task.DisplayName, task.RunID)
				}
			case run_model.V2beta1RuntimeStateFAILED:
				{
					agentLogsNotYetAcquired := true
					logger.Log("%s - Task %s for Run %s did not complete successfully", *task.State, task.DisplayName, task.RunID)
					for _, childTask := range task.ChildTasks {
						podName := childTask.PodName
						isDriver := strings.Contains(task.DisplayName, "-driver")
						if podName != "" {
							if isDriver {
								if agentLogsNotYetAcquired {
									logger.Log("Capturing pod logs for task executor agent, with pod name %s", podName)
									podName += "-agent"
									agentLogsNotYetAcquired = false
								} else {
									logger.Log("Logs from agent pod was already captured")
									continue
								}
							} else {
								logger.Log("Capturing pod logs for task %s, with pod name %s", task.DisplayName, podName)
							}
							podLog := testutil.ReadPodLogs(k8Client, *config.Namespace, podName, nil, &testContext.TestStartTimeUTC, config.PodLogLimit)
							logger.Log("Pod logs captured for task %s in pod %s", task.DisplayName, podName)
							logger.Log("Attaching pod logs to the report")
							ginkgo.AddReportEntry(fmt.Sprintf("Failing '%s' Component Log", task.DisplayName), podLog)
							logger.Log("Attached pod logs to the report")
						}
					}
					failedTasks[task.DisplayName] = string(*task.State)
				}
			default:
				{
					logger.Log("UNKNOWN state - Task %s for Run %s has an UNKNOWN state", task.DisplayName, task.RunID)
				}
			}
		}
	}
	if len(failedTasks) > 0 {
		logger.Log("Found failed tasks: %v", maps.Keys(failedTasks))
	}
}

type TaskDetails struct {
	TaskName  string
	Task      v1alpha1.DAGTask
	Container v1.Container
	DependsOn string
}

// GetTasksFromWorkflow - Get tasks from a compiled workflow
func GetTasksFromWorkflow(workflow *v1alpha1.Workflow) []TaskDetails {
	var containers = make(map[string]*v1.Container)
	var tasks []TaskDetails
	for _, template := range workflow.Spec.Templates {
		if template.Container != nil {
			containers[template.Name] = template.Container
		}
	}
	for _, template := range workflow.Spec.Templates {
		if template.DAG != nil {
			for _, task := range template.DAG.Tasks {
				if task.When == "" {
					continue
				}
				container, containerExists := containers[task.Template]
				taskToAppend := TaskDetails{
					TaskName:  task.Name,
					Task:      task,
					DependsOn: task.Depends,
				}
				if containerExists {
					taskToAppend.Container = *container
				}
				tasks = append(tasks, taskToAppend)
			}
		}
	}
	return tasks
}
