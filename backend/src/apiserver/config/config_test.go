// Copyright 2025 The Kubeflow Authors
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

package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/kubeflow/pipelines/backend/src/apiserver/list"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
)

func fakeResourceManager() *resource.ResourceManager {
	clientManager := resource.NewFakeClientManagerOrFatalV2()
	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})
	return resourceManager
}

func TestLoadSamplesConfigBackwardsCompatibility(t *testing.T) {
	viper.Set("POD_NAMESPACE", "Test")
	rm := fakeResourceManager()
	pc := []deprecatedConfig{
		{
			Name:        "Pipeline 1",
			Description: "test description",
			File:        "testdata/sample_pipeline.yaml",
		},
		{
			Name:        "Pipeline 2",
			Description: "test description",
			File:        "testdata/sample_pipeline.yaml",
		},
	}

	path, err := writeSampleConfigDeprecated(t, pc, "sample.json")
	require.NoError(t, err)
	err = LoadSamples(rm, path, "")
	require.NoError(t, err)

	_, err = rm.GetPipelineByNameAndNamespace(pc[0].Name, "")
	require.NoError(t, err)
	_, err = rm.GetPipelineByNameAndNamespace(pc[1].Name, "")
	require.NoError(t, err)

	_, err = rm.GetPipelineVersionByName(pc[0].Name)
	require.NoError(t, err)
	_, err = rm.GetPipelineVersionByName(pc[1].Name)
	require.NoError(t, err)

	// Update pipeline version for Pipeline 1
	pc[0].Name = "Pipeline 3"
	path, err = writeSampleConfigDeprecated(t, pc, "sample.json")
	require.NoError(t, err)

	// Loading samples should result in no pipeline uploaded
	err = LoadSamples(rm, path, "")
	require.NoError(t, err)
	_, err = rm.GetPipelineByNameAndNamespace(pc[0].Name, "")
	var userErr *util.UserError
	if assert.ErrorAs(t, err, &userErr) {
		require.Equal(t, codes.NotFound, userErr.ExternalStatusCode())
	}
}

