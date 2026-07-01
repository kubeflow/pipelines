// Copyright 2025 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package apiclient

import (
	"testing"
	"time"

	"google.golang.org/grpc/backoff"
)

func TestFromEnv_Defaults(t *testing.T) {
	cfg := FromEnv()
	if cfg.Endpoint != "ml-pipeline.kubeflow:8887" {
		t.Fatalf("unexpected default endpoint: %s", cfg.Endpoint)
	}
	if cfg.BackoffBaseDelay != "" || cfg.BackoffMultiplier != "" || cfg.BackoffJitter != "" ||
		cfg.BackoffMaxDelay != "" || cfg.MinConnectTimeout != "" {
		t.Fatalf("expected empty gRPC backoff config by default, got %+v", cfg)
	}
}

func TestFromEnv_WithCustomBackoffConfig(t *testing.T) {
	t.Setenv(KFPAPIAddressEnvVar, "ml-pipeline.kubeflow")
	t.Setenv(KFPAPIPortEnvVar, "9999")
	t.Setenv(KFPAPIGRPCBackoffBaseDelayEnvVar, "2s")
	t.Setenv(KFPAPIGRPCBackoffMultiplierEnvVar, "1.8")
	t.Setenv(KFPAPIGRPCBackoffJitterEnvVar, "0.4")
	t.Setenv(KFPAPIGRPCBackoffMaxDelayEnvVar, "45s")
	t.Setenv(KFPAPIGRPCMinConnectTimeoutEnvVar, "25s")

	cfg := FromEnv()
	if cfg.Endpoint != "ml-pipeline.kubeflow:9999" {
		t.Fatalf("unexpected custom endpoint: %s", cfg.Endpoint)
	}
	if cfg.BackoffBaseDelay != "2s" ||
		cfg.BackoffMultiplier != "1.8" ||
		cfg.BackoffJitter != "0.4" ||
		cfg.BackoffMaxDelay != "45s" ||
		cfg.MinConnectTimeout != "25s" {
		t.Fatalf("unexpected backoff config from env: %+v", cfg)
	}
}

func TestFromEnvWithEndpointOverride_PrefersExplicitEndpoint(t *testing.T) {
	t.Setenv(KFPAPIAddressEnvVar, "ignored-address")
	t.Setenv(KFPAPIPortEnvVar, "9999")

	cfg := FromEnvWithEndpointOverride("custom-address", "7777")
	if cfg.Endpoint != "custom-address:7777" {
		t.Fatalf("unexpected override endpoint: %s", cfg.Endpoint)
	}
}

func TestFromEnvWithEndpointOverride_FallsBackToEnv(t *testing.T) {
	t.Setenv(KFPAPIAddressEnvVar, "ml-pipeline.custom")
	t.Setenv(KFPAPIPortEnvVar, "9999")

	cfg := FromEnvWithEndpointOverride("", "")
	if cfg.Endpoint != "ml-pipeline.custom:9999" {
		t.Fatalf("unexpected env fallback endpoint: %s", cfg.Endpoint)
	}
}

func TestConfigConnectParams_DefaultsToNilWhenUnset(t *testing.T) {
	params, err := (&Config{}).connectParams()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if params != nil {
		t.Fatalf("expected nil connect params when unset, got %+v", params)
	}
}

func TestConfigConnectParams_ParsesConfiguredValues(t *testing.T) {
	params, err := (&Config{
		BackoffBaseDelay:  "3s",
		BackoffMultiplier: "2.0",
		BackoffJitter:     "0.1",
		BackoffMaxDelay:   "1m",
		MinConnectTimeout: "15s",
	}).connectParams()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if params == nil {
		t.Fatal("expected connect params")
	}
	if params.Backoff.BaseDelay != 3*time.Second {
		t.Fatalf("unexpected base delay: %v", params.Backoff.BaseDelay)
	}
	if params.Backoff.Multiplier != 2.0 {
		t.Fatalf("unexpected multiplier: %v", params.Backoff.Multiplier)
	}
	if params.Backoff.Jitter != 0.1 {
		t.Fatalf("unexpected jitter: %v", params.Backoff.Jitter)
	}
	if params.Backoff.MaxDelay != time.Minute {
		t.Fatalf("unexpected max delay: %v", params.Backoff.MaxDelay)
	}
	if params.MinConnectTimeout != 15*time.Second {
		t.Fatalf("unexpected min connect timeout: %v", params.MinConnectTimeout)
	}
}

func TestConfigConnectParams_UsesGRPCDefaultsForUnsetFields(t *testing.T) {
	params, err := (&Config{BackoffBaseDelay: "5s"}).connectParams()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if params == nil {
		t.Fatal("expected connect params")
	}
	if params.Backoff.BaseDelay != 5*time.Second {
		t.Fatalf("unexpected base delay: %v", params.Backoff.BaseDelay)
	}
	if params.Backoff.Multiplier != backoff.DefaultConfig.Multiplier {
		t.Fatalf("unexpected default multiplier: %v", params.Backoff.Multiplier)
	}
	if params.Backoff.Jitter != backoff.DefaultConfig.Jitter {
		t.Fatalf("unexpected default jitter: %v", params.Backoff.Jitter)
	}
	if params.Backoff.MaxDelay != backoff.DefaultConfig.MaxDelay {
		t.Fatalf("unexpected default max delay: %v", params.Backoff.MaxDelay)
	}
	if params.MinConnectTimeout != defaultMinConnectTimeout {
		t.Fatalf("unexpected default min connect timeout: %v", params.MinConnectTimeout)
	}
}

func TestConfigConnectParams_ReturnsErrorForInvalidValues(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
	}{
		{
			name: "invalid base delay",
			cfg:  Config{BackoffBaseDelay: "not-a-duration"},
		},
		{
			name: "invalid multiplier",
			cfg:  Config{BackoffMultiplier: "not-a-number"},
		},
		{
			name: "invalid jitter",
			cfg:  Config{BackoffJitter: "not-a-number"},
		},
		{
			name: "invalid max delay",
			cfg:  Config{BackoffMaxDelay: "not-a-duration"},
		},
		{
			name: "invalid min connect timeout",
			cfg:  Config{MinConnectTimeout: "not-a-duration"},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			params, err := testCase.cfg.connectParams()
			if err == nil {
				t.Fatalf("expected error, got params=%+v", params)
			}
		})
	}
}
