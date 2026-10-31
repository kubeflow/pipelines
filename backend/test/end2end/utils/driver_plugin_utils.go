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

package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	argoclient "github.com/argoproj/argo-workflows/v4/pkg/client/clientset/versioned"
	argocommon "github.com/argoproj/argo-workflows/v4/workflow/common"
	"github.com/kubeflow/pipelines/backend/api/v2/go_http_client/run_model"
	apiserver "github.com/kubeflow/pipelines/backend/src/common/client/api_server/v2"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/test/logger"
	"github.com/kubeflow/pipelines/backend/test/testutil"
	"github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	// DriverPluginName is the executor plugin key in compiled driver templates
	// and the sidecar container name in the Argo agent pod.
	DriverPluginName = "driver-plugin"
	// DriverLogsURIProperty is the task status_metadata custom property the
	// plugin registers after uploading its log (see
	// backend/src/driver/executorplugin/log_artifact.go).
	DriverLogsURIProperty = "driver_logs_uri"
	// DAGDriverTemplate is the plugin template every compiled workflow uses
	// for the root DAG driver.
	DAGDriverTemplate = "system-dag-driver"
	// driverImageMarker identifies the KFP driver image independent of registry and tag.
	driverImageMarker = "kfp-driver"
	// argoAgentComponent is the LabelKeyComponent value Argo puts on the agent pod.
	argoAgentComponent = "agent"
	// publishLogsArg is the driver plugin argument that enables log publication.
	publishLogsArg = "publish_logs"
	// driverTaskSuffix is appended to a task name for its driver DAG task.
	driverTaskSuffix = "-driver"
)

// DriverPluginArgs returns the "driver-plugin" arguments of a template, or
// false when the template is not implemented by the driver plugin.
func DriverPluginArgs(template v1alpha1.Template) (map[string]interface{}, bool) {
	if template.Plugin == nil || len(template.Plugin.Value) == 0 {
		return nil, false
	}
	var plugin map[string]struct {
		Args map[string]interface{} `json:"args"`
	}
	if err := json.Unmarshal(template.Plugin.Value, &plugin); err != nil {
		return nil, false
	}
	config, ok := plugin[DriverPluginName]
	if !ok {
		return nil, false
	}
	return config.Args, true
}

// DriverPluginTemplates returns the names of the workflow templates the
// driver executor plugin implements.
func DriverPluginTemplates(workflow *v1alpha1.Workflow) []string {
	var names []string
	for _, template := range workflow.Spec.Templates {
		if _, ok := DriverPluginArgs(template); ok {
			names = append(names, template.Name)
		}
	}
	sort.Strings(names)
	return names
}

// DriverPluginPublishesLogs reports whether the compiled driver templates
// ask the plugin to publish driver logs.
func DriverPluginPublishesLogs(workflow *v1alpha1.Workflow) bool {
	for _, template := range workflow.Spec.Templates {
		args, ok := DriverPluginArgs(template)
		if !ok {
			continue
		}
		if value, ok := args[publishLogsArg].(string); ok && strings.EqualFold(value, "true") {
			return true
		}
	}
	return false
}

// DriverNodes returns the workflow nodes that ran a driver plugin template.
func DriverNodes(workflow *v1alpha1.Workflow) []v1alpha1.NodeStatus {
	driverTemplates := make(map[string]bool)
	for _, name := range DriverPluginTemplates(workflow) {
		driverTemplates[name] = true
	}
	var nodes []v1alpha1.NodeStatus
	for _, node := range workflow.Status.Nodes {
		if driverTemplates[node.TemplateName] {
			nodes = append(nodes, node)
		}
	}
	sort.Slice(nodes, func(i, j int) bool { return nodes[i].Name < nodes[j].Name })
	return nodes
}

// containerTemplateViolations lists the ways a compiled template still runs
// the driver as a container instead of through the executor plugin.
func containerTemplateViolations(template v1alpha1.Template) []string {
	var violations []string
	if role := template.Metadata.Annotations[util.AnnotationKeyRuntimeRole]; role == string(util.ExecutionRuntimeRoleDriver) {
		violations = append(violations, fmt.Sprintf("template %s declares runtime role %q", template.Name, role))
	}
	var containers []v1.Container
	if template.Container != nil {
		containers = append(containers, *template.Container)
	}
	for _, initContainer := range template.InitContainers {
		containers = append(containers, initContainer.Container)
	}
	for _, sidecar := range template.Sidecars {
		containers = append(containers, sidecar.Container)
	}
	for _, container := range containers {
		if strings.Contains(container.Image, driverImageMarker) {
			violations = append(violations, fmt.Sprintf("template %s runs driver image %s in container %q", template.Name, container.Image, container.Name))
		}
		if container.Name == DriverPluginName {
			violations = append(violations, fmt.Sprintf("template %s defines a %q container", template.Name, DriverPluginName))
		}
	}
	return violations
}