func TestLoadSamples(t *testing.T) {
	viper.Set("POD_NAMESPACE", "")
	rm := fakeResourceManager()
	pc := config{
		LoadSamplesOnRestart: true,
		Pipelines: []configPipelines{
			{
				Name:               "Pipeline 1",
				Description:        "test description",
				File:               "testdata/sample_pipeline.yaml",
				VersionName:        "Pipeline 1 - Ver 1",
				VersionDescription: "Pipeline 1 - Ver 1 Description",
			},
			{
				Name:               "Pipeline 2",
				Description:        "test description",
				File:               "testdata/sample_pipeline.yaml",
				VersionName:        "Pipeline 2 - Ver 1",
				VersionDescription: "Pipeline 2 - Ver 1 Description",
			},
			// Test case: Pipeline with display name but no version display name
			{
				Name:               "Pipeline 3",
				DisplayName:        "Display Pipeline 3",
				Description:        "test description",
				File:               "testdata/sample_pipeline.yaml",
				VersionName:        "Pipeline 3 - Ver 1",
				VersionDescription: "Pipeline 3 - Ver 1 Description",
			},
			// Test case: Pipeline with version display name but no pipeline display name
			{
				Name:               "Pipeline 4",
				Description:        "test description",
				File:               "testdata/sample_pipeline.yaml",
				VersionName:        "Pipeline 4 - Ver 1",
				VersionDisplayName: "Display Pipeline 4 - Version 1",
				VersionDescription: "Pipeline 4 - Ver 1 Description",
			},
			// Test case: Pipeline with both display names
			{
				Name:               "Pipeline 5",
				DisplayName:        "Display Pipeline 5",
				Description:        "test description",
				File:               "testdata/sample_pipeline.yaml",
				VersionName:        "Pipeline 5 - Ver 1",
				VersionDisplayName: "Display Pipeline 5 - Version 1",
				VersionDescription: "Pipeline 5 - Ver 1 Description",
			},
			// Test case: Pipeline with only pipeline name, no version name or display names
			{
				Name:               "Pipeline 6",
				Description:        "test description",
				File:               "testdata/sample_pipeline.yaml",
				VersionDescription: "Pipeline 6 - Ver 1 Description",
			},
		},
	}

	path, err := writeSampleConfig(t, pc, "sample.json")
	require.NoError(t, err)
	err = LoadSamples(rm, path, "")
	require.NoError(t, err)

	var pipeline1 *model.Pipeline
	pipeline1, err = rm.GetPipelineByNameAndNamespace(pc.Pipelines[0].Name, "")
	require.NoError(t, err)
	version1, err := rm.GetPipelineVersionByName(pc.Pipelines[0].VersionName)
	require.NoError(t, err)
	// Verify that the pipeline display name is set to the pipeline name if not provided
	assert.Equal(t, pipeline1.DisplayName, pc.Pipelines[0].Name)
	assert.Equal(t, version1.DisplayName, pc.Pipelines[0].VersionName)
	var pipeline2 *model.Pipeline
	pipeline2, err = rm.GetPipelineByNameAndNamespace(pc.Pipelines[1].Name, "")
	require.NoError(t, err)
	version2, err := rm.GetPipelineVersionByName(pc.Pipelines[1].VersionName)
	// Verify that the pipeline display name is set to the pipeline name if not provided
	assert.Equal(t, pipeline2.DisplayName, pc.Pipelines[1].Name)
	assert.Equal(t, version2.DisplayName, pc.Pipelines[1].VersionName)
	require.NoError(t, err)

	// Test display name for Pipeline 3 (pipeline display name only)
	pipeline3, err := rm.GetPipelineByNameAndNamespace(pc.Pipelines[2].Name, "")
	require.NoError(t, err)
	assert.Equal(t, pc.Pipelines[2].DisplayName, pipeline3.DisplayName)
	version3, err := rm.GetPipelineVersionByName(pc.Pipelines[2].VersionName)
	require.NoError(t, err)
	assert.Equal(t, pc.Pipelines[2].VersionName, version3.DisplayName) // Should use version name when no version display name is provided

	// Test display name for Pipeline 4 (version display name only)
	pipeline4, err := rm.GetPipelineByNameAndNamespace(pc.Pipelines[3].Name, "")
	require.NoError(t, err)
	assert.Equal(t, pc.Pipelines[3].Name, pipeline4.DisplayName) // Should use pipeline name
	version4, err := rm.GetPipelineVersionByName(pc.Pipelines[3].VersionName)
	require.NoError(t, err)
	assert.Equal(t, pc.Pipelines[3].VersionDisplayName, version4.DisplayName)

	// Test display name for Pipeline 5 (both display names)
	pipeline5, err := rm.GetPipelineByNameAndNamespace(pc.Pipelines[4].Name, "")
	require.NoError(t, err)
	assert.Equal(t, pc.Pipelines[4].DisplayName, pipeline5.DisplayName)
	version5, err := rm.GetPipelineVersionByName(pc.Pipelines[4].VersionName)
	require.NoError(t, err)
	assert.Equal(t, pc.Pipelines[4].VersionDisplayName, version5.DisplayName)

	// Test display name for Pipeline 6 (only pipeline name)
	pipeline6, err := rm.GetPipelineByNameAndNamespace(pc.Pipelines[5].Name, "")
	require.NoError(t, err)
	assert.Equal(t, pc.Pipelines[5].Name, pipeline6.DisplayName)       // Should use pipeline name
	version6, err := rm.GetPipelineVersionByName(pc.Pipelines[5].Name) // Version name should default to pipeline name
	require.NoError(t, err)
	assert.Equal(t, pc.Pipelines[5].Name, version6.DisplayName) // Version display name should default to pipeline name

	_, err = rm.GetPipelineVersionByName(pc.Pipelines[0].VersionName)
	require.NoError(t, err)

	// Update pipeline version for Pipeline 1
	pc.Pipelines[0].VersionName = "Pipeline 1 - Ver 2"
	path, err = writeSampleConfig(t, pc, "sample.json")
	require.NoError(t, err)
	err = LoadSamples(rm, path, "")
	require.NoError(t, err)

	// Expect another Pipeline version added for Pipeline 1
	opts, err := list.NewOptions(&model.PipelineVersion{}, 10, "id", nil)
	require.NoError(t, err)
	_, totalSize, _, err := rm.ListPipelineVersions(pipeline1.UUID, opts, nil)
	require.NoError(t, err)
	require.Equal(t, totalSize, 2)

	// Update pipeline version for Pipeline 2
	pc.Pipelines[1].VersionName = "Pipeline 2 - Ver 2"
	path, err = writeSampleConfig(t, pc, "sample.json")
	require.NoError(t, err)
	err = LoadSamples(rm, path, "")
	require.NoError(t, err)

	// Expect another Pipeline version added for Pipeline 2
	_, err = rm.GetPipelineVersionByName(pc.Pipelines[1].VersionName)
	require.NoError(t, err)
	_, totalSize, _, err = rm.ListPipelineVersions(pipeline2.UUID, opts, nil)
	require.NoError(t, err)
	require.Equal(t, totalSize, 2)

	// Confirm previous pipeline version count has not been affected
	_, totalSize, _, err = rm.ListPipelineVersions(pipeline1.UUID, opts, nil)
	require.NoError(t, err)
	require.Equal(t, totalSize, 2)

	// When LoadSamplesOnRestart is false, changes to config should
	// result in no new pipelines update upon restart
	pc.LoadSamplesOnRestart = false

	// Update pipeline version for Pipeline 2
	pc.Pipelines[1].VersionName = "Pipeline 2 - Ver 3"
	path, err = writeSampleConfig(t, pc, "sample.json")
	require.NoError(t, err)
	err = LoadSamples(rm, path, "")
	require.NoError(t, err)

	// Expect no change
	_, totalSize, _, err = rm.ListPipelineVersions(pipeline2.UUID, opts, nil)
	require.NoError(t, err)
	require.Equal(t, totalSize, 2)
}

