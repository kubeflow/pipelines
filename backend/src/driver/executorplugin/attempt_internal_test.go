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
	"errors"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	wf "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	argofake "github.com/argoproj/argo-workflows/v4/pkg/client/clientset/versioned/fake"
	"github.com/kubeflow/pipelines/backend/src/driver/driverapi"
	"github.com/kubeflow/pipelines/backend/src/v2/driver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	ktesting "k8s.io/client-go/testing"
)

// fakeAgent loads the test agent Pod through the real bootstrap validation.
func fakeAgent(t *testing.T, argo *argofake.Clientset) (*agentClients, agentLoader) {
	t.Helper()
	pod := agentPodForAuth()
	agent, err := loadAgent(context.Background(), fake.NewSimpleClientset(pod), argo, pod.Namespace, pod.Name)
	require.NoError(t, err)
	return agent, func(context.Context) (*agentClients, error) { return agent, nil }
}

func tasksetFromClientset(ctx context.Context, agent *agentClients) (*wf.WorkflowTaskSet, error) {
	return agent.argo.ArgoprojV1alpha1().WorkflowTaskSets(agent.pod.Namespace).Get(ctx, agent.workflow.Name, metav1.GetOptions{})
}

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
				plugin := map[string]any{driverapi.PluginName: map[string]any{"args": *args}}
				raw, _ := json.Marshal(plugin)
				// DTO omitempty removes required empty fields; keep them from the wire shape.
				var decoded map[string]any
				require.NoError(t, json.Unmarshal(raw, &decoded))
				candidate := decoded[driverapi.PluginName].(map[string]any)["args"].(map[string]any)
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
			argo := argofake.NewSimpleClientset(taskset) //nolint:staticcheck // SA1019: Argo v4.1.2 does not generate the replacement NewClientset.
			source := tasksetFromClientset
			if mode == "read failure" {
				source = func(context.Context, *agentClients) (*wf.WorkflowTaskSet, error) {
					return nil, context.DeadlineExceeded
				}
			}
			if mode == "namespace" {
				// Namespace-scoped lookups cannot return it; a lister bug would.
				source = func(context.Context, *agentClients) (*wf.WorkflowTaskSet, error) { return taskset, nil }
			}
			_, agent := fakeAgent(t, argo)
			key, err := tasksetAttemptResolver(agent, source)(context.Background(), *args)
			switch mode {
			case "active", "canonical":
				require.NoError(t, err)
				require.Equal(t, "workflow-uid/node-0", key)
			case "retry":
				require.NoError(t, err)
				require.Equal(t, "workflow-uid/node-1", key)
			default:
				require.Error(t, err)
				require.Empty(t, key)
			}
		})
	}
}

func TestTasksetAttemptResolutionRejectsForeignRequests(t *testing.T) {
	_, agent := fakeAgent(t, argofake.NewSimpleClientset()) //nolint:staticcheck // SA1019: Argo v4.1.2 does not generate the replacement NewClientset.
	args := driverArgsForAuth()
	args.RunName = "other-workflow"
	resolve := tasksetAttemptResolver(agent, func(context.Context, *agentClients) (*wf.WorkflowTaskSet, error) {
		t.Fatal("foreign requests must not read the taskset")
		return nil, nil
	})
	_, err := resolve(context.Background(), args)
	require.ErrorContains(t, err, "does not own the agent Pod")
}

func TestCachedAgentRetriesBootstrapUntilSuccess(t *testing.T) {
	var loads atomic.Int32
	agent, _ := fakeAgent(t, nil)
	load := cachedAgent(func(context.Context) (*agentClients, error) {
		if loads.Add(1) == 1 {
			return nil, errors.New("not ready")
		}
		return agent, nil
	})
	_, err := load(context.Background())
	require.ErrorContains(t, err, "not ready")
	for range 3 {
		loaded, err := load(context.Background())
		require.NoError(t, err)
		assert.Same(t, agent, loaded)
	}
	assert.EqualValues(t, 2, loads.Load(), "successful bootstrap is reused")
}

func TestLoadAgentRequiresWorkflowOwner(t *testing.T) {
	for _, tc := range []struct {
		name, message string
		mutate        func(*agentClients)
	}{
		{"missing owner", "not controlled by an Argo Workflow", func(a *agentClients) { a.pod.OwnerReferences = nil }},
		{"wrong owner kind", "not controlled by an Argo Workflow", func(a *agentClients) { a.pod.OwnerReferences[0].Kind = "Deployment" }},
		{"missing owner UID", "not controlled by an Argo Workflow", func(a *agentClients) { a.pod.OwnerReferences[0].UID = "" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pod := agentPodForAuth()
			tc.mutate(&agentClients{pod: pod})
			_, err := loadAgent(context.Background(), fake.NewSimpleClientset(pod), nil, pod.Namespace, pod.Name)
			require.ErrorContains(t, err, tc.message)
		})
	}
	_, err := loadAgent(context.Background(), fake.NewSimpleClientset(), nil, "run-namespace", "missing-pod")
	require.ErrorContains(t, err, "failed to get executor plugin Pod")
	_, err = loadAgent(context.Background(), fake.NewSimpleClientset(), nil, "", "pod")
	require.ErrorContains(t, err, "identity is incomplete")
}

func TestWatchedTasksetServesCacheWithoutPolling(t *testing.T) {
	pod := agentPodForAuth()
	taskset := &wf.WorkflowTaskSet{ObjectMeta: metav1.ObjectMeta{Name: "workflow", Namespace: pod.Namespace}}
	argo := argofake.NewSimpleClientset(taskset) //nolint:staticcheck // SA1019: Argo v4.1.2 does not generate the replacement NewClientset.
	var reads atomic.Int32
	argo.PrependReactor("get", "workflowtasksets", func(ktesting.Action) (bool, runtime.Object, error) {
		reads.Add(1)
		return false, nil, nil
	})
	agent, _ := fakeAgent(t, argo)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	source := watchedTaskset(ctx)
	for range 5 {
		current, err := source(context.Background(), agent)
		require.NoError(t, err)
		assert.Equal(t, "workflow", current.Name)
	}
	assert.Zero(t, reads.Load(), "polls are served from the informer cache")
	taskset.Spec.Tasks = map[string]wf.Template{"node-1": {}}
	_, err := argo.ArgoprojV1alpha1().WorkflowTaskSets(pod.Namespace).Update(ctx, taskset, metav1.UpdateOptions{})
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		current, err := source(context.Background(), agent)
		return err == nil && len(current.Spec.Tasks) == 1
	}, 5*time.Second, 10*time.Millisecond)
}

func TestWatchedTasksetReportsMissingObject(t *testing.T) {
	agent, _ := fakeAgent(t, argofake.NewSimpleClientset()) //nolint:staticcheck // SA1019: Argo v4.1.2 does not generate the replacement NewClientset.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_, err := watchedTaskset(ctx)(context.Background(), agent)
	require.ErrorContains(t, err, "does not exist yet")
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
