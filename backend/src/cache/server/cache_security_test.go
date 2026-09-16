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

package server

import (
	"bytes"
	"log"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func unsetCacheSecurityEnv(t *testing.T) {
	t.Helper()
	for _, name := range []string{cacheSecurityModeEnv, legacyCacheFallbackEnv} {
		t.Setenv(name, "")
		require.NoError(t, os.Unsetenv(name))
	}
}

func TestGetCacheSecurityMode(t *testing.T) {
	for _, tc := range []struct {
		name    string
		env     map[string]string
		want    string
		invalid bool
	}{
		{name: "default", want: "enforce"},
		{name: "empty", env: map[string]string{cacheSecurityModeEnv: ""}, want: "enforce"},
		{name: "enforce", env: map[string]string{cacheSecurityModeEnv: "enforce"}, want: "enforce"},
		{name: "audit", env: map[string]string{cacheSecurityModeEnv: "audit"}, want: "audit"},
		{name: "legacy true", env: map[string]string{legacyCacheFallbackEnv: "true"}, want: "audit"},
		{name: "legacy false", env: map[string]string{legacyCacheFallbackEnv: "false"}, want: "enforce"},
		{name: "legacy numeric true", env: map[string]string{legacyCacheFallbackEnv: "1"}, want: "audit"},
		{name: "legacy numeric false", env: map[string]string{legacyCacheFallbackEnv: "0"}, want: "enforce"},
		{name: "matching audit", env: map[string]string{cacheSecurityModeEnv: "audit", legacyCacheFallbackEnv: "true"}, want: "audit"},
		{name: "matching enforce", env: map[string]string{cacheSecurityModeEnv: "enforce", legacyCacheFallbackEnv: "false"}, want: "enforce"},
		{name: "conflicting audit", env: map[string]string{cacheSecurityModeEnv: "audit", legacyCacheFallbackEnv: "false"}, invalid: true},
		{name: "conflicting enforce", env: map[string]string{cacheSecurityModeEnv: "enforce", legacyCacheFallbackEnv: "true"}, invalid: true},
		{name: "empty canonical conflicts", env: map[string]string{cacheSecurityModeEnv: "", legacyCacheFallbackEnv: "true"}, invalid: true},
		{name: "invalid canonical", env: map[string]string{cacheSecurityModeEnv: "legacy"}, invalid: true},
		{name: "invalid canonical with legacy", env: map[string]string{cacheSecurityModeEnv: "typo", legacyCacheFallbackEnv: "true"}, invalid: true},
		{name: "invalid legacy with canonical", env: map[string]string{cacheSecurityModeEnv: "audit", legacyCacheFallbackEnv: "typo"}, invalid: true},
		{name: "invalid legacy", env: map[string]string{legacyCacheFallbackEnv: "typo"}, invalid: true},
		{name: "empty legacy", env: map[string]string{legacyCacheFallbackEnv: ""}, invalid: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			unsetCacheSecurityEnv(t)
			for name, value := range tc.env {
				t.Setenv(name, value)
			}
			mode, err := getCacheSecurityMode()
			if tc.invalid {
				require.Error(t, err)
				require.Error(t, InitializeCacheSecurityMode())
				m := legacyCacheManager(t)
				require.NoError(t, m.DB().Create(legacyCacheRow(t)).Error)
				pod := cachePod("tenant")
				req := GetFakeRequestFromPod(pod)
				req.Namespace = pod.Namespace
				patches, admissionErr := MutatePodIfCached(req, m)
				require.Error(t, admissionErr)
				require.Empty(t, patches)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.want, mode)
				require.NoError(t, InitializeCacheSecurityMode())
			}
		})
	}
}

func TestInitializeCacheSecurityModeWarnings(t *testing.T) {
	for _, tc := range []struct {
		name       string
		env        map[string]string
		audit      bool
		deprecated bool
	}{
		{name: "default"},
		{name: "audit", env: map[string]string{cacheSecurityModeEnv: "audit"}, audit: true},
		{name: "deprecated enabled", env: map[string]string{legacyCacheFallbackEnv: "true"}, audit: true, deprecated: true},
		{name: "deprecated disabled", env: map[string]string{legacyCacheFallbackEnv: "false"}, deprecated: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			unsetCacheSecurityEnv(t)
			for name, value := range tc.env {
				t.Setenv(name, value)
			}
			var logs bytes.Buffer
			original := log.Writer()
			log.SetOutput(&logs)
			t.Cleanup(func() { log.SetOutput(original) })
			require.NoError(t, InitializeCacheSecurityMode())
			if tc.audit {
				require.Contains(t, logs.String(), "unknown ownership")
				require.Contains(t, logs.String(), "namespace isolation")
				require.Contains(t, logs.String(), "3.0")
			} else {
				require.NotContains(t, logs.String(), "unknown ownership")
			}
			if tc.deprecated {
				require.Contains(t, logs.String(), "deprecated")
			} else {
				require.NotContains(t, logs.String(), "deprecated")
			}
		})
	}
}
