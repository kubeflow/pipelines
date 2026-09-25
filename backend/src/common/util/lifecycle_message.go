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

import "strings"

// transientNodeMessages are Kubernetes pod-startup reasons, not failures.
var transientNodeMessages = map[string]bool{
	"PodInitializing":   true,
	"ContainerCreating": true,
}

// terminalSuccessStates are states that should not surface a lifecycle warning.
var terminalSuccessStates = map[string]bool{
	"Succeeded": true,
	"Skipped":   true,
	"Omitted":   true,
}

func lifecycleMessageReason(message string) string {
	reason, _, _ := strings.Cut(message, ":")
	return strings.TrimSpace(reason)
}

// NormalizeLifecycleMessage drops transient startup reasons, messages from successful nodes,
// and user-code exit messages (non-goal per KEP-12843).
func NormalizeLifecycleMessage(message string, state string) string {
	if transientNodeMessages[message] || transientNodeMessages[lifecycleMessageReason(message)] {
		return ""
	}
	if terminalSuccessStates[state] {
		return ""
	}
	// Argo records "Error (exit code N)" for normal non-zero exits; these are user-script
	// failures, not infrastructure lifecycle events.
	if strings.HasPrefix(message, "Error (exit code ") {
		return ""
	}
	return message
}

// ResolveNodeLifecycleMessages propagates executor-pod messages up to parent task nodes.
// Terminal-success nodes return empty and do not inherit child messages.
func ResolveNodeLifecycleMessages(nodes map[string]NodeStatus) map[string]string {
	resolved := make(map[string]string, len(nodes))
	onStack := make(map[string]bool)

	var resolve func(id string) string
	resolve = func(id string) string {
		if msg, ok := resolved[id]; ok {
			return msg
		}
		if onStack[id] {
			return ""
		}
		onStack[id] = true
		defer delete(onStack, id)

		node, ok := nodes[id]
		if !ok {
			resolved[id] = ""
			return ""
		}
		if terminalSuccessStates[node.State] {
			resolved[id] = ""
			return ""
		}
		msg := NormalizeLifecycleMessage(node.Message, node.State)
		if msg == "" {
			for _, childID := range node.Children {
				if childMsg := resolve(childID); childMsg != "" {
					msg = childMsg
					break
				}
			}
		}
		resolved[id] = msg
		return msg
	}

	for id := range nodes {
		resolve(id)
	}
	return resolved
}
