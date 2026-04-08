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
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/backend/src/apiserver/client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/kubeflow/pipelines/backend/src/apiserver/server"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"google.golang.org/grpc/codes"
)

const managedPipelinesUploadTagsEnv = "MANAGED_PIPELINES_UPLOAD_TAGS"

// parseManagedPipelinesTags reads MANAGED_PIPELINES_UPLOAD_TAGS from the
// environment and returns the parsed key=value pairs. Returns nil when the
// variable is unset or empty (backward-compatible no-op).
func parseManagedPipelinesTags() (map[string]string, error) {
	raw := os.Getenv(managedPipelinesUploadTagsEnv)
	if raw == "" {
		return nil, nil
	}
	tags := make(map[string]string)
	for _, entry := range strings.Split(raw, ",") {
		idx := strings.Index(entry, "=")
		if idx < 0 {
			return nil, fmt.Errorf("malformed %s entry %q: missing '='", managedPipelinesUploadTagsEnv, entry)
		}
		key := entry[:idx]
		if key == "" {
			return nil, fmt.Errorf("malformed %s entry %q: empty key", managedPipelinesUploadTagsEnv, entry)
		}
		tags[key] = entry[idx+1:]
	}
	return tags, nil
}

// deprecated
type deprecatedConfig struct {
	Name        string
	Description string
	File        string
}

type configPipelines struct {
	Name string
	// optional, Name is used for Pipeline if not provided
	DisplayName string
	Description string
	File        string
	// optional, Name is used for PipelineVersion if not provided
	VersionName string
	// optional, VersionName, DisplayName, or Name is used for PipelineVersion if not provided
	VersionDisplayName string
	// optional, Description is used for PipelineVersion if not provided
	VersionDescription string
}

type config struct {
	// If pipeline version already exists and
	// LoadSamplesOnRestart is enabled, then the pipeline
	// version is uploaded again on server restart
	// if it does not already exist
	LoadSamplesOnRestart bool
	Pipelines            []configPipelines
}

type managedPipelineManifestEntry struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Path        string `json:"path"`
	Stability   string `json:"stability"`
}

// loadManagedPipelinesManifest reads a managed-pipelines.json manifest and
// returns configPipelines entries for any pipeline whose name is not already
// in the existing set. Returns nil without error when the manifest file does
// not exist (volume not mounted / flag not set).
func loadManagedPipelinesManifest(manifestPath string, existing map[string]bool) ([]configPipelines, error) {
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read managed pipelines manifest %s: %w", manifestPath, err)
	}

	var entries []managedPipelineManifestEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("failed to parse managed pipelines manifest %s: %w", manifestPath, err)
	}

	dir := filepath.Dir(manifestPath)
	resolvedDir, err := filepath.EvalSymlinks(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve managed pipelines directory %s: %w", dir, err)
	}

	seen := make(map[string]bool, len(entries))
	var pipelines []configPipelines
	for _, entry := range entries {
		if entry.Name == "" {
			return nil, fmt.Errorf("managed pipelines manifest %s contains entry with empty name", manifestPath)
		}
		if entry.Path == "" {
			return nil, fmt.Errorf("managed pipelines manifest %s contains entry %q with empty path", manifestPath, entry.Name)
		}
		if seen[entry.Name] {
			return nil, fmt.Errorf("managed pipelines manifest %s contains duplicate name %q", manifestPath, entry.Name)
		}
		seen[entry.Name] = true
		if strings.ContainsAny(entry.Name, "/\\") || strings.Contains(entry.Name, "..") {
			return nil, fmt.Errorf("managed pipeline name %q contains invalid character (/, \\, or ..)", entry.Name)
		}
		if existing[entry.Name] {
			glog.Infof("Skipping managed pipeline %q: already in sample config", entry.Name)
			continue
		}
		fileName := entry.Name + ".yaml"
		filePath := filepath.Join(resolvedDir, fileName)
		if !strings.HasPrefix(filePath, resolvedDir+string(filepath.Separator)) {
			return nil, fmt.Errorf("managed pipeline %q: constructed path escapes directory %q", entry.Name, resolvedDir)
		}
		resolvedPath, evalErr := filepath.EvalSymlinks(filePath)
		if evalErr != nil && !os.IsNotExist(evalErr) {
			return nil, fmt.Errorf("failed to resolve path for managed pipeline %q: %w", entry.Name, evalErr)
		}
		if evalErr == nil {
			rel, relErr := filepath.Rel(resolvedDir, resolvedPath)
			if relErr != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
				return nil, fmt.Errorf("managed pipeline %q: resolved path %q escapes directory %q", entry.Name, resolvedPath, resolvedDir)
			}
			filePath = resolvedPath
		}
		pipelines = append(pipelines, configPipelines{
			Name:        entry.Name,
			Description: entry.Description,
			File:        filePath,
		})
	}
	return pipelines, nil
}