func TestLoadSamplesMultiplePipelineVersionsInConfig(t *testing.T) {
	rm := fakeResourceManager()
	pc := config{
		LoadSamplesOnRestart: true,
		Pipelines: []configPipelines{
			{
				Name:               "Pipeline 1",
				Description:        "test description",
				File:               "testdata/sample_pipeline.yaml",
				VersionName:        "Pipeline 1 - Ver 1",
				VersionDescription: "Pipeline 1 - Ver 1 Description",
			},
			{
				Name:               "Pipeline 1",
				Description:        "test description",
				File:               "testdata/sample_pipeline.yaml",
				VersionName:        "Pipeline 1 - Ver 2",
				VersionDescription: "Pipeline 1 - Ver 2 Description",
			},
		},
	}

	path, err := writeSampleConfig(t, pc, "sample.json")
	require.NoError(t, err)
	err = LoadSamples(rm, path, "")
	require.NoError(t, err)

	// Expect both versions to be added
	var pipeline *model.Pipeline
	pipeline, err = rm.GetPipelineByNameAndNamespace(pc.Pipelines[0].Name, "")
	require.NoError(t, err)

	_, err = rm.GetPipelineVersionByName(pc.Pipelines[0].VersionName)
	require.NoError(t, err)
	_, err = rm.GetPipelineVersionByName(pc.Pipelines[1].VersionName)
	require.NoError(t, err)

	opts, err := list.NewOptions(&model.PipelineVersion{}, 10, "id", nil)
	require.NoError(t, err)

	_, totalSize, _, err := rm.ListPipelineVersions(pipeline.UUID, opts, nil)
	require.NoError(t, err)
	require.Equal(t, totalSize, 2)
}

