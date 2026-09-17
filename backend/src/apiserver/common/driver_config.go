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
	apivalidation "k8s.io/apimachinery/pkg/api/validation"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

const (
	// Config keys. Viper derives the environment variable name from the key, so these
	// are also the names the API server Deployment must use when it wires the values
	// in from the pipeline-install-config ConfigMap.
	DriverPodLabels      = "DRIVER_POD_LABELS"
	DriverPodAnnotations = "DRIVER_POD_ANNOTATIONS"

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
	rawLabels, err := parseMapConfig(DriverPodLabels)
	if err != nil {
		return nil, err
	}
	rawAnnotations, err := parseMapConfig(DriverPodAnnotations)
	if err != nil {
		return nil, err
	}

	labels := filterReservedEntries(rawLabels)
	if err := validateLabels(labels); err != nil {
		return nil, err
	}

	annotations := filterReservedEntries(rawAnnotations)
	if err := validateAnnotations(annotations); err != nil {
		return nil, err
	}

	return &DriverPodConfig{
		Labels:      labels,
		Annotations: annotations,
	}, nil
}

// validateLabels rejects label keys or values that Kubernetes would not accept, so a
// bad entry fails at API server startup rather than later when the driver pod is
// created and the pod is rejected by the API.
func validateLabels(labels map[string]string) error {
	for k, v := range labels {
		if errs := validation.IsQualifiedName(k); len(errs) > 0 {
			return fmt.Errorf("%s has an invalid label key %q: %s", DriverPodLabels, k, strings.Join(errs, "; "))
		}
		if errs := validation.IsValidLabelValue(v); len(errs) > 0 {
			return fmt.Errorf("%s has an invalid label value %q for key %q: %s", DriverPodLabels, v, k, strings.Join(errs, "; "))
		}
	}
	return nil
}

// validateAnnotations uses the Kubernetes annotation validator, which lowers keys before
// checking syntax (so Example.com/Name is valid unlike for labels) and enforces the 256 KiB
// total size limit. Only the configured annotations are measured; compiler-added ones are not.
func validateAnnotations(annotations map[string]string) error {
	// The field path already names the setting, so the result is returned unwrapped.
	return apivalidation.ValidateAnnotations(annotations, field.NewPath(DriverPodAnnotations)).ToAggregate()
}

// parseMapConfig reads and parses the configured value in one step so the validated map is
// the one cached. A JSON object in the config file is refused because Viper lowercases its
// keys, silently merging case-differing entries; the string form keeps keys as written.
func parseMapConfig(name string) (map[string]string, error) {
	switch value := viper.Get(name).(type) {
	case nil:
		return nil, nil
	case string:
		parsed, err := jsonToMap(value)
		if err != nil {
			return nil, fmt.Errorf("%s %w", name, err)
		}
		return parsed, nil
	case map[string]string:
		return value, nil
	case map[string]any:
		return nil, fmt.Errorf("%s must be a JSON string, not an object; write it as \"{\\\"app\\\":\\\"driver\\\"}\"", name)
	default:
		return nil, fmt.Errorf("%s must be a JSON string, but it is a %T", name, value)
	}
}

// jsonToMap parses a JSON object whose values are all strings. Unmarshalling straight into a
// map[string]string would turn a null into an empty string without an error, so the values are
// checked before conversion.
func jsonToMap(value string) (map[string]string, error) {
	raw := strings.TrimSpace(value)
	if raw == "" {
		return nil, nil
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return nil, fmt.Errorf("could not be parsed as a JSON object: %w (raw value: %q)", err, raw)
	}
	if m == nil {
		return nil, fmt.Errorf("must be a JSON object, not null (raw value: %q)", raw)
	}
	parsed := make(map[string]string, len(m))
	for k, v := range m {
		s, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("value for key %q must be a string (raw value: %q)", k, raw)
		}
		parsed[k] = s
	}
	if len(parsed) == 0 {
		return nil, nil
	}
	return parsed, nil
}

// GetDriverPodConfig returns a copy of the cached driver pod configuration, or nil when
// no labels or annotations are configured. The returned value is safe to mutate.
//
// Both maps are copied under a single read lock. Calling the two getters in turn would
// take the lock twice, which would let a concurrent InitDriverPodConfig slip in between
// and hand back labels from one version of the config and annotations from another. That
// would defeat the point of swapping the cache as one atomic snapshot.
func GetDriverPodConfig() *DriverPodConfig {
	driverConfigMutex.RLock()
	defer driverConfigMutex.RUnlock()

	if cachedDriverPodConfig == nil {
		return nil
	}

	labels := copyMap(cachedDriverPodConfig.Labels)
	annotations := copyMap(cachedDriverPodConfig.Annotations)
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
