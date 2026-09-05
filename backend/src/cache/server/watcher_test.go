// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// https://www.apache.org/licenses/LICENSE-2.0

package server

import (
	"context"
	"runtime"
	"testing"
	"time"

	"github.com/kubeflow/pipelines/backend/src/cache/client"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

type testPodClient struct {
	v1.PodInterface
	watcher watch.Interface
	calls   int
}

func (f *testPodClient) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	f.calls++
	if f.calls > 1 {
		runtime.Goexit()
	}
	return f.watcher, nil
}

type testKubernetesCore struct {
	podClient v1.PodInterface
}

func (c *testKubernetesCore) PodClient(namespace string) v1.PodInterface {
	return c.podClient
}

type testClientManager struct {
	*FakeClientManager
	kubernetesCore client.KubernetesCoreInterface
}

func (m *testClientManager) KubernetesCoreClient() client.KubernetesCoreInterface {
	return m.kubernetesCore
}

func TestWatchPodsIgnoresErrorEventsWithoutPanic(t *testing.T) {
	watcher := watch.NewRaceFreeFake()

	clientManager := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())

	podClient := &testPodClient{
		PodInterface: clientManager.KubernetesCoreClient().PodClient("default"),
		watcher:      watcher,
	}

	testManager := &testClientManager{
		FakeClientManager: clientManager,
		kubernetesCore: &testKubernetesCore{
			podClient: podClient,
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})

	go func() {
		defer close(done)
		WatchPods(ctx, "default", testManager)
	}()
	watcher.Error(&metav1.Status{
		Status:  metav1.StatusFailure,
		Message: "test watch error",
	})

	watcher.Add(&corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-pod",
			Labels: map[string]string{
				ArgoCompleteLabelKey: "false",
			},
		},
	})
	// Cancel context and stop watcher to trigger clean shutdown of WatchPods.
	cancel()
	watcher.Stop()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("WatchPods failed to exit after context cancellation")
	}
}