func TestParseManagedPipelinesTags(t *testing.T) {
	tests := []struct {
		name    string
		envVal  string
		want    map[string]string
		wantErr bool
	}{
		{
			name:   "multiple tags",
			envVal: "managed=true,platform-version=v2.18.0",
			want:   map[string]string{"managed": "true", "platform-version": "v2.18.0"},
		},
		{
			name:   "single tag",
			envVal: "managed=true",
			want:   map[string]string{"managed": "true"},
		},
		{
			name:   "empty string returns nil",
			envVal: "",
			want:   nil,
		},
		{
			name:   "value containing equals splits on first only",
			envVal: "key=val=ue",
			want:   map[string]string{"key": "val=ue"},
		},
		{
			name:    "missing equals is malformed",
			envVal:  "badentry",
			wantErr: true,
		},
		{
			name:    "empty key is malformed",
			envVal:  "=value",
			wantErr: true,
		},
		{
			name:   "whitespace in key and value is preserved",
			envVal: " key = value ",
			want:   map[string]string{" key ": " value "},
		},
		{
			name:    "trailing comma produces empty entry",
			envVal:  "managed=true,",
			wantErr: true,
		},
		{
			name:   "empty value is valid",
			envVal: "key=",
			want:   map[string]string{"key": ""},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(managedPipelinesUploadTagsEnv, tt.envVal)
			got, err := parseManagedPipelinesTags()
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestLoadSamples_TagsApplied(t *testing.T) {
	viper.Set("POD_NAMESPACE", "")
	t.Setenv(managedPipelinesUploadTagsEnv, "managed=true,platform-version=v2.18.0")
	wantTags := map[string]string{"managed": "true", "platform-version": "v2.18.0"}

	rm := fakeResourceManager()
	pc := config{
		LoadSamplesOnRestart: true,
		Pipelines: []configPipelines{
			{
				Name:        "Tagged Pipeline",
				Description: "test",
				File:        "testdata/sample_pipeline.yaml",
				VersionName: "Tagged Pipeline - Ver 1",
			},
		},
	}

	path, err := writeSampleConfig(t, pc, "sample.json")
	require.NoError(t, err)
	require.NoError(t, LoadSamples(rm, path, ""))

	pipeline, err := rm.GetPipelineByNameAndNamespace("Tagged Pipeline", "")
	require.NoError(t, err)
	assert.Equal(t, wantTags, pipeline.Tags)

	version, err := rm.GetPipelineVersionByName("Tagged Pipeline - Ver 1")
	require.NoError(t, err)
	assert.Empty(t, version.Tags)
}

func TestLoadSamples_NoTagsWhenEnvUnset(t *testing.T) {
	viper.Set("POD_NAMESPACE", "")
	t.Setenv(managedPipelinesUploadTagsEnv, "")

	rm := fakeResourceManager()
	pc := config{
		LoadSamplesOnRestart: true,
		Pipelines: []configPipelines{
			{
				Name:        "Untagged Pipeline",
				Description: "test",
				File:        "testdata/sample_pipeline.yaml",
				VersionName: "Untagged Pipeline - Ver 1",
			},
		},
	}

	path, err := writeSampleConfig(t, pc, "sample.json")
	require.NoError(t, err)
	require.NoError(t, LoadSamples(rm, path, ""))

	pipeline, err := rm.GetPipelineByNameAndNamespace("Untagged Pipeline", "")
	require.NoError(t, err)
	assert.Empty(t, pipeline.Tags)

	version, err := rm.GetPipelineVersionByName("Untagged Pipeline - Ver 1")
	require.NoError(t, err)
	assert.Empty(t, version.Tags)
}

func TestLoadSamples_MalformedTagsReturnsError(t *testing.T) {
	viper.Set("POD_NAMESPACE", "")
	t.Setenv(managedPipelinesUploadTagsEnv, "badentry")

	rm := fakeResourceManager()
	pc := config{
		LoadSamplesOnRestart: true,
		Pipelines: []configPipelines{
			{
				Name:        "Should Not Load",
				Description: "test",
				File:        "testdata/sample_pipeline.yaml",
				VersionName: "Should Not Load - Ver 1",
			},
		},
	}

	path, err := writeSampleConfig(t, pc, "sample.json")
	require.NoError(t, err)
	err = LoadSamples(rm, path, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "badentry")
}

func TestLoadSamples_MalformedTagsIgnoredWhenLoadingSkipped(t *testing.T) {
	viper.Set("POD_NAMESPACE", "")
	rm := fakeResourceManager()

	pc := config{
		LoadSamplesOnRestart: false,
		Pipelines: []configPipelines{
			{
				Name:        "Already Loaded Pipeline",
				Description: "test",
				File:        "testdata/sample_pipeline.yaml",
				VersionName: "Already Loaded Pipeline - Ver 1",
			},
		},
	}

	// First load succeeds (no malformed tags, samples not yet loaded).
	t.Setenv(managedPipelinesUploadTagsEnv, "")
	path, err := writeSampleConfig(t, pc, "sample.json")
	require.NoError(t, err)
	require.NoError(t, LoadSamples(rm, path, ""))

	// Second load: samples already loaded + LoadSamplesOnRestart=false → skip.
	// A malformed env var must NOT cause an error because loading is skipped.
	t.Setenv(managedPipelinesUploadTagsEnv, "badentry")
	path, err = writeSampleConfig(t, pc, "sample.json")
	require.NoError(t, err)
	require.NoError(t, LoadSamples(rm, path, ""))
}

func TestLoadSamples_ExistingPipelineNotRetagged(t *testing.T) {
	viper.Set("POD_NAMESPACE", "")
	rm := fakeResourceManager()

	// First load: create pipeline + version without tags
	t.Setenv(managedPipelinesUploadTagsEnv, "")
	pc := config{
		LoadSamplesOnRestart: true,
		Pipelines: []configPipelines{
			{
				Name:        "Existing Pipeline",
				Description: "test",
				File:        "testdata/sample_pipeline.yaml",
				VersionName: "Existing Pipeline - Ver 1",
			},
		},
	}
	path, err := writeSampleConfig(t, pc, "sample.json")
	require.NoError(t, err)
	require.NoError(t, LoadSamples(rm, path, ""))

	// Second load: add a new version with tags enabled
	t.Setenv(managedPipelinesUploadTagsEnv, "managed=true")
	pc.Pipelines[0].VersionName = "Existing Pipeline - Ver 2"
	path, err = writeSampleConfig(t, pc, "sample.json")
	require.NoError(t, err)
	require.NoError(t, LoadSamples(rm, path, ""))

	// Existing pipeline should NOT have been re-tagged
	pipeline, err := rm.GetPipelineByNameAndNamespace("Existing Pipeline", "")
	require.NoError(t, err)
	assert.Empty(t, pipeline.Tags)
}

func writeSampleConfig(t *testing.T, config config, path string) (string, error) {
	return writeContents(t, config, path)
}

func writeSampleConfigDeprecated(t *testing.T, config []deprecatedConfig, path string) (string, error) {
	return writeContents(t, config, path)
}

func writeContents(t *testing.T, content interface{}, path string) (string, error) {
	tempDir := t.TempDir()
	sampleFilePath := filepath.Join(tempDir, path)
	marshal, err := json.Marshal(content)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(sampleFilePath, marshal, 0644); err != nil {
		t.Fatalf("Failed to create %v file: %v", path, err)
	}
	return sampleFilePath, nil
}

func writeManagedPipelinesManifest(t *testing.T, dir string, entries []managedPipelineManifestEntry) {
	t.Helper()
	data, err := json.Marshal(entries)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "managed-pipelines.json"), data, 0644))
}