// ValidateDriverPluginWorkflow asserts the run's Argo Workflow drives every
// task through the executor plugin: driver templates are "driver-plugin"
// plugin templates, no template runs the driver as a container, and every
// driver node executed as a Plugin node that did not fail. It returns the
// driver template names.
func ValidateDriverPluginWorkflow(workflow *v1alpha1.Workflow) []string {
	logger.Log("Validating driver plugin templates and nodes of workflow %s", workflow.Name)
	driverTemplates := DriverPluginTemplates(workflow)
	gomega.Expect(driverTemplates).NotTo(gomega.BeEmpty(), "Workflow %s has no %q plugin templates", workflow.Name, DriverPluginName)
	gomega.Expect(driverTemplates).To(gomega.ContainElement(DAGDriverTemplate), "Workflow %s does not drive its root DAG through the plugin", workflow.Name)
	for _, template := range workflow.Spec.Templates {
		gomega.Expect(containerTemplateViolations(template)).To(gomega.BeEmpty(), "Workflow %s still runs the driver as a container", workflow.Name)
	}

	driverNodes := DriverNodes(workflow)
	gomega.Expect(driverNodes).NotTo(gomega.BeEmpty(), "Workflow %s has no nodes for driver templates %v (status has %d nodes)", workflow.Name, driverTemplates, len(workflow.Status.Nodes))
	succeeded := 0
	for _, node := range driverNodes {
		gomega.Expect(node.Type).To(gomega.Equal(v1alpha1.NodeTypePlugin), "Driver node %s (template %s) ran as %s instead of a plugin node", node.Name, node.TemplateName, node.Type)
		gomega.Expect(node.FailedOrError()).To(gomega.BeFalse(), "Driver node %s failed: %s", node.Name, node.Message)
		if node.Phase == v1alpha1.NodeSucceeded {
			succeeded++
		}
	}
	gomega.Expect(succeeded).To(gomega.BeNumerically(">", 0), "No driver plugin node of workflow %s succeeded", workflow.Name)
	logger.Log("Workflow %s drove %d node(s) through plugin templates %v (%d succeeded)", workflow.Name, len(driverNodes), driverTemplates, succeeded)
	return driverTemplates
}

// WorkflowPodObserver records every pod of a run while it executes. Argo
// deletes the agent pod as soon as the workflow finishes, so the driver plugin
// sidecar can only be observed while the run is live. Check is meant to be
// passed to testutil.WaitForRunToBeInState; it only observes and never fails
// the wait.
type WorkflowPodObserver struct {
	k8sClient    kubernetes.Interface
	argoClient   argoclient.Interface
	namespace    string
	runID        string
	mu           sync.Mutex
	workflowName string
	pods         map[string]v1.Pod
}

// NewWorkflowPodObserver returns an observer for the pods of a run.
func NewWorkflowPodObserver(k8sClient kubernetes.Interface, argoClient argoclient.Interface, namespace, runID string) *WorkflowPodObserver {
	return &WorkflowPodObserver{
		k8sClient:  k8sClient,
		argoClient: argoClient,
		namespace:  namespace,
		runID:      runID,
		pods:       make(map[string]v1.Pod),
	}
}

// Check snapshots the pods that currently belong to the run.
func (o *WorkflowPodObserver) Check() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.workflowName == "" && o.argoClient != nil {
		workflows, err := o.argoClient.ArgoprojV1alpha1().Workflows(o.namespace).List(ctx, metav1.ListOptions{LabelSelector: testutil.RunWorkflowSelector(o.runID)})
		if err != nil {
			logger.Log("Pod observer for run %s: cannot list workflows: %v", o.runID, err)
		} else if len(workflows.Items) > 0 {
			o.workflowName = workflows.Items[0].Name
		}
	}
	selectors := []string{testutil.RunWorkflowSelector(o.runID)}
	if o.workflowName != "" {
		// The agent pod carries Argo's workflow label but not the run's podMetadata labels.
		selectors = append(selectors, argocommon.LabelKeyWorkflow+"="+o.workflowName)
	}
	for _, selector := range selectors {
		pods, err := o.k8sClient.CoreV1().Pods(o.namespace).List(ctx, metav1.ListOptions{LabelSelector: selector})
		if err != nil {
			logger.Log("Pod observer for run %s: cannot list pods with selector %q: %v", o.runID, selector, err)
			continue
		}
		for _, pod := range pods.Items {
			o.pods[pod.Name] = pod
		}
	}
	return nil
}

// WorkflowName returns the observed Argo Workflow name, or "" if none was seen.
func (o *WorkflowPodObserver) WorkflowName() string {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.workflowName
}

