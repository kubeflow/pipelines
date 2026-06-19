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

	recurringrunparams "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/recurring_run_client/recurring_run_service"
	"github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/recurring_run_model"
	runparams "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/run_client/run_service"
	"github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/run_model"
	commonutil "github.com/kubeflow/pipelines/backend/src/common/util"
	swfclientset "github.com/kubeflow/pipelines/backend/src/crd/pkg/client/clientset/versioned"
	"github.com/kubeflow/pipelines/backend/test/config"
	"github.com/kubeflow/pipelines/backend/test/constants"
	"github.com/kubeflow/pipelines/backend/test/testutil"

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

var _ = Describe("Argo runtime compatibility >", Label(constants.POSITIVE, constants.APIServerTests, "ArgoCompatibility"), func() {
	BeforeEach(func() {
		if os.Getenv(argoCompatibilityTestsEnvironmentVariable) != "true" {
			Skip("Argo compatibility tests run only in the canonical Argo 4 API test job")
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

	It("preserves task metadata, artifact IDs, and archived logs after pod deletion", func() {
		pipelineFile := filepath.Join(pipelineFilesRootDir, "argo_compatibility", "fast_artifact.yaml")
		createdExperiment := createExperiment(experimentName)
		createdPipeline := uploadAPipeline(pipelineFile, &testContext.Pipeline.PipelineGeneratedName)
		createdPipelineVersion := testutil.GetLatestPipelineVersion(pipelineClient, &createdPipeline.PipelineID)
		createdRun := createPipelineRun(
			&createdPipeline.PipelineID,
			&createdPipelineVersion.PipelineVersionID,
			&createdExperiment.ExperimentID,
			testutil.GetPipelineRunTimeInputs(pipelineFile),
		)

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
			if _, err := k8Client.CoreV1().Pods(testutil.GetNamespace()).Get(context.Background(), logPodName, metav1.GetOptions{}); err != nil {
				return ""
			}
			logContents, err := readArgoCompatibilityRunLog(createdRun.RunID, logPodName)
			if err != nil {
				return ""
			}
			return logContents
		}, "180s", "1s").Should(ContainSubstring("input:  foo"))

		artifactTimeout := time.Duration(300)
		testutil.WaitForRunToBeInState(
			runClient,
			&createdRun.RunID,
			[]run_model.V2beta1RuntimeState{run_model.V2beta1RuntimeStateSUCCEEDED},
			&artifactTimeout,
		)

		Eventually(func() bool {
			storedRun, err := runClient.Get(&runparams.RunServiceGetRunParams{RunID: createdRun.RunID})
			if err != nil {
				return false
			}
			return argoCompatibilityMetadataReady(storedRun)
		}, "120s", "2s").Should(BeTrue())

		archivePipelineRun(&createdRun.RunID)
		storedRun := testutil.GetPipelineRun(runClient, &createdRun.RunID)
		Expect(storedRun.StorageState).NotTo(BeNil())
		Expect(*storedRun.StorageState).To(Equal(run_model.V2beta1RunStorageStateARCHIVED))

		zeroGracePeriod := int64(0)
		err := k8Client.CoreV1().Pods(testutil.GetNamespace()).Delete(context.Background(), logPodName, metav1.DeleteOptions{
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

func argoCompatibilityMetadataReady(run *run_model.V2beta1Run) bool {
	if run.RunDetails == nil {
		return false
	}

	for _, task := range run.RunDetails.TaskDetails {
		if task == nil || task.ExecutionID == "" {
			continue
		}
		for _, artifacts := range task.Outputs {
			if len(artifacts.ArtifactIds) > 0 {
				return true
			}
		}
	}
	return false
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
