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
	"io"
	"strings"
	"sync"

	argoconfig "github.com/argoproj/argo-workflows/v4/config"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/workflow/hydrator"
	hydratorfake "github.com/argoproj/argo-workflows/v4/workflow/hydrator/fake"
	"github.com/argoproj/argo-workflows/v4/workflow/packer"
	"k8s.io/client-go/kubernetes"
)

type lazyOffloadHydrator struct {
	mu sync.Mutex
	// loadConfig reads persistence settings and a revision token (ConfigMap
	// resourceVersion). Used by production NewLazyOffloadHydrator.
	loadConfig PersistConfigLoader
	kube       kubernetes.Interface
	// initFn is a test-only builder that returns (inner, revision). Empty
	// revision disables revision-based cache invalidation.
	initFn         func(context.Context) (hydrator.Interface, string, error)
	inner          hydrator.Interface
	configRevision string
}

// PersistConfigLoader loads Argo persistence config, the namespace for persistence
// secrets, and a revision token (typically the workflow-controller ConfigMap
// resourceVersion) so enabled hydrators can pick up config changes.
type PersistConfigLoader func(ctx context.Context) (*argoconfig.PersistConfig, string, string, error)

// NewLazyOffloadHydrator returns a hydrator that lazily connects to Argo's offload database.
// Initialization is retried on each offload read/write until it succeeds. When the
// ConfigMap revision changes, a previously cached real hydrator is discarded.
func NewLazyOffloadHydrator(kube kubernetes.Interface, load PersistConfigLoader) hydrator.Interface {
	return &lazyOffloadHydrator{kube: kube, loadConfig: load}
}

func newLazyOffloadHydrator(initFn func(context.Context) (hydrator.Interface, string, error)) hydrator.Interface {
	return &lazyOffloadHydrator{initFn: initFn}
}

// TryInitLazyOffloadHydrator attempts to initialize the underlying hydrator once at startup.
// The returned bool is true when node status offload is enabled and a real hydrator was created.
func TryInitLazyOffloadHydrator(h hydrator.Interface, ctx context.Context) (bool, error) {
	lazy, ok := h.(*lazyOffloadHydrator)
	if !ok {
		return false, nil
	}
	inner, err := lazy.ensureInner(ctx)
	if err != nil {
		return false, err
	}
	return inner != hydratorfake.Noop, nil
}

func closeHydratorIfPossible(h hydrator.Interface) {
	if h == nil || h == hydratorfake.Noop {
		return
	}
	if closer, ok := h.(io.Closer); ok {
		_ = closer.Close()
	}
}

func (l *lazyOffloadHydrator) replaceInnerLocked(next hydrator.Interface, revision string) hydrator.Interface {
	if l.inner != nil && l.inner != next {
		closeHydratorIfPossible(l.inner)
	}
	l.inner = nil
	l.configRevision = revision
	if next == hydratorfake.Noop {
		return next
	}
	l.inner = next
	return l.inner
}

func (l *lazyOffloadHydrator) ensureInner(ctx context.Context) (hydrator.Interface, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.loadConfig != nil {
		return l.ensureInnerFromConfigLocked(ctx)
	}
	return l.ensureInnerFromInitFnLocked(ctx)
}

func (l *lazyOffloadHydrator) ensureInnerFromConfigLocked(ctx context.Context) (hydrator.Interface, error) {
	persist, secretsNamespace, revision, err := l.loadConfig(ctx)
	if err != nil {
		if l.inner != nil {
			// Keep serving the cached hydrator if the ConfigMap is temporarily
			// unreadable; session-recovery paths still clear on hard DB errors.
			return l.inner, nil
		}
		return nil, err
	}
	if l.inner != nil && revision != "" && revision == l.configRevision {
		return l.inner, nil
	}
	if persist == nil || !persist.NodeStatusOffload {
		return l.replaceInnerLocked(hydratorfake.Noop, revision), nil
	}
	inner, err := CreateWorkflowHydrator(ctx, l.kube, persist, secretsNamespace)
	if err != nil {
		return nil, err
	}
	return l.replaceInnerLocked(inner, revision), nil
}

