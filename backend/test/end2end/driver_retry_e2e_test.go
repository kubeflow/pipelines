// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package end2end

import (
	"context"
	_ "embed"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	wfapi "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	argoclient "github.com/argoproj/argo-workflows/v4/pkg/client/clientset/versioned"
	"github.com/kubeflow/pipelines/backend/api/v2/go_http_client/run_model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	e2eutils "github.com/kubeflow/pipelines/backend/test/end2end/utils"
	"github.com/kubeflow/pipelines/backend/test/testutil"
	apitests "github.com/kubeflow/pipelines/backend/test/v2/api"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/yaml"
)

//go:embed test_data/driver_retry.yaml
var driverRetryPipeline string

// Serial is required because the fault image replaces the namespace's executor
// plugin configmap. The dedicated CI lane uses a disposable KFP deployment.
var _ = Describe("Driver retry runtime", Serial, Label("E2EDriverRetry"), func() {
	var tc *apitests.TestContext
	var argo argoclient.Interface
	BeforeEach(func() {
		image := os.Getenv("KFP_DRIVER_RETRY_TEST_IMAGE")
		Expect(image).NotTo(BeEmpty(), "Build backend/test/driverretry/faultdriver and set KFP_DRIVER_RETRY_TEST_IMAGE; see backend/test/driverretry/README.md")
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		configs := k8Client.CoreV1().ConfigMaps(testutil.GetNamespace())
		original, err := configs.Get(ctx, "ml-pipeline-driver-agent", metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() {
			cleanupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			current, err := configs.Get(cleanupCtx, original.Name, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			current.Data = original.Data
			_, err = configs.Update(cleanupCtx, current, metav1.UpdateOptions{})
			Expect(err).NotTo(HaveOccurred())
			restartDriverRetryController()
		})
		updated := original.DeepCopy()
		var sidecar corev1.Container
		Expect(yaml.Unmarshal([]byte(updated.Data["sidecar.container"]), &sidecar)).To(Succeed())
		sidecar.Image = image
		sidecar.Command = []string{"/bin/driver-retry-test"}
		sidecar.Args = nil
		encoded, err := yaml.Marshal(sidecar)
		Expect(err).NotTo(HaveOccurred())
		updated.Data["sidecar.container"] = string(encoded)
		_, err = configs.Update(ctx, updated, metav1.UpdateOptions{})
		Expect(err).NotTo(HaveOccurred())
		restartDriverRetryController()
		argo, err = testutil.CreateArgoClient()
		Expect(err).NotTo(HaveOccurred())
		tc = &apitests.TestContext{TestStartTimeUTC: time.Now()}
		tc.Pipeline.PipelineGeneratedName = "driver-retry-" + time.Now().Format("150405.000000000")
		DeferCleanup(cleanupE2ETestResources, tc)
	})

	AfterEach(func() {
		if CurrentSpecReport().Failed() && tc != nil && len(tc.PipelineRun.CreatedRunIds) > 0 {
			report, _ := testutil.BuildArchivedWorkflowLogsReport(k8Client, tc.PipelineRun.CreatedRunIds)
			AddReportEntry(testutil.ArchivedWorkflowLogsReportTitle, report)
		}
	})

	It("replays committed driver output after its plugin process dies before responding", func() {
		runID, observer := startDriverRetryRun(tc, argo, "restart-driver")
		waitDriverRetryRun(runID, run_model.V2RuntimeStateSUCCEEDED, observer.Check)
		workflow := getDriverRetryWorkflow(argo, runID)
		task := uniqueDriverRetryTask(runID, "restart-driver")
		Expect(*task.State).To(Equal(run_model.PipelineTaskTaskStateSUCCEEDED))
		Expect(task.Inputs.Parameters).To(HaveLen(1))
		Expect(task.Inputs.Parameters[0].Value).To(Equal("preserved-input"))
		attempts, succeeded := 0, 0
		for _, node := range workflow.Status.Nodes {
			if node.Type == wfapi.NodeTypePlugin && strings.HasPrefix(node.DisplayName, "restart-driver-driver(") && strings.HasPrefix(node.TemplateName, "retry-") {
				attempts++
				if node.Phase != wfapi.NodeSucceeded {
					Expect(node.Phase).To(Equal(wfapi.NodeError), "Only transport failures may precede recovery")
					continue
				}
				succeeded++
				Expect(node.Outputs).NotTo(BeNil())
				foundTaskID := false
				for _, parameter := range node.Outputs.Parameters {
					if parameter.Name == "task-id" {
						Expect(parameter.Value.String()).To(Equal(task.TaskID))
						foundTaskID = true
					}
				}
				Expect(foundTaskID).To(BeTrue())
			}
		}
		// A slow container restart can outlast Argo's bounded HTTP retry, in
		// which case the infrastructure Error consumes a configured attempt.
		Expect(attempts).To(BeNumerically(">=", 1))
		Expect(attempts).To(BeNumerically("<=", 3))
		Expect(succeeded).To(Equal(1))
		restarted := false
		for _, pod := range observer.Pods() {
			if !e2eutils.IsArgoAgentPod(pod) {
				continue
			}
			for _, container := range pod.Status.ContainerStatuses {
				if container.Name == e2eutils.DriverPluginName && container.RestartCount > 0 {
					restarted = true
					Expect(container.LastTerminationState.Terminated).NotTo(BeNil())
					Expect(container.LastTerminationState.Terminated.ExitCode).To(Equal(int32(86)))
				}
			}
		}
		Expect(restarted).To(BeTrue(), "The test must observe an actual plugin process crash after the committed response")
		Expect(e2eutils.TaskAttemptNodes(workflow, "restart-driver")).To(HaveLen(1), "Replay must not launch duplicate executors")
	})

	It("finalizes exhausted drivers before cleanup and recovers on manual retry", func() {
		runID, observer := startDriverRetryRun(tc, argo, "exhaust-driver")
		observedCleanup := false
		checkCleanup := func() error {
			run := testutil.GetPipelineRun(runClient, &runID)
			for _, task := range run.Tasks {
				if task.DisplayName != "cleanup" || task.State == nil || *task.State != run_model.PipelineTaskTaskStateRUNNING {
					continue
				}
				failed := uniqueDriverRetryTask(runID, "exhaust-driver")
				Expect(*failed.State).To(Equal(run_model.PipelineTaskTaskStateFAILED), "Driver finalization must precede exit-handler execution")
				Expect(*run.State).To(Equal(run_model.V2RuntimeStateRUNNING), "Check must happen before workflow terminal persistence")
				expectDriverRetryCleanupStatus(task, "FAILED")
				observedCleanup = true
			}
			return observer.Check()
		}
		waitDriverRetryRun(runID, run_model.V2RuntimeStateFAILED, checkCleanup)
		Expect(observedCleanup).To(BeTrue(), "Must observe the exit handler while the workflow is still running")
		workflow := getDriverRetryWorkflow(argo, runID)
		originalTaskID := uniqueDriverRetryTask(runID, "exhaust-driver").TaskID
		driverAttempts, finalizers := 0, 0
		for _, node := range workflow.Status.Nodes {
			if node.Type != wfapi.NodeTypePlugin {
				continue
			}
			if strings.HasPrefix(node.DisplayName, "exhaust-driver-driver(") && !strings.Contains(node.TemplateName, "finalize-") {
				driverAttempts++
				Expect(node.Phase).To(Equal(wfapi.NodeFailed))
				Expect(node.Message).To(ContainSubstring("lost committed response"))
			}
			if strings.HasPrefix(node.TemplateName, "finalize-") {
				finalizers++
				Expect(node.Phase).To(Equal(wfapi.NodeSucceeded))
			}
		}
		Expect(driverAttempts).To(Equal(3), "Initial execution plus two configured retries")
		Expect(finalizers).To(Equal(1))
		Expect(e2eutils.TaskAttemptNodes(workflow, "exhaust-driver")).To(BeEmpty(), "An exhausted driver must not launch the component")
		e2eutils.RetryPipelineRun(runClient, runID)
		waitDriverRetryRun(runID, run_model.V2RuntimeStateSUCCEEDED, observer.Check)
		retried := uniqueDriverRetryTask(runID, "exhaust-driver")
		Expect(retried.TaskID).To(Equal(originalTaskID), "Manual retry must reuse native task identity")
		Expect(*retried.State).To(Equal(run_model.PipelineTaskTaskStateSUCCEEDED))
		workflow = getDriverRetryWorkflow(argo, runID)
		Expect(workflow.Annotations[util.AnnotationKeyRetryGeneration]).To(Equal("1"))
		Expect(e2eutils.TaskAttemptNodes(workflow, "exhaust-driver")).To(HaveLen(1))
	})
})

func startDriverRetryRun(tc *apitests.TestContext, argo argoclient.Interface, task string) (string, *e2eutils.WorkflowPodObserver) {
	directory, err := os.MkdirTemp("", "driver-retry-pipeline-")
	Expect(err).NotTo(HaveOccurred())
	DeferCleanup(func() { Expect(os.RemoveAll(directory)).To(Succeed()) })
	path := filepath.Join(directory, "pipeline.yaml")
	Expect(os.WriteFile(path, []byte(strings.ReplaceAll(driverRetryPipeline, "TARGET_TASK", task)), 0600)).To(Succeed())
	uploaded, err := testutil.UploadPipeline(pipelineUploadClient, path, &tc.Pipeline.PipelineGeneratedName, nil)
	Expect(err).NotTo(HaveOccurred())
	tc.Pipeline.CreatedPipelines = append(tc.Pipeline.CreatedPipelines, uploaded)
	version := testutil.GetLatestPipelineVersion(pipelineClient, &uploaded.PipelineID)
	run := e2eutils.CreatePipelineRun(runClient, tc, &uploaded.PipelineID, &version.PipelineVersionID, experimentID, nil)
	return run.RunID, e2eutils.NewWorkflowPodObserver(k8Client, argo, testutil.GetNamespace(), run.RunID)
}

func waitDriverRetryRun(runID string, state run_model.V2RuntimeState, checks ...func() error) {
	timeout := time.Duration(maxPipelineWaitTime)
	testutil.WaitForRunToBeInState(runClient, &runID, []run_model.V2RuntimeState{state}, &timeout, checks...)
}

func getDriverRetryWorkflow(argo argoclient.Interface, runID string) *wfapi.Workflow {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	workflow, err := testutil.GetWorkflowByRunID(ctx, argo, testutil.GetNamespace(), runID)
	Expect(err).NotTo(HaveOccurred())
	AddReportEntry("Driver retry workflow", workflow.Status.Nodes)
	return workflow
}

func uniqueDriverRetryTask(runID, name string) *run_model.V2PipelineTask {
	var matches []*run_model.V2PipelineTask
	for _, task := range testutil.GetPipelineRun(runClient, &runID).Tasks {
		if task.DisplayName == name {
			matches = append(matches, task)
		}
	}
	Expect(matches).To(HaveLen(1), "Expected one stable task identity for %s", name)
	Expect(matches[0].State).NotTo(BeNil())
	return matches[0]
}

func expectDriverRetryCleanupStatus(task *run_model.V2PipelineTask, state string) {
	Expect(task.Inputs).NotTo(BeNil())
	for _, parameter := range task.Inputs.Parameters {
		if parameter.ParameterKey != "status" {
			continue
		}
		status, ok := parameter.Value.(map[string]any)
		Expect(ok).To(BeTrue(), "TASK_FINAL_STATUS is a JSON object in the native task API")
		Expect(status["state"]).To(Equal(state))
		return
	}
	Fail("Cleanup task has no final-status input")
}

// The controller caches executor-plugin ConfigMaps asynchronously. Rolling it
// after the update creates an ordering barrier: its initial informer list sees
// the new ConfigMap before any workflow work is processed. The deployment name
// comes from manifests/kustomize/third-party/argo/base; this suite deliberately
// requires the dedicated KFP deployment used by the E2E lanes.
func restartDriverRetryController() {
	const restartAnnotation = "pipelines.kubeflow.org/driver-retry-test-restart"
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	deployments := k8Client.AppsV1().Deployments(testutil.GetNamespace())
	marker := time.Now().UTC().Format(time.RFC3339Nano)
	patch, err := json.Marshal(map[string]any{"spec": map[string]any{"template": map[string]any{
		"metadata": map[string]any{"annotations": map[string]string{restartAnnotation: marker}},
	}}})
	Expect(err).NotTo(HaveOccurred())
	deployment, err := deployments.Patch(ctx, "workflow-controller", types.MergePatchType, patch, metav1.PatchOptions{})
	Expect(err).NotTo(HaveOccurred(), "This dedicated suite requires the KFP workflow-controller deployment")
	replicas := int32(1)
	if deployment.Spec.Replicas != nil {
		replicas = *deployment.Spec.Replicas
	}
	Expect(replicas).To(BeNumerically(">", 0))
	selector, err := metav1.LabelSelectorAsSelector(deployment.Spec.Selector)
	Expect(err).NotTo(HaveOccurred())
	err = wait.PollUntilContextCancel(ctx, time.Second, true, func(ctx context.Context) (bool, error) {
		current, err := deployments.Get(ctx, deployment.Name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		status := current.Status
		if status.ObservedGeneration < deployment.Generation || status.UpdatedReplicas != replicas ||
			status.ReadyReplicas != replicas || status.AvailableReplicas != replicas || status.Replicas != replicas {
			return false, nil
		}
		pods, err := k8Client.CoreV1().Pods(testutil.GetNamespace()).List(ctx, metav1.ListOptions{LabelSelector: selector.String()})
		if err != nil {
			return false, err
		}
		if len(pods.Items) != int(replicas) {
			return false, nil
		}
		for _, pod := range pods.Items {
			// Wait for old leaders to disappear, including terminating pods that
			// Deployment replica counts may already exclude.
			if pod.DeletionTimestamp != nil || pod.Annotations[restartAnnotation] != marker {
				return false, nil
			}
		}
		return true, nil
	})
	Expect(err).NotTo(HaveOccurred(), "Wait for the restarted controller and its initial ConfigMap informer list")
}
