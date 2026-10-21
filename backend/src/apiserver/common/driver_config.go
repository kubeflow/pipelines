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

package common

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// ValidateDriverPodMetadataConfig rejects legacy driver-only metadata settings
// that cannot be applied to executor-plugin agent Pods without affecting tasks.
func ValidateDriverPodMetadataConfig() error {
	for _, name := range []string{"DRIVER_POD_LABELS", "DRIVER_POD_ANNOTATIONS"} {
		if hasDriverPodMetadata(viper.Get(name)) {
			return fmt.Errorf("%s is no longer supported by executor-plugin drivers; remove it and configure agent-targeted Pod admission instead (see manifests/kustomize/README.md)", name)
		}
	}
	return nil
}

func hasDriverPodMetadata(value interface{}) bool {
	switch value := value.(type) {
	case nil:
		return false
	case string:
		if strings.TrimSpace(value) == "" {
			return false
		}
		var entries map[string]interface{}
		return json.Unmarshal([]byte(value), &entries) != nil || len(entries) != 0
	case map[string]string:
		return len(value) != 0
	case map[string]interface{}:
		return len(value) != 0
	default:
		return true
	}
}
