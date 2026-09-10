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

package util

import (
	"testing"

	workflowapi "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/stretchr/testify/assert"
)

func TestResolveNodeLifecycleMessage(t *testing.T) {
	tests := []struct {
		name    string
		nodes   map[string]workflowapi.NodeStatus
		nodeID  string
		want    string
		comment string
	}{
		{
			name: "pod node keeps its own message",
			nodes: map[string]workflowapi.NodeStatus{
				"pod": {Type: workflowapi.NodeTypePod, Phase: workflowapi.NodeFailed, Message: "OOMKilled"},
			},
			nodeID: "pod",
			want:   "OOMKilled",
		},
		{
			name: "task node bubbles up its pod child's message",
			nodes: map[string]workflowapi.NodeStatus{
				"task": {Type: workflowapi.NodeTypePod, Phase: workflowapi.NodePending, Children: []string{"executor"}},
				"executor": {
					Type: workflowapi.NodeTypePod, Phase: workflowapi.NodePending,
					Message: "Back-off pulling image \"does-not-exist:v1\"",
				},
			},
			nodeID: "task",
			want:   "Back-off pulling image \"does-not-exist:v1\"",
		},
		{
			name: "retry node's own exhausted-retries message does not shadow the real failure",
			nodes: map[string]workflowapi.NodeStatus{
				"retry": {
					Type: workflowapi.NodeTypeRetry, Phase: workflowapi.NodeFailed,
					Message: "No more retries left", Children: []string{"retry(0)"},
				},
				"retry(0)": {Type: workflowapi.NodeTypePod, Phase: workflowapi.NodeFailed, Message: "OOMKilled"},
			},
			nodeID:  "retry",
			want:    "OOMKilled",
			comment: "a Retry node's own message is control-flow prose, never a pod failure",
		},
		{
			name: "max-duration message on a DAG node is not a pod failure",
			nodes: map[string]workflowapi.NodeStatus{
				"dag": {Type: workflowapi.NodeTypeDAG, Phase: workflowapi.NodeFailed, Message: "Max duration limit exceeded"},
			},
			nodeID: "dag",
			want:   "",
		},
		{
			name: "running parent shows the latest attempt, not a stale earlier failure",
			nodes: map[string]workflowapi.NodeStatus{
				"task": {Type: workflowapi.NodeTypePod, Phase: workflowapi.NodeRunning, Children: []string{"retry"}},
				"retry": {
					Type: workflowapi.NodeTypeRetry, Phase: workflowapi.NodeRunning,
					// Argo appends retry children oldest-first: retry(0) is the failed first
					// attempt, retry(1) is the current, healthy attempt.
					Children: []string{"retry(0)", "retry(1)"},
				},
				"retry(0)": {Type: workflowapi.NodeTypePod, Phase: workflowapi.NodeFailed, Message: "OOMKilled"},
				"retry(1)": {Type: workflowapi.NodeTypePod, Phase: workflowapi.NodeRunning, Message: ""},
			},
			nodeID:  "task",
			want:    "",
			comment: "must not show attempt 0's stale OOMKilled while attempt 1 is live and healthy",
		},
		{
			name: "a later failed attempt is preferred over an earlier failed attempt",
			nodes: map[string]workflowapi.NodeStatus{
				"retry": {
					Type: workflowapi.NodeTypeRetry, Phase: workflowapi.NodeFailed,
					Children: []string{"retry(0)", "retry(1)"},
				},
				"retry(0)": {Type: workflowapi.NodeTypePod, Phase: workflowapi.NodeFailed, Message: "OOMKilled"},
				"retry(1)": {Type: workflowapi.NodeTypePod, Phase: workflowapi.NodeFailed, Message: "ImagePullBackOff"},
			},
			nodeID: "retry",
			want:   "ImagePullBackOff",
		},
		{
			name: "succeeded node does not inherit a failed descendant's message",
			nodes: map[string]workflowapi.NodeStatus{
				"task":     {Type: workflowapi.NodeTypePod, Phase: workflowapi.NodeSucceeded, Children: []string{"executor"}},
				"executor": {Type: workflowapi.NodeTypePod, Phase: workflowapi.NodeFailed, Message: "OOMKilled"},
			},
			nodeID: "task",
			want:   "",
		},
		{
			name: "skipped node's skip-reason is not a failure",
			nodes: map[string]workflowapi.NodeStatus{
				"skipped": {
					Type: workflowapi.NodeTypePod, Phase: workflowapi.NodeSkipped,
					Message: "when 'true != true' evaluated false",
				},
			},
			nodeID: "skipped",
			want:   "",
		},
		{
			name:   "missing node returns empty",
			nodes:  map[string]workflowapi.NodeStatus{},
			nodeID: "does-not-exist",
			want:   "",
		},
		{
			name: "a NodeError on a Pod node is Argo's own error, not a pod failure",
			nodes: map[string]workflowapi.NodeStatus{
				"pod": {
					Type: workflowapi.NodeTypePod, Phase: workflowapi.NodeError,
					Message: "pods \"my-task\" is forbidden: error looking up service account",
				},
			},
			nodeID:  "pod",
			want:    "",
			comment: "NodeError means the controller failed to manage the pod, not the pod itself",
		},
		{
			name: "a NodeFailed Pod message still surfaces normally",
			nodes: map[string]workflowapi.NodeStatus{
				"pod": {Type: workflowapi.NodeTypePod, Phase: workflowapi.NodeFailed, Message: "OOMKilled"},
			},
			nodeID:  "pod",
			want:    "OOMKilled",
			comment: "regression guard: NodeFailed must still work after the NodeError exclusion",
		},
		{
			name: "cyclic children do not cause infinite recursion",
			nodes: map[string]workflowapi.NodeStatus{
				"a": {Type: workflowapi.NodeTypeDAG, Phase: workflowapi.NodeFailed, Children: []string{"b"}},
				"b": {Type: workflowapi.NodeTypeDAG, Phase: workflowapi.NodeFailed, Children: []string{"a"}},
			},
			nodeID: "a",
			want:   "",
		},
		{
			name: "three-level nesting (DAG -> Retry -> Pod) bubbles up correctly",
			nodes: map[string]workflowapi.NodeStatus{
				"stepgroup": {Type: workflowapi.NodeTypeStepGroup, Phase: workflowapi.NodeFailed, Children: []string{"retry"}},
				"retry":     {Type: workflowapi.NodeTypeRetry, Phase: workflowapi.NodeFailed, Children: []string{"retry(0)"}},
				"retry(0)":  {Type: workflowapi.NodeTypePod, Phase: workflowapi.NodeFailed, Message: "Unschedulable"},
			},
			nodeID: "stepgroup",
			want:   "Unschedulable",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveNodeLifecycleMessage(tt.nodes, tt.nodeID)
			assert.Equal(t, tt.want, got, tt.comment)
		})
	}
}

