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

package api

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	argoclient "github.com/argoproj/argo-workflows/v4/pkg/client/clientset/versioned"
	recurringrunparams "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/recurring_run_client/recurring_run_service"
	"github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/recurring_run_model"
	runparams "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/run_client/run_service"
	"github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/run_model"
	commonutil "github.com/kubeflow/pipelines/backend/src/common/util"
	swfclientset "github.com/kubeflow/pipelines/backend/src/crd/pkg/client/clientset/versioned"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata/testutils"
	"github.com/kubeflow/pipelines/backend/test/config"
	"github.com/kubeflow/pipelines/backend/test/constants"
	"github.com/kubeflow/pipelines/backend/test/testutil"
	pb "github.com/kubeflow/pipelines/third_party/ml-metadata/go/ml_metadata"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const argoCompatibilityTestsEnvironmentVariable = "ARGO_COMPATIBILITY_TESTS"

const (
	argoNodeNameAnnotation = "workflows.argoproj.io/node-name"
	artifactTaskName       = "write-artifact"
	artifactContainerImage = "alpine:3.23"
)

var _ = Describe("Argo runtime compatibility >", Serial, Label(constants.POSITIVE, constants.APIServerTests, "ArgoCompatibility"), func() {
	var diagnosticRunID string

	BeforeEach(func() {
		diagnosticRunID = ""
		if os.Getenv(argoCompatibilityTestsEnvironmentVariable) != "true" {
			Skip("Argo compatibility tests run only in the canonical Argo 4 API test job")
		}
	})

	AfterEach(func() {
		if CurrentSpecReport().Failed() && diagnosticRunID != "" {
			AddReportEntry("Argo compatibility orchestration state", collectArgoCompatibilityDiagnostics(diagnosticRunID))
		}
	})

	It("creates a disabled recurring run and its ScheduledWorkflow", func() {
		pipelineFile := filepath.Join(testutil.GetValidPipelineFilesDir(), helloWorldPipelineFileName)
		createdExperiment := createExperiment(experimentName)
		createdPipeline := uploadAPipeline(pipelineFile, &testContext.Pipeline.PipelineGeneratedName)
		createdPipelineVersion := testutil.GetLatestPipelineVersion(pipelineClient, &createdPipeline.PipelineID)

		recurringRun := &recurring_run_model.V2beta1RecurringRun{
			DisplayName:    "Argo compatibility recurring run - " + randomName,
			Description:    "Validates recurring-run creation against the supported Argo version",
			ExperimentID:   createdExperiment.ExperimentID,
			ServiceAccount: testutil.GetDefaultPipelineRunnerServiceAccount(),
			PipelineVersionReference: &recurring_run_model.V2beta1PipelineVersionReference{
				PipelineID:        createdPipeline.PipelineID,
				PipelineVersionID: createdPipelineVersion.PipelineVersionID,
			},
			RuntimeConfig: &recurring_run_model.V2beta1RuntimeConfig{
				Parameters: testutil.GetPipelineRunTimeInputs(pipelineFile),
			},
			Trigger: &recurring_run_model.V2beta1Trigger{
				CronSchedule: &recurring_run_model.V2beta1CronSchedule{Cron: "0 0 1 1 *"},
			},
			Mode:           recurring_run_model.RecurringRunModeDISABLE.Pointer(),
			MaxConcurrency: 1,
			NoCatchup:      true,
		}

		createdRecurringRun, err := recurringRunClient.Create(&recurringrunparams.RecurringRunServiceCreateRecurringRunParams{
			RecurringRun: recurringRun,
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(createdRecurringRun.RecurringRunID).NotTo(BeEmpty())
		DeferCleanup(func() {
			err := recurringRunClient.Delete(&recurringrunparams.RecurringRunServiceDeleteRecurringRunParams{
				RecurringRunID: createdRecurringRun.RecurringRunID,
			})
			Expect(err).NotTo(HaveOccurred())
		})

		storedRecurringRun, err := recurringRunClient.Get(&recurringrunparams.RecurringRunServiceGetRecurringRunParams{
			RecurringRunID: createdRecurringRun.RecurringRunID,
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(storedRecurringRun.Mode).To(Equal(recurring_run_model.RecurringRunModeDISABLE.Pointer()))
		Expect(storedRecurringRun.Trigger.CronSchedule.Cron).To(Equal("0 0 1 1 *"))

		restConfig, err := commonutil.GetKubernetesConfig()
		Expect(err).NotTo(HaveOccurred())
		scheduledWorkflowClientSet, err := swfclientset.NewForConfig(restConfig)
		Expect(err).NotTo(HaveOccurred())

		Eventually(func(g Gomega) {
			scheduledWorkflows, err := scheduledWorkflowClientSet.ScheduledworkflowV1beta1().
				ScheduledWorkflows(testutil.GetNamespace()).List(context.Background(), metav1.ListOptions{})
			g.Expect(err).NotTo(HaveOccurred())

			found := false
			for index := range scheduledWorkflows.Items {
				scheduledWorkflow := &scheduledWorkflows.Items[index]
				if string(scheduledWorkflow.UID) != createdRecurringRun.RecurringRunID {
					continue
				}
				found = true
				g.Expect(scheduledWorkflow.Spec.Enabled).To(BeFalse())
				g.Expect(scheduledWorkflow.Spec.PipelineId).To(Equal(createdPipeline.PipelineID))
				g.Expect(scheduledWorkflow.Spec.PipelineVersionId).To(Equal(createdPipelineVersion.PipelineVersionID))
			}
			g.Expect(found).To(BeTrue())
		}, "30s", "1s").Should(Succeed())
	})

	It("retries a failed run through the run API", func() {
		pipelineFile := filepath.Join(testutil.GetValidPipelineFilesDir(), "failing", "fail_v2.yaml")
		createdExperiment := createExperiment(experimentName)
		createdPipeline := uploadAPipeline(pipelineFile, &testContext.Pipeline.PipelineGeneratedName)
		createdPipelineVersion := testutil.GetLatestPipelineVersion(pipelineClient, &createdPipeline.PipelineID)
		createdRun := createPipelineRun(
			&createdPipeline.PipelineID,
			&createdPipelineVersion.PipelineVersionID,
			&createdExperiment.ExperimentID,
			testutil.GetPipelineRunTimeInputs(pipelineFile),
		)
		diagnosticRunID = createdRun.RunID

		retryTimeout := time.Duration(180)
		testutil.WaitForRunToBeInState(
			runClient,
			&createdRun.RunID,
			[]run_model.V2beta1RuntimeState{run_model.V2beta1RuntimeStateFAILED},
			&retryTimeout,
		)
		failedRun := testutil.GetPipelineRun(runClient, &createdRun.RunID)
		stateHistoryLengthBeforeRetry := len(failedRun.StateHistory)

		err := runClient.Retry(&runparams.RunServiceRetryRunParams{RunID: createdRun.RunID})
		Expect(err).NotTo(HaveOccurred())

		testutil.WaitForRunToBeInState(
			runClient,
			&createdRun.RunID,
			[]run_model.V2beta1RuntimeState{run_model.V2beta1RuntimeStateFAILED},
			&retryTimeout,
		)
		retriedRun := testutil.GetPipelineRun(runClient, &createdRun.RunID)
		Expect(len(retriedRun.StateHistory)).To(BeNumerically(">", stateHistoryLengthBeforeRetry))
		Expect(runtimeStateAppearsAfter(retriedRun.StateHistory, stateHistoryLengthBeforeRetry, run_model.V2beta1RuntimeStateRUNNING)).To(BeTrue())
	})

	It("writes execution and artifact metadata and serves archived logs after pod deletion", func() {
		pipelineFile := filepath.Join(pipelineFilesRootDir, "argo_compatibility", "fast_artifact.yaml")
		mlmdClient, err := testutils.NewTestMlmdClient(
			"127.0.0.1",
			metadata.GetMetadataConfig().Port,
			*config.TLSEnabled,
			*config.CaCertPath,
		)
		Expect(err).NotTo(HaveOccurred())
		createdExperiment := createExperiment(experimentName)
		createdPipeline := uploadAPipeline(pipelineFile, &testContext.Pipeline.PipelineGeneratedName)
		createdPipelineVersion := testutil.GetLatestPipelineVersion(pipelineClient, &createdPipeline.PipelineID)
		createdRun := createPipelineRun(
			&createdPipeline.PipelineID,
			&createdPipelineVersion.PipelineVersionID,
			&createdExperiment.ExperimentID,
			testutil.GetPipelineRunTimeInputs(pipelineFile),
		)
		diagnosticRunID = createdRun.RunID

		var logPodName string
		Eventually(func() string {
			pods, err := k8Client.CoreV1().Pods(testutil.GetNamespace()).List(context.Background(), metav1.ListOptions{
				LabelSelector: fmt.Sprintf("%s=%s", commonutil.LabelKeyWorkflowRunId, createdRun.RunID),
			})
			if err != nil {
				return ""
			}
			logPodName = findArgoCompatibilityPodName(pods.Items)
			return logPodName
		}, "180s", "1s").ShouldNot(BeEmpty())

		Eventually(func() string {
			logContents, err := k8Client.CoreV1().Pods(testutil.GetNamespace()).
				GetLogs(logPodName, &corev1.PodLogOptions{Container: "main"}).
				DoRaw(context.Background())
			if err != nil {
				return ""
			}
			return string(logContents)
		}, "180s", "1s").Should(ContainSubstring("input:  foo"))

		artifactTimeout := time.Duration(300)
		testutil.WaitForRunToBeInState(
			runClient,
			&createdRun.RunID,
			[]run_model.V2beta1RuntimeState{run_model.V2beta1RuntimeStateSUCCEEDED},
			&artifactTimeout,
		)

		Eventually(func() bool {
			requestContext, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			contextsFilterQuery := fmt.Sprintf("name = '%s'", createdRun.RunID)
			contexts, err := mlmdClient.GetContexts(requestContext, &pb.GetContextsRequest{
				Options: &pb.ListOperationOptions{FilterQuery: &contextsFilterQuery},
			})
			if err != nil {
				return false
			}

			var executions []*pb.Execution
			for _, metadataContext := range contexts.GetContexts() {
				contextID := metadataContext.GetId()
				executionsResponse, err := mlmdClient.GetExecutionsByContext(requestContext, &pb.GetExecutionsByContextRequest{
					ContextId: &contextID,
				})
				if err != nil {
					return false
				}
				executions = append(executions, executionsResponse.GetExecutions()...)
			}

			executionIDs := make([]int64, 0, len(executions))
			for _, execution := range executions {
				executionIDs = append(executionIDs, execution.GetId())
			}
			events, err := mlmdClient.GetEventsByExecutionIDs(requestContext, &pb.GetEventsByExecutionIDsRequest{
				ExecutionIds: executionIDs,
			})
			if err != nil {
				return false
			}

			executionID, artifactID := findArgoCompatibilityMetadataIDs(executions, events.GetEvents())
			if executionID == 0 || artifactID == 0 {
				return false
			}
			artifacts, err := mlmdClient.GetArtifactsByID(requestContext, &pb.GetArtifactsByIDRequest{
				ArtifactIds: []int64{artifactID},
			})
			return err == nil && len(artifacts.GetArtifacts()) == 1 && artifacts.GetArtifacts()[0].GetId() == artifactID
		}, "120s", "2s").Should(BeTrue())

		archivePipelineRun(&createdRun.RunID)
		storedRun := testutil.GetPipelineRun(runClient, &createdRun.RunID)
		Expect(storedRun.StorageState).NotTo(BeNil())
		Expect(*storedRun.StorageState).To(Equal(run_model.V2beta1RunStorageStateARCHIVED))

		zeroGracePeriod := int64(0)
		err = k8Client.CoreV1().Pods(testutil.GetNamespace()).Delete(context.Background(), logPodName, metav1.DeleteOptions{
			GracePeriodSeconds: &zeroGracePeriod,
		})
		Expect(err == nil || apierrors.IsNotFound(err)).To(BeTrue())
		Eventually(func() bool {
			_, err := k8Client.CoreV1().Pods(testutil.GetNamespace()).Get(context.Background(), logPodName, metav1.GetOptions{})
			return apierrors.IsNotFound(err)
		}, "30s", "1s").Should(BeTrue())

		Eventually(func() string {
			logContents, err := readArgoCompatibilityRunLog(createdRun.RunID, logPodName)
			if err != nil {
				return ""
			}
			return logContents
		}, "90s", "3s").Should(ContainSubstring("input:  foo"))
	})
})

func collectArgoCompatibilityDiagnostics(runID string) string {
	var report strings.Builder
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	fmt.Fprintf(&report, "Run ID: %s\n", runID)
	run, err := runClient.Get(&runparams.RunServiceGetRunParams{RunID: runID})
	if err != nil {
		fmt.Fprintf(&report, "Run lookup error: %v\n", err)
	} else {
		runState := "<unset>"
		if run.State != nil {
			runState = string(*run.State)
		}
		fmt.Fprintf(&report, "Run state: %s\n", runState)
		fmt.Fprintln(&report, "Run state history:")
		for _, status := range run.StateHistory {
			if status == nil {
				continue
			}
			state := "<unset>"
			if status.State != nil {
				state = string(*status.State)
			}
			fmt.Fprintf(&report, "- state=%s updated=%s error=%v\n", state, status.UpdateTime, status.Error)
		}
	}

	restConfig, err := commonutil.GetKubernetesConfig()
	if err != nil {
		fmt.Fprintf(&report, "Kubernetes config error: %v\n", err)
		return report.String()
	}

	namespace := testutil.GetNamespace()
	selector := fmt.Sprintf("%s=%s", commonutil.LabelKeyWorkflowRunId, runID)
	objectNames := make(map[string]struct{})

	argoClient, err := argoclient.NewForConfig(restConfig)
	if err != nil {
		fmt.Fprintf(&report, "Argo client error: %v\n", err)
	} else {
		workflows, listErr := argoClient.ArgoprojV1alpha1().Workflows(namespace).List(ctx, metav1.ListOptions{LabelSelector: selector})
		switch {
		case listErr != nil:
			fmt.Fprintf(&report, "Workflow lookup error: %v\n", listErr)
		case len(workflows.Items) == 0:
			fmt.Fprintln(&report, "Workflows: none")
		default:
			fmt.Fprintln(&report, "Workflows:")
			for index := range workflows.Items {
				workflow := &workflows.Items[index]
				objectNames[workflow.Name] = struct{}{}
				fmt.Fprintf(
					&report,
					"- name=%s phase=%s message=%q created=%s nodes=%d\n",
					workflow.Name,
					workflow.Status.Phase,
					workflow.Status.Message,
					workflow.CreationTimestamp.Time.UTC().Format(time.RFC3339),
					len(workflow.Status.Nodes),
				)
			}
		}
	}

	pods, err := k8Client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{LabelSelector: selector})
	switch {
	case err != nil:
		fmt.Fprintf(&report, "Pod lookup error: %v\n", err)
	case len(pods.Items) == 0:
		fmt.Fprintln(&report, "Pods: none")
	default:
		fmt.Fprintln(&report, "Pods:")
		for index := range pods.Items {
			pod := &pods.Items[index]
			objectNames[pod.Name] = struct{}{}
			fmt.Fprintf(&report, "- name=%s phase=%s node=%s reason=%q message=%q\n", pod.Name, pod.Status.Phase, pod.Spec.NodeName, pod.Status.Reason, pod.Status.Message)
			for _, container := range pod.Status.ContainerStatuses {
				fmt.Fprintf(&report, "  container=%s ready=%t restarts=%d state=%v\n", container.Name, container.Ready, container.RestartCount, container.State)
			}
		}
	}

	events, err := k8Client.CoreV1().Events(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		fmt.Fprintf(&report, "Event lookup error: %v\n", err)
		return report.String()
	}
	fmt.Fprintln(&report, "Relevant events:")
	eventCount := 0
	for index := range events.Items {
		event := &events.Items[index]
		_, objectIsRelevant := objectNames[event.InvolvedObject.Name]
		isRecentWarning := len(objectNames) == 0 && event.Type == corev1.EventTypeWarning &&
			event.LastTimestamp.After(testContext.TestStartTimeUTC)
		if !objectIsRelevant && !isRecentWarning {
			continue
		}
		fmt.Fprintf(&report, "- type=%s reason=%s object=%s/%s count=%d message=%q\n", event.Type, event.Reason, event.InvolvedObject.Kind, event.InvolvedObject.Name, event.Count, event.Message)
		eventCount++
		if eventCount == 50 {
			fmt.Fprintln(&report, "- additional events omitted")
			break
		}
	}
	if eventCount == 0 {
		fmt.Fprintln(&report, "- none")
	}

	return report.String()
}

func runtimeStateAppearsAfter(
	stateHistory []*run_model.V2beta1RuntimeStatus,
	startIndex int,
	expectedState run_model.V2beta1RuntimeState,
) bool {
	if startIndex < 0 || startIndex > len(stateHistory) {
		return false
	}
	for _, runtimeStatus := range stateHistory[startIndex:] {
		if runtimeStatus.State != nil && *runtimeStatus.State == expectedState {
			return true
		}
	}
	return false
}

func findArgoCompatibilityPodName(pods []corev1.Pod) string {
	for _, pod := range pods {
		nodeName := pod.Annotations[argoNodeNameAnnotation]
		if nodeName == artifactTaskName || strings.HasSuffix(nodeName, "."+artifactTaskName) {
			return pod.Name
		}
		for _, container := range pod.Spec.Containers {
			if container.Image == artifactContainerImage || strings.HasSuffix(container.Image, "/"+artifactContainerImage) {
				return pod.Name
			}
		}
	}
	return ""
}

func findArgoCompatibilityMetadataIDs(executions []*pb.Execution, events []*pb.Event) (int64, int64) {
	containerExecutionIDs := make(map[int64]struct{})
	for _, execution := range executions {
		if execution.GetId() > 0 && execution.GetType() == string(metadata.ContainerExecutionTypeName) {
			containerExecutionIDs[execution.GetId()] = struct{}{}
		}
	}

	for _, event := range events {
		if event.GetType() != pb.Event_OUTPUT || event.GetArtifactId() == 0 {
			continue
		}
		if _, exists := containerExecutionIDs[event.GetExecutionId()]; exists {
			return event.GetExecutionId(), event.GetArtifactId()
		}
	}
	return 0, 0
}

func readArgoCompatibilityRunLog(runID string, nodeID string) (string, error) {
	logURL := fmt.Sprintf(
		"%s/apis/v1alpha1/runs/%s/nodes/%s/log?follow=false",
		strings.TrimRight(*config.ApiUrl, "/"),
		url.PathEscape(runID),
		url.PathEscape(nodeID),
	)
	request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, logURL, nil)
	if err != nil {
		return "", err
	}
	if *config.AuthToken != "" {
		request.Header.Set("Authorization", "Bearer "+*config.AuthToken)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	response, err := client.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	contents, err := io.ReadAll(response.Body)
	if err != nil {
		return "", err
	}
	if response.StatusCode != http.StatusOK {
		return string(contents), fmt.Errorf("run log request returned %s", response.Status)
	}
	return string(contents), nil
}