func TestLoadManagedPipelinesManifest_HappyPath(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "pipeline-a.yaml"), []byte("apiVersion: v2"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "pipeline-b.yaml"), []byte("apiVersion: v2"), 0644))

	entries := []managedPipelineManifestEntry{
		{Name: "pipeline-a", Description: "Pipeline A", Path: "pipelines/a/pipeline.py"},
		{Name: "pipeline-b", Description: "Pipeline B", Path: "pipelines/b/pipeline.py"},
	}
	writeManagedPipelinesManifest(t, dir, entries)

	got, err := loadManagedPipelinesManifest(filepath.Join(dir, "managed-pipelines.json"), nil)
	require.NoError(t, err)
	require.Len(t, got, 2)

	resolvedDir, err := filepath.EvalSymlinks(dir)
	require.NoError(t, err)

	assert.Equal(t, "pipeline-a", got[0].Name)
	assert.Equal(t, "Pipeline A", got[0].Description)
	assert.Equal(t, filepath.Join(resolvedDir, "pipeline-a.yaml"), got[0].File)
	assert.Equal(t, "pipeline-b", got[1].Name)
	assert.Equal(t, "Pipeline B", got[1].Description)
	assert.Equal(t, filepath.Join(resolvedDir, "pipeline-b.yaml"), got[1].File)
}

func TestLoadManagedPipelinesManifest_DirNotExist(t *testing.T) {
	got, err := loadManagedPipelinesManifest("/nonexistent/managed-pipelines.json", nil)
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestLoadManagedPipelinesManifest_SkipDuplicates(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "new-pipeline.yaml"), []byte("apiVersion: v2"), 0644))

	entries := []managedPipelineManifestEntry{
		{Name: "existing-pipeline", Description: "Already exists", Path: "pipelines/existing/pipeline.py"},
		{Name: "new-pipeline", Description: "New one", Path: "pipelines/new/pipeline.py"},
	}
	writeManagedPipelinesManifest(t, dir, entries)

	existing := map[string]bool{"existing-pipeline": true}
	got, err := loadManagedPipelinesManifest(filepath.Join(dir, "managed-pipelines.json"), existing)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "new-pipeline", got[0].Name)
}

func TestLoadManagedPipelinesManifest_EmptyManifest(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "managed-pipelines.json"), []byte("[]"), 0644))

	got, err := loadManagedPipelinesManifest(filepath.Join(dir, "managed-pipelines.json"), nil)
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestLoadManagedPipelinesManifest_EmptyNameRejected(t *testing.T) {
	dir := t.TempDir()
	entries := []managedPipelineManifestEntry{
		{Name: "", Description: "No name", Path: "no-name.yaml"},
	}
	writeManagedPipelinesManifest(t, dir, entries)

	_, err := loadManagedPipelinesManifest(filepath.Join(dir, "managed-pipelines.json"), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty name")
}

func TestLoadManagedPipelinesManifest_EmptyPathRejected(t *testing.T) {
	dir := t.TempDir()
	entries := []managedPipelineManifestEntry{
		{Name: "has-name", Description: "No path", Path: ""},
	}
	writeManagedPipelinesManifest(t, dir, entries)

	_, err := loadManagedPipelinesManifest(filepath.Join(dir, "managed-pipelines.json"), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty path")
}

func TestLoadManagedPipelinesManifest_MalformedJSON(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "managed-pipelines.json"), []byte("{bad json"), 0644))

	_, err := loadManagedPipelinesManifest(filepath.Join(dir, "managed-pipelines.json"), nil)
	require.Error(t, err)
}

func TestLoadManagedPipelinesManifest_DangerousPathFieldIgnored(t *testing.T) {
	cases := []struct {
		name      string
		entryName string
		pathField string
	}{
		{"traversal path", "safe-name", "../escape.yaml"},
		{"absolute path", "absolute", "/etc/passwd"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			require.NoError(t, os.WriteFile(filepath.Join(dir, tc.entryName+".yaml"), []byte("apiVersion: v2"), 0644))

			entries := []managedPipelineManifestEntry{
				{Name: tc.entryName, Description: "Path field ignored", Path: tc.pathField},
			}
			writeManagedPipelinesManifest(t, dir, entries)

			got, err := loadManagedPipelinesManifest(filepath.Join(dir, "managed-pipelines.json"), nil)
			require.NoError(t, err, "Path field is metadata-only; dangerous values do not affect file resolution")
			require.Len(t, got, 1)
			assert.Equal(t, tc.entryName+".yaml", filepath.Base(got[0].File))
		})
	}
}

