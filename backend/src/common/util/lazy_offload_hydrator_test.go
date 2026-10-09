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

package util

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	argoconfig "github.com/argoproj/argo-workflows/v4/config"
	workflowapi "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/workflow/hydrator"
	hydratorfake "github.com/argoproj/argo-workflows/v4/workflow/hydrator/fake"
	"github.com/argoproj/argo-workflows/v4/workflow/packer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfake "k8s.io/client-go/kubernetes/fake"
)

func TestLazyOffloadHydrator_InitFailureThenSuccess(t *testing.T) {
	repo := NewMemoryOffloadNodeStatusRepo()
	repo.Put("wf-uid", "offload-hash", workflowapi.Nodes{
		"ok": {ID: "ok", Name: "my-wf", Phase: workflowapi.NodeSucceeded, Type: workflowapi.NodeTypePod},
	})
	inner := NewMemoryWorkflowHydrator(repo)

	var attempts atomic.Int32
	failInit := true
	var failMu sync.Mutex

	lazy := newLazyOffloadHydrator(func(ctx context.Context) (hydrator.Interface, error) {
		attempts.Add(1)
		failMu.Lock()
		defer failMu.Unlock()
		if failInit {
			return nil, errors.New("db unavailable")
		}
		return inner, nil
	})

	wf := &workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{Name: "my-wf", UID: "wf-uid"},
		Status: workflowapi.WorkflowStatus{
			OffloadNodeStatusVersion: "offload-hash",
		},
	}

	err := lazy.Hydrate(context.Background(), wf)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not ready")
	assert.Equal(t, int32(1), attempts.Load())

	failMu.Lock()
	failInit = false
	failMu.Unlock()

	require.NoError(t, lazy.Hydrate(context.Background(), wf))
	assert.Equal(t, int32(2), attempts.Load())
	assert.Empty(t, wf.Status.OffloadNodeStatusVersion)
	assert.Equal(t, workflowapi.NodeSucceeded, wf.Status.Nodes["ok"].Phase)
}

func TestLazyOffloadHydrator_HydrateInlineWithoutInit(t *testing.T) {
	var initCalls atomic.Int32
	lazy := newLazyOffloadHydrator(func(ctx context.Context) (hydrator.Interface, error) {
		initCalls.Add(1)
		return nil, errors.New("db unavailable")
	})

	wf := &workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{Name: "my-wf", UID: "wf-uid"},
		Status: workflowapi.WorkflowStatus{
			Nodes: workflowapi.Nodes{
				"ok": {ID: "ok", Name: "my-wf", Phase: workflowapi.NodeSucceeded, Type: workflowapi.NodeTypePod},
			},
		},
	}

	require.NoError(t, lazy.Hydrate(context.Background(), wf))
	assert.Equal(t, int32(0), initCalls.Load())
	assert.Equal(t, workflowapi.NodeSucceeded, wf.Status.Nodes["ok"].Phase)
}

func TestLazyOffloadHydrator_HydrateCompressedWithoutInit(t *testing.T) {
	cleanup := packer.SetMaxWorkflowSize(230)
	t.Cleanup(cleanup)

	ctx := withArgoLogger(context.Background())
	wf := &workflowapi.Workflow{
		Status: workflowapi.WorkflowStatus{
			Nodes: workflowapi.Nodes{
				"foo": {},
				"bar": {},
			},
		},
	}
	require.NoError(t, packer.CompressWorkflowIfNeeded(ctx, wf))
	require.NotEmpty(t, wf.Status.CompressedNodes)
	require.Empty(t, wf.Status.Nodes)

	var initCalls atomic.Int32
	lazy := newLazyOffloadHydrator(func(ctx context.Context) (hydrator.Interface, error) {
		initCalls.Add(1)
		return nil, errors.New("db unavailable")
	})

	require.NoError(t, lazy.Hydrate(ctx, wf))
	assert.Equal(t, int32(0), initCalls.Load())
	assert.Len(t, wf.Status.Nodes, 2)
}

