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
	if isUserScriptExit(message) {
		return ""
	}
	return message
}

const (
	userExitMessagePrefix = "Error (exit code "
	// mainContainerName is the container that runs the user's code. Argo's own containers
	// (wait, init) report through the same pod, so their failures are not user-script failures.
	mainContainerName = "main"
)

// isUserScriptExit reports whether message is Argo's record of a non-zero exit from the user's
// code: "Error (exit code N)", optionally prefixed with the container name and followed by its log
// output, e.g. "main: Error (exit code 1): <logs>". Those are user-script failures, not
// infrastructure lifecycle events. The same text from another container, such as
// "wait: Error (exit code 1)" when the wait sidecar cannot write its task result, is a real
// diagnostic and is kept.
func isUserScriptExit(message string) bool {
	container, rest := splitContainerPrefix(message)
	if container != "" && container != mainContainerName {
		return false
	}
	return strings.HasPrefix(rest, userExitMessagePrefix)
}

// splitContainerPrefix splits a leading "<container>: " off an Argo node message. It returns an
// empty container name when the message has no such prefix.
func splitContainerPrefix(message string) (container, rest string) {
	name, after, ok := strings.Cut(message, ": ")
	if !ok || name == "" || strings.ContainsAny(name, " \t\n") {
		return "", message
	}
	return name, after
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
			children := node.Children
			// For Retry nodes, only the current (last) attempt is relevant.
			// Earlier attempts may have failed; the last attempt reflects current status.
			if node.Type == "Retry" && len(children) > 0 {
				children = children[len(children)-1:]
			}
			for _, childID := range children {
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
