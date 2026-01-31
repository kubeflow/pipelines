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

// Package common provides shared utilities and configuration for the KFP API server.
package common

import (
	"strings"
	"sync"

	"github.com/golang/glog"
)

const (
	DriverPodLabels      = "DriverPodLabels"
	DriverPodAnnotations = "DriverPodAnnotations"

	// Reserved label prefix that will be filtered out to prevent
	// overriding system labels that control workflow behavior
	ReservedLabelPrefix = "pipelines.kubeflow.org/"
)

var (
	// cachedDriverPodLabels stores the driver pod labels loaded at startup
	cachedDriverPodLabels map[string]string
	// cachedDriverPodAnnotations stores the driver pod annotations loaded at startup
	cachedDriverPodAnnotations map[string]string
	// driverConfigInitialized indicates whether InitDriverPodConfig has been called
	driverConfigInitialized bool
	// driverConfigMutex protects the cached config from concurrent access
	driverConfigMutex sync.RWMutex
)

// InitDriverPodConfig loads and caches driver pod configuration at API server startup.
// This should be called once after Viper configuration is loaded to catch configuration
// errors early and avoid recomputing the configuration on every pipeline run submission.
func InitDriverPodConfig() {
	driverConfigMutex.Lock()
	defer driverConfigMutex.Unlock()

	// Load labels from Viper configuration
	rawLabels := GetMapConfig(DriverPodLabels)
	cachedDriverPodLabels = filterReservedLabels(rawLabels)

	// Load annotations from Viper configuration
	cachedDriverPodAnnotations = GetMapConfig(DriverPodAnnotations)

	driverConfigInitialized = true

	// Log the loaded configuration for visibility
	glog.Infof("Driver pod configuration initialized: %d labels, %d annotations",
		len(cachedDriverPodLabels), len(cachedDriverPodAnnotations))

	if len(cachedDriverPodLabels) > 0 {
		glog.V(1).Infof("Driver pod labels: %v", cachedDriverPodLabels)
	}
	if len(cachedDriverPodAnnotations) > 0 {
		glog.V(1).Infof("Driver pod annotations: %v", cachedDriverPodAnnotations)
	}
}

// GetDriverPodLabels returns cached driver pod labels from configuration.
// Labels with pipelines.kubeflow.org/ prefix are filtered out during initialization.
// Returns nil if InitDriverPodConfig has not been called.
func GetDriverPodLabels() map[string]string {
	driverConfigMutex.RLock()
	defer driverConfigMutex.RUnlock()

	if !driverConfigInitialized {
		glog.Warning("Driver pod config not initialized, returning nil labels")
		return nil
	}
	return cachedDriverPodLabels
}

// GetDriverPodAnnotations returns cached driver pod annotations from configuration.
// Returns nil if InitDriverPodConfig has not been called.
func GetDriverPodAnnotations() map[string]string {
	driverConfigMutex.RLock()
	defer driverConfigMutex.RUnlock()

	if !driverConfigInitialized {
		glog.Warning("Driver pod config not initialized, returning nil annotations")
		return nil
	}
	return cachedDriverPodAnnotations
}

// filterReservedLabels removes labels with reserved prefix
func filterReservedLabels(labels map[string]string) map[string]string {
	if labels == nil {
		return nil
	}

	filtered := make(map[string]string, len(labels))
	for k, v := range labels {
		if strings.HasPrefix(k, ReservedLabelPrefix) {
			glog.Warningf("Ignoring reserved label %s (prefix %s is reserved)", k, ReservedLabelPrefix)
			continue
		}
		filtered[k] = v
	}
	return filtered
}
