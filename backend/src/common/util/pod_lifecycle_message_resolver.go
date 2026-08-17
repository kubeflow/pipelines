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

import workflowapi "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"

// nonFailureNodePhases are terminal/benign Argo node phases that never represent a pod lifecycle
// failure, even when the node has a leftover message (e.g. a Skipped node's skip-reason, or a
// Succeeded node that previously failed and later completed on retry).
var nonFailureNodePhases = map[workflowapi.NodePhase]bool{
	workflowapi.NodeSucceeded: true,
	workflowapi.NodeSkipped:   true,
	workflowapi.NodeOmitted:   true,
}

// ResolveNodeLifecycleMessage returns the pod lifecycle failure message that should be
// attributed to nodeID, or an empty string if none applies. Feed the result into
// ClassifyPodFailure to get a category.
//
// Two correctness properties are enforced here, neither handled by the raw-message approach in
// the KEP under review on kubeflow/pipelines#12843 (persisting Argo's node.Message as-is for UI
// display):
//
//  1. Only a Pod-type node's own message is ever treated as a pod lifecycle failure. Argo's
//     control-flow nodes (Retry, DAG, StepGroup) carry their own orchestration messages -- e.g.
//     "No more retries left" on a Retry node once all attempts are exhausted, or "Max duration
//     limit exceeded" -- which are not pod failures and must not be classified as one. A rule of
//     "any non-empty message on any non-terminal node" mislabels a task's own exhausted retry
//     policy as an infrastructure failure, and can actively hide the real underlying reason (a
//     Retry node's own "No more retries left" message takes precedence over its last attempt's
//     real "OOMKilled" message under a naive own-message-wins rule).
//  2. For a Retry-type node, only the most recent attempt's message is ever consulted, and it is
//     authoritative even when empty. Argo appends retry attempt children oldest-first (confirmed
//     against argo-workflows' workflow/controller/operator.go, which always does
//     `node.Children = append(node.Children, childID)`), so a lookup that searches children for
//     the first non-empty message -- in either direction -- can resurface a stale failure from
//     an earlier attempt even while the latest attempt is currently healthy and running. Only
//     the latest attempt reflects what is happening right now.
func ResolveNodeLifecycleMessage(nodes map[string]workflowapi.NodeStatus, nodeID string) string {
	resolved := make(map[string]string, len(nodes))
	visiting := make(map[string]bool, len(nodes))
	return resolveNodeLifecycleMessage(nodes, nodeID, resolved, visiting)
}

func resolveNodeLifecycleMessage(
	nodes map[string]workflowapi.NodeStatus,
	nodeID string,
	resolved map[string]string,
	visiting map[string]bool,
) string {
	if message, ok := resolved[nodeID]; ok {
		return message
	}
	node, ok := nodes[nodeID]
	if !ok {
		return ""
	}
	if visiting[nodeID] {
		// Cycle guard: the Argo node graph is not expected to contain cycles, but don't hang if
		// one somehow exists.
		return ""
	}
	visiting[nodeID] = true
	defer delete(visiting, nodeID)

	if nonFailureNodePhases[node.Phase] {
		resolved[nodeID] = ""
		return ""
	}

	// Only a Pod-type node's own message is a candidate pod lifecycle failure. Control-flow
	// nodes (Retry, DAG, StepGroup, ...) have their own message semantics and are never treated
	// as a pod failure here, even when non-empty.
	if node.Type == workflowapi.NodeTypePod && node.Message != "" {
		resolved[nodeID] = node.Message
		return node.Message
	}

	if node.Type == workflowapi.NodeTypeRetry && len(node.Children) > 0 {
		// The Argo controller appends retry attempt children oldest-first (confirmed against
		// argo-workflows' operator.go), so the last child is the most recent attempt. That
		// attempt is authoritative: if it produced no message, the retry group currently has no
		// pod-level failure to report, even if an earlier attempt failed. Do NOT fall through to
		// earlier children here -- doing so would resurface a stale failure from an attempt that
		// has already been superseded by a healthy one.
		latest := node.Children[len(node.Children)-1]
		message := resolveNodeLifecycleMessage(nodes, latest, resolved, visiting)
		resolved[nodeID] = message
		return message
	}

	// For non-Retry parents (DAG, StepGroup, ...), children represent genuinely different
	// things, not sequential attempts of the same thing, so search all of them for the first
	// non-empty message.
	for _, childID := range node.Children {
		if message := resolveNodeLifecycleMessage(nodes, childID, resolved, visiting); message != "" {
			resolved[nodeID] = message
			return message
		}
	}

	resolved[nodeID] = ""
	return ""
}