func TestLazyOffloadHydrator_DehydrateSmallWithoutInit(t *testing.T) {
	cleanup := packer.SetMaxWorkflowSize(230)
	t.Cleanup(cleanup)

	var initCalls atomic.Int32
	lazy := newLazyOffloadHydrator(func(ctx context.Context) (hydrator.Interface, error) {
		initCalls.Add(1)
		return nil, errors.New("db unavailable")
	})

	wf := &workflowapi.Workflow{
		Status: workflowapi.WorkflowStatus{
			Nodes: workflowapi.Nodes{
				"foo": {},
				"bar": {},
			},
		},
	}

	require.NoError(t, lazy.Dehydrate(withArgoLogger(context.Background()), wf))
	assert.Equal(t, int32(0), initCalls.Load())
	assert.NotEmpty(t, wf.Status.CompressedNodes)
	assert.Empty(t, wf.Status.Nodes)
}

func TestLazyOffloadHydrator_HydrateWithNodesLocal(t *testing.T) {
	var initCalls atomic.Int32
	lazy := newLazyOffloadHydrator(func(ctx context.Context) (hydrator.Interface, error) {
		initCalls.Add(1)
		return nil, errors.New("db unavailable")
	})

	wf := &workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{Name: "my-wf", UID: "wf-uid"},
		Status: workflowapi.WorkflowStatus{
			OffloadNodeStatusVersion: "offload-hash",
		},
	}
	nodes := workflowapi.Nodes{
		"ok": {ID: "ok", Name: "my-wf", Phase: workflowapi.NodeSucceeded, Type: workflowapi.NodeTypePod},
	}

	lazy.HydrateWithNodes(wf, nodes)
	assert.Equal(t, int32(0), initCalls.Load())
	assert.Equal(t, nodes, wf.Status.Nodes)
	assert.Empty(t, wf.Status.CompressedNodes)
	assert.Empty(t, wf.Status.OffloadNodeStatusVersion)
}

func TestNewLazyOffloadHydrator_PersistLoaderRetries(t *testing.T) {
	var loadCalls atomic.Int32
	failLoad := true
	persistYAML := []byte(`nodeStatusOffLoad: true
postgresql:
  host: postgres.example.invalid
  port: 5432
  database: argo
  tableName: argo_workflows
`)

	lazy := NewLazyOffloadHydrator(k8sfake.NewClientset(), func(ctx context.Context) (*argoconfig.PersistConfig, string, error) {
		loadCalls.Add(1)
		if failLoad {
			return nil, "", errors.New("configmap not found")
		}
		persist, err := ParseArgoPersistConfig(persistYAML)
		if err != nil {
			return nil, "", err
		}
		return persist, "ns", nil
	})

	wf := &workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{Name: "my-wf", UID: "wf-uid"},
		Status: workflowapi.WorkflowStatus{
			OffloadNodeStatusVersion: "offload-hash",
		},
	}

	err := lazy.Hydrate(context.Background(), wf)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not ready")
	assert.Equal(t, int32(1), loadCalls.Load())

	failLoad = false
	err = lazy.Hydrate(context.Background(), wf)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not ready")
	assert.Equal(t, int32(2), loadCalls.Load())
}

func TestLazyOffloadHydrator_DisabledOffloadReloadable(t *testing.T) {
	repo := NewMemoryOffloadNodeStatusRepo()
	repo.Put("wf-uid", "offload-hash", workflowapi.Nodes{
		"ok": {ID: "ok", Name: "my-wf", Phase: workflowapi.NodeSucceeded, Type: workflowapi.NodeTypePod},
	})
	inner := NewMemoryWorkflowHydrator(repo)

	offloadEnabled := false
	var loadCalls atomic.Int32
	lazy := newLazyOffloadHydrator(func(ctx context.Context) (hydrator.Interface, error) {
		loadCalls.Add(1)
		if !offloadEnabled {
			return hydratorfake.Noop, nil
		}
		return inner, nil
	})

	ctx := context.Background()
	offloadEnabled, err := TryInitLazyOffloadHydrator(lazy, ctx)
	require.NoError(t, err)
	assert.False(t, offloadEnabled)
	assert.Equal(t, int32(1), loadCalls.Load())

	wf := &workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{Name: "my-wf", UID: "wf-uid"},
		Status: workflowapi.WorkflowStatus{
			OffloadNodeStatusVersion: "offload-hash",
		},
	}

	require.NoError(t, lazy.Hydrate(ctx, wf))
	assert.Equal(t, "offload-hash", wf.Status.OffloadNodeStatusVersion)
	assert.Empty(t, wf.Status.Nodes)
	assert.Equal(t, int32(2), loadCalls.Load())

	offloadEnabled = true
	require.NoError(t, lazy.Hydrate(ctx, wf))
	assert.Equal(t, int32(3), loadCalls.Load())
	assert.Empty(t, wf.Status.OffloadNodeStatusVersion)
	assert.Equal(t, workflowapi.NodeSucceeded, wf.Status.Nodes["ok"].Phase)

	require.NoError(t, lazy.Hydrate(ctx, wf))
	assert.Equal(t, int32(3), loadCalls.Load())
}

