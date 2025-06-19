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
	err = LoadSamples(rm, path)
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
	err = LoadSamples(rm, path)
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
	err = LoadSamples(rm, path)
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
	err = LoadSamples(rm, path)
	require.NoError(t, err)

	// Expect another Pipeline version added for Pipeline 1
	opts, err := list.NewOptions(&model.PipelineVersion{}, 10, "id", nil)
	require.NoError(t, err)
	_, totalSize, _, err := rm.ListPipelineVersions(pipeline1.UUID, opts)
	require.NoError(t, err)
	require.Equal(t, totalSize, 2)

	// Update pipeline version for Pipeline 2
	pc.Pipelines[1].VersionName = "Pipeline 2 - Ver 2"
	path, err = writeSampleConfig(t, pc, "sample.json")
	require.NoError(t, err)
	err = LoadSamples(rm, path)
	require.NoError(t, err)

	// Expect another Pipeline version added for Pipeline 2
	_, err = rm.GetPipelineVersionByName(pc.Pipelines[1].VersionName)
	require.NoError(t, err)
	_, totalSize, _, err = rm.ListPipelineVersions(pipeline2.UUID, opts)
	require.NoError(t, err)
	require.Equal(t, totalSize, 2)

	// Confirm previous pipeline version count has not been affected
	_, totalSize, _, err = rm.ListPipelineVersions(pipeline1.UUID, opts)
	require.NoError(t, err)
	require.Equal(t, totalSize, 2)

	// When LoadSamplesOnRestart is false, changes to config should
	// result in no new pipelines update upon restart
	pc.LoadSamplesOnRestart = false

	// Update pipeline version for Pipeline 2
	pc.Pipelines[1].VersionName = "Pipeline 2 - Ver 3"
	path, err = writeSampleConfig(t, pc, "sample.json")
	require.NoError(t, err)
	err = LoadSamples(rm, path)
	require.NoError(t, err)

	// Expect no change
	_, totalSize, _, err = rm.ListPipelineVersions(pipeline2.UUID, opts)
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
	err = LoadSamples(rm, path)
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

	_, totalSize, _, err := rm.ListPipelineVersions(pipeline.UUID, opts)
	require.Equal(t, totalSize, 2)
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
