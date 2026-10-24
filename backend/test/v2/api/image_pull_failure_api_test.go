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
	"os"
	"path/filepath"
	"strconv"
	"time"

	argoclient "github.com/argoproj/argo-workflows/v4/pkg/client/clientset/versioned"
	"github.com/kubeflow/pipelines/backend/api/v2/go_http_client/run_model"
	commonutil "github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/test/config"
	"github.com/kubeflow/pipelines/backend/test/constants"
	"github.com/kubeflow/pipelines/backend/test/testutil"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	persistenceAgentDeploymentName     = "ml-pipeline-persistenceagent"
	imagePullFailureEnabledEnvVar      = "IMAGE_PULL_FAILURE_HANDLING_ENABLED"
	imagePullFailureGracePeriodEnvVar  = "IMAGE_PULL_FAILURE_GRACE_PERIOD_SEC"
	imagePullFailureDefaultGracePeriod = 300 * time.Second
	// imagePullFailureExpectedEnvVar is set to "true" by CI jobs whose overlay
	// opts into image pull failure handling. In those jobs a disabled feature is
	// a failure, not a skip, so the end-to-end contract cannot silently stop
	// being exercised if the component or its env patch is dropped.
	imagePullFailureExpectedEnvVar = "IMAGE_PULL_FAILURE_HANDLING_EXPECTED"
	unpullableImage                = "registry.invalid/kubeflow/pipelines-unpullable:does-not-exist"
	// imagePullFailureDetectionMargin is added to the configured grace period
	// to cover pod scheduling, the kubelet reporting the pull error, and the
	// persistence agent's 30s resync before and after the grace period.
	imagePullFailureDetectionMargin = 4 * time.Minute
)

// These tests run against a deployment that opted in to image pull failure
// handling (see manifests/kustomize/components/image-pull-failure-handling).
// They verify both production paths end to end: the persistence agent
// terminates Argo Workflows with the failure annotations, and the Kubernetes
// coordinator confirms termination and records the managed task's failure.
// Overlays that leave the feature disabled skip these tests, unless the job
// declares it expected (see imagePullFailureExpectedEnvVar).
var _ = Describe("Image pull failure handling >", Serial, Label(constants.POSITIVE, constants.APIServerTests, "ImagePullFailure"), func() {
	var diagnosticRunID string
	var gracePeriod time.Duration

	BeforeEach(func() {
		diagnosticRunID = ""
		enabled, configuredGracePeriod := imagePullFailureHandlingConfig()
		if !enabled {
			if os.Getenv(imagePullFailureExpectedEnvVar) == "true" {
				Fail(fmt.Sprintf("%s=true but %s is not enabled on the selected execution backend deployment: this job's overlay must opt in through manifests/kustomize/components/image-pull-failure-handling",
					imagePullFailureExpectedEnvVar, imagePullFailureEnabledEnvVar))
			}
			Skip("Image pull failure handling is not enabled for the selected execution backend")
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

		waitTimeout := imagePullFailureWaitTimeout(gracePeriod)
		testutil.WaitForRunToBeInState(
			runClient,
			&createdRun.RunID,
			[]run_model.V2RuntimeState{run_model.V2RuntimeStateFAILED},
			&waitTimeout,
		)

		restConfig, err := commonutil.GetKubernetesConfig()
		Expect(err).NotTo(HaveOccurred())
		argoClientSet, err := argoclient.NewForConfig(restConfig)
		Expect(err).NotTo(HaveOccurred())
		workflows, err := argoClientSet.ArgoprojV1alpha1().Workflows(testutil.GetNamespace()).List(context.Background(), metav1.ListOptions{
			LabelSelector: fmt.Sprintf("%s=%s", commonutil.LabelKeyWorkflowRunId, createdRun.RunID),
		})
		Expect(err).NotTo(HaveOccurred())
		if testutil.ShouldUseCoordinatorRuntimeSelector() {
			Expect(workflows.Items).To(BeEmpty(), "coordinator-managed runs must not create Argo workflows")
			run := testutil.GetPipelineRun(runClient, &createdRun.RunID)
			Expect(run.ExecutorType).To(Equal("kubernetes"))
			Expect(run.Tasks).To(ContainElement(SatisfyAll(
				HaveField("State", Equal(run_model.PipelineTaskTaskStateFAILED.Pointer())),
				HaveField("StatusMetadata", Not(BeNil())),
				HaveField("StatusMetadata.CustomProperties", SatisfyAll(
					HaveKeyWithValue("terminationReason", commonutil.TerminationReasonImagePullFailure),
					HaveKeyWithValue("failedImage", unpullableImage),
				)),
			)), "the task must retain the image-pull failure cause")
			return
		}
		Expect(workflows.Items).To(HaveLen(1), "expected exactly one workflow for the run")
		workflow := workflows.Items[0]

		// The persistence agent terminates through KFP's activeDeadlineSeconds=0
		// patch and records why on the workflow so the cause is visible.
		Expect(workflow.Spec.ActiveDeadlineSeconds).NotTo(BeNil())
		Expect(*workflow.Spec.ActiveDeadlineSeconds).To(BeZero())
		Expect(workflow.Annotations).To(HaveKeyWithValue(commonutil.AnnotationKeyTerminationReason, commonutil.TerminationReasonImagePullFailure))
		Expect(workflow.Annotations).To(HaveKeyWithValue(commonutil.AnnotationKeyFailedImage, unpullableImage))
		Expect(workflow.Status.Phase.Completed()).To(BeTrue())
	})
})

func imagePullFailureWaitTimeout(gracePeriod time.Duration) time.Duration {
	return gracePeriod + imagePullFailureDetectionMargin
}

func imagePullFailureBackendDeployment(coordinator bool) (string, string) {
	if coordinator {
		return "ml-pipeline", "ml-pipeline-api-server"
	}
	return persistenceAgentDeploymentName, persistenceAgentDeploymentName
}

// imagePullFailureHandlingConfig reads the selected engine's deployment so the
// test uses the grace period of the component that actually owns the workload.
func imagePullFailureHandlingConfig() (bool, time.Duration) {
	deploymentName, containerName := imagePullFailureBackendDeployment(testutil.ShouldUseCoordinatorRuntimeSelector())
	deployment, err := k8Client.AppsV1().Deployments(*config.Namespace).Get(context.Background(), deploymentName, metav1.GetOptions{})
	Expect(err).NotTo(HaveOccurred(), "failed to read the workload owner's deployment")

	enabled := false
	gracePeriod := imagePullFailureDefaultGracePeriod
	for _, container := range deployment.Spec.Template.Spec.Containers {
		if container.Name != containerName {
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