func TestLoadManagedPipelinesManifest_DuplicateNamesRejected(t *testing.T) {
	dir := t.TempDir()
	entries := []managedPipelineManifestEntry{
		{Name: "dup", Description: "first", Path: "a.yaml"},
		{Name: "dup", Description: "second", Path: "b.yaml"},
	}
	writeManagedPipelinesManifest(t, dir, entries)

	_, err := loadManagedPipelinesManifest(filepath.Join(dir, "managed-pipelines.json"), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate name")
}

func TestLoadManagedPipelinesManifest_SymlinkEscapeRejected(t *testing.T) {
	dir := t.TempDir()
	outside := filepath.Join(t.TempDir(), "escape.yaml")
	require.NoError(t, os.WriteFile(outside, []byte("apiVersion: v2"), 0644))
	require.NoError(t, os.Symlink(outside, filepath.Join(dir, "symlink-escape.yaml")))

	entries := []managedPipelineManifestEntry{
		{Name: "symlink-escape", Description: "escape", Path: "pipelines/x/pipeline.py"},
	}
	writeManagedPipelinesManifest(t, dir, entries)

	_, err := loadManagedPipelinesManifest(filepath.Join(dir, "managed-pipelines.json"), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "escapes directory")
}

func TestLoadSamples_MergesManagedManifest(t *testing.T) {
	viper.Set("POD_NAMESPACE", "")
	t.Setenv(managedPipelinesUploadTagsEnv, "")
	rm := fakeResourceManager()

	pc := config{
		LoadSamplesOnRestart: true,
		Pipelines: []configPipelines{
			{
				Name:        "Explicit Pipeline",
				Description: "from sample_config.json",
				File:        "testdata/sample_pipeline.yaml",
				VersionName: "Explicit Pipeline - Ver 1",
			},
		},
	}
	samplePath, err := writeSampleConfig(t, pc, "sample.json")
	require.NoError(t, err)

	managedDir := t.TempDir()
	entries := []managedPipelineManifestEntry{
		{Name: "managed-a", Description: "Managed A", Path: "managed-a.yaml"},
		{Name: "managed-b", Description: "Managed B", Path: "managed-b.yaml"},
	}
	writeManagedPipelinesManifest(t, managedDir, entries)
	sampleYAML, err := os.ReadFile("testdata/sample_pipeline.yaml")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(managedDir, "managed-a.yaml"), sampleYAML, 0644))
	require.NoError(t, os.WriteFile(filepath.Join(managedDir, "managed-b.yaml"), sampleYAML, 0644))

	require.NoError(t, LoadSamples(rm, samplePath, managedDir))

	_, err = rm.GetPipelineByNameAndNamespace("Explicit Pipeline", "")
	require.NoError(t, err)
	_, err = rm.GetPipelineByNameAndNamespace("managed-a", "")
	require.NoError(t, err)
	_, err = rm.GetPipelineByNameAndNamespace("managed-b", "")
	require.NoError(t, err)

	_, err = rm.GetPipelineVersionByName("Explicit Pipeline - Ver 1")
	require.NoError(t, err)
	_, err = rm.GetPipelineVersionByName("managed-a")
	require.NoError(t, err)
	_, err = rm.GetPipelineVersionByName("managed-b")
	require.NoError(t, err)
}

func TestLoadSamples_TagsAppliedToManagedPipelines(t *testing.T) {
	viper.Set("POD_NAMESPACE", "")
	t.Setenv(managedPipelinesUploadTagsEnv, "managed=true")
	rm := fakeResourceManager()

	pc := config{
		LoadSamplesOnRestart: true,
		Pipelines:            []configPipelines{},
	}
	samplePath, err := writeSampleConfig(t, pc, "sample.json")
	require.NoError(t, err)

	managedDir := t.TempDir()
	entries := []managedPipelineManifestEntry{
		{Name: "tagged-managed", Description: "Tagged", Path: "tagged-managed.yaml"},
	}
	writeManagedPipelinesManifest(t, managedDir, entries)
	sampleYAML, err := os.ReadFile("testdata/sample_pipeline.yaml")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(managedDir, "tagged-managed.yaml"), sampleYAML, 0644))

	require.NoError(t, LoadSamples(rm, samplePath, managedDir))

	pipeline, err := rm.GetPipelineByNameAndNamespace("tagged-managed", "")
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"managed": "true"}, pipeline.Tags)
}

