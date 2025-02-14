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
	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/backend/src/apiserver/client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/kubeflow/pipelines/backend/src/apiserver/server"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"google.golang.org/grpc/codes"
	"os"
	"time"
)

// deprecated
type deprecatedConfig struct {
	Name        string
	Description string
	File        string
}

type configPipelines struct {
	Name        string
	Description string
	File        string
	// optional, Name is used for PipelineVersion if not provided
	VersionName string
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

// LoadSamples preloads a collection of pipeline samples
//
// If LoadSamplesOnRestart is false then Samples are only
// loaded once when the pipeline system is initially installed.
// They won't be loaded on upgrade or pod restart, to
// prevent them from reappearing if user explicitly deletes the
// samples. If LoadSamplesOnRestart is true then PipelineVersions
// are uploaded if they do not already exist upon upgrade or pod
// restart.
func LoadSamples(resourceManager *resource.ResourceManager, sampleConfigPath string) error {
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

		// Create pipeline if it does not already exist
		p, fetchErr := resourceManager.GetPipelineByNameAndNamespace(cfg.Name, "")
		if fetchErr != nil {
			if util.IsUserErrorCodeMatch(fetchErr, codes.NotFound) {
				p, configErr = resourceManager.CreatePipeline(&model.Pipeline{
					Name:        cfg.Name,
					Description: cfg.Description,
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

		// If the Pipeline Version exists, do nothing
		// Otherwise upload new Pipeline Version for
		// this pipeline.
		_, fetchErr = resourceManager.GetPipelineVersionByName(pvName)
		if fetchErr != nil {
			if util.IsUserErrorCodeMatch(fetchErr, codes.NotFound) {
				_, configErr = resourceManager.CreatePipelineVersion(
					&model.PipelineVersion{
						Name:         pvName,
						Description:  pvDescription,
						PipelineId:   p.UUID,
						PipelineSpec: string(pipelineFile),
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
