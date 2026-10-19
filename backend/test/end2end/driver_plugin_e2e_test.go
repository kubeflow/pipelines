// Copyright 2026 The Kubeflow Authors
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

package end2end

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"time"

	"github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	argoclient "github.com/argoproj/argo-workflows/v4/pkg/client/clientset/versioned"
	upload_params "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_upload_client/pipeline_upload_service"
	"github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_upload_model"
	"github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/run_model"
	workflowutils "github.com/kubeflow/pipelines/backend/test/compiler/utils"
	. "github.com/kubeflow/pipelines/backend/test/constants"
	e2e_utils "github.com/kubeflow/pipelines/backend/test/end2end/utils"
	"github.com/kubeflow/pipelines/backend/test/logger"
	"github.com/kubeflow/pipelines/backend/test/testutil"
	apitests "github.com/kubeflow/pipelines/backend/test/v2/api"

	"github.com/go-openapi/strfmt"
	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/ginkgo/v2/types"
	. "github.com/onsi/gomega"
)

// Fixtures for the driver plugin scenarios of
// proposals/separate-standalone-driver/README.md ("Test Plan").
const (
	driverPluginPipelineDir = "valid/driver_plugin"
	// retryPipelineFile has a component that fails on its first attempt and
	// succeeds on retry, feeding its output to a consumer that checks it.
	retryPipelineFile = "pipeline_with_flaky_retry.yaml"
	retryTaskName     = "flaky-add"
	retryConsumerName = "check-sum"
)

