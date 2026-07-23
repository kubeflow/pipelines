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

// validateAnnotations defers to the validator Kubernetes applies to an object's annotations, so
// a key it would accept is not refused here and the combined size is held to the same limit.
// Reimplementing either rule would drift from Kubernetes over time, and the key rule in
// particular is not the one used for labels: an annotation key is lowered before its syntax is
// checked, so Example.com/name is a valid annotation key even though the same key would be
// refused as a label. Only the syntax check is case insensitive; the keys themselves keep the
// case they were written with.
//
// The size is worth checking at startup because Kubernetes only applies it when the pod is
// created, which is long after the API server has started and the workflow has compiled. What is
// measured here is the configured annotations alone. The driver pod also carries annotations
// added by the compiler and possibly by admission, so passing this check bounds the operator's
// own input rather than guaranteeing the pod will be accepted.
func validateAnnotations(annotations map[string]string) error {
	if errs := apivalidation.ValidateAnnotations(annotations, field.NewPath(DriverPodAnnotations)); len(errs) > 0 {
		return fmt.Errorf("%s is not valid: %s", DriverPodAnnotations, errs.ToAggregate())
	}
	return nil
}

// parseMapConfig reads the configured value once and returns the map it describes, so the map
// that was checked is the map that ends up in the cache. Reading it a second time to convert it
// would leave a window for the configuration watcher to reload between the two reads and cache
// something that was never checked.
//
// A value written as a JSON object in the configuration file is refused rather than parsed. The
// configuration loader lowers the keys of such an object while reading the file, before any of
// this code runs, and that is destructive in a way nothing downstream can repair. Two keys
// differing only in case collapse into one, so a value this function would refuse can be
// discarded before it is ever seen, and when both keys need lowering the survivor depends on map
// iteration order, which makes the same file start the API server on one restart and fail on the
// next. Writing the object as a JSON string, the way the ConfigMap does, keeps the keys exactly
// as they were written and gives every value to the check below.
func parseMapConfig(name string) (map[string]string, error) {
	// One read, deliberately. Asking IsSet first and fetching afterwards would be two, and the
	// nil case below already covers a key that is not set, since IsSet is defined as the same
	// lookup returning something.
	switch value := viper.Get(name).(type) {
	case nil:
		// Not configured. A configuration file value of null reports as nothing here too, and
		// Viper offers no way to tell that apart from an absent key.
		return nil, nil
	case string:
		return parseMapConfigString(name, value)
	case map[string]string:
		// Only reachable when the value is set programmatically, which a configuration file
		// cannot do, so the keys are as written and the values are already strings.
		return copyMap(value), nil
	case map[string]interface{}:
		return nil, fmt.Errorf("%s must be a JSON object written as a string, for example \"{\\\"app\\\":\\\"driver\\\"}\", because a JSON object written directly in the configuration file has its keys lowered before it can be checked and two keys differing only in case are merged", name)
	default:
		return nil, fmt.Errorf("%s must be a JSON object written as a string, but it is a %T", name, value)
	}
}

// parseMapConfigString parses the form a ConfigMap or an environment variable delivers, where
// the whole object arrives as one string. Every value must be a JSON string, because the weak
// conversion this replaces coerced a boolean or a number to its text and turned a null, a list
// or a nested object into an empty string, none of it producing an error. An empty label value
// is the worst of those, since it does not mean the same thing as no label at all.
func parseMapConfigString(name, value string) (map[string]string, error) {
	// A blank or whitespace only value means no configuration.
	raw := strings.TrimSpace(value)
	if raw == "" {
		return nil, nil
	}

	var probe map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &probe); err != nil {
		return nil, fmt.Errorf("%s could not be parsed as a JSON object: %w (raw value: %q)", name, err, raw)
	}
	// A bare null unmarshals into a nil map without error, so the loop below would have
	// nothing to walk and the value would be accepted while the feature stayed off. Every
	// other value that is not an object already fails above.
	if probe == nil {
		return nil, fmt.Errorf("%s must be a JSON object, but the value is null; leave it empty or use {} to configure nothing (raw value: %q)", name, raw)
	}

	parsed := make(map[string]string, len(probe))
	for k, v := range probe {
		s, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("%s has a value that is not a JSON string for key %q; every value must be quoted, for example \"true\" rather than true or null (raw value: %q)", name, k, raw)
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
