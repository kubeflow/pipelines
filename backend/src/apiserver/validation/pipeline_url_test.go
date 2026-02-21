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

package validation

import (
	"net"
	"sync"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

// resetURLConfig resets the sync.Once to allow re-initialization with new config
func resetURLConfig() {
	urlConfigInit = sync.Once{}
	blockedNets = nil
	allowedDomains = nil
}

func TestValidatePipelineURL_AllowedDomains(t *testing.T) {
	viper.Reset()
	resetURLConfig()

	// Default allowed domains should pass
	assert.NoError(t, ValidatePipelineURL("https://storage.googleapis.com/bucket/file.yaml"))
	assert.NoError(t, ValidatePipelineURL("https://github.com/kubeflow/pipelines/raw/main/test.yaml"))
	assert.NoError(t, ValidatePipelineURL("https://s3.amazonaws.com/bucket/file.yaml"))

	// Disallowed domain should fail
	err := ValidatePipelineURL("https://evil.com/malicious.yaml")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not in allowlist")
}

func TestValidatePipelineURL_BlockedDomain(t *testing.T) {
	viper.Reset()
	resetURLConfig()

	// localhost should fail (not in allowlist)
	err := ValidatePipelineURL("https://localhost/file.yaml")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not in allowlist")
}

func TestValidatePipelineURL_SchemeRestriction(t *testing.T) {
	viper.Reset()
	resetURLConfig()

	// HTTP blocked by default
	err := ValidatePipelineURL("http://storage.googleapis.com/bucket/file.yaml")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "scheme")

	// HTTPS allowed
	assert.NoError(t, ValidatePipelineURL("https://storage.googleapis.com/bucket/file.yaml"))
}

func TestValidatePipelineURL_HTTPAllowedWhenConfigured(t *testing.T) {
	viper.Reset()
	viper.Set("PIPELINE_URL_ALLOW_HTTP", "true")
	resetURLConfig()

	// HTTP allowed when configured
	assert.NoError(t, ValidatePipelineURL("http://storage.googleapis.com/bucket/file.yaml"))
}

func TestValidatePipelineURL_PortRestriction(t *testing.T) {
	viper.Reset()
	resetURLConfig()

	// Default port (no port specified) should pass
	assert.NoError(t, ValidatePipelineURL("https://storage.googleapis.com/bucket/file.yaml"))

	// Explicit standard port should pass
	assert.NoError(t, ValidatePipelineURL("https://storage.googleapis.com:443/bucket/file.yaml"))

	// Non-standard port should fail
	err := ValidatePipelineURL("https://storage.googleapis.com:8080/bucket/file.yaml")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "port")
}

func TestValidatePipelineURL_PortMustMatchScheme(t *testing.T) {
	viper.Reset()
	viper.Set("PIPELINE_URL_ALLOW_HTTP", "true")
	resetURLConfig()

	// HTTPS on port 80 should fail
	err := ValidatePipelineURL("https://storage.googleapis.com:80/bucket/file.yaml")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "HTTPS requires port 443")

	// HTTP on port 443 should fail
	err = ValidatePipelineURL("http://storage.googleapis.com:443/bucket/file.yaml")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "HTTP requires port 80")
}

func TestValidatePipelineURL_UserConfiguredDomains(t *testing.T) {
	viper.Reset()
	viper.Set("PIPELINE_URL_ALLOWED_DOMAINS", "example.com")
	resetURLConfig()

	// User-configured domain should pass
	assert.NoError(t, ValidatePipelineURL("https://example.com/test.yaml"))

	// Default domains should still pass
	assert.NoError(t, ValidatePipelineURL("https://storage.googleapis.com/bucket/file.yaml"))
}

func TestValidatePipelineURL_InvalidURL(t *testing.T) {
	viper.Reset()
	resetURLConfig()

	err := ValidatePipelineURL("not-a-valid-url")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid URL")
}

func TestValidatePipelineURL_FileSchemeBlocked(t *testing.T) {
	viper.Reset()
	resetURLConfig()

	err := ValidatePipelineURL("file:///etc/passwd")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "scheme")
}

func TestIsBlockedIP(t *testing.T) {
	viper.Reset()
	resetURLConfig()
	initURLConfig()

	tests := []struct {
		ip      string
		blocked bool
		desc    string
	}{
		// Loopback
		{"127.0.0.1", true, "IPv4 loopback"},
		{"127.255.255.255", true, "IPv4 loopback range end"},
		{"::1", true, "IPv6 loopback"},

		// Private IPv4
		{"10.0.0.1", true, "Private Class A"},
		{"10.255.255.255", true, "Private Class A end"},
		{"172.16.0.1", true, "Private Class B start"},
		{"172.31.255.255", true, "Private Class B end"},
		{"192.168.1.1", true, "Private Class C"},
		{"192.168.255.255", true, "Private Class C end"},

		// Link-local / metadata
		{"169.254.0.1", true, "Link-local"},
		{"169.254.169.254", true, "Cloud metadata endpoint"},

		// Public IPs (should NOT be blocked)
		{"8.8.8.8", false, "Google DNS"},
		{"1.1.1.1", false, "Cloudflare DNS"},
		{"142.250.80.46", false, "Google public"},
		{"151.101.1.140", false, "GitHub"},
	}

	for _, tt := range tests {
		ip := net.ParseIP(tt.ip)
		assert.NotNil(t, ip, "Failed to parse IP: %s", tt.ip)
		assert.Equal(t, tt.blocked, isBlockedIP(ip), "%s: IP %s should be blocked=%v", tt.desc, tt.ip, tt.blocked)
	}
}

func TestSafePipelineHTTPClient_HasTimeout(t *testing.T) {
	viper.Reset()
	resetURLConfig()

	client := SafePipelineHTTPClient()
	assert.NotNil(t, client)
	assert.True(t, client.Timeout > 0, "Client should have a timeout configured")
}

func TestSafePipelineHTTPClient_CustomTimeout(t *testing.T) {
	viper.Reset()
	viper.Set("PIPELINE_URL_TIMEOUT", "60")
	resetURLConfig()

	client := SafePipelineHTTPClient()
	assert.NotNil(t, client)
	// 60 seconds = 60,000,000,000 nanoseconds
	assert.Equal(t, int64(60_000_000_000), int64(client.Timeout))
}