// Driver plugin end-to-end coverage: the driver runs as the "driver-plugin"
// executor plugin sidecar of Argo's per-workflow agent pod instead of a pod
// per task. Filter with --label-filter=E2EDriverPlugin.
var _ = Describe("Driver plugin >", Label(FullRegression, E2eDriverPlugin), func() {
	var testContext *apitests.TestContext
	var argoClient argoclient.Interface

	// ################## SET AND TEARDOWN ##################

	BeforeEach(func() {
		logger.Log("################### Setup before each driver plugin test #####################")
		testContext = &apitests.TestContext{
			TestStartTimeUTC: time.Now(),
		}
		randomName = strconv.FormatInt(time.Now().UnixNano(), 10)
		testContext.Pipeline.UploadParams = upload_params.NewUploadPipelineParams()
		testContext.Pipeline.PipelineGeneratedName = "e2e-driver-plugin-" + randomName
		testContext.Pipeline.CreatedPipelines = make([]*pipeline_upload_model.V2beta1Pipeline, 0)
		testContext.PipelineRun.CreatedRunIds = make([]string, 0)
		testContext.Pipeline.ExpectedPipeline = new(pipeline_upload_model.V2beta1Pipeline)
		testContext.Pipeline.ExpectedPipeline.CreatedAt = strfmt.DateTime(testContext.TestStartTimeUTC)
		var err error
		argoClient, err = testutil.CreateArgoClient()
		Expect(err).NotTo(HaveOccurred(), "Failed to initialize Argo Workflows client")

		DeferCleanup(cleanupE2ETestResources, testContext)
	})

	ReportAfterEach(func(specReport types.SpecReport) {
		if testContext == nil {
			return
		}
		if specReport.Failed() && len(testContext.PipelineRun.CreatedRunIds) > 0 {
			report, _ := testutil.BuildArchivedWorkflowLogsReport(k8Client, testContext.PipelineRun.CreatedRunIds)
			AddReportEntry(testutil.ArchivedWorkflowLogsReportTitle, report)
		}
	})

	// ################## TESTS ##################

	Context("Run a pipeline through the driver plugin and verify that it succeeds >", FlakeAttempts(2), func() {
		var pipelineDir = "valid/essential"
		pipelineFiles := []string{
			// A single component.
			"container_no_input.yaml",
			// A chain of components passing parameters and artifacts.
			"lightweight_python_functions_with_outputs.yaml",
		}
		for _, pipelineFile := range pipelineFiles {
			It(fmt.Sprintf("Upload %s pipeline and verify the driver ran as a plugin", pipelineFile), func() {
				runID, observer := runPipelineObservingPods(pipelineFile, pipelineDir, testContext, argoClient, true)
				compiledWorkflow := workflowutils.UnmarshallWorkflowYAML(filepath.Join(testutil.GetCompiledWorkflowsFilesDir(), pipelineFile))
				e2e_utils.ValidateComponentStatuses(runClient, k8Client, testContext, runID, compiledWorkflow)
				validateDriverPluginRun(runID, observer, argoClient)
			})
		}
	})

	Context("Run a failing pipeline through the driver plugin and verify that it fails >", func() {
		var pipelineDir = "valid/failing"
		pipelineFile := "fail_v2.yaml"
		It(fmt.Sprintf("Upload %s pipeline and verify the run fails while the driver ran as a plugin", pipelineFile), func() {
			// The executor fails by design; the drivers around it must still succeed as plugin nodes.
			runID, observer := runPipelineObservingPods(pipelineFile, pipelineDir, testContext, argoClient, false)
			updatedRun := testutil.GetPipelineRun(runClient, &runID)
			Expect(updatedRun.State).NotTo(BeNil(), "Updated pipeline run state is Nil")
			Expect(*updatedRun.State).To(Equal(run_model.V2beta1RuntimeStateFAILED), "Pipeline run was expected to fail, but is "+*updatedRun.State)
			validateDriverPluginRun(runID, observer, argoClient)
		})
	})

	Context("Run a pipeline with a component that fails and succeeds on retry >", FlakeAttempts(2), func() {
		It(fmt.Sprintf("Upload %s pipeline and verify the run succeeds after retrying the executor", retryPipelineFile), func() {
			runID, observer := runPipelineObservingPods(retryPipelineFile, driverPluginPipelineDir, testContext, argoClient, true)
			updatedRun := testutil.GetPipelineRun(runClient, &runID)
			Expect(updatedRun.State).NotTo(BeNil(), "Updated pipeline run state is Nil")
			if *updatedRun.State != run_model.V2beta1RuntimeStateSUCCEEDED {
				e2e_utils.CapturePodLogsForUnsuccessfulTasks(k8Client, testContext, updatedRun.Tasks)
			}
			Expect(*updatedRun.State).To(Equal(run_model.V2beta1RuntimeStateSUCCEEDED), "Pipeline run with retries was expected to succeed, but is "+*updatedRun.State)

			// Task persistence may lag the terminal run update.
			Eventually(func(g Gomega) {
				run := testutil.GetPipelineRun(runClient, &runID)
				states := map[string]run_model.PipelineTaskTaskState{}
				for _, task := range run.Tasks {
					if task.State != nil && task.Type != nil && *task.Type == run_model.PipelineTaskTaskTypeRUNTIME {
						states[task.DisplayName] = *task.State
					}
				}
				g.Expect(states).To(HaveKeyWithValue(retryTaskName, run_model.PipelineTaskTaskStateSUCCEEDED), "Retried task did not succeed: %v", states)
				g.Expect(states).To(HaveKeyWithValue(retryConsumerName, run_model.PipelineTaskTaskStateSUCCEEDED), "Consumer of the retried task did not succeed: %v", states)
			}, time.Minute, 2*time.Second).Should(Succeed())

			workflow := validateDriverPluginRun(runID, observer, argoClient)
			attempts := e2e_utils.ValidateTaskRetried(workflow, retryTaskName, 2)
			AddReportEntry("Retry attempts of "+retryTaskName, attempts)
		})
	})
})

