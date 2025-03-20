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

import "os"

const (
	HttpProxyEnv        = "HTTP_PROXY"
	HttpsProxyEnv       = "HTTPS_PROXY"
	NoProxyEnv          = "NO_PROXY"
	defaultNoProxyValue = "localhost,127.0.0.1,.svc.cluster.local,kubernetes.default.svc,metadata-grpc-service,0,1,2,3,4,5,6,7,8,9"
)

type Config interface {
	GetHttpProxy() string
	GetHttpsProxy() string
	GetNoProxy() string
}

func NewConfig(httpProxy string, httpsProxy string, noProxy string) Config {
	return &config{
		httpProxy:  httpProxy,
		httpsProxy: httpsProxy,
		noProxy:    noProxy,
	}
}

func NewConfigFromEnv() Config {
	httpProxyValue, isHttpProxySet := os.LookupEnv(HttpProxyEnv)
	httpsProxyValue, isHttpsProxySet := os.LookupEnv(HttpsProxyEnv)
	noProxyValue, isNoProxySet := os.LookupEnv(NoProxyEnv)

	if (isHttpProxySet || isHttpsProxySet) && !isNoProxySet {
		return NewConfig(httpProxyValue, httpsProxyValue, defaultNoProxyValue)
	}

	return NewConfig(httpProxyValue, httpsProxyValue, noProxyValue)
}

func EmptyConfig() Config {
	return emptyConfig
}

var emptyConfig = NewConfig("", "", "")

type config struct {
	httpProxy  string
	httpsProxy string
	noProxy    string
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
