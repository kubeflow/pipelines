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
	"path/filepath"
	"strconv"
	"time"

	argoclient "github.com/argoproj/argo-workflows/v4/pkg/client/clientset/versioned"
	"github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/run_model"
	commonutil "github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/test/config"
	"github.com/kubeflow/pipelines/backend/test/constants"
	"github.com/kubeflow/pipelines/backend/test/testutil"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	persistenceAgentDeploymentName       = "ml-pipeline-persistenceagent"
	imagePullFailureEnabledEnvVar        = "IMAGE_PULL_FAILURE_HANDLING_ENABLED"
	imagePullFailureGracePeriodEnvVar    = "IMAGE_PULL_FAILURE_GRACE_PERIOD_SEC"
	imagePullFailureDefaultGracePeriod   = 300 * time.Second
	imagePullFailureTerminationReasonKey = "pipelines.kubeflow.org/termination-reason"
	imagePullFailureFailedImageKey       = "pipelines.kubeflow.org/failed-image"
	unpullableImage                      = "registry.invalid/kubeflow/pipelines-unpullable:does-not-exist"
	// imagePullFailureDetectionMargin is added to the configured grace period
	// to cover pod scheduling, the kubelet reporting the pull error, and the
	// persistence agent's 30s resync before and after the grace period.
	imagePullFailureDetectionMargin = 4 * time.Minute
)

// These tests run against a deployment that opted in to image pull failure
// handling (see manifests/kustomize/components/image-pull-failure-handling).
// They verify the production contract end to end: the persistence agent
// observes a workflow pod stuck pulling its image, the conditional merge patch
// it sends is accepted by the API server, Argo finishes the workflow, and the
// run is reported as failed with the image pull failure recorded on the
// workflow.
var _ = Describe("Image pull failure handling >", Serial, Label(constants.POSITIVE, constants.APIServerTests, "ImagePullFailure"), func() {
	var diagnosticRunID string
	var gracePeriod time.Duration

	BeforeEach(func() {
		diagnosticRunID = ""
		enabled, configuredGracePeriod := imagePullFailureHandlingConfig()
		if !enabled {
			Skip("Image pull failure handling is not enabled on the persistence agent")
		}
		gracePeriod = configuredGracePeriod
	})

	AfterEach(func() {
		if CurrentSpecReport().Failed() && diagnosticRunID != "" {
			AddReportEntry("Image pull failure orchestration state", collectArgoCompatibilityDiagnostics(diagnosticRunID))
		}
	})

	It("fails a run whose container image cannot be pulled", func() {
		pipelineFile := filepath.Join(pipelineFilesRootDir, "image_pull_failure", "unpullable_image.yaml")
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

		timeoutInSeconds := time.Duration((gracePeriod + imagePullFailureDetectionMargin) / time.Second)
		testutil.WaitForRunToBeInState(
			runClient,
			&createdRun.RunID,
			[]run_model.V2beta1RuntimeState{run_model.V2beta1RuntimeStateFAILED},
			&timeoutInSeconds,
		)

		restConfig, err := commonutil.GetKubernetesConfig()
		Expect(err).NotTo(HaveOccurred())
		argoClientSet, err := argoclient.NewForConfig(restConfig)
		Expect(err).NotTo(HaveOccurred())
		workflows, err := argoClientSet.ArgoprojV1alpha1().Workflows(testutil.GetNamespace()).List(context.Background(), metav1.ListOptions{
			LabelSelector: fmt.Sprintf("%s=%s", commonutil.LabelKeyWorkflowRunId, createdRun.RunID),
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(workflows.Items).To(HaveLen(1), "expected exactly one workflow for the run")
		workflow := workflows.Items[0]

		// The persistence agent terminates through KFP's activeDeadlineSeconds=0
		// patch and records why on the workflow so the cause is visible.
		Expect(workflow.Spec.ActiveDeadlineSeconds).NotTo(BeNil())
		Expect(*workflow.Spec.ActiveDeadlineSeconds).To(BeZero())
		Expect(workflow.Annotations).To(HaveKeyWithValue(imagePullFailureTerminationReasonKey, "ImagePullFailure"))
		Expect(workflow.Annotations).To(HaveKeyWithValue(imagePullFailureFailedImageKey, unpullableImage))
		Expect(workflow.Status.Phase.Completed()).To(BeTrue())
	})
})

// imagePullFailureHandlingConfig reads the persistence agent deployment and
// reports whether image pull failure handling is enabled and with which grace
// period, so the test adapts to the deployment under test instead of assuming
// a particular overlay.
func imagePullFailureHandlingConfig() (bool, time.Duration) {
	deployment, err := k8Client.AppsV1().Deployments(*config.Namespace).Get(context.Background(), persistenceAgentDeploymentName, metav1.GetOptions{})
	Expect(err).NotTo(HaveOccurred(), "failed to read the persistence agent deployment")

	enabled := false
	gracePeriod := imagePullFailureDefaultGracePeriod
	for _, container := range deployment.Spec.Template.Spec.Containers {
		if container.Name != persistenceAgentDeploymentName {
			continue
		}
		for _, envVar := range container.Env {
			switch envVar.Name {
			case imagePullFailureEnabledEnvVar:
				enabled = envVar.Value == "true"
			case imagePullFailureGracePeriodEnvVar:
				seconds, err := strconv.Atoi(envVar.Value)
				Expect(err).NotTo(HaveOccurred(), "invalid %s on the persistence agent", imagePullFailureGracePeriodEnvVar)
				gracePeriod = time.Duration(seconds) * time.Second
			}
		}
	}
	return enabled, gracePeriod
}
