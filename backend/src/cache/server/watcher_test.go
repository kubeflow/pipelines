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
	"context"
	"testing"
	"time"

	cacheclient "github.com/kubeflow/pipelines/backend/src/cache/client"
	"github.com/kubeflow/pipelines/backend/src/cache/storage"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

type watchErrorK8sClient struct {
	podClient v1.PodInterface
}

func (c *watchErrorK8sClient) PodClient(namespace string) v1.PodInterface {
	return c.podClient
}

type watchErrorClientMgr struct {
	k8sClient cacheclient.KubernetesCoreInterface
}

func (m *watchErrorClientMgr) CacheStore() storage.ExecutionCacheStoreInterface {
	return nil
}

func (m *watchErrorClientMgr) KubernetesCoreClient() cacheclient.KubernetesCoreInterface {
	return m.k8sClient
}

func TestWatchPods_WatchErrorDoesNotPanic(t *testing.T) {
	badPodClient := &cacheclient.FakeBadPodClient{}
	k8sClient := &watchErrorK8sClient{podClient: badPodClient}
	clientMgr := &watchErrorClientMgr{k8sClient: k8sClient}

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		WatchPods(ctx, "test-ns", clientMgr)
		close(done)
	}()

	time.Sleep(100 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("WatchPods did not exit after context cancellation")
	}
}
