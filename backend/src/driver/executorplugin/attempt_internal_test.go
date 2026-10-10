// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package executorplugin

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	wf "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	argofake "github.com/argoproj/argo-workflows/v4/pkg/client/clientset/versioned/fake"
	"github.com/kubeflow/pipelines/backend/src/driver/driverapi"
	"github.com/kubeflow/pipelines/backend/src/v2/driver"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	ktesting "k8s.io/client-go/testing"
)

func TestTasksetAttemptResolution(t *testing.T) {
	fields := validContainerDriverArgs()
	fields["namespace"] = "run-namespace"
	fields["run_id"] = "run-id"
	fields["run_name"] = "workflow"
	body := driverRequestBody(t, fields)
	args, err := parseDriverRequestArgs(authenticatedRequest("POST", body))
	require.NoError(t, err)
	var request struct {
		Template wf.Template `json:"template"`
	}
	require.NoError(t, json.Unmarshal([]byte(body), &request))
	for _, mode := range []string{"active", "retry", "duplicate", "completed", "removed", "owner", "namespace", "read failure", "different args", "canonical"} {
		t.Run(mode, func(t *testing.T) {
			pod := agentPodForAuth()
			owner := pod.OwnerReferences[0]
			owner.Controller = nil
			taskset := &wf.WorkflowTaskSet{ObjectMeta: metav1.ObjectMeta{Name: "workflow", Namespace: pod.Namespace, OwnerReferences: []metav1.OwnerReference{owner}}, Spec: wf.WorkflowTaskSetSpec{Tasks: map[string]wf.Template{"node-0": request.Template}}, Status: wf.WorkflowTaskSetStatus{Nodes: map[string]wf.NodeResult{}}}
			switch mode {
			case "retry":
				taskset.Status.Nodes["node-0"] = wf.NodeResult{Phase: wf.NodeFailed}
				taskset.Spec.Tasks["node-1"] = request.Template
			case "duplicate":
				taskset.Spec.Tasks["node-1"] = request.Template
			case "completed":
				taskset.Status.Nodes["node-0"] = wf.NodeResult{Phase: wf.NodeSucceeded}
			case "removed":
				delete(taskset.Spec.Tasks, "node-0")
			case "owner":
				taskset.OwnerReferences[0].UID = "other"
			case "namespace":
				taskset.Namespace = "other"
			case "different args":
				changed := *args
				changed.TaskName = "other"
				args = &changed
				defer func() { args, _ = parseDriverRequestArgs(authenticatedRequest("POST", body)) }()
			case "canonical":
				// Omitted optional fields vs explicit empty fields compare as the validated DTO.
				plugin := map[string]any{"driver-plugin": map[string]any{"args": *args}}
				raw, _ := json.Marshal(plugin)
				// DTO omitempty removes required empty fields; keep them from the wire shape.
				var decoded map[string]any
				require.NoError(t, json.Unmarshal(raw, &decoded))
				candidate := decoded["driver-plugin"].(map[string]any)["args"].(map[string]any)
				for key, value := range fields {
					if _, ok := candidate[key]; !ok {
						candidate[key] = value
					}
				}
				raw, _ = json.MarshalIndent(decoded, "", "  ")
				tmpl := request.Template.DeepCopy()
				tmpl.Plugin.Value = raw
				taskset.Spec.Tasks["node-0"] = *tmpl
			}
			argo := argofake.NewSimpleClientset(taskset)
			if mode == "read failure" {
				argo.PrependReactor("get", "workflowtasksets", func(ktesting.Action) (bool, runtime.Object, error) { return true, nil, context.DeadlineExceeded })
			}
			key, err := tasksetAttemptResolver(fake.NewSimpleClientset(pod), argo, pod.Namespace, pod.Name)(context.Background(), *args)
			if mode == "active" || mode == "canonical" {
				require.NoError(t, err)
				require.Equal(t, "workflow-uid/node-0", key)
			} else if mode == "retry" {
				require.NoError(t, err)
				require.Equal(t, "workflow-uid/node-1", key)
			} else {
				require.Error(t, err)
				require.Empty(t, key)
			}
		})
	}
}

func TestAttemptLookupFailureNeverStartsDriver(t *testing.T) {
	h := asyncHandler(t, context.Background(), func(context.Context, driverapi.DriverPluginArgs) (*driver.Execution, error) {
		t.Error("must not execute")
		return nil, nil
	}, func(context.Context, driverapi.DriverPluginArgs) (string, error) { return "", context.DeadlineExceeded })
	recorder := httptest.NewRecorder()
	h.ServeHTTP(recorder, authenticatedRequest("POST", requestBody(t, "CONTAINER")))
	require.Equal(t, 503, recorder.Code)
}
