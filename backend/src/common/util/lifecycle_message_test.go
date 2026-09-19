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

	"github.com/stretchr/testify/assert"
)

func TestNormalizeLifecycleMessage(t *testing.T) {
	tests := []struct {
		name    string
		message string
		state   string
		want    string
	}{
		{"empty message", "", "Pending", ""},
		{"transient PodInitializing", "PodInitializing", "Pending", ""},
		{"transient ContainerCreating", "ContainerCreating", "Pending", ""},
		{"Argo reason with detail", "ContainerCreating: Container is creating", "Pending", ""},
		{"Argo PodInitializing with detail", "PodInitializing: Waiting for init", "Pending", ""},
		{"real failure kept", `Back-off pulling image "ghcr.io/example/missing:v1"`, "Pending", `Back-off pulling image "ghcr.io/example/missing:v1"`},
		{"succeeded node suppressed", "some leftover message", "Succeeded", ""},
		{"skipped node suppressed", "some leftover message", "Skipped", ""},
		{"omitted node suppressed", "some leftover message", "Omitted", ""},
		{"failed node kept", "OOMKilled", "Failed", "OOMKilled"},
		{"running node kept", "ImagePullBackOff", "Running", "ImagePullBackOff"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, NormalizeLifecycleMessage(tt.message, tt.state))
		})
	}
}

func TestResolveNodeLifecycleMessages(t *testing.T) {
	t.Run("propagates child message to parent", func(t *testing.T) {
		nodes := map[string]NodeStatus{
			"parent": {ID: "parent", State: "Running", DisplayName: "dag-node", Children: []string{"child"}},
			"child":  {ID: "child", State: "Pending", DisplayName: "executor", Message: `Back-off pulling image "img:bad"`},
		}
		resolved := ResolveNodeLifecycleMessages(nodes)
		assert.Equal(t, `Back-off pulling image "img:bad"`, resolved["parent"])
		assert.Equal(t, `Back-off pulling image "img:bad"`, resolved["child"])
	})

	t.Run("parent own message takes precedence over child", func(t *testing.T) {
		nodes := map[string]NodeStatus{
			"parent": {ID: "parent", State: "Failed", DisplayName: "dag", Message: "parent error", Children: []string{"child"}},
			"child":  {ID: "child", State: "Pending", DisplayName: "exec", Message: "child error"},
		}
		resolved := ResolveNodeLifecycleMessages(nodes)
		assert.Equal(t, "parent error", resolved["parent"])
		assert.Equal(t, "child error", resolved["child"])
	})

	t.Run("transient child message not propagated", func(t *testing.T) {
		nodes := map[string]NodeStatus{
			"parent": {ID: "parent", State: "Running", Children: []string{"child"}},
			"child":  {ID: "child", State: "Pending", Message: "ContainerCreating: Container is creating"},
		}
		resolved := ResolveNodeLifecycleMessages(nodes)
		assert.Equal(t, "", resolved["parent"])
		assert.Equal(t, "", resolved["child"])
	})

	t.Run("succeeded child message suppressed", func(t *testing.T) {
		nodes := map[string]NodeStatus{
			"parent": {ID: "parent", State: "Running", Children: []string{"child"}},
			"child":  {ID: "child", State: "Succeeded", Message: "leftover"},
		}
		resolved := ResolveNodeLifecycleMessages(nodes)
		assert.Equal(t, "", resolved["parent"])
	})

	t.Run("succeeded parent does not inherit failed child attempt", func(t *testing.T) {
		nodes := map[string]NodeStatus{
			"parent": {ID: "parent", State: "Succeeded", Children: []string{"child"}},
			"child":  {ID: "child", State: "Failed", Message: "ImagePullBackOff"},
		}
		resolved := ResolveNodeLifecycleMessages(nodes)
		assert.Equal(t, "", resolved["parent"])
		assert.Equal(t, "ImagePullBackOff", resolved["child"])
	})

	t.Run("handles cycles without infinite loop", func(t *testing.T) {
		nodes := map[string]NodeStatus{
			"a": {ID: "a", State: "Running", Children: []string{"b"}},
			"b": {ID: "b", State: "Running", Children: []string{"a"}},
		}
		resolved := ResolveNodeLifecycleMessages(nodes)
		assert.Equal(t, "", resolved["a"])
		assert.Equal(t, "", resolved["b"])
	})

	t.Run("shared descendant resolved correctly", func(t *testing.T) {
		nodes := map[string]NodeStatus{
			"p1":     {ID: "p1", State: "Running", Children: []string{"shared"}},
			"p2":     {ID: "p2", State: "Running", Children: []string{"shared"}},
			"shared": {ID: "shared", State: "Pending", Message: "ImagePullBackOff"},
		}
		resolved := ResolveNodeLifecycleMessages(nodes)
		assert.Equal(t, "ImagePullBackOff", resolved["p1"])
		assert.Equal(t, "ImagePullBackOff", resolved["p2"])
		assert.Equal(t, "ImagePullBackOff", resolved["shared"])
	})
}