// runPipelineObservingPods uploads a pipeline file, runs it and waits for a
// terminal state while recording the run's pods, which is the only time the
// Argo agent pod (and its driver plugin sidecar) can be observed.
// checkImagePulls opts into failing fast on persistent image-pull failures;
// negative fixtures may intentionally use images that cannot be pulled.
func runPipelineObservingPods(pipelineFile string, pipelineDir string, testContext *apitests.TestContext, argoClient argoclient.Interface, checkImagePulls bool) (string, *e2e_utils.WorkflowPodObserver) {
	testutil.CheckIfSkipping(pipelineFile)
	pipelineFilePath := filepath.Join(testutil.GetPipelineFilesDir(), pipelineDir, pipelineFile)
	logger.Log("Uploading pipeline file %s", pipelineFile)
	uploadedPipeline, uploadErr := testutil.UploadPipeline(pipelineUploadClient, pipelineFilePath, &testContext.Pipeline.PipelineGeneratedName, nil)
	Expect(uploadErr).To(BeNil(), "Failed to upload pipeline %s", pipelineFile)
	testContext.Pipeline.CreatedPipelines = append(testContext.Pipeline.CreatedPipelines, uploadedPipeline)
	logger.Log("Upload of pipeline file '%s' successful", pipelineFile)
	uploadedPipelineVersion := testutil.GetLatestPipelineVersion(pipelineClient, &uploadedPipeline.PipelineID)
	pipelineRuntimeInputs := testutil.GetPipelineRunTimeInputs(pipelineFilePath)

	createdRun := e2e_utils.CreatePipelineRun(runClient, testContext, &uploadedPipeline.PipelineID, &uploadedPipelineVersion.PipelineVersionID, experimentID, pipelineRuntimeInputs)
	observer := e2e_utils.NewWorkflowPodObserver(k8Client, argoClient, testutil.GetNamespace(), createdRun.RunID)
	checks := []func() error{observer.Check}
	if checkImagePulls {
		checks = append(checks, testutil.NewRunImagePullCheck(k8Client, testutil.GetNamespace(), createdRun.RunID))
	}
	timeout := time.Duration(maxPipelineWaitTime)
	testutil.WaitForRunToBeInState(runClient, &createdRun.RunID, []run_model.V2beta1RuntimeState{
		run_model.V2beta1RuntimeStateSUCCEEDED,
		run_model.V2beta1RuntimeStateSKIPPED,
		run_model.V2beta1RuntimeStateFAILED,
		run_model.V2beta1RuntimeStateCANCELED,
	}, &timeout, checks...)
	// Pods that outlive the workflow (the task pods) are still worth a final look.
	Expect(observer.Check()).To(Succeed())
	return createdRun.RunID, observer
}

// validateDriverPluginRun asserts the plugin model on a finished run: the
// workflow drives tasks through "driver-plugin" templates, the driver only
// ran as the agent pod sidecar, and driver logs were published when enabled.
func validateDriverPluginRun(runID string, observer *e2e_utils.WorkflowPodObserver, argoClient argoclient.Interface) *v1alpha1.Workflow {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	workflow, err := testutil.GetWorkflowByRunID(ctx, argoClient, testutil.GetNamespace(), runID)
	Expect(err).NotTo(HaveOccurred(), "Failed to fetch the Argo Workflow of run %s", runID)
	if observed := observer.WorkflowName(); observed != "" {
		Expect(workflow.Name).To(Equal(observed), "Observed pods belong to a different workflow than run %s", runID)
	}
	AddReportEntry("Driver plugin workflow "+workflow.Name, workflow.Status.Nodes)

	e2e_utils.ValidateDriverPluginWorkflow(workflow)
	report := e2e_utils.ValidateDriverPluginPods(observer.Pods())
	AddReportEntry("Driver plugin pods of run "+runID, report)

	if e2e_utils.DriverPluginPublishesLogs(workflow) {
		e2e_utils.ValidateDriverLogsPublished(runClient, runID)
	} else {
		logger.Log("Driver templates of workflow %s compiled with publish_logs disabled; skipping %s check", workflow.Name, e2e_utils.DriverLogsURIProperty)
	}
	return workflow
}