func TestTryInitLazyOffloadHydrator_OffloadEnabled(t *testing.T) {
	repo := NewMemoryOffloadNodeStatusRepo()
	inner := NewMemoryWorkflowHydrator(repo)
	lazy := newLazyOffloadHydrator(func(ctx context.Context) (hydrator.Interface, error) {
		return inner, nil
	})

	offloadEnabled, err := TryInitLazyOffloadHydrator(lazy, context.Background())
	require.NoError(t, err)
	assert.True(t, offloadEnabled)
}

// TestLazyOffloadHydrator_SecretMissingThenPresent models API-server startup when
// persistence is enabled in the ConfigMap but the credential Secret is not yet
// readable. Compressed hydrate must still succeed; offloaded hydrate retries init
// once the Secret appears (memory hydrator stands in for CreateWorkflowHydrator).
func TestLazyOffloadHydrator_SecretMissingThenPresent(t *testing.T) {
	const (
		ns         = "kubeflow-pipelines"
		secretName = "argo-postgres-config"
	)
	repo := NewMemoryOffloadNodeStatusRepo()
	repo.Put("wf-uid", "offload-hash", workflowapi.Nodes{
		"ok": {ID: "ok", Name: "my-wf", Phase: workflowapi.NodeSucceeded, Type: workflowapi.NodeTypePod},
	})
	memoryHydrator := NewMemoryWorkflowHydrator(repo)

	kube := k8sfake.NewClientset()
	var initCalls atomic.Int32
	lazy := newLazyOffloadHydrator(func(ctx context.Context) (hydrator.Interface, error) {
		initCalls.Add(1)
		_, err := kube.CoreV1().Secrets(ns).Get(ctx, secretName, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			return nil, fmt.Errorf("failed to create Argo offload DB session: secrets %q not found", secretName)
		}
		if err != nil {
			return nil, err
		}
		return memoryHydrator, nil
	})

	ctx := context.Background()
	offloadEnabled, err := TryInitLazyOffloadHydrator(lazy, ctx)
	require.Error(t, err)
	assert.False(t, offloadEnabled)
	assert.Contains(t, err.Error(), secretName)

	cleanup := packer.SetMaxWorkflowSize(230)
	t.Cleanup(cleanup)
	compressed := &workflowapi.Workflow{
		Status: workflowapi.WorkflowStatus{
			Nodes: workflowapi.Nodes{"foo": {}, "bar": {}},
		},
	}
	require.NoError(t, packer.CompressWorkflowIfNeeded(withArgoLogger(ctx), compressed))
	require.NotEmpty(t, compressed.Status.CompressedNodes)
	require.NoError(t, lazy.Hydrate(ctx, compressed), "compressed hydrate must not require Secret/DB")
	assert.Len(t, compressed.Status.Nodes, 2)

	offloaded := &workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{Name: "my-wf", UID: "wf-uid"},
		Status: workflowapi.WorkflowStatus{
			OffloadNodeStatusVersion: "offload-hash",
		},
	}
	require.Error(t, lazy.Hydrate(ctx, offloaded))
	assert.Equal(t, "offload-hash", offloaded.Status.OffloadNodeStatusVersion)

	_, err = kube.CoreV1().Secrets(ns).Create(ctx, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: secretName, Namespace: ns},
		Data: map[string][]byte{
			"username": []byte("argo"),
			"password": []byte("secret"),
		},
	}, metav1.CreateOptions{})
	require.NoError(t, err)

	require.NoError(t, lazy.Hydrate(ctx, offloaded))
	assert.Empty(t, offloaded.Status.OffloadNodeStatusVersion)
	assert.Equal(t, workflowapi.NodeSucceeded, offloaded.Status.Nodes["ok"].Phase)
	assert.GreaterOrEqual(t, initCalls.Load(), int32(2))
}

