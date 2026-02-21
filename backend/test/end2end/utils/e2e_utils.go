// Package utils provides helpers shared across end-to-end tests.
package utils

import (
	"context"
	"fmt"
	"maps"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"time"

	runparams "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/run_client/run_service"
	"github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/run_model"
	apiserver "github.com/kubeflow/pipelines/backend/src/common/client/api_server/v2"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/test/config"
	"github.com/kubeflow/pipelines/backend/test/logger"
	"github.com/kubeflow/pipelines/backend/test/testutil"
	apitests "github.com/kubeflow/pipelines/backend/test/v2/api"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var (
	workflowClient     versioned.Interface
	workflowClientOnce sync.Once
	workflowClientErr  error
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
		logger.Log("Pipeline run did not finish, checking workflow controller logs")
		podLog := testutil.ReadContainerLogs(k8Client, *config.Namespace, "workflow-controller", nil, &testContext.TestStartTimeUTC, config.PodLogLimit)
		logger.Log("Attaching Workflow Controller logs to the report")
		ginkgo.AddReportEntry("Workflow Controller Logs", podLog)
		ginkgo.Fail("Pipeline run did not complete, it stayed in RUNNING state")

	} else {
		if *updatedRun.State != run_model.V2beta1RuntimeStateSUCCEEDED {
			logger.Log("Looks like the run %s FAILED, so capture pod logs for the failed task", runID)
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
	archivedLogCache := make(map[string]string)
	namespace := testutil.GetNamespace()
	sort.Slice(taskDetails, func(i, j int) bool {
		return time.Time(taskDetails[i].EndTime).After(time.Time(taskDetails[j].EndTime)) // Sort Tasks by End Time in descending order
	})
	for _, task := range taskDetails {
		if task.State == nil {
			continue
		}

		switch *task.State {
		case run_model.V2beta1RuntimeStateSUCCEEDED:
			logger.Log("SUCCEEDED - Task %s for run %s has finished successfully", task.DisplayName, task.RunID)
		case run_model.V2beta1RuntimeStateRUNNING:
			logger.Log("RUNNING - Task %s for Run %s is running", task.DisplayName, task.RunID)
		case run_model.V2beta1RuntimeStateSKIPPED:
			logger.Log("SKIPPED - Task %s for Run %s skipped", task.DisplayName, task.RunID)
		case run_model.V2beta1RuntimeStateCANCELED:
			logger.Log("CANCELED - Task %s for Run %s canceled", task.DisplayName, task.RunID)
		case run_model.V2beta1RuntimeStateFAILED:
			logger.Log("%s - Task %s for Run %s did not complete successfully", *task.State, task.DisplayName, task.RunID)

			podNames := map[string]struct{}{}
			if task.PodName != "" {
				podNames[task.PodName] = struct{}{}
			}
			for _, childTask := range task.ChildTasks {
				if childTask.PodName != "" {
					podNames[childTask.PodName] = struct{}{}
				}
			}

			if len(podNames) == 0 {
				logger.Log("Task %s for Run %s did not report any pod names", task.DisplayName, task.RunID)
				failedTasks[task.DisplayName] = string(*task.State)
				continue
			}

			var combinedLog strings.Builder
			for podName := range podNames {
				logger.Log("Collecting logs for task %s pod %s", task.DisplayName, podName)
				combinedLog.WriteString(fmt.Sprintf("===== Pod: %s =====\n", podName))

				podLog := testutil.ReadPodLogs(k8Client, namespace, podName, nil, &testContext.TestStartTimeUTC, config.PodLogLimit)
				missingPod := false
				meaningful, missingPod := hasMeaningfulLogs(podLog)
				switch {
				case meaningful:
					combinedLog.WriteString("----- Live Logs (kubectl) -----\n")
					combinedLog.WriteString(podLog)
					if !strings.HasSuffix(podLog, "\n") {
						combinedLog.WriteString("\n")
					}
				case strings.TrimSpace(podLog) != "":
					combinedLog.WriteString(podLog)
					if !strings.HasSuffix(podLog, "\n") {
						combinedLog.WriteString("\n")
					}
					if missingPod {
						combinedLog.WriteString("Pod logs unavailable; pod not found. Falling back to archived logs.\n")
					} else {
						combinedLog.WriteString("Live logs unavailable via kubectl logs.\n")
					}
				default:
					if missingPod {
						combinedLog.WriteString("Pod logs unavailable; pod not found. Falling back to archived logs.\n")
					} else {
						combinedLog.WriteString("Live logs unavailable via kubectl logs.\n")
					}
				}

				if missingPod {
					archivedLog, err := getArchivedLogWithCache(archivedLogCache, k8Client, namespace, task.RunID, podName)
					if err != nil {
						logger.Log("Failed to retrieve archived logs for pod %s: %v", podName, err)
						combinedLog.WriteString(fmt.Sprintf("Failed to retrieve archived logs via Argo Workflows: %v\n", err))
					} else if strings.TrimSpace(archivedLog) != "" {
						combinedLog.WriteString("----- Archived Logs (Argo) -----\n")
						combinedLog.WriteString(archivedLog)
						if !strings.HasSuffix(archivedLog, "\n") {
							combinedLog.WriteString("\n")
						}
					} else {
						combinedLog.WriteString("Archived logs were empty.\n")
					}
				}

				combinedLog.WriteString("\n")
			}

			entryContent := combinedLog.String()
			if strings.TrimSpace(entryContent) == "" {
				entryContent = fmt.Sprintf("No logs were available for failed task %s", task.DisplayName)
			}

			logger.Log("Attaching logs to report for task %s", task.DisplayName)
			ginkgo.AddReportEntry(fmt.Sprintf("Failing '%s' Component Log", task.DisplayName), entryContent)
			logger.Log("Attached logs to the report for task %s", task.DisplayName)

			failedTasks[task.DisplayName] = string(*task.State)
		default:
			logger.Log("UNKNOWN state - Task %s for Run %s has an UNKNOWN state", task.DisplayName, task.RunID)
		}
	}
	if len(failedTasks) > 0 {
		logger.Log("Found failed tasks: %v", maps.Keys(failedTasks))
	}
}

func getArchivedLogWithCache(cache map[string]string, k8Client *kubernetes.Clientset, namespace, runID, podName string) (string, error) {
	cacheKey := fmt.Sprintf("%s::%s", runID, podName)
	if val, ok := cache[cacheKey]; ok {
		return val, nil
	}

	logContent, err := retrieveArchivedLogs(k8Client, namespace, runID, podName)
	if err != nil {
		return "", err
	}

	cache[cacheKey] = logContent
	return logContent, nil
}

func retrieveArchivedLogs(k8Client *kubernetes.Clientset, namespace, runID, podName string) (string, error) {
	workflowName, err := resolveWorkflowNameForRun(k8Client, namespace, runID)
	if err != nil {
		return "", err
	}

	cliPath, err := ensureArgoCLI()
	if err != nil {
		return "", err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, cliPath, "logs", workflowName, podName, "-n", namespace, "--no-color")
	output, err := cmd.CombinedOutput()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("argo logs command timed out after 30s\n%s", string(output))
		}
		return "", fmt.Errorf("argo logs command failed: %w\n%s", err, string(output))
	}

	return string(output), nil
}

