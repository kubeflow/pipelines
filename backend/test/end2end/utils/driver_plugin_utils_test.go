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

package utils

import (
	"encoding/json"
	"slices"
	"testing"
	"time"

	"github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	argocommon "github.com/argoproj/argo-workflows/v4/workflow/common"
	"github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/run_model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func pluginTemplate(name string, args map[string]interface{}) v1alpha1.Template {
	value, err := json.Marshal(map[string]interface{}{DriverPluginName: map[string]interface{}{"args": args}})
	if err != nil {
		panic(err)
	}
	return v1alpha1.Template{Name: name, Plugin: &v1alpha1.Plugin{Object: v1alpha1.Object{Value: value}}}
}

func TestDriverPluginTemplates(t *testing.T) {
	workflow := &v1alpha1.Workflow{Spec: v1alpha1.WorkflowSpec{Templates: []v1alpha1.Template{
		{Name: "system-container-impl", Container: &v1.Container{Image: "ghcr.io/kubeflow/kfp-launcher:latest"}},
		pluginTemplate(DAGDriverTemplate, map[string]interface{}{"publish_logs": "true"}),
		pluginTemplate("system-container-driver", map[string]interface{}{"publish_logs": "false"}),
		{Name: "other-plugin", Plugin: &v1alpha1.Plugin{Object: v1alpha1.Object{Value: []byte(`{"http":{"url":"x"}}`)}}},
	}}}
	got := DriverPluginTemplates(workflow)
	want := []string{"system-container-driver", DAGDriverTemplate}
	if !slices.Equal(got, want) {
		t.Fatalf("DriverPluginTemplates() = %v, want %v", got, want)
	}
	if !DriverPluginPublishesLogs(workflow) {
		t.Fatal("DriverPluginPublishesLogs() = false, want true")
	}
	workflow.Spec.Templates[1] = pluginTemplate(DAGDriverTemplate, map[string]interface{}{"publish_logs": "false"})
	if DriverPluginPublishesLogs(workflow) {
		t.Fatal("DriverPluginPublishesLogs() = true, want false")
	}
}

func TestContainerTemplateViolations(t *testing.T) {
	testCases := []struct {
		name     string
		template v1alpha1.Template
		want     int
	}{
		{
			name:     "launcher template",
			template: v1alpha1.Template{Name: "impl", Container: &v1.Container{Name: "main", Image: "python:3.11"}, InitContainers: []v1alpha1.UserContainer{{Container: v1.Container{Name: "kfp-launcher", Image: "ghcr.io/kubeflow/kfp-launcher:latest"}}}},
			want:     0,
		},
		{
			name: "legacy driver container template",
			template: v1alpha1.Template{
				Name:      "system-dag-driver",
				Container: &v1.Container{Name: "main", Image: "ghcr.io/kubeflow/kfp-driver:latest"},
				Metadata:  v1alpha1.Metadata{Annotations: map[string]string{util.AnnotationKeyRuntimeRole: string(util.ExecutionRuntimeRoleDriver)}},
			},
			want: 2,
		},
		{
			name:     "driver plugin as sidecar",
			template: v1alpha1.Template{Name: "impl", Container: &v1.Container{Name: "main", Image: "python:3.11"}, Sidecars: []v1alpha1.UserContainer{{Container: v1.Container{Name: DriverPluginName, Image: "example.com/driver:1"}}}},
			want:     1,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			got := containerTemplateViolations(testCase.template)
			if len(got) != testCase.want {
				t.Fatalf("containerTemplateViolations() = %v, want %d violation(s)", got, testCase.want)
			}
		})
	}
}

func agentPod(name, image string) v1.Pod {
	return v1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: name, Labels: map[string]string{argocommon.LabelKeyComponent: "agent"}},
		Spec: v1.PodSpec{Containers: []v1.Container{
			{Name: "main", Image: "quay.io/argoproj/argoexec:latest"},
			{Name: DriverPluginName, Image: image},
		}},
	}
}

