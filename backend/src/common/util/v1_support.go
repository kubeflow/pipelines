// Copyright 2026 The Kubeflow Authors
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

package util

import (
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

const (
	BlockV1Pipelines = "BLOCK_V1_PIPELINES"
	// v1AllowedNamespaces is unexported; test-only mirrors exist in:
	//   - backend/src/apiserver/resource/resource_manager_test.go
	//   - backend/src/crd/controller/scheduledworkflow/controller_test.go
	v1AllowedNamespaces = "V1_ALLOWED_NAMESPACES"
)

func IsV1PipelinesBlocked(namespace string) bool {
	if !viper.IsSet(BlockV1Pipelines) {
		return false
	}
	blockV1Value := viper.GetString(BlockV1Pipelines)
	blockV1, err := strconv.ParseBool(blockV1Value)
	if err != nil {
		log.Fatalf("Failed converting string to bool %s", blockV1Value)
	}
	if !blockV1 {
		return false
	}

	allowedNamespaces := viper.GetString(v1AllowedNamespaces)
	if allowedNamespaces == "" {
		return true
	}

	targetNamespace := strings.ToLower(strings.TrimSpace(namespace))
	for _, n := range strings.Split(allowedNamespaces, ",") {
		if strings.ToLower(strings.TrimSpace(n)) == targetNamespace {
			return false
		}
	}
	return true
}
