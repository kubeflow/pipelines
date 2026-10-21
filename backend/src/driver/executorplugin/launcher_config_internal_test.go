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
	"testing"

	"github.com/kubeflow/pipelines/backend/src/driver/driverapi"
	"github.com/kubeflow/pipelines/backend/src/driver/driverlogs"
	"github.com/kubeflow/pipelines/backend/src/v2/client_manager"
	"github.com/kubeflow/pipelines/backend/src/v2/config"
	"github.com/kubeflow/pipelines/backend/src/v2/driver"
	drivercommon "github.com/kubeflow/pipelines/backend/src/v2/driver/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

func TestLauncherConfigCacheLoadsOncePerNamespace(t *testing.T) {
	loads := map[string]int{}
	fail := map[string]bool{"broken": true}
	cache := newLauncherConfigCache()
	cache.load = func(_ context.Context, _ kubernetes.Interface, namespace string) (*config.Config, error) {
		loads[namespace]++
		if fail[namespace] {
			return nil, assert.AnError
		}
		if namespace == "absent" {
			return nil, nil // the ConfigMap is optional
		}
		return &config.Config{}, nil
	}
	load := cache.loader(fake.NewSimpleClientset())
	var wg sync.WaitGroup
	for range 50 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := load(context.Background(), "team-a")
			assert.NoError(t, err)
		}()
	}
	wg.Wait()
	for range 3 {
		cfg, err := load(context.Background(), "absent")
		require.NoError(t, err)
		assert.Nil(t, cfg)
		_, err = load(context.Background(), "broken")
		require.ErrorIs(t, err, assert.AnError)
	}
	assert.Equal(t, map[string]int{"team-a": 1, "absent": 1, "broken": 3}, loads, "successes are cached per namespace; failures retry")
}

func TestDriveReadsLauncherConfigOnceAcrossInvocations(t *testing.T) {
	runner, args, api, manager := invocationFixture(t, driverapi.ContainerDriver)
	api.run.RuntimeConfig = nil // resolve the root from the launcher config
	reads := 0
	manager.k8s.(*fake.Clientset).PrependReactor("get", "configmaps", func(k8stesting.Action) (bool, runtime.Object, error) {
		reads++
		return false, nil, nil
	})
	var loaders []config.LauncherConfigLoader
	runner.execute = func(ctx context.Context, options drivercommon.Options, _ client_manager.ClientManagerInterface) (*driver.Execution, error) {
		require.NotNil(t, options.LauncherConfig, "drivers must resolve the root through the shared cache")
		loaders = append(loaders, options.LauncherConfig)
		_, err := options.LauncherConfig(ctx, options.Namespace)
		require.NoError(t, err)
		return &driver.Execution{TaskID: "task"}, nil
	}
	runner.uploadLogs = func(context.Context, *driverlogs.ArtifactContext, kubernetes.Interface) error { return nil }
	for range 3 {
		_, err := runner.drive(context.Background(), args)
		require.NoError(t, err)
	}
	assert.Len(t, loaders, 3)
	assert.Equal(t, 1, reads, "one ConfigMap read serves every invocation of the agent process")
}