func (l *lazyOffloadHydrator) ensureInnerFromInitFnLocked(ctx context.Context) (hydrator.Interface, error) {
	if l.initFn == nil {
		return nil, fmt.Errorf("argo offload hydrator is not initialized")
	}
	// When a revision-aware test initFn is cached, re-probe on each ensure so
	// config flips are observed. Unversioned initFns (empty revision) keep the
	// previous "init once" behavior for session-recovery tests.
	if l.inner != nil && l.configRevision == "" {
		return l.inner, nil
	}
	inner, revision, err := l.initFn(ctx)
	if err != nil {
		if l.inner != nil {
			return l.inner, nil
		}
		return nil, err
	}
	if l.inner != nil && revision != "" && revision == l.configRevision {
		if inner != l.inner {
			closeHydratorIfPossible(inner)
		}
		return l.inner, nil
	}
	// Offload may be toggled; do not cache the no-op hydrator so later reads
	// can pick up an enabled persistence config without restart. Still record
	// the revision so an enabled→disabled flip clears a real cached hydrator.
	if inner == hydratorfake.Noop {
		return l.replaceInnerLocked(inner, revision), nil
	}
	return l.replaceInnerLocked(inner, revision), nil
}

// clearInnerIfSame drops the cached hydrator only when it is still the instance
// that failed. Concurrent recovery against the same closed session must not
// discard a healthy replacement another goroutine already installed.
func (l *lazyOffloadHydrator) clearInnerIfSame(failed hydrator.Interface) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.inner == failed {
		closeHydratorIfPossible(l.inner)
		l.inner = nil
		l.configRevision = ""
	}
}

// isUnusableOffloadSessionError reports errors that mean the cached Argo SessionProxy
// (or similar DB session) can no longer serve hydrate/dehydrate until re-initialized.
func isUnusableOffloadSessionError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "session proxy is closed") ||
		strings.Contains(msg, "sql: database is closed")
}

func (l *lazyOffloadHydrator) callWithSessionRecovery(
	ctx context.Context,
	op func(hydrator.Interface) error,
) error {
	inner, err := l.ensureInner(ctx)
	if err != nil {
		return fmt.Errorf("argo offload hydrator is not ready: %w", err)
	}
	err = op(inner)
	if err == nil || !isUnusableOffloadSessionError(err) {
		return err
	}
	// Argo's SessionProxy closes itself after exhausted reconnect attempts and
	// stays closed. Clear only the failed instance (compare-and-swap under the
	// lock) so a concurrent recovery cannot discard a healthy replacement, then
	// rebuild at most once on this call.
	failed := inner
	l.clearInnerIfSame(failed)
	inner, initErr := l.ensureInner(ctx)
	if initErr != nil {
		return fmt.Errorf("argo offload hydrator is not ready: %w", initErr)
	}
	return op(inner)
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
	return l.callWithSessionRecovery(ctx, func(inner hydrator.Interface) error {
		return inner.Hydrate(ctx, wf)
	})
}

func (l *lazyOffloadHydrator) Dehydrate(ctx context.Context, wf *wfv1.Workflow) error {
	if !l.IsHydrated(wf) {
		return nil
	}
	ctx = withArgoLogger(ctx)
	compressErr := packer.CompressWorkflowIfNeeded(ctx, wf)
	if compressErr == nil {
		wf.Status.OffloadNodeStatusVersion = ""
		return nil
	}
	if !packer.IsTooLargeError(compressErr) {
		return compressErr
	}
	err := l.callWithSessionRecovery(ctx, func(inner hydrator.Interface) error {
		if inner == hydratorfake.Noop {
			return compressErr
		}
		return inner.Dehydrate(ctx, wf)
	})
	return err
}

func (l *lazyOffloadHydrator) HydrateWithNodes(wf *wfv1.Workflow, nodes wfv1.Nodes) {
	wf.Status.Nodes = nodes
	wf.Status.CompressedNodes = ""
	wf.Status.OffloadNodeStatusVersion = ""
}
