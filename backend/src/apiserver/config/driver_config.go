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
	"strings"

	"github.com/golang/glog"
)

const (
	// Environment variable names for driver pod configuration
	EnvDriverPodLabels      = "DRIVER_POD_LABELS"
	EnvDriverPodAnnotations = "DRIVER_POD_ANNOTATIONS"

	// Reserved label prefix that should be filtered out
	ReservedLabelPrefix = "pipelines.kubeflow.org/"
)

// DriverPodConfig holds the configuration for driver pod labels and annotations
type DriverPodConfig struct {
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
}

// GetDriverPodConfig reads driver pod configuration from environment variables.
// It supports both JSON format and comma-separated key=value format.
// Reserved labels with prefix "pipelines.kubeflow.org/" are filtered out.
// Returns an empty config (not nil) on errors to allow graceful degradation.
func GetDriverPodConfig() (*DriverPodConfig, error) {
	config := &DriverPodConfig{
		Labels:      make(map[string]string),
		Annotations: make(map[string]string),
	}

	// Read and parse labels from environment variable
	labelsEnv := os.Getenv(EnvDriverPodLabels)
	if labelsEnv != "" {
		labels, err := parseConfigValue(labelsEnv)
		if err != nil {
			glog.Warningf("Failed to parse %s: %v. Using empty labels.", EnvDriverPodLabels, err)
		} else {
			config.Labels = labels
		}
	}

	// Read and parse annotations from environment variable
	annotationsEnv := os.Getenv(EnvDriverPodAnnotations)
	if annotationsEnv != "" {
		annotations, err := parseConfigValue(annotationsEnv)
		if err != nil {
			glog.Warningf("Failed to parse %s: %v. Using empty annotations.", EnvDriverPodAnnotations, err)
		} else {
			config.Annotations = annotations
		}
	}

	// Filter out reserved system labels
	validateSystemLabels(config)

	return config, nil
}

// parseConfigValue attempts to parse the input as JSON first,
// then falls back to parsing as comma-separated key=value pairs
func parseConfigValue(input string) (map[string]string, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return make(map[string]string), nil
	}

	// Try parsing as JSON first
	if strings.HasPrefix(input, "{") {
		var result map[string]string
		if err := json.Unmarshal([]byte(input), &result); err == nil {
			return result, nil
		}
		// If JSON parsing fails, log and fall through to k=v parsing
		glog.V(4).Infof("Failed to parse as JSON, trying key=value format: %v", input)
	}

	// Parse as comma-separated key=value pairs
	return parseKVPairs(input), nil
}

// parseKVPairs parses comma-separated key=value pairs into a map.
// Format: "key1=value1,key2=value2,key3=value3"
// Invalid pairs (missing '=' or empty key/value) are skipped with a warning.
func parseKVPairs(input string) map[string]string {
	result := make(map[string]string)
	input = strings.TrimSpace(input)

	if input == "" {
		return result
	}

	pairs := strings.Split(input, ",")
	for _, pair := range pairs {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}

		parts := strings.SplitN(pair, "=", 2)
		if len(parts) != 2 {
			glog.Warningf("Invalid key=value pair, skipping: %s", pair)
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		if key == "" {
			glog.Warningf("Empty key in pair, skipping: %s", pair)
			continue
		}

		if value == "" {
			glog.Warningf("Empty value for key '%s', skipping", key)
			continue
		}

		result[key] = value
	}

	return result
}

// validateSystemLabels removes reserved labels that start with the reserved prefix.
// This prevents users from overriding system-managed labels.
func validateSystemLabels(config *DriverPodConfig) {
	if config == nil || config.Labels == nil {
		return
	}

	for key := range config.Labels {
		if strings.HasPrefix(key, ReservedLabelPrefix) {
			glog.Warningf("Removing reserved label with prefix '%s': %s", ReservedLabelPrefix, key)
			delete(config.Labels, key)
		}
	}
}
