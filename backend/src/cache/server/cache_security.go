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

// Package server implements legacy pipeline cache admission and pod observation.
package server

import (
	"fmt"
	"log"
	"os"
)

const cacheSecurityModeEnv = "KFP_SECURITY_LEGACY_CACHE_MODE"

func getCacheSecurityMode() (string, error) {
	mode := os.Getenv(cacheSecurityModeEnv)
	switch mode {
	case "":
		mode = "enforce"
	case "enforce", "audit":
	default:
		return "", fmt.Errorf("%s must be enforce or audit", cacheSecurityModeEnv)
	}
	return mode, nil
}

// InitializeCacheSecurityMode validates migration settings before serving requests.
func InitializeCacheSecurityMode() error {
	mode, err := getCacheSecurityMode()
	if err != nil {
		return err
	}
	if mode == "audit" {
		log.Printf("WARNING: %s=audit permits reuse of legacy cache entries with unknown ownership, weakening namespace isolation; audit mode is planned for removal in 3.0.0 (https://github.com/kubeflow/pipelines/issues/14367)", cacheSecurityModeEnv)
	}
	return nil
}
