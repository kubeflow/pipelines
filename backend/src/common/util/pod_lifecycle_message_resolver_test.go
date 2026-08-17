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