// Pods returns the latest snapshot of every pod observed, sorted by name.
func (o *WorkflowPodObserver) Pods() []v1.Pod {
	o.mu.Lock()
	defer o.mu.Unlock()
	pods := make([]v1.Pod, 0, len(o.pods))
	for _, pod := range o.pods {
		pods = append(pods, pod)
	}
	sort.Slice(pods, func(i, j int) bool { return pods[i].Name < pods[j].Name })
	return pods
}

// DriverPluginPodReport summarizes how the pods of a run were classified.
type DriverPluginPodReport struct {
	AgentPods    []string
	LauncherPods []string
	DriverImages []string
}

// IsArgoAgentPod reports whether a pod is Argo's per-workflow agent pod.
func IsArgoAgentPod(pod v1.Pod) bool {
	return pod.Labels[argocommon.LabelKeyComponent] == argoAgentComponent
}

func podContainers(pod v1.Pod) []v1.Container {
	containers := append([]v1.Container(nil), pod.Spec.InitContainers...)
	return append(containers, pod.Spec.Containers...)
}

// taskPodViolations lists the ways a non-agent pod still runs the driver, or
// is not a launcher pod at all.
func taskPodViolations(pod v1.Pod, driverImages map[string]bool) []string {
	var violations []string
	role := pod.Annotations[util.AnnotationKeyRuntimeRole]
	if role != string(util.ExecutionRuntimeRoleLauncher) {
		violations = append(violations, fmt.Sprintf("pod %s has runtime role %q, expected %q", pod.Name, role, util.ExecutionRuntimeRoleLauncher))
	}
	for _, container := range podContainers(pod) {
		if container.Name == DriverPluginName {
			violations = append(violations, fmt.Sprintf("pod %s runs a %q container outside the agent pod", pod.Name, DriverPluginName))
		}
		if driverImages[container.Image] || strings.Contains(container.Image, driverImageMarker) {
			violations = append(violations, fmt.Sprintf("pod %s runs the driver image %s in container %q", pod.Name, container.Image, container.Name))
		}
	}
	return violations
}

// classifyPods splits observed pods into agent pods, launcher pods and the
// images the driver plugin sidecars run; it returns a violation for every pod
// that does not fit the plugin model.
func classifyPods(pods []v1.Pod) (DriverPluginPodReport, []string) {
	var report DriverPluginPodReport
	var violations []string
	driverImages := make(map[string]bool)
	for _, pod := range pods {
		if !IsArgoAgentPod(pod) {
			continue
		}
		report.AgentPods = append(report.AgentPods, pod.Name)
		found := false
		for _, container := range pod.Spec.Containers {
			if container.Name != DriverPluginName {
				continue
			}
			found = true
			if !driverImages[container.Image] {
				driverImages[container.Image] = true
				report.DriverImages = append(report.DriverImages, container.Image)
			}
		}
		if !found {
			violations = append(violations, fmt.Sprintf("agent pod %s has no %q sidecar container", pod.Name, DriverPluginName))
		}
	}
	for _, pod := range pods {
		if IsArgoAgentPod(pod) {
			continue
		}
		podViolations := taskPodViolations(pod, driverImages)
		if len(podViolations) == 0 {
			report.LauncherPods = append(report.LauncherPods, pod.Name)
		}
		violations = append(violations, podViolations...)
	}
	sort.Strings(report.DriverImages)
	return report, violations
}

// ValidateDriverPluginPods asserts the pods observed for a run follow the
// plugin model: the driver runs only as the "driver-plugin" sidecar of the
// Argo agent pod, and every other pod is a launcher pod that neither runs the
// driver image nor carries the driver runtime role.
func ValidateDriverPluginPods(pods []v1.Pod) DriverPluginPodReport {
	logger.Log("Validating %d observed pod(s) against the driver plugin model", len(pods))
	report, violations := classifyPods(pods)
	gomega.Expect(violations).To(gomega.BeEmpty(), "Observed pods do not follow the driver plugin model")
	gomega.Expect(report.AgentPods).NotTo(gomega.BeEmpty(), "No Argo agent pod (label %s=%s) was observed; the driver plugin sidecar never ran", argocommon.LabelKeyComponent, argoAgentComponent)
	gomega.Expect(report.LauncherPods).NotTo(gomega.BeEmpty(), "No launcher pod was observed for the run")
	logger.Log("Agent pod(s) %v ran the driver plugin with image(s) %v; launcher pod(s): %v", report.AgentPods, report.DriverImages, report.LauncherPods)
	return report
}

// DriverLogsURI returns the driver_logs_uri custom property of a task, or "".
func DriverLogsURI(task *run_model.V2PipelineTask) string {
	if task == nil || task.StatusMetadata == nil {
		return ""
	}
	uri, _ := task.StatusMetadata.CustomProperties[DriverLogsURIProperty].(string)
	return uri
}

