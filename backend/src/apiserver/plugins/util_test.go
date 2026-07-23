package plugins

import (
	"encoding/json"
	"testing"
	"time"

	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestBuildKFPRunURL_EmptyRunID(t *testing.T) {
	url := BuildKFPRunURL("", "ns1", "https://kfp.example.com", "")
	assert.Empty(t, url)
}

func TestBuildKFPRunURL_EmptyBaseURL(t *testing.T) {
	url := BuildKFPRunURL("run-1", "ns1", "", "")
	assert.Empty(t, url)
}

func TestBuildKFPRunURL_DefaultTemplate(t *testing.T) {
	url := BuildKFPRunURL("run-1", "ns1", "https://kfp.example.com", "")
	assert.Equal(t, "https://kfp.example.com/#/runs/details/run-1", url)
}

func TestBuildKFPRunURL_DefaultTemplate_TrailingSlash(t *testing.T) {
	url := BuildKFPRunURL("run-1", "ns1", "https://kfp.example.com/", "")
	assert.Equal(t, "https://kfp.example.com/#/runs/details/run-1", url)
}

func TestBuildKFPRunURL_CustomTemplate(t *testing.T) {
	tmpl := "/pipelines/{namespace}/runs/{run_id}"
	url := BuildKFPRunURL("run-2", "proj-x", "https://console.example.com", tmpl)
	assert.Equal(t, "https://console.example.com/pipelines/proj-x/runs/run-2", url)
}

func TestBuildKFPRunURL_CustomTemplate_WithHash(t *testing.T) {
	tmpl := "#/namespaces/{namespace}/runs/{run_id}"
	url := BuildKFPRunURL("run-3", "team-a", "https://app.example.com", tmpl)
	assert.Equal(t, "https://app.example.com#/namespaces/team-a/runs/run-3", url)
}

func TestBuildKFPRunURL_CustomTemplate_MissingNamespace(t *testing.T) {
	tmpl := "/ns/{namespace}/runs/{run_id}"
	url := BuildKFPRunURL("run-4", "", "https://app.example.com", tmpl)
	assert.Empty(t, url)
}

func TestBuildKFPRunURL_URLEscaping(t *testing.T) {
	url := BuildKFPRunURL("run/with/slashes", "ns-1", "https://kfp.example.com", "")
	assert.Equal(t, "https://kfp.example.com/#/runs/details/run%2Fwith%2Fslashes", url)
}

// Serialization Tests

func TestDeserializePluginsOutput_Empty(t *testing.T) {
	result, err := DeserializePluginsOutput(nil)
	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestDeserializePluginsOutput_EmptyString(t *testing.T) {
	empty := model.LargeText("")
	result, err := DeserializePluginsOutput(&empty)
	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestDeserializePluginsOutput_SinglePlugin(t *testing.T) {
	pluginOutput := &apiv2beta1.PluginOutput{
		State:        apiv2beta1.PluginState_PLUGIN_SUCCEEDED,
		StateMessage: "success",
		Entries: map[string]*apiv2beta1.MetadataValue{
			"key1": {Value: structpb.NewStringValue("value1")},
		},
	}
	serialized, err := SerializePluginsOutput(map[string]*apiv2beta1.PluginOutput{
		"test-plugin": pluginOutput,
	})
	require.NoError(t, err)

	result, err := DeserializePluginsOutput(serialized)
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, apiv2beta1.PluginState_PLUGIN_SUCCEEDED, result["test-plugin"].State)
	assert.Equal(t, "success", result["test-plugin"].StateMessage)
	assert.Equal(t, "value1", result["test-plugin"].Entries["key1"].Value.GetStringValue())
}

func TestDeserializePluginsOutput_MultiplePlugins(t *testing.T) {
	outputs := map[string]*apiv2beta1.PluginOutput{
		"plugin1": {
			State: apiv2beta1.PluginState_PLUGIN_SUCCEEDED,
			Entries: map[string]*apiv2beta1.MetadataValue{
				"key1": {Value: structpb.NewStringValue("value1")},
			},
		},
		"plugin2": {
			State: apiv2beta1.PluginState_PLUGIN_FAILED,
			Entries: map[string]*apiv2beta1.MetadataValue{
				"key2": {Value: structpb.NewStringValue("value2")},
			},
		},
	}
	serialized, err := SerializePluginsOutput(outputs)
	require.NoError(t, err)

	result, err := DeserializePluginsOutput(serialized)
	require.NoError(t, err)
	require.Len(t, result, 2)
	assert.Equal(t, apiv2beta1.PluginState_PLUGIN_SUCCEEDED, result["plugin1"].State)
	assert.Equal(t, apiv2beta1.PluginState_PLUGIN_FAILED, result["plugin2"].State)
}

func TestDeserializePluginsOutput_MalformedJSON(t *testing.T) {
	malformed := model.LargeText(`{"invalid": json`)
	_, err := DeserializePluginsOutput(&malformed)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to unmarshal plugins_output")
}

func TestDeserializePluginsOutput_InvalidPluginData(t *testing.T) {
	// JSON with invalid plugin payload that can't unmarshal to PluginOutput
	invalidData := model.LargeText(`{"plugin1": "not-a-valid-plugin-output"}`)
	result, err := DeserializePluginsOutput(&invalidData)
	require.NoError(t, err)
	// Should skip invalid entries silently
	assert.Empty(t, result)
}

func TestSerializePluginsOutput_Empty(t *testing.T) {
	result, err := SerializePluginsOutput(nil)
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestSerializePluginsOutput_EmptyMap(t *testing.T) {
	result, err := SerializePluginsOutput(map[string]*apiv2beta1.PluginOutput{})
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestSerializePluginsOutput_SinglePlugin(t *testing.T) {
	outputs := map[string]*apiv2beta1.PluginOutput{
		"test-plugin": {
			State:        apiv2beta1.PluginState_PLUGIN_SUCCEEDED,
			StateMessage: "test message",
			Entries: map[string]*apiv2beta1.MetadataValue{
				"key1": {Value: structpb.NewStringValue("value1")},
			},
		},
	}
	result, err := SerializePluginsOutput(outputs)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Verify it's valid JSON
	var envelope PluginsOutputEnvelope
	err = json.Unmarshal([]byte(*result), &envelope)
	require.NoError(t, err)
	assert.Len(t, envelope.Plugins, 1)
}

func TestSerializePluginsOutput_MultiplePlugins(t *testing.T) {
	outputs := map[string]*apiv2beta1.PluginOutput{
		"plugin1": {State: apiv2beta1.PluginState_PLUGIN_SUCCEEDED},
		"plugin2": {State: apiv2beta1.PluginState_PLUGIN_FAILED},
		"plugin3": {State: apiv2beta1.PluginState_PLUGIN_RUNNING},
	}
	result, err := SerializePluginsOutput(outputs)
	require.NoError(t, err)
	require.NotNil(t, result)

	var envelope PluginsOutputEnvelope
	err = json.Unmarshal([]byte(*result), &envelope)
	require.NoError(t, err)
	assert.Len(t, envelope.Plugins, 3)
}

func TestUpsertPluginOutput_NewPlugin(t *testing.T) {
	output := &apiv2beta1.PluginOutput{
		State:        apiv2beta1.PluginState_PLUGIN_SUCCEEDED,
		StateMessage: "success",
		Entries: map[string]*apiv2beta1.MetadataValue{
			"key1": {Value: structpb.NewStringValue("value1")},
		},
	}

	result, err := upsertPluginOutput(nil, "test-plugin", output)
	require.NoError(t, err)
	assert.NotEmpty(t, result)

	// Verify the result
	var envelope PluginsOutputEnvelope
	err = json.Unmarshal([]byte(result), &envelope)
	require.NoError(t, err)
	assert.Len(t, envelope.Plugins, 1)
}

func TestUpsertPluginOutput_UpdateExisting(t *testing.T) {
	// Create initial output with one plugin
	initial := map[string]*apiv2beta1.PluginOutput{
		"plugin1": {
			State: apiv2beta1.PluginState_PLUGIN_RUNNING,
			Entries: map[string]*apiv2beta1.MetadataValue{
				"key1": {Value: structpb.NewStringValue("old-value")},
			},
		},
	}
	serialized, err := SerializePluginsOutput(initial)
	require.NoError(t, err)

	// Update plugin1
	updated := &apiv2beta1.PluginOutput{
		State: apiv2beta1.PluginState_PLUGIN_SUCCEEDED,
		Entries: map[string]*apiv2beta1.MetadataValue{
			"key1": {Value: structpb.NewStringValue("new-value")},
		},
	}

	serializedStr := string(*serialized)
	result, err := upsertPluginOutput(&serializedStr, "plugin1", updated)
	require.NoError(t, err)

	// Verify plugin1 was updated
	resultLT := model.LargeText(result)
	deserialized, err := DeserializePluginsOutput(&resultLT)
	require.NoError(t, err)
	assert.Equal(t, apiv2beta1.PluginState_PLUGIN_SUCCEEDED, deserialized["plugin1"].State)
	assert.Equal(t, "new-value", deserialized["plugin1"].Entries["key1"].Value.GetStringValue())
}

func TestUpsertPluginOutput_AddToExisting(t *testing.T) {
	// Create initial output with one plugin
	initial := map[string]*apiv2beta1.PluginOutput{
		"plugin1": {State: apiv2beta1.PluginState_PLUGIN_SUCCEEDED},
	}
	serialized, err := SerializePluginsOutput(initial)
	require.NoError(t, err)

	// Add plugin2
	plugin2 := &apiv2beta1.PluginOutput{
		State: apiv2beta1.PluginState_PLUGIN_RUNNING,
	}

	serializedStr := string(*serialized)
	result, err := upsertPluginOutput(&serializedStr, "plugin2", plugin2)
	require.NoError(t, err)

	// Verify both plugins exist
	resultLT := model.LargeText(result)
	deserialized, err := DeserializePluginsOutput(&resultLT)
	require.NoError(t, err)
	assert.Len(t, deserialized, 2)
	assert.Equal(t, apiv2beta1.PluginState_PLUGIN_SUCCEEDED, deserialized["plugin1"].State)
	assert.Equal(t, apiv2beta1.PluginState_PLUGIN_RUNNING, deserialized["plugin2"].State)
}

func TestUpsertPluginOutput_MalformedExisting(t *testing.T) {
	malformed := `{"invalid": json`
	_, err := upsertPluginOutput(&malformed, "test-plugin", &apiv2beta1.PluginOutput{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to unmarshal existing plugins_output")
}

func TestSetPendingRunPluginOutput_NilRun(t *testing.T) {
	err := SetPendingRunPluginOutput(nil, "test", &apiv2beta1.PluginOutput{})
	require.NoError(t, err)
}

func TestSetPendingRunPluginOutput_NilOutput(t *testing.T) {
	run := &PendingRun{RunID: "run-1"}
	err := SetPendingRunPluginOutput(run, "test", nil)
	require.NoError(t, err)
	assert.Nil(t, run.PluginsOutput)
}

func TestSetPendingRunPluginOutput_EmptyPluginName(t *testing.T) {
	run := &PendingRun{RunID: "run-1"}
	err := SetPendingRunPluginOutput(run, "", &apiv2beta1.PluginOutput{})
	require.NoError(t, err)
	assert.Nil(t, run.PluginsOutput)
}

func TestSetPendingRunPluginOutput_FirstPlugin(t *testing.T) {
	run := &PendingRun{RunID: "run-1"}
	output := &apiv2beta1.PluginOutput{
		State: apiv2beta1.PluginState_PLUGIN_SUCCEEDED,
		Entries: map[string]*apiv2beta1.MetadataValue{
			"key1": {Value: structpb.NewStringValue("value1")},
		},
	}

	err := SetPendingRunPluginOutput(run, "test-plugin", output)
	require.NoError(t, err)
	require.NotNil(t, run.PluginsOutput)

	// Verify the output was set
	deserialized, err := DeserializePluginsOutput((*model.LargeText)(run.PluginsOutput))
	require.NoError(t, err)
	assert.Len(t, deserialized, 1)
	assert.Equal(t, apiv2beta1.PluginState_PLUGIN_SUCCEEDED, deserialized["test-plugin"].State)
}

func TestSetPendingRunPluginOutput_UpdateExisting(t *testing.T) {
	run := &PendingRun{RunID: "run-1"}

	// Add first plugin
	output1 := &apiv2beta1.PluginOutput{
		State: apiv2beta1.PluginState_PLUGIN_RUNNING,
	}
	err := SetPendingRunPluginOutput(run, "plugin1", output1)
	require.NoError(t, err)

	// Add second plugin
	output2 := &apiv2beta1.PluginOutput{
		State: apiv2beta1.PluginState_PLUGIN_SUCCEEDED,
	}
	err = SetPendingRunPluginOutput(run, "plugin2", output2)
	require.NoError(t, err)

	// Verify both plugins exist
	deserialized, err := DeserializePluginsOutput((*model.LargeText)(run.PluginsOutput))
	require.NoError(t, err)
	assert.Len(t, deserialized, 2)
	assert.Equal(t, apiv2beta1.PluginState_PLUGIN_RUNNING, deserialized["plugin1"].State)
	assert.Equal(t, apiv2beta1.PluginState_PLUGIN_SUCCEEDED, deserialized["plugin2"].State)
}

func TestModelToPersistedRun_NilModel(t *testing.T) {
	_, err := ModelToPersistedRun(nil, "ns1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "model.Run is nil")
}

func TestModelToPersistedRun_BasicConversion(t *testing.T) {
	modelRun := &model.Run{
		UUID: "run-123",
		RunDetails: model.RunDetails{
			State: model.RuntimeStateSucceeded,
		},
	}

	result, err := ModelToPersistedRun(modelRun, "ns1")
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "run-123", result.RunID)
	assert.Equal(t, "ns1", result.Namespace)
	assert.Equal(t, "SUCCEEDED", result.State)
	assert.Nil(t, result.FinishedAt)
}

func TestModelToPersistedRun_WithFinishedAt(t *testing.T) {
	finishedAt := int64(1234567890)
	modelRun := &model.Run{
		UUID: "run-123",
		RunDetails: model.RunDetails{
			State:           model.RuntimeStateFailed,
			FinishedAtInSec: finishedAt,
		},
	}

	result, err := ModelToPersistedRun(modelRun, "ns1")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.FinishedAt)
	assert.Equal(t, time.Unix(finishedAt, 0), *result.FinishedAt)
}

func TestModelToPersistedRun_WithPluginsOutput(t *testing.T) {
	outputs := map[string]*apiv2beta1.PluginOutput{
		"test-plugin": {
			State: apiv2beta1.PluginState_PLUGIN_SUCCEEDED,
			Entries: map[string]*apiv2beta1.MetadataValue{
				"key1": {Value: structpb.NewStringValue("value1")},
			},
		},
	}
	serialized, err := SerializePluginsOutput(outputs)
	require.NoError(t, err)

	modelRun := &model.Run{
		UUID: "run-123",
		RunDetails: model.RunDetails{
			State:               model.RuntimeStateSucceeded,
			PluginsOutputString: serialized,
		},
	}

	result, err := ModelToPersistedRun(modelRun, "ns1")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.PluginsOutput)
	assert.Len(t, result.PluginsOutput, 1)
	assert.Equal(t, apiv2beta1.PluginState_PLUGIN_SUCCEEDED, result.PluginsOutput["test-plugin"].State)
}

func TestModelToPersistedRun_InvalidPluginsOutput(t *testing.T) {
	malformed := model.LargeText(`{"invalid": json`)
	modelRun := &model.Run{
		UUID: "run-123",
		RunDetails: model.RunDetails{
			State:               model.RuntimeStateSucceeded,
			PluginsOutputString: &malformed,
		},
	}

	_, err := ModelToPersistedRun(modelRun, "ns1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to deserialize plugins_output")
}

func TestGetStringEntry_NilOutput(t *testing.T) {
	result := GetStringEntry(nil, "key")
	assert.Empty(t, result)
}

func TestGetStringEntry_EmptyKey(t *testing.T) {
	output := &apiv2beta1.PluginOutput{
		Entries: map[string]*apiv2beta1.MetadataValue{
			"key1": {Value: structpb.NewStringValue("value1")},
		},
	}
	result := GetStringEntry(output, "")
	assert.Empty(t, result)
}

func TestGetStringEntry_NilEntries(t *testing.T) {
	output := &apiv2beta1.PluginOutput{}
	result := GetStringEntry(output, "key")
	assert.Empty(t, result)
}

func TestGetStringEntry_MissingKey(t *testing.T) {
	output := &apiv2beta1.PluginOutput{
		Entries: map[string]*apiv2beta1.MetadataValue{
			"key1": {Value: structpb.NewStringValue("value1")},
		},
	}
	result := GetStringEntry(output, "missing-key")
	assert.Empty(t, result)
}

func TestGetStringEntry_NilValue(t *testing.T) {
	output := &apiv2beta1.PluginOutput{
		Entries: map[string]*apiv2beta1.MetadataValue{
			"key1": nil,
		},
	}
	result := GetStringEntry(output, "key1")
	assert.Empty(t, result)
}

func TestGetStringEntry_Success(t *testing.T) {
	output := &apiv2beta1.PluginOutput{
		Entries: map[string]*apiv2beta1.MetadataValue{
			"key1": {Value: structpb.NewStringValue("expected-value")},
		},
	}
	result := GetStringEntry(output, "key1")
	assert.Equal(t, "expected-value", result)
}

func TestGetParentRunID_NilOutput(t *testing.T) {
	result := GetParentRunID(nil)
	assert.Empty(t, result)
}

func TestGetParentRunID_MissingEntry(t *testing.T) {
	output := &apiv2beta1.PluginOutput{
		Entries: map[string]*apiv2beta1.MetadataValue{
			"other-key": {Value: structpb.NewStringValue("value")},
		},
	}
	result := GetParentRunID(output)
	assert.Empty(t, result)
}

func TestGetParentRunID_Success(t *testing.T) {
	output := &apiv2beta1.PluginOutput{
		Entries: map[string]*apiv2beta1.MetadataValue{
			EntryRootRunID: {Value: structpb.NewStringValue("parent-run-123")},
		},
	}
	result := GetParentRunID(output)
	assert.Equal(t, "parent-run-123", result)
}

// Mock store for testing PersistPluginsOutput
type mockRunPluginOutputStore struct {
	updateCalled  bool
	runID         string
	pluginsOutput *model.LargeText
	updateError   error
}

func (m *mockRunPluginOutputStore) UpdateRunPluginsOutput(runID string, output *model.LargeText) error {
	m.updateCalled = true
	m.runID = runID
	m.pluginsOutput = output
	return m.updateError
}

func TestPersistPluginsOutput_Success(t *testing.T) {
	run := &PersistedRun{
		RunID: "run-123",
		PluginsOutput: map[string]*apiv2beta1.PluginOutput{
			"test-plugin": {
				State: apiv2beta1.PluginState_PLUGIN_SUCCEEDED,
				Entries: map[string]*apiv2beta1.MetadataValue{
					"key1": {Value: structpb.NewStringValue("value1")},
				},
			},
		},
	}

	store := &mockRunPluginOutputStore{}
	err := PersistPluginsOutput(run, store)
	require.NoError(t, err)

	assert.True(t, store.updateCalled)
	assert.Equal(t, "run-123", store.runID)
	require.NotNil(t, store.pluginsOutput)

	// Verify the serialized output
	deserialized, err := DeserializePluginsOutput(store.pluginsOutput)
	require.NoError(t, err)
	assert.Len(t, deserialized, 1)
	assert.Equal(t, apiv2beta1.PluginState_PLUGIN_SUCCEEDED, deserialized["test-plugin"].State)
}

func TestPersistPluginsOutput_EmptyPluginsOutput(t *testing.T) {
	run := &PersistedRun{
		RunID:         "run-123",
		PluginsOutput: map[string]*apiv2beta1.PluginOutput{},
	}

	store := &mockRunPluginOutputStore{}
	err := PersistPluginsOutput(run, store)
	require.NoError(t, err)

	assert.True(t, store.updateCalled)
	assert.Nil(t, store.pluginsOutput)
}

func TestPersistPluginsOutput_StoreError(t *testing.T) {
	run := &PersistedRun{
		RunID: "run-123",
		PluginsOutput: map[string]*apiv2beta1.PluginOutput{
			"test-plugin": {State: apiv2beta1.PluginState_PLUGIN_SUCCEEDED},
		},
	}

	store := &mockRunPluginOutputStore{
		updateError: assert.AnError,
	}
	err := PersistPluginsOutput(run, store)
	require.Error(t, err)
	assert.Equal(t, assert.AnError, err)
}