func TestLoadSamples_ManagedDedupAgainstExplicit(t *testing.T) {
	viper.Set("POD_NAMESPACE", "")
	t.Setenv(managedPipelinesUploadTagsEnv, "")
	rm := fakeResourceManager()

	pc := config{
		LoadSamplesOnRestart: true,
		Pipelines: []configPipelines{
			{
				Name:        "shared-name",
				Description: "explicit version",
				File:        "testdata/sample_pipeline.yaml",
				VersionName: "shared-name - Ver 1",
			},
		},
	}
	samplePath, err := writeSampleConfig(t, pc, "sample.json")
	require.NoError(t, err)

	managedDir := t.TempDir()
	entries := []managedPipelineManifestEntry{
		{Name: "shared-name", Description: "managed version", Path: "shared-name.yaml"},
		{Name: "unique-managed", Description: "only in manifest", Path: "unique-managed.yaml"},
	}
	writeManagedPipelinesManifest(t, managedDir, entries)
	sampleYAML, err := os.ReadFile("testdata/sample_pipeline.yaml")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(managedDir, "shared-name.yaml"), sampleYAML, 0644))
	require.NoError(t, os.WriteFile(filepath.Join(managedDir, "unique-managed.yaml"), sampleYAML, 0644))

	require.NoError(t, LoadSamples(rm, samplePath, managedDir))

	pipeline, err := rm.GetPipelineByNameAndNamespace("shared-name", "")
	require.NoError(t, err)
	opts, err := list.NewOptions(&model.PipelineVersion{}, 10, "id", nil)
	require.NoError(t, err)
	_, totalSize, _, err := rm.ListPipelineVersions(pipeline.UUID, opts, nil)
	require.NoError(t, err)
	require.Equal(t, 1, totalSize)

	_, err = rm.GetPipelineByNameAndNamespace("unique-managed", "")
	require.NoError(t, err)
}

func TestLoadManagedPipelinesManifest_AllDuplicates(t *testing.T) {
	dir := t.TempDir()
	entries := []managedPipelineManifestEntry{
		{Name: "dup-a", Description: "Dup A", Path: "dup-a.yaml"},
		{Name: "dup-b", Description: "Dup B", Path: "dup-b.yaml"},
	}
	writeManagedPipelinesManifest(t, dir, entries)

	existing := map[string]bool{"dup-a": true, "dup-b": true}
	got, err := loadManagedPipelinesManifest(filepath.Join(dir, "managed-pipelines.json"), existing)
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestLoadSamples_ManagedSkippedWhenLoadSamplesOnRestartFalse(t *testing.T) {
	viper.Set("POD_NAMESPACE", "")
	t.Setenv(managedPipelinesUploadTagsEnv, "")
	rm := fakeResourceManager()

	managedDir := t.TempDir()
	entries := []managedPipelineManifestEntry{
		{Name: "managed-restart", Description: "Managed", Path: "managed-restart.yaml"},
	}
	writeManagedPipelinesManifest(t, managedDir, entries)
	sampleYAML, err := os.ReadFile("testdata/sample_pipeline.yaml")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(managedDir, "managed-restart.yaml"), sampleYAML, 0644))

	pc := config{
		LoadSamplesOnRestart: false,
		Pipelines: []configPipelines{
			{
				Name:        "Initial Pipeline",
				Description: "test",
				File:        "testdata/sample_pipeline.yaml",
				VersionName: "Initial Pipeline - Ver 1",
			},
		},
	}

	path, err := writeSampleConfig(t, pc, "sample.json")
	require.NoError(t, err)
	require.NoError(t, LoadSamples(rm, path, managedDir))

	_, err = rm.GetPipelineByNameAndNamespace("Initial Pipeline", "")
	require.NoError(t, err)
	_, err = rm.GetPipelineByNameAndNamespace("managed-restart", "")
	require.NoError(t, err)

	pc.Pipelines = append(pc.Pipelines, configPipelines{
		Name:        "New Explicit",
		Description: "added after first load",
		File:        "testdata/sample_pipeline.yaml",
		VersionName: "New Explicit - Ver 1",
	})
	newManagedEntries := []managedPipelineManifestEntry{
		{Name: "managed-restart", Description: "Managed", Path: "managed-restart.yaml"},
		{Name: "managed-new", Description: "New managed", Path: "managed-new.yaml"},
	}
	writeManagedPipelinesManifest(t, managedDir, newManagedEntries)
	require.NoError(t, os.WriteFile(filepath.Join(managedDir, "managed-new.yaml"), sampleYAML, 0644))

	path, err = writeSampleConfig(t, pc, "sample.json")
	require.NoError(t, err)
	require.NoError(t, LoadSamples(rm, path, managedDir))

	_, err = rm.GetPipelineByNameAndNamespace("New Explicit", "")
	require.Error(t, err, "new explicit pipeline should not be loaded on restart with LoadSamplesOnRestart=false")
	_, err = rm.GetPipelineByNameAndNamespace("managed-new", "")
	require.Error(t, err, "new managed pipeline should not be loaded on restart with LoadSamplesOnRestart=false")
}

func TestLoadSamples_EmptyManagedDir(t *testing.T) {
	viper.Set("POD_NAMESPACE", "")
	t.Setenv(managedPipelinesUploadTagsEnv, "")
	rm := fakeResourceManager()

	pc := config{
		LoadSamplesOnRestart: true,
		Pipelines: []configPipelines{
			{
				Name:        "Solo Pipeline",
				Description: "test",
				File:        "testdata/sample_pipeline.yaml",
				VersionName: "Solo Pipeline - Ver 1",
			},
		},
	}
	path, err := writeSampleConfig(t, pc, "sample.json")
	require.NoError(t, err)

	require.NoError(t, LoadSamples(rm, path, ""))

	_, err = rm.GetPipelineByNameAndNamespace("Solo Pipeline", "")
	require.NoError(t, err)
}