func TestLazyOffloadHydrator_DehydrateTooLargeOffloadDisabled(t *testing.T) {
	cleanup := packer.SetMaxWorkflowSize(1)
	t.Cleanup(cleanup)

	lazy := newLazyOffloadHydrator(func(ctx context.Context) (hydrator.Interface, error) {
		return hydratorfake.Noop, nil
	})

	nodes := make(workflowapi.Nodes, 32)
	for i := 0; i < 32; i++ {
		id := fmt.Sprintf("node-%d", i)
		nodes[id] = workflowapi.NodeStatus{
			ID:   id,
			Name: fmt.Sprintf("my-wf.%s", id),
			Type: workflowapi.NodeTypePod,
		}
	}
	wf := &workflowapi.Workflow{
		Status: workflowapi.WorkflowStatus{Nodes: nodes},
	}

	err := lazy.Dehydrate(withArgoLogger(context.Background()), wf)
	require.Error(t, err)
	assert.True(t, packer.IsTooLargeError(err))
	assert.NotEmpty(t, wf.Status.Nodes)
}

type closedSessionHydrator struct {
	inner        hydrator.Interface
	failOnce     atomic.Bool
	hydrateCalls atomic.Int32
}

func (h *closedSessionHydrator) IsHydrated(wf *workflowapi.Workflow) bool {
	return h.inner.IsHydrated(wf)
}

func (h *closedSessionHydrator) Hydrate(ctx context.Context, wf *workflowapi.Workflow) error {
	h.hydrateCalls.Add(1)
	if h.failOnce.CompareAndSwap(true, false) {
		return errors.New("session proxy is closed")
	}
	return h.inner.Hydrate(ctx, wf)
}

func (h *closedSessionHydrator) Dehydrate(ctx context.Context, wf *workflowapi.Workflow) error {
	return h.inner.Dehydrate(ctx, wf)
}

func (h *closedSessionHydrator) HydrateWithNodes(wf *workflowapi.Workflow, nodes workflowapi.Nodes) {
	h.inner.HydrateWithNodes(wf, nodes)
}

func TestLazyOffloadHydrator_ReinitAfterClosedSession(t *testing.T) {
	repo := NewMemoryOffloadNodeStatusRepo()
	repo.Put("wf-uid", "offload-hash", workflowapi.Nodes{
		"ok": {ID: "ok", Name: "my-wf", Phase: workflowapi.NodeSucceeded, Type: workflowapi.NodeTypePod},
	})
	inner := NewMemoryWorkflowHydrator(repo)
	closed := &closedSessionHydrator{inner: inner}
	closed.failOnce.Store(true)

	var initCalls atomic.Int32
	lazy := newLazyOffloadHydrator(func(ctx context.Context) (hydrator.Interface, error) {
		initCalls.Add(1)
		return closed, nil
	})

	wf := &workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{Name: "my-wf", UID: "wf-uid"},
		Status: workflowapi.WorkflowStatus{
			OffloadNodeStatusVersion: "offload-hash",
		},
	}

	require.NoError(t, lazy.Hydrate(context.Background(), wf))
	assert.Equal(t, int32(2), initCalls.Load(), "closed session must clear the cache and re-init")
	assert.Equal(t, int32(2), closed.hydrateCalls.Load())
	assert.Empty(t, wf.Status.OffloadNodeStatusVersion)
	assert.Equal(t, workflowapi.NodeSucceeded, wf.Status.Nodes["ok"].Phase)

	// A healthy cached hydrator should not re-init on the next call.
	wf.Status.OffloadNodeStatusVersion = "offload-hash"
	wf.Status.Nodes = nil
	require.NoError(t, lazy.Hydrate(context.Background(), wf))
	assert.Equal(t, int32(2), initCalls.Load())
	assert.Equal(t, int32(3), closed.hydrateCalls.Load())
}