// LoadSamples preloads a collection of pipeline samples
//
// If LoadSamplesOnRestart is false then Samples are only
// loaded once when the pipeline system is initially installed.
// They won't be loaded on upgrade or pod restart, to
// prevent them from reappearing if user explicitly deletes the
// samples. If LoadSamplesOnRestart is true then PipelineVersions
// are uploaded if they do not already exist upon upgrade or pod
// restart.
func LoadSamples(resourceManager *resource.ResourceManager, sampleConfigPath string, managedPipelinesDir string) error {
	pathExists, err := client.PathExists(sampleConfigPath)
	if err != nil {
		return err
	}

	if !pathExists {
		glog.Infof("No samples path provided, skipping loading samples..")
		return nil
	}

	configBytes, err := os.ReadFile(sampleConfigPath)
	if err != nil {
		return fmt.Errorf("failed to read sample configurations file. Err: %v", err)
	}

	var pipelineConfig config
	if configErr := json.Unmarshal(configBytes, &pipelineConfig); configErr != nil {
		// Attempt to parse to deprecated config version:
		var deprecatedCfg []deprecatedConfig
		if depConfigErr := json.Unmarshal(configBytes, &deprecatedCfg); depConfigErr != nil {
			return fmt.Errorf("failed to read sample configurations. Err: %v", configErr)
		}
		glog.Warningf("encountered deprecated version of samples config, please update to the newer version to " +
			"ensure future compatibility")
		for _, cfg := range deprecatedCfg {
			pipelineConfig.Pipelines = append(pipelineConfig.Pipelines, configPipelines{
				Name:        cfg.Name,
				DisplayName: cfg.Name,
				File:        cfg.File,
				Description: cfg.Description,
			})
		}
		pipelineConfig.LoadSamplesOnRestart = false
	}

	// Check if sample has been loaded already and skip loading if true.
	haveSamplesLoaded, err := resourceManager.HaveSamplesLoaded()
	if err != nil {
		return err
	}

	if !pipelineConfig.LoadSamplesOnRestart && haveSamplesLoaded {
		glog.Infof("Samples already loaded in the past. Skip loading.")
		return nil
	}

	if managedPipelinesDir != "" {
		existing := make(map[string]bool, len(pipelineConfig.Pipelines))
		for _, p := range pipelineConfig.Pipelines {
			existing[p.Name] = true
		}
		manifestPath := filepath.Join(managedPipelinesDir, "managed-pipelines.json")
		managedPipelines, mergeErr := loadManagedPipelinesManifest(manifestPath, existing)
		if mergeErr != nil {
			return mergeErr
		}
		pipelineConfig.Pipelines = append(pipelineConfig.Pipelines, managedPipelines...)
	}

	tags, err := parseManagedPipelinesTags()
	if err != nil {
		return err
	}
	if len(tags) > 0 {
		glog.Infof("Parsed %d managed pipeline upload tag(s) from %s", len(tags), managedPipelinesUploadTagsEnv)
	}

	processedPipelines := map[string]bool{}

	for _, cfg := range pipelineConfig.Pipelines {
		// Track if this is the first upload of this pipeline
		reader, configErr := os.Open(cfg.File)
		if configErr != nil {
			return fmt.Errorf("failed to load sample %s. Error: %v", cfg.Name, configErr)
		}
		pipelineFile, configErr := server.ReadPipelineFile(cfg.File, reader, common.MaxFileLength)
		if configErr != nil {
			return fmt.Errorf("failed to load sample %s. Error: %v", cfg.Name, configErr)
		}

		pipelineDisplayName := cfg.DisplayName
		if pipelineDisplayName == "" {
			pipelineDisplayName = cfg.Name
		}

		// Create pipeline if it does not already exist
		p, fetchErr := resourceManager.GetPipelineByNameAndNamespace(cfg.Name, common.GetPodNamespace())
		if fetchErr != nil {
			if util.IsUserErrorCodeMatch(fetchErr, codes.NotFound) {
				p, configErr = resourceManager.CreatePipeline(&model.Pipeline{
					Name:        cfg.Name,
					DisplayName: pipelineDisplayName,
					Description: model.LargeText(cfg.Description),
					Tags:        tags,
				})
				if configErr != nil {
					// Log the error but not fail. The API Server pod can restart and it could potentially cause
					// name collision. In the future, we might consider loading samples during deployment, instead
					// of when API server starts.
					glog.Warningf(fmt.Sprintf(
						"Failed to create pipeline for %s. Error: %v", cfg.Name, configErr))
					continue
				} else {
					glog.Info(fmt.Sprintf("Successfully uploaded Pipeline %s.", cfg.Name))
				}
			} else {
				return fmt.Errorf(
					"Failed to handle load sample for Pipeline: %s. Error: %v", cfg.Name, fetchErr)
			}
		}

		// Use Pipeline Version Name/Description if provided
		// Otherwise fallback to owning Pipeline's Name/Description
		pvDescription := cfg.Description
		if cfg.VersionDescription != "" {
			pvDescription = cfg.VersionDescription
		}
		pvName := cfg.Name
		if cfg.VersionName != "" {
			pvName = cfg.VersionName
		}

		// Use Pipeline Version Display Name if provided, if not use Pipeline Version Name, and if not provided use
		// Pipeline Display Name (which defaults to Pipeline Name).
		var pvDisplayName string
		if cfg.VersionDisplayName != "" {
			pvDisplayName = cfg.VersionDisplayName
		} else if cfg.VersionName != "" {
			pvDisplayName = cfg.VersionName
		} else {
			pvDisplayName = p.DisplayName
		}

		// If the Pipeline Version exists, do nothing
		// Otherwise upload new Pipeline Version for
		// this pipeline.
		_, fetchErr = resourceManager.GetPipelineVersionByName(pvName)
		if fetchErr != nil {
			if util.IsUserErrorCodeMatch(fetchErr, codes.NotFound) {
				_, configErr = resourceManager.CreatePipelineVersion(
					&model.PipelineVersion{
						Name:         pvName,
						DisplayName:  pvDisplayName,
						Description:  model.LargeText(pvDescription),
						PipelineId:   p.UUID,
						PipelineSpec: model.LargeText(string(pipelineFile)),
					},
				)
				if configErr != nil {
					// Log the error but not fail. The API Server pod can restart and it could potentially cause name collision.
					// In the future, we might consider loading samples during deployment, instead of when API server starts.
					glog.Warningf(fmt.Sprintf("Failed to create pipeline for %s. Error: %v", pvName, configErr))

					continue
				} else {
					glog.Info(fmt.Sprintf("Successfully uploaded PipelineVersion %s.", pvName))
				}

				if processedPipelines[pvName] {
					// Since the default sorting is by create time,
					// Sleep one second makes sure the samples are
					// showing up in the same order as they are added.
					time.Sleep(1 * time.Second)
				}
			} else {
				return fmt.Errorf(
					"Failed to handle load sample for PipelineVersion: %s. Error: %v", pvName, fetchErr)
			}
		} else {
			// pipeline version already exists, do nothing
			continue
		}

		processedPipelines[pvName] = true
	}

	if !haveSamplesLoaded {
		err = resourceManager.MarkSampleLoaded()
		if err != nil {
			return err
		}
	}

	glog.Info("All samples are loaded.")
	return nil
}
