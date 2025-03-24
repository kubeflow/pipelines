package proxy

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewProxyConfigFromEnvVars(t *testing.T) {
	tests := []struct {
		envVars             map[string]interface{}
		expectedProxyConfig ProxyConfig
	}{
		{
			envVars:             map[string]interface{}{},
			expectedProxyConfig: NewProxyConfig("", "", ""),
		},
		{
			envVars: map[string]interface{}{
				HttpProxyEnv: "http_proxy",
			},
			expectedProxyConfig: NewProxyConfig("http_proxy", "", defaultNoProxyValue),
		},
		{
			envVars: map[string]interface{}{
				HttpProxyEnv:  "http_proxy",
				HttpsProxyEnv: "https_proxy",
			},
			expectedProxyConfig: NewProxyConfig("http_proxy", "https_proxy", defaultNoProxyValue),
		},
		{
			envVars: map[string]interface{}{
				HttpProxyEnv: "http_proxy",
				NoProxyEnv:   "no_proxy",
			},
			expectedProxyConfig: NewProxyConfig("http_proxy", "", "no_proxy"),
		},
		{
			envVars: map[string]interface{}{
				HttpsProxyEnv: "https_proxy",
			},
			expectedProxyConfig: NewProxyConfig("", "https_proxy", defaultNoProxyValue),
		},
		{
			envVars: map[string]interface{}{
				HttpsProxyEnv: "https_proxy",
				NoProxyEnv:    "no_proxy",
			},
			expectedProxyConfig: NewProxyConfig("", "https_proxy", "no_proxy"),
		},
		{
			envVars: map[string]interface{}{
				NoProxyEnv: "no_proxy",
			},
			expectedProxyConfig: NewProxyConfig("", "", "no_proxy"),
		},
		{
			envVars: map[string]interface{}{
				HttpProxyEnv:  "http_proxy",
				HttpsProxyEnv: "https_proxy",
				NoProxyEnv:    "no_proxy",
			},
			expectedProxyConfig: NewProxyConfig("http_proxy", "https_proxy", "no_proxy"),
		},
		{
			envVars: map[string]interface{}{
				HttpProxyEnv:  "",
				HttpsProxyEnv: "",
				NoProxyEnv:    "",
			},
			expectedProxyConfig: NewProxyConfig("", "", ""),
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%+v", tt), func(t *testing.T) {
			actualProxyConfig := NewProxyConfigFromEnvVars(tt.envVars)
			assert.Equal(t, tt.expectedProxyConfig, actualProxyConfig)
		})
	}
}
