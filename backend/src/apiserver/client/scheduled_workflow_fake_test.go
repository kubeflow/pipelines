// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package client

import (
	"context"
	"testing"

	swfapi "github.com/kubeflow/pipelines/backend/src/crd/pkg/apis/scheduledworkflow/v1beta1"
	"github.com/stretchr/testify/require"
	k8errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestFakeScheduledWorkflowMergePatch(t *testing.T) {
	client := NewScheduledWorkflowClientFake()
	ctx := context.Background()
	created, err := client.Create(ctx, &swfapi.ScheduledWorkflow{
		ObjectMeta: metav1.ObjectMeta{GenerateName: "schedule", Labels: map[string]string{"keep": "yes", "remove": "yes"}},
		Spec:       swfapi.ScheduledWorkflowSpec{Enabled: true, ServiceAccount: "runner"},
	})
	require.NoError(t, err)
	patched, err := client.Patch(ctx, created.Name, types.MergePatchType, []byte(`{"spec":{"enabled":false},"metadata":{"labels":{"remove":null,"added":"yes"}}}`))
	require.NoError(t, err)
	require.False(t, patched.Spec.Enabled)
	require.Equal(t, "runner", patched.Spec.ServiceAccount)
	require.Equal(t, map[string]string{"keep": "yes", "added": "yes"}, patched.Labels)
	stored, err := client.Get(ctx, created.Name, metav1.GetOptions{})
	require.NoError(t, err)
	require.Equal(t, patched, stored)

	patched, err = client.Patch(ctx, created.Name, types.MergePatchType, []byte(`{"spec":{"enabled":true}}`))
	require.NoError(t, err)
	require.True(t, patched.Spec.Enabled)
	before := patched.DeepCopy()
	for _, test := range []struct {
		name         string
		patchType    types.PatchType
		data         string
		subresources []string
	}{
		{name: "malformed", patchType: types.MergePatchType, data: `{`},
		{name: "invalid field type", patchType: types.MergePatchType, data: `{"spec":{"enabled":"yes"}}`},
		{name: "identity change", patchType: types.MergePatchType, data: `{"metadata":{"name":"other"}}`},
		{name: "unsupported patch", patchType: types.JSONPatchType, data: `[]`},
		{name: "unsupported subresource", patchType: types.MergePatchType, data: `{}`, subresources: []string{"status"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := client.Patch(ctx, created.Name, test.patchType, []byte(test.data), test.subresources...)
			require.Error(t, err)
			stored, err := client.Get(ctx, created.Name, metav1.GetOptions{})
			require.NoError(t, err)
			require.Equal(t, before, stored)
		})
	}
	_, err = client.Patch(ctx, "missing", types.MergePatchType, []byte(`{}`))
	require.True(t, k8errors.IsNotFound(err))
}
