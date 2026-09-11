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

package plugins

import (
	"context"
	"testing"

	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
)

func creationPluginOutput(parentID string) *apiv2beta1.PluginOutput {
	return &apiv2beta1.PluginOutput{Entries: map[string]*apiv2beta1.MetadataValue{
		EntryRootRunID: {Value: structpb.NewStringValue(parentID)},
	}}
}

func TestSetExecutionPluginParents(t *testing.T) {
	for _, populated := range []bool{false, true} {
		run := &PendingRun{}
		execution := newFakeExecutionSpec()
		execution.SetAnnotations(AnnotationKeyPluginParents, `{"spoofed":"parent"}`)
		expected := `{}`
		if populated {
			output := creationPluginOutput("parent-1")
			output.StateMessage = "not runtime ownership metadata"
			require.NoError(t, SetPendingRunPluginOutput(run, "test", output))
			expected = `{"test":"parent-1"}`
		}
		require.NoError(t, SetExecutionPluginParents(run, execution))
		require.JSONEq(t, expected, execution.ExecutionObjectMeta().Annotations[AnnotationKeyPluginParents])
	}
}

type discardedCreationHandler struct {
	fakeHandler
	ended []*PersistedRun
}

func (h *discardedCreationHandler) OnRunEnd(_ context.Context, run *PersistedRun, _ interface{}) (bool, error) {
	h.ended = append(h.ended, run)
	if h.endBool {
		run.PluginsOutput[h.name].State = apiv2beta1.PluginState_PLUGIN_FAILED
	}
	return h.endBool, h.endErr
}

func TestOnRunCreationDiscardedPreservesWinningParentsAndOutput(t *testing.T) {
	for _, test := range []struct {
		name       string
		parents    string
		wantCalls  int
		wantError  bool
		cleanupErr bool
	}{
		{name: "distinct parent", parents: `{"test":"parent-1"}`, wantCalls: 1},
		{name: "shared parent", parents: `{"test":"parent-2"}`},
		{name: "creator had no parents", parents: `{}`, wantCalls: 1},
		{name: "legacy execution", wantError: true},
		{name: "malformed metadata", parents: `{`, wantError: true},
		{name: "null metadata", parents: `null`, wantError: true},
		{name: "cleanup failure", parents: `{}`, wantCalls: 1, wantError: true, cleanupErr: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			handler := &discardedCreationHandler{fakeHandler: fakeHandler{
				name: "test", pluginConfig: &PluginConfig{}, endBool: test.cleanupErr,
			}}
			store := &fakeRunPluginOutputStoreWithError{}
			dispatcher, err := NewRunPluginDispatcherImpl([]RunPluginHandler{handler}, &fakeKubeClientProvider{}, store)
			require.NoError(t, err)
			run := &PendingRun{RunID: "run-1", Namespace: "ns1"}
			require.NoError(t, SetPendingRunPluginOutput(run, "test", creationPluginOutput("parent-2")))
			execution := newFakeExecutionSpec()
			if test.parents != "" {
				execution.SetAnnotations(AnnotationKeyPluginParents, test.parents)
			}
			err = dispatcher.OnRunCreationDiscarded(context.Background(), run, execution)
			if test.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.Len(t, handler.ended, test.wantCalls)
			if test.wantCalls > 0 {
				require.Equal(t, string(model.RuntimeStateCanceled), handler.ended[0].State)
				require.Equal(t, "parent-2", GetParentRunID(handler.ended[0].PluginsOutput["test"]))
			}
			require.Zero(t, store.callCount, "cleanup must never persist the discarded request's output")
		})
	}
}
