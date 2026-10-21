// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package executorplugin

import (
	"context"
	"sync"

	"github.com/kubeflow/pipelines/backend/src/v2/config"
	"k8s.io/client-go/kubernetes"
)

// launcherConfigCache serves the kfp-launcher config once per namespace for
// the life of the agent process, which is the life of one workflow. The
// per-task driver read it for every task; here that would be three reads per
// task. Loading under the lock also keeps a wide fan-out from stampeding the
// API server on the first task. Failures are not cached.
type launcherConfigCache struct {
	load    func(context.Context, kubernetes.Interface, string) (*config.Config, error)
	mu      sync.Mutex
	configs map[string]*config.Config
}

func newLauncherConfigCache() *launcherConfigCache {
	return &launcherConfigCache{load: config.LoadLauncherConfig, configs: map[string]*config.Config{}}
}

func (c *launcherConfigCache) loader(k8sClient kubernetes.Interface) config.LauncherConfigLoader {
	return func(ctx context.Context, namespace string) (*config.Config, error) {
		c.mu.Lock()
		defer c.mu.Unlock()
		if cfg, ok := c.configs[namespace]; ok {
			return cfg, nil
		}
		cfg, err := c.load(ctx, k8sClient, namespace)
		if err != nil {
			return nil, err
		}
		c.configs[namespace] = cfg
		return cfg, nil
	}
}