func hasMeaningfulLogs(logText string) (bool, bool) {
	trimmed := strings.TrimSpace(logText)
	if trimmed == "" {
		return false, false
	}
	lower := strings.ToLower(trimmed)

	missingPod := strings.Contains(lower, "not found")
	if strings.Contains(lower, "no pod logs available") ||
		strings.Contains(lower, "could not find pod containing container") ||
		strings.Contains(lower, "failed to stream pod logs") {
		return false, missingPod
	}

	return true, missingPod
}

func ensureArgoCLI() (string, error) {
	return exec.LookPath("argo")
}

func resolveWorkflowNameForRun(k8Client *kubernetes.Clientset, namespace, runID string) (string, error) {
	if runID == "" {
		return "", fmt.Errorf("run ID is empty")
	}

	wfClient, err := getWorkflowClient()
	if err != nil {
		return "", err
	}

	labelSelector := fmt.Sprintf("%s=%s", util.LabelKeyWorkflowRunId, runID)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	workflows, err := wfClient.ArgoprojV1alpha1().Workflows(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("listing workflows timed out after 10s for run ID %s", runID)
		}
		return "", fmt.Errorf("failed to list workflows: %w", err)
	}

	if len(workflows.Items) == 0 {
		return "", fmt.Errorf("no workflow found in namespace %s with run ID %s", namespace, runID)
	}

	return workflows.Items[0].Name, nil
}

func getWorkflowClient() (versioned.Interface, error) {
	workflowClientOnce.Do(func() {
		restConfig, err := util.GetKubernetesConfig()
		if err != nil {
			workflowClientErr = fmt.Errorf("failed to create kubernetes config: %w", err)
			return
		}

		workflowClient, workflowClientErr = versioned.NewForConfig(restConfig)
	})
	return workflowClient, workflowClientErr
}

type TaskDetails struct {
	TaskName  string
	Task      v1alpha1.DAGTask
	Container v1.Container
	DependsOn string
}

// GetTasksFromWorkflow - Get tasks from a compiled workflow
func GetTasksFromWorkflow(workflow *v1alpha1.Workflow) []TaskDetails {
	containers := make(map[string]*v1.Container)
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
