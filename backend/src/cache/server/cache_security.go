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
	"strconv"
)

const (
	cacheSecurityModeEnv   = "KFP_SECURITY_LEGACY_CACHE_MODE"
	legacyCacheFallbackEnv = "ALLOW_LEGACY_CACHE_FALLBACK"
)

func getCacheSecurityMode() (string, error) {
	mode, configured := os.LookupEnv(cacheSecurityModeEnv)
	switch mode {
	case "":
		mode = "enforce"
	case "enforce", "audit":
	default:
		return "", fmt.Errorf("%s must be enforce or audit", cacheSecurityModeEnv)
	}
	if value, present := os.LookupEnv(legacyCacheFallbackEnv); present {
		enabled, err := strconv.ParseBool(value)
		if err != nil {
			return "", fmt.Errorf("%s must be true or false; migrate to %s=enforce or audit", legacyCacheFallbackEnv, cacheSecurityModeEnv)
		}
		legacyMode := "enforce"
		if enabled {
			legacyMode = "audit"
		}
		if configured && mode != legacyMode {
			return "", fmt.Errorf("%s conflicts with %s; remove the deprecated %s setting", cacheSecurityModeEnv, legacyCacheFallbackEnv, legacyCacheFallbackEnv)
		}
		mode = legacyMode
	}
	return mode, nil
}

// InitializeCacheSecurityMode validates migration settings before serving requests.
func InitializeCacheSecurityMode() error {
	mode, err := getCacheSecurityMode()
	if err != nil {
		return err
	}
	if _, present := os.LookupEnv(legacyCacheFallbackEnv); present {
		log.Printf("WARNING: %s is deprecated; use %s=%s and remove the old setting", legacyCacheFallbackEnv, cacheSecurityModeEnv, mode)
	}
	if mode == "audit" {
		log.Printf("WARNING: %s=audit permits reuse of legacy cache entries with unknown ownership, weakening namespace isolation; audit mode is planned for removal in 3.0.0 (https://github.com/kubeflow/pipelines/issues/14367)", cacheSecurityModeEnv)
	}
	return nil
}