func launcherPod(name string) v1.Pod {
	return v1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: name, Annotations: map[string]string{util.AnnotationKeyRuntimeRole: string(util.ExecutionRuntimeRoleLauncher)}},
		Spec: v1.PodSpec{
			InitContainers: []v1.Container{{Name: "kfp-launcher", Image: "ghcr.io/kubeflow/kfp-launcher:latest"}},
			Containers:     []v1.Container{{Name: "main", Image: "python:3.11"}},
		},
	}
}

func TestClassifyPods(t *testing.T) {
	driverImage := "example.com/custom-driver:dev"
	t.Run("plugin model", func(t *testing.T) {
		report, violations := classifyPods([]v1.Pod{agentPod("wf-agent", driverImage), launcherPod("wf-task-1"), launcherPod("wf-task-2")})
		if len(violations) != 0 {
			t.Fatalf("unexpected violations: %v", violations)
		}
		if !slices.Equal(report.AgentPods, []string{"wf-agent"}) || !slices.Equal(report.LauncherPods, []string{"wf-task-1", "wf-task-2"}) || !slices.Equal(report.DriverImages, []string{driverImage}) {
			t.Fatalf("unexpected report: %+v", report)
		}
	})
	t.Run("driver pod per task", func(t *testing.T) {
		driverPod := launcherPod("wf-task-driver")
		driverPod.Annotations[util.AnnotationKeyRuntimeRole] = string(util.ExecutionRuntimeRoleDriver)
		driverPod.Spec.InitContainers = nil
		driverPod.Spec.Containers = []v1.Container{{Name: "main", Image: driverImage}}
		_, violations := classifyPods([]v1.Pod{agentPod("wf-agent", driverImage), driverPod})
		if len(violations) != 2 {
			t.Fatalf("classifyPods() violations = %v, want 2", violations)
		}
	})
	t.Run("driver image by marker", func(t *testing.T) {
		pod := launcherPod("wf-task")
		pod.Spec.Containers[0].Image = "ghcr.io/kubeflow/kfp-driver:latest"
		_, violations := classifyPods([]v1.Pod{agentPod("wf-agent", driverImage), pod})
		if len(violations) != 1 {
			t.Fatalf("classifyPods() violations = %v, want 1", violations)
		}
	})
	t.Run("agent without sidecar", func(t *testing.T) {
		pod := agentPod("wf-agent", driverImage)
		pod.Spec.Containers = pod.Spec.Containers[:1]
		_, violations := classifyPods([]v1.Pod{pod, launcherPod("wf-task")})
		if len(violations) != 1 {
			t.Fatalf("classifyPods() violations = %v, want 1", violations)
		}
	})
}