// Contract 1: The pipelines-components init container copies compiled YAMLs as
// <name>.yaml (flat) and copies managed-pipelines.json unchanged. The manifest
// path field is a repo-relative source location, NOT a file path in the managed
// dir. The API server must locate files by <name>.yaml.
func TestLoadManagedPipelinesManifest_InitContainerContract(t *testing.T) {
	dir := t.TempDir()

	entries := []managedPipelineManifestEntry{
		{
			Name:        "my-training-pipeline",
			Description: "A training pipeline",
			Path:        "pipelines/training/pipeline.py",
		},
		{
			Name:        "my-evaluation-pipeline",
			Description: "An eval pipeline",
			Path:        "pipelines/evaluation/pipeline.py",
		},
	}
	writeManagedPipelinesManifest(t, dir, entries)

	require.NoError(t, os.WriteFile(filepath.Join(dir, "my-training-pipeline.yaml"), []byte("apiVersion: v2"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "my-evaluation-pipeline.yaml"), []byte("apiVersion: v2"), 0644))

	got, err := loadManagedPipelinesManifest(filepath.Join(dir, "managed-pipelines.json"), nil)
	require.NoError(t, err)
	require.Len(t, got, 2)

	assert.Equal(t, "my-training-pipeline", got[0].Name)
	assert.Equal(t, "A training pipeline", got[0].Description)
	assert.True(t, filepath.IsAbs(got[0].File), "File should be absolute")
	assert.Equal(t, "my-training-pipeline.yaml", filepath.Base(got[0].File))

	assert.Equal(t, "my-evaluation-pipeline", got[1].Name)
	assert.Equal(t, "An eval pipeline", got[1].Description)
	assert.True(t, filepath.IsAbs(got[1].File), "File should be absolute")
	assert.Equal(t, "my-evaluation-pipeline.yaml", filepath.Base(got[1].File))
}

// Contract 2: LoadSamples expects the manifest at <dir>/managed-pipelines.json.
func TestLoadSamples_ManifestFilename(t *testing.T) {
	viper.Set("POD_NAMESPACE", "")
	t.Setenv(managedPipelinesUploadTagsEnv, "")
	rm := fakeResourceManager()

	pc := config{LoadSamplesOnRestart: true, Pipelines: []configPipelines{}}
	samplePath, err := writeSampleConfig(t, pc, "sample.json")
	require.NoError(t, err)

	managedDir := t.TempDir()
	entries := []managedPipelineManifestEntry{
		{Name: "contract-pipe", Description: "d", Path: "pipelines/x/pipeline.py"},
	}
	writeManagedPipelinesManifest(t, managedDir, entries)
	sampleYAML, err := os.ReadFile("testdata/sample_pipeline.yaml")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(managedDir, "contract-pipe.yaml"), sampleYAML, 0644))

	require.NoError(t, LoadSamples(rm, samplePath, managedDir))

	_, err = rm.GetPipelineByNameAndNamespace("contract-pipe", "")
	require.NoError(t, err)

	wrongDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(wrongDir, "pipelines.json"), []byte("[]"), 0644))

	require.NoError(t, LoadSamples(rm, samplePath, wrongDir),
		"missing managed-pipelines.json is silently ignored (volume not mounted)")
}

// Contract 3: The manifest schema matches the Python ManagedPipelineEntry
// dataclass: name, description, path (all lowercase). Extra fields like
// "stability" are ignored by the Go consumer.
func TestLoadManagedPipelinesManifest_SchemaContract(t *testing.T) {
	dir := t.TempDir()

	rawJSON := `[{
		"name": "schema-test",
		"description": "Validates schema contract",
		"path": "pipelines/demo/pipeline.py"
	}]`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "managed-pipelines.json"), []byte(rawJSON), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "schema-test.yaml"), []byte("apiVersion: v2"), 0644))

	got, err := loadManagedPipelinesManifest(filepath.Join(dir, "managed-pipelines.json"), nil)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "schema-test", got[0].Name)
	assert.Equal(t, "Validates schema contract", got[0].Description)
}

func TestLoadManagedPipelinesManifest_InvalidNamesRejected(t *testing.T) {
	cases := []struct {
		name    string
		badName string
	}{
		{"traversal with slashes", "../../etc/passwd"},
		{"forward slash", "sub/dir"},
		{"backslash", "pipe\\line"},
		{"dot-dot without separator", "pipe..line"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			entries := []managedPipelineManifestEntry{
				{Name: tc.badName, Description: "bad name", Path: "p.py"},
			}
			writeManagedPipelinesManifest(t, dir, entries)

			_, err := loadManagedPipelinesManifest(filepath.Join(dir, "managed-pipelines.json"), nil)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "invalid character")
		})
	}
}
