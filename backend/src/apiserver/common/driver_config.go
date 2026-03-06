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
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/golang/glog"
	"github.com/spf13/viper"
)

const (
	DriverPodLabels      = "DriverPodLabels"
	DriverPodAnnotations = "DriverPodAnnotations"

	// Reserved label prefix that will be filtered out to prevent
	// overriding system labels that control workflow behavior
	ReservedLabelPrefix = "pipelines.kubeflow.org/"
)

// DriverPodConfig holds the labels and annotations applied to driver pods.
type DriverPodConfig struct {
	Labels      map[string]string
	Annotations map[string]string
}

var (
	// cachedDriverPodConfig holds the driver pod configuration loaded at startup.
	// It is swapped atomically by InitDriverPodConfig so readers never observe a
	// partly updated state. A nil pointer means the config has not been loaded yet.
	cachedDriverPodConfig *DriverPodConfig
	// driverConfigMutex protects the cached config from concurrent access.
	driverConfigMutex sync.RWMutex
)

// InitDriverPodConfig loads, validates, and caches driver pod configuration at API
// server startup. This should be called once after Viper configuration is loaded.
// It returns an error when the configuration is present but cannot be parsed, so the
// API server fails fast instead of starting with a silently empty configuration.
func InitDriverPodConfig() error {
	// Build and validate the new config before taking the lock for the swap.
	cfg, err := newDriverPodConfig()
	if err != nil {
		return err
	}

	driverConfigMutex.Lock()
	defer driverConfigMutex.Unlock()

	if cachedDriverPodConfig != nil {
		glog.Info("Re-initializing driver pod configuration")
	}
	cachedDriverPodConfig = cfg

	glog.Infof("Driver pod configuration initialized: %d labels, %d annotations",
		len(cfg.Labels), len(cfg.Annotations))
	if len(cfg.Labels) > 0 {
		glog.V(1).Infof("Driver pod labels: %v", cfg.Labels)
	}
	if len(cfg.Annotations) > 0 {
		glog.V(1).Infof("Driver pod annotations: %v", cfg.Annotations)
	}
	return nil
}

// newDriverPodConfig reads the driver pod labels and annotations from Viper,
// validates them, and returns a fully populated config or an error. Keeping
// construction separate from the cache swap guarantees the update is applied
// completely or not at all.
func newDriverPodConfig() (*DriverPodConfig, error) {
	rawLabels := GetMapConfig(DriverPodLabels)
	if err := validateMapConfig(DriverPodLabels, rawLabels); err != nil {
		return nil, err
	}

	rawAnnotations := GetMapConfig(DriverPodAnnotations)
	if err := validateMapConfig(DriverPodAnnotations, rawAnnotations); err != nil {
		return nil, err
	}

	return &DriverPodConfig{
		Labels:      filterReservedEntries(rawLabels),
		Annotations: filterReservedEntries(rawAnnotations),
	}, nil
}

// validateMapConfig surfaces the silent failure case on the ConfigMap string path.
// When values arrive as a JSON string, viper.GetStringMapString returns an empty map
// (not an error) if the JSON is malformed or holds values that are not strings, such
// as a boolean or a number. Probing the raw string with json.Unmarshal tells apart a
// genuinely empty config from a misconfiguration that should fail startup.
//
// Note that Viper coerces a JSON null to an empty string, so a value like {"k":null}
// yields the key k with an empty value rather than an error.
func validateMapConfig(name string, parsed map[string]string) error {
	// A parsed map with entries means Viper handled it; an unset key is a valid default.
	if len(parsed) > 0 || !viper.IsSet(name) {
		return nil
	}

	// Native map values (config.json) report an empty string here, and a value that is
	// blank or only whitespace is treated as no configuration rather than an error.
	raw := strings.TrimSpace(viper.GetString(name))
	if raw == "" {
		return nil
	}

	var probe map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &probe); err != nil {
		return fmt.Errorf("%s could not be parsed as a JSON object: %w (raw value: %q)", name, err, raw)
	}
	if len(probe) > 0 {
		return fmt.Errorf("%s parsed as JSON but all values were dropped; ensure every value is a string, not a boolean or number (raw value: %q)", name, raw)
	}
	return nil
}

// GetDriverPodConfig returns a copy of the cached driver pod configuration, or nil
// when no labels or annotations are configured. The returned value is safe to mutate.
func GetDriverPodConfig() *DriverPodConfig {
	labels := GetDriverPodLabels()
	annotations := GetDriverPodAnnotations()
	if len(labels) == 0 && len(annotations) == 0 {
		return nil
	}
	return &DriverPodConfig{
		Labels:      labels,
		Annotations: annotations,
	}
}

// GetDriverPodLabels returns a copy of cached driver pod labels from configuration.
// Labels with pipelines.kubeflow.org/ prefix are filtered out during initialization.
// Returns nil if InitDriverPodConfig has not been called or no labels are configured.
func GetDriverPodLabels() map[string]string {
	driverConfigMutex.RLock()
	defer driverConfigMutex.RUnlock()

	if cachedDriverPodConfig == nil {
		return nil
	}
	return copyMap(cachedDriverPodConfig.Labels)
}

// GetDriverPodAnnotations returns a copy of cached driver pod annotations from configuration.
// Returns nil if InitDriverPodConfig has not been called or no annotations are configured.
func GetDriverPodAnnotations() map[string]string {
	driverConfigMutex.RLock()
	defer driverConfigMutex.RUnlock()

	if cachedDriverPodConfig == nil {
		return nil
	}
	return copyMap(cachedDriverPodConfig.Annotations)
}

// copyMap returns a shallow copy of the given map, or nil if the input is nil.
func copyMap(m map[string]string) map[string]string {
	if m == nil {
		return nil
	}
	cp := make(map[string]string, len(m))
	for k, v := range m {
		cp[k] = v
	}
	return cp
}

// filterReservedEntries removes entries whose key starts with the reserved prefix.
// Used for both labels and annotations to prevent overriding metadata managed by the system.
// Returns nil if the input is nil or the result is empty after filtering.
func filterReservedEntries(entries map[string]string) map[string]string {
	if entries == nil {
		return nil
	}

	filtered := make(map[string]string, len(entries))
	for k, v := range entries {
		if strings.HasPrefix(k, ReservedLabelPrefix) {
			glog.Warningf("Ignoring reserved key %s (prefix %s is reserved)", k, ReservedLabelPrefix)
			continue
		}
		filtered[k] = v
	}
	if len(filtered) == 0 {
		return nil
	}
	return filtered
}