// HasDriverPod reports whether the driver plugin registered a DRIVER pod for the task.
func HasDriverPod(task *run_model.V2PipelineTask) bool {
	if task == nil {
		return false
	}
	for _, pod := range task.Pods {
		if pod != nil && pod.Type != nil && *pod.Type == run_model.PipelineTaskTaskPodTypeDRIVER {
			return true
		}
	}
	return false
}

// ValidateDriverLogsPublished asserts every task the plugin drove carries the
// driver_logs_uri custom property. Log publication and task persistence lag
// the run's terminal state, so the check polls.
func ValidateDriverLogsPublished(runClient *apiserver.RunClient, runID string) {
	logger.Log("Validating that driver logs were published for the tasks of run %s", runID)
	gomega.Eventually(func(g gomega.Gomega) {
		run := testutil.GetPipelineRun(runClient, &runID)
		driven := 0
		for _, task := range run.Tasks {
			if !HasDriverPod(task) {
				continue
			}
			driven++
			g.Expect(DriverLogsURI(task)).NotTo(gomega.BeEmpty(), "Task %s (%s) of run %s has no %s custom property", task.DisplayName, task.TaskID, runID, DriverLogsURIProperty)
		}
		g.Expect(driven).To(gomega.BeNumerically(">", 0), "Run %s has no task with a DRIVER pod", runID)
	}, 2*time.Minute, 5*time.Second).Should(gomega.Succeed())
}

// podDescendants returns the Pod nodes below a node, following child IDs.
func podDescendants(workflow *v1alpha1.Workflow, node v1alpha1.NodeStatus, visited map[string]bool) []v1alpha1.NodeStatus {
	var pods []v1alpha1.NodeStatus
	for _, childID := range node.Children {
		if visited[childID] {
			continue
		}
		visited[childID] = true
		child, ok := workflow.Status.Nodes[childID]
		if !ok {
			continue
		}
		if child.Type == v1alpha1.NodeTypePod {
			pods = append(pods, child)
		}
		pods = append(pods, podDescendants(workflow, child, visited)...)
	}
	return pods
}

// TaskAttemptNodes returns the Pod nodes executed for a KFP task, one per
// Argo retry attempt, ordered by start time.
func TaskAttemptNodes(workflow *v1alpha1.Workflow, taskName string) []v1alpha1.NodeStatus {
	visited := make(map[string]bool)
	var attempts []v1alpha1.NodeStatus
	for _, node := range workflow.Status.Nodes {
		if node.DisplayName != taskName {
			continue
		}
		attempts = append(attempts, podDescendants(workflow, node, visited)...)
	}
	sort.SliceStable(attempts, func(i, j int) bool {
		if !attempts[i].StartedAt.Equal(&attempts[j].StartedAt) {
			return attempts[i].StartedAt.Before(&attempts[j].StartedAt)
		}
		return attempts[i].Name < attempts[j].Name
	})
	return attempts
}

// ValidateTaskRetried asserts a task's executor ran at least minAttempts
// times, failed before its final attempt and succeeded on the last one, while
// its driver ran exactly once through the plugin.
func ValidateTaskRetried(workflow *v1alpha1.Workflow, taskName string, minAttempts int) []v1alpha1.NodeStatus {
	logger.Log("Validating retry attempts of task %s in workflow %s", taskName, workflow.Name)
	attempts := TaskAttemptNodes(workflow, taskName)
	gomega.Expect(len(attempts)).To(gomega.BeNumerically(">=", minAttempts), "Task %s ran %d attempt(s), expected at least %d", taskName, len(attempts), minAttempts)
	last := attempts[len(attempts)-1]
	gomega.Expect(last.Phase).To(gomega.Equal(v1alpha1.NodeSucceeded), "Final attempt %s of task %s is %s: %s", last.Name, taskName, last.Phase, last.Message)
	gomega.Expect(attempts[0].FailedOrError()).To(gomega.BeTrue(), "First attempt %s of task %s should have failed but is %s", attempts[0].Name, taskName, attempts[0].Phase)

	driverRuns := 0
	for _, node := range workflow.Status.Nodes {
		if node.DisplayName == taskName+driverTaskSuffix && node.Type == v1alpha1.NodeTypePlugin {
			driverRuns++
		}
	}
	gomega.Expect(driverRuns).To(gomega.Equal(1), "Driver of task %s ran %d time(s); retries must only re-run the executor", taskName, driverRuns)
	for _, attempt := range attempts {
		logger.Log("Task %s attempt %s: phase=%s", taskName, attempt.Name, attempt.Phase)
	}
	return attempts
}