func TestResolveNodeLifecycleMessage_IntegratesWithClassifyPodFailure(t *testing.T) {
	nodes := map[string]workflowapi.NodeStatus{
		"retry": {
			Type: workflowapi.NodeTypeRetry, Phase: workflowapi.NodeFailed,
			Message: "No more retries left", Children: []string{"retry(0)"},
		},
		"retry(0)": {Type: workflowapi.NodeTypePod, Phase: workflowapi.NodeFailed, Message: "OOMKilled"},
	}
	message := ResolveNodeLifecycleMessage(nodes, "retry")
	category, reason := ClassifyPodFailure(message)
	assert.Equal(t, PodFailureCategoryRuntime, category)
	assert.Equal(t, "OOMKilled", reason)
}

func TestLastNameSegment(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"my-wf(0).root(0).heavy-task-driver", "heavy-task-driver"},
		{"my-wf(0).root(0).heavy-task-driver(0)", "heavy-task-driver(0)"},
		{"my-wf(0).root(0).heavy-task", "heavy-task"},
		{"my-wf", "my-wf"},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, lastNameSegment(tt.name))
		})
	}
}

// Node shapes below are modeled directly on a real workflow's live status.nodes, captured with
// `kubectl get wf -o json` against a KFP v2 pipeline with a single component task named
// "heavy-task", not reconstructed from compiler source alone:
//
//	id=...-2792668430 name=wf(0).root(0).heavy-task-driver         type=Retry  phase=Succeeded
//	id=...-297853757  name=wf(0).root(0).heavy-task-driver(0)      type=Pod    phase=Succeeded
//	id=...-435981245  name=wf(0).root(0).heavy-task                type=Retry  phase=Running
//
// The driver is itself Retry-wrapped, not a bare Pod node directly -- both the "-driver" group
// node and its "(0)" attempt child exist, same as any other retried task.
func TestResolveMissingTaskDriverFailure(t *testing.T) {
	// realDagChildren mirrors "root(0)"'s actual Children from the same captured workflow: the
	// driver group node's ID, and (only once the driver succeeds) the executor group node's ID.
	t.Run("driver failed: no executor node exists yet, driver's message surfaces", func(t *testing.T) {
		nodes := map[string]workflowapi.NodeStatus{
			"root": {
				Type: workflowapi.NodeTypeDAG, Phase: workflowapi.NodeRunning,
				// Only the driver child exists -- the executor was never created, since it
				// Depends on the driver and the driver never succeeded.
				Children: []string{"driver-group"},
			},
			"driver-group": {
				Type: workflowapi.NodeTypeRetry, Phase: workflowapi.NodeFailed,
				Name:     "wf(0).root(0).heavy-task-driver",
				Children: []string{"driver-attempt-0"},
			},
			"driver-attempt-0": {
				Type: workflowapi.NodeTypePod, Phase: workflowapi.NodeFailed,
				Name: "wf(0).root(0).heavy-task-driver(0)",
				Message: "failed to create PVC \"typo-name\": " +
					"persistentvolumeclaims \"typo-name\" not found",
			},
		}
		got := ResolveMissingTaskDriverFailure(nodes, "root", "heavy-task")
		assert.Contains(t, got, "persistentvolumeclaims \"typo-name\" not found")
	})

	t.Run("driver succeeded and executor already exists: defers, returns empty", func(t *testing.T) {
		nodes := map[string]workflowapi.NodeStatus{
			"root": {
				Type:     workflowapi.NodeTypeDAG,
				Phase:    workflowapi.NodeRunning,
				Children: []string{"driver-group", "executor-group"},
			},
			"driver-group": {
				Type: workflowapi.NodeTypeRetry, Phase: workflowapi.NodeSucceeded,
				Name:     "wf(0).root(0).heavy-task-driver",
				Children: []string{"driver-attempt-0"},
			},
			"driver-attempt-0": {
				Type: workflowapi.NodeTypePod, Phase: workflowapi.NodeSucceeded,
				Name: "wf(0).root(0).heavy-task-driver(0)",
			},
			"executor-group": {
				Type: workflowapi.NodeTypeRetry, Phase: workflowapi.NodeRunning,
				Name: "wf(0).root(0).heavy-task",
			},
		}
		got := ResolveMissingTaskDriverFailure(nodes, "root", "heavy-task")
		assert.Equal(t, "", got,
			"the executor node exists, so ResolveNodeLifecycleMessage owns this, not the driver check")
	})

	t.Run("driver still running, executor doesn't exist yet: not a failure, returns empty", func(t *testing.T) {
		nodes := map[string]workflowapi.NodeStatus{
			"root": {
				Type: workflowapi.NodeTypeDAG, Phase: workflowapi.NodeRunning,
				Children: []string{"driver-group"},
			},
			"driver-group": {
				Type: workflowapi.NodeTypeRetry, Phase: workflowapi.NodeRunning,
				Name:     "wf(0).root(0).heavy-task-driver",
				Children: []string{"driver-attempt-0"},
			},
			"driver-attempt-0": {
				Type: workflowapi.NodeTypePod, Phase: workflowapi.NodeRunning,
				Name: "wf(0).root(0).heavy-task-driver(0)",
			},
		}
		got := ResolveMissingTaskDriverFailure(nodes, "root", "heavy-task")
		assert.Equal(t, "", got)
	})

	t.Run("unrelated task name under the same parent: no match, returns empty", func(t *testing.T) {
		nodes := map[string]workflowapi.NodeStatus{
			"root": {
				Type: workflowapi.NodeTypeDAG, Phase: workflowapi.NodeRunning,
				Children: []string{"driver-group"},
			},
			"driver-group": {
				Type: workflowapi.NodeTypeRetry, Phase: workflowapi.NodeFailed,
				Name:     "wf(0).root(0).heavy-task-driver",
				Children: []string{"driver-attempt-0"},
			},
			"driver-attempt-0": {
				Type: workflowapi.NodeTypePod, Phase: workflowapi.NodeFailed,
				Name: "wf(0).root(0).heavy-task-driver(0)", Message: "some driver failure",
			},
		}
		got := ResolveMissingTaskDriverFailure(nodes, "root", "some-other-task")
		assert.Equal(t, "", got)
	})

	t.Run("unknown parent node: returns empty rather than panicking", func(t *testing.T) {
		got := ResolveMissingTaskDriverFailure(map[string]workflowapi.NodeStatus{}, "does-not-exist", "heavy-task")
		assert.Equal(t, "", got)
	})

	t.Run("two tasks sharing a parent are not confused with each other", func(t *testing.T) {
		nodes := map[string]workflowapi.NodeStatus{
			"root": {
				Type: workflowapi.NodeTypeDAG, Phase: workflowapi.NodeRunning,
				Children: []string{"a-driver-group", "b-driver-group", "b-executor-group"},
			},
			"a-driver-group": {
				Type: workflowapi.NodeTypeRetry, Phase: workflowapi.NodeFailed,
				Name:     "wf(0).root(0).task-a-driver",
				Children: []string{"a-driver-attempt-0"},
			},
			"a-driver-attempt-0": {
				Type: workflowapi.NodeTypePod, Phase: workflowapi.NodeFailed,
				Name: "wf(0).root(0).task-a-driver(0)", Message: "task A's driver failed",
			},
			"b-driver-group": {
				Type: workflowapi.NodeTypeRetry, Phase: workflowapi.NodeSucceeded,
				Name: "wf(0).root(0).task-b-driver",
			},
			"b-executor-group": {
				Type: workflowapi.NodeTypeRetry, Phase: workflowapi.NodeRunning,
				Name: "wf(0).root(0).task-b",
			},
		}
		assert.Contains(t, ResolveMissingTaskDriverFailure(nodes, "root", "task-a"), "task A's driver failed")
		assert.Equal(t, "", ResolveMissingTaskDriverFailure(nodes, "root", "task-b"))
	})
}
