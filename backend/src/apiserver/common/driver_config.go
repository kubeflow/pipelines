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
	"k8s.io/apimachinery/pkg/util/validation"
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
	if err := validateMapConfig(DriverPodLabels); err != nil {
		return nil, err
	}
	if err := validateMapConfig(DriverPodAnnotations); err != nil {
		return nil, err
	}

	labels := filterReservedEntries(GetMapConfig(DriverPodLabels))
	if err := validateLabels(labels); err != nil {
		return nil, err
	}

	annotations := filterReservedEntries(GetMapConfig(DriverPodAnnotations))
	if err := validateAnnotationKeys(annotations); err != nil {
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

// validateAnnotationKeys rejects annotation keys that Kubernetes would not accept.
// Annotation values are free form, so only the keys are checked.
func validateAnnotationKeys(annotations map[string]string) error {
	for k := range annotations {
		if errs := validation.IsQualifiedName(k); len(errs) > 0 {
			return fmt.Errorf("%s has an invalid annotation key %q: %s", DriverPodAnnotations, k, strings.Join(errs, "; "))
		}
	}
	return nil
}

// validateMapConfig rejects a ConfigMap value that Viper would not turn into the map the
// operator meant. On the ConfigMap string path viper.GetStringMapString swallows problems
// in two different ways, and neither produces an error: a boolean or a number makes it
// drop the whole map, while a JSON null is quietly coerced to an empty string. Both leave
// the operator with a configuration that looks accepted but silently does not do what
// they asked for, so the raw value is probed here and every entry must be a JSON string.
func validateMapConfig(name string) error {
	if !viper.IsSet(name) {
		return nil
	}

	// A native map (config.json) reports an empty string here, and a blank or whitespace
	// only value means no configuration. Neither needs probing.
	raw := strings.TrimSpace(viper.GetString(name))
	if raw == "" {
		return nil
	}

	var probe map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &probe); err != nil {
		return fmt.Errorf("%s could not be parsed as a JSON object: %w (raw value: %q)", name, err, raw)
	}
	for k, v := range probe {
		if _, ok := v.(string); !ok {
			return fmt.Errorf("%s has a value that is not a JSON string for key %q; every value must be quoted, for example \"true\" rather than true or null (raw value: %q)", name, k, raw)
		}
	}
	return nil
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
