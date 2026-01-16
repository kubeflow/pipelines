// Copyright 2025 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package proxy

import (
	"os"

	k8score "k8s.io/api/core/v1"
)

const (
	HttpProxyEnv        = "HTTP_PROXY"
	HttpsProxyEnv       = "HTTPS_PROXY"
	NoProxyEnv          = "NO_PROXY"
	defaultNoProxyValue = "localhost,127.0.0.1,.svc.cluster.local,kubernetes.default.svc,seaweedfs.kubeflow,metadata-grpc-service,metadata-grpc-service.kubeflow,ml-pipeline.kubeflow"
)

type Config interface {
	GetHttpProxy() string
	GetHttpsProxy() string
	GetNoProxy() string
	GetEnvVars() []k8score.EnvVar
}

var (
	configInstance Config = &nonInitializedConfig{}
	emptyConfig           = newConfig("", "", "")
)

func InitializeConfig(httpProxy string, httpsProxy string, noProxy string) {
	configInstance = newConfig(httpProxy, httpsProxy, noProxy)
}

func InitializeConfigWithEnv() {
	configInstance = newConfigFromEnv()
}

func InitializeConfigWithEmptyForTests() {
	configInstance = EmptyConfig()
}

func GetConfig() Config {
	return configInstance
}

func newConfig(httpProxy string, httpsProxy string, noProxy string) Config {
	return &config{
		httpProxy:  httpProxy,
		httpsProxy: httpsProxy,
		noProxy:    noProxy,
	}
}

func newConfigFromEnv() Config {
	httpProxyValue, isHttpProxySet := os.LookupEnv(HttpProxyEnv)
	httpsProxyValue, isHttpsProxySet := os.LookupEnv(HttpsProxyEnv)
	noProxyValue, isNoProxySet := os.LookupEnv(NoProxyEnv)

	if (isHttpProxySet || isHttpsProxySet) && !isNoProxySet {
		return newConfig(httpProxyValue, httpsProxyValue, defaultNoProxyValue)
	}

	return newConfig(httpProxyValue, httpsProxyValue, noProxyValue)
}

func EmptyConfig() Config {
	return emptyConfig
}

type config struct {
	httpProxy  string
	httpsProxy string
	noProxy    string
}

func (c *config) GetEnvVars() []k8score.EnvVar {
	var envVars []k8score.EnvVar

	if c.httpProxy != "" {
		envVars = append(envVars, k8score.EnvVar{Name: "http_proxy", Value: c.httpProxy})
		envVars = append(envVars, k8score.EnvVar{Name: "HTTP_PROXY", Value: c.httpProxy})
	}

	if c.httpsProxy != "" {
		envVars = append(envVars, k8score.EnvVar{Name: "https_proxy", Value: c.httpsProxy})
		envVars = append(envVars, k8score.EnvVar{Name: "HTTPS_PROXY", Value: c.httpsProxy})
	}

	if c.noProxy != "" {
		envVars = append(envVars, k8score.EnvVar{Name: "no_proxy", Value: c.noProxy})
		envVars = append(envVars, k8score.EnvVar{Name: "NO_PROXY", Value: c.noProxy})
	}

	return envVars
}

func (c *config) GetHttpProxy() string {
	return c.httpProxy
}

func (c *config) GetHttpsProxy() string {
	return c.httpsProxy
}

func (c *config) GetNoProxy() string {
	return c.noProxy
}

type nonInitializedConfig struct {
}

func (c *nonInitializedConfig) GetEnvVars() []k8score.EnvVar {
	panic("Attempt to use a non-initialized proxy config")
}

func (c *nonInitializedConfig) GetHttpProxy() string {
	panic("Attempt to use a non-initialized proxy config")
}

func (c *nonInitializedConfig) GetHttpsProxy() string {
	panic("Attempt to use a non-initialized proxy config")
}

func (c *nonInitializedConfig) GetNoProxy() string {
	panic("Attempt to use a non-initialized proxy config")
}
