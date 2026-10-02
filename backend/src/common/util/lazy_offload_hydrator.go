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
	"fmt"
	"sync"

	argoconfig "github.com/argoproj/argo-workflows/v4/config"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/workflow/hydrator"
	hydratorfake "github.com/argoproj/argo-workflows/v4/workflow/hydrator/fake"
	"github.com/argoproj/argo-workflows/v4/workflow/packer"
	"k8s.io/client-go/kubernetes"
)

type lazyOffloadHydrator struct {
	mu     sync.Mutex
	initFn func(context.Context) (hydrator.Interface, error)
	inner  hydrator.Interface
}

// PersistConfigLoader loads Argo persistence config and the namespace for persistence secrets.
type PersistConfigLoader func(ctx context.Context) (*argoconfig.PersistConfig, string, error)

// NewLazyOffloadHydrator returns a hydrator that lazily connects to Argo's offload database.
// Initialization is retried on each offload read/write until it succeeds.
func NewLazyOffloadHydrator(kube kubernetes.Interface, load PersistConfigLoader) hydrator.Interface {
	return newLazyOffloadHydrator(func(ctx context.Context) (hydrator.Interface, error) {
		persist, secretsNamespace, err := load(ctx)
		if err != nil {
			return nil, err
		}
		if persist == nil || !persist.NodeStatusOffload {
			return hydratorfake.Noop, nil
		}
		return CreateWorkflowHydrator(ctx, kube, persist, secretsNamespace)
	})
}

func newLazyOffloadHydrator(initFn func(context.Context) (hydrator.Interface, error)) hydrator.Interface {
	return &lazyOffloadHydrator{initFn: initFn}
}

// TryInitLazyOffloadHydrator attempts to initialize the underlying hydrator once at startup.
func TryInitLazyOffloadHydrator(h hydrator.Interface, ctx context.Context) error {
	lazy, ok := h.(*lazyOffloadHydrator)
	if !ok {
		return nil
	}
	_, err := lazy.ensureInner(ctx)
	return err
}

func (l *lazyOffloadHydrator) ensureInner(ctx context.Context) (hydrator.Interface, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.inner != nil {
		return l.inner, nil
	}
	if l.initFn == nil {
		return nil, fmt.Errorf("argo offload hydrator is not initialized")
	}
	inner, err := l.initFn(ctx)
	if err != nil {
		return nil, err
	}
	l.inner = inner
	return l.inner, nil
}

func (l *lazyOffloadHydrator) IsHydrated(wf *wfv1.Workflow) bool {
	return wf.Status.CompressedNodes == "" && !wf.Status.IsOffloadNodeStatus()
}

func (l *lazyOffloadHydrator) Hydrate(ctx context.Context, wf *wfv1.Workflow) error {
	ctx = withArgoLogger(ctx)
	if err := packer.DecompressWorkflow(ctx, wf); err != nil {
		return err
	}
	if !wf.Status.IsOffloadNodeStatus() {
		return nil
	}
	inner, err := l.ensureInner(ctx)
	if err != nil {
		return fmt.Errorf("argo offload hydrator is not ready: %w", err)
	}
	return inner.Hydrate(ctx, wf)
}

func (l *lazyOffloadHydrator) Dehydrate(ctx context.Context, wf *wfv1.Workflow) error {
	if !l.IsHydrated(wf) {
		return nil
	}
	ctx = withArgoLogger(ctx)
	err := packer.CompressWorkflowIfNeeded(ctx, wf)
	if err == nil {
		wf.Status.OffloadNodeStatusVersion = ""
		return nil
	}
	if !packer.IsTooLargeError(err) {
		return err
	}
	inner, err := l.ensureInner(ctx)
	if err != nil {
		return fmt.Errorf("argo offload hydrator is not ready: %w", err)
	}
	return inner.Dehydrate(ctx, wf)
}

func (l *lazyOffloadHydrator) HydrateWithNodes(wf *wfv1.Workflow, nodes wfv1.Nodes) {
	wf.Status.Nodes = nodes
	wf.Status.CompressedNodes = ""
	wf.Status.OffloadNodeStatusVersion = ""
}