func TestLazyOffloadHydrator_ConcurrentInit(t *testing.T) {
	repo := NewMemoryOffloadNodeStatusRepo()
	inner := NewMemoryWorkflowHydrator(repo)

	var initCalls atomic.Int32
	start := make(chan struct{})
	lazy := newLazyOffloadHydrator(func(ctx context.Context) (hydrator.Interface, error) {
		initCalls.Add(1)
		<-start
		return inner, nil
	})

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			wf := &workflowapi.Workflow{
				ObjectMeta: metav1.ObjectMeta{Name: "my-wf", UID: "wf-uid"},
				Status: workflowapi.WorkflowStatus{
					OffloadNodeStatusVersion: "offload-hash",
				},
			}
			_ = lazy.Hydrate(context.Background(), wf)
		}()
	}
	close(start)
	wg.Wait()
	assert.Equal(t, int32(1), initCalls.Load())
}

// syncClosedSessionHydrator always reports a closed session, and blocks inside
// Hydrate until release is closed so concurrent callers can fail against the
// same instance before either recovers.
type syncClosedSessionHydrator struct {
	entered chan<- struct{}
	release <-chan struct{}
}

func (h *syncClosedSessionHydrator) IsHydrated(wf *workflowapi.Workflow) bool {
	return wf.Status.CompressedNodes == "" && !wf.Status.IsOffloadNodeStatus()
}

func (h *syncClosedSessionHydrator) Hydrate(context.Context, *workflowapi.Workflow) error {
	h.entered <- struct{}{}
	<-h.release
	return errors.New("session proxy is closed")
}

func (h *syncClosedSessionHydrator) Dehydrate(context.Context, *workflowapi.Workflow) error {
	return nil
}

func (h *syncClosedSessionHydrator) HydrateWithNodes(wf *workflowapi.Workflow, nodes workflowapi.Nodes) {
	wf.Status.Nodes = nodes
	wf.Status.CompressedNodes = ""
	wf.Status.OffloadNodeStatusVersion = ""
}

func TestLazyOffloadHydrator_ConcurrentClosedSessionRecovery(t *testing.T) {
	repo := NewMemoryOffloadNodeStatusRepo()
	repo.Put("wf-uid", "offload-hash", workflowapi.Nodes{
		"ok": {ID: "ok", Name: "my-wf", Phase: workflowapi.NodeSucceeded, Type: workflowapi.NodeTypePod},
	})
	healthy := NewMemoryWorkflowHydrator(repo)

	entered := make(chan struct{}, 2)
	release := make(chan struct{})
	failed := &syncClosedSessionHydrator{entered: entered, release: release}

	var initCalls atomic.Int32
	lazy := newLazyOffloadHydrator(func(ctx context.Context) (hydrator.Interface, error) {
		n := initCalls.Add(1)
		if n == 1 {
			return failed, nil
		}
		return healthy, nil
	})

	// Install the failing session once so both callers share the same pointer.
	lazyConcrete := lazy.(*lazyOffloadHydrator)
	inner, err := lazyConcrete.ensureInner(context.Background())
	require.NoError(t, err)
	require.Equal(t, failed, inner)
	require.Equal(t, int32(1), initCalls.Load())

	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			wf := &workflowapi.Workflow{
				ObjectMeta: metav1.ObjectMeta{Name: "my-wf", UID: "wf-uid"},
				Status: workflowapi.WorkflowStatus{
					OffloadNodeStatusVersion: "offload-hash",
				},
			}
			errs <- lazy.Hydrate(context.Background(), wf)
		}()
	}

	<-entered
	<-entered
	close(release)
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}

	assert.Equal(t, int32(2), initCalls.Load(),
		"concurrent closed-session recovery must re-init once, not discard a healthy replacement")
	lazyConcrete.mu.Lock()
	finalInner := lazyConcrete.inner
	lazyConcrete.mu.Unlock()
	assert.Equal(t, healthy, finalInner, "final cached hydrator must be the healthy replacement")
}