func TestTaskAttemptNodes(t *testing.T) {
	start := metav1.NewTime(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	later := metav1.NewTime(start.Add(time.Minute))
	workflow := &v1alpha1.Workflow{Status: v1alpha1.WorkflowStatus{Nodes: v1alpha1.Nodes{
		"wf":                {ID: "wf", Name: "wf", DisplayName: "wf", Type: v1alpha1.NodeTypeDAG, Children: []string{"root-driver", "root"}},
		"root-driver":       {ID: "root-driver", Name: "wf.root-driver", DisplayName: "root-driver", Type: v1alpha1.NodeTypePlugin, TemplateName: DAGDriverTemplate, Phase: v1alpha1.NodeSucceeded},
		"root":              {ID: "root", Name: "wf.root", DisplayName: "root", Type: v1alpha1.NodeTypeDAG, Children: []string{"task-driver", "task"}},
		"task-driver":       {ID: "task-driver", Name: "wf.root.flaky-add-driver", DisplayName: "flaky-add-driver", Type: v1alpha1.NodeTypePlugin, TemplateName: "system-container-driver", Phase: v1alpha1.NodeSucceeded},
		"task":              {ID: "task", Name: "wf.root.flaky-add", DisplayName: "flaky-add", Type: v1alpha1.NodeTypeRetry, Children: []string{"task-0"}},
		"task-0":            {ID: "task-0", Name: "wf.root.flaky-add(0)", DisplayName: "flaky-add(0)", Type: v1alpha1.NodeTypeDAG, Children: []string{"executor"}},
		"executor":          {ID: "executor", Name: "wf.root.flaky-add(0).executor", DisplayName: "executor", Type: v1alpha1.NodeTypeRetry, Children: []string{"executor-1", "executor-0"}},
		"executor-0":        {ID: "executor-0", Name: "wf.root.flaky-add(0).executor(0)", DisplayName: "executor(0)", Type: v1alpha1.NodeTypePod, Phase: v1alpha1.NodeFailed, StartedAt: start},
		"executor-1":        {ID: "executor-1", Name: "wf.root.flaky-add(0).executor(1)", DisplayName: "executor(1)", Type: v1alpha1.NodeTypePod, Phase: v1alpha1.NodeSucceeded, StartedAt: later},
		"consumer":          {ID: "consumer", Name: "wf.root.check-sum", DisplayName: "check-sum", Type: v1alpha1.NodeTypeRetry, Children: []string{"consumer-0"}},
		"consumer-0":        {ID: "consumer-0", Name: "wf.root.check-sum(0)", DisplayName: "check-sum(0)", Type: v1alpha1.NodeTypeDAG, Children: []string{"consumer-executor"}},
		"consumer-executor": {ID: "consumer-executor", Name: "wf.root.check-sum(0).executor", DisplayName: "executor", Type: v1alpha1.NodeTypePod, Phase: v1alpha1.NodeSucceeded, StartedAt: later},
	}}}

	attempts := TaskAttemptNodes(workflow, "flaky-add")
	var names []string
	for _, attempt := range attempts {
		names = append(names, attempt.ID)
	}
	if !slices.Equal(names, []string{"executor-0", "executor-1"}) {
		t.Fatalf("TaskAttemptNodes(flaky-add) = %v, want [executor-0 executor-1]", names)
	}
	if attempts := TaskAttemptNodes(workflow, "check-sum"); len(attempts) != 1 {
		t.Fatalf("TaskAttemptNodes(check-sum) returned %d attempt(s), want 1", len(attempts))
	}
	if attempts := TaskAttemptNodes(workflow, "missing"); len(attempts) != 0 {
		t.Fatalf("TaskAttemptNodes(missing) returned %d attempt(s), want 0", len(attempts))
	}

	workflow.Spec.Templates = []v1alpha1.Template{pluginTemplate(DAGDriverTemplate, nil), pluginTemplate("system-container-driver", nil)}
	driverNodes := DriverNodes(workflow)
	if len(driverNodes) != 2 {
		t.Fatalf("DriverNodes() returned %d node(s), want 2", len(driverNodes))
	}
}

func TestDriverLogsURI(t *testing.T) {
	driverType := run_model.PipelineTaskTaskPodTypeDRIVER
	executorType := run_model.PipelineTaskTaskPodTypeEXECUTOR
	task := &run_model.V2beta1PipelineTask{
		Pods:           []*run_model.PipelineTaskTaskPod{{Name: "wf-agent", Type: &driverType}, {Name: "wf-task", Type: &executorType}},
		StatusMetadata: &run_model.PipelineTaskStatusMetadata{CustomProperties: map[string]any{DriverLogsURIProperty: "minio://mlpipeline/v2/artifacts/p/r/flaky-add/driver-logs"}},
	}
	if !HasDriverPod(task) {
		t.Fatal("HasDriverPod() = false, want true")
	}
	if got := DriverLogsURI(task); got == "" {
		t.Fatal("DriverLogsURI() is empty")
	}
	if got := DriverLogsURI(&run_model.V2beta1PipelineTask{}); got != "" {
		t.Fatalf("DriverLogsURI() of task without metadata = %q, want empty", got)
	}
	if HasDriverPod(&run_model.V2beta1PipelineTask{Pods: []*run_model.PipelineTaskTaskPod{{Name: "wf-task", Type: &executorType}}}) {
		t.Fatal("HasDriverPod() = true for executor-only task")
	}
}
