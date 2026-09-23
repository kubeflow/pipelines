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
	"sort"
	"sync"
	"testing"

	argoconfig "github.com/argoproj/argo-workflows/v4/config"
	persistsqldb "github.com/argoproj/argo-workflows/v4/persist/sqldb"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	argologging "github.com/argoproj/argo-workflows/v4/util/logging"
	argosqldb "github.com/argoproj/argo-workflows/v4/util/sqldb"
	"github.com/argoproj/argo-workflows/v4/workflow/hydrator"
	hydratorfake "github.com/argoproj/argo-workflows/v4/workflow/hydrator/fake"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/yaml"
)

var workflowHydrator hydrator.Interface = hydratorfake.Noop

// SetWorkflowHydrator replaces the hydrator used to restore and persist Argo node status.
// Passing nil restores the no-op hydrator.
func SetWorkflowHydrator(h hydrator.Interface) {
	if h == nil {
		workflowHydrator = hydratorfake.Noop
		return
	}
	workflowHydrator = h
}

// SetWorkflowHydratorForTest installs a hydrator for the duration of a test.
func SetWorkflowHydratorForTest(t *testing.T, h hydrator.Interface) {
	t.Helper()
	previous := workflowHydrator
	SetWorkflowHydrator(h)
	t.Cleanup(func() {
		SetWorkflowHydrator(previous)
	})
}

func withArgoLogger(ctx context.Context) context.Context {
	if ctx == nil {
		return ArgoContext()
	}
	return argologging.WithLogger(ctx, argologging.RequireLoggerFromContext(ArgoContext()))
}

func (w *Workflow) Hydrate(ctx context.Context) error {
	if err := workflowHydrator.Hydrate(withArgoLogger(ctx), w.Workflow); err != nil {
		return NewInternalServerError(err, "Failed to hydrate offloaded workflow node status")
	}
	return nil
}

func (w *Workflow) Dehydrate(ctx context.Context) error {
	if err := workflowHydrator.Dehydrate(withArgoLogger(ctx), w.Workflow); err != nil {
		return NewInternalServerError(err, "Failed to dehydrate workflow node status before updating Kubernetes")
	}
	return nil
}

func (w *Workflow) ClearPersistedNodeStatus() {
	if w == nil || w.Workflow == nil {
		return
	}
	w.Status.Nodes = nil
	w.Status.CompressedNodes = ""
	w.Status.OffloadNodeStatusVersion = ""
}

// ParseArgoPersistConfig unmarshals the workflow-controller `persistence` YAML.
func ParseArgoPersistConfig(persistenceYAML []byte) (*argoconfig.PersistConfig, error) {
	if len(persistenceYAML) == 0 {
		return nil, fmt.Errorf("persistence config is empty")
	}
	var persist argoconfig.PersistConfig
	if err := yaml.Unmarshal(persistenceYAML, &persist); err != nil {
		return nil, fmt.Errorf("failed to parse Argo persistence config: %w", err)
	}
	return &persist, nil
}

// ArgoPersistSecretNames returns unique Secret names referenced by Argo persistence credentials.
func ArgoPersistSecretNames(persist *argoconfig.PersistConfig) []string {
	if persist == nil {
		return nil
	}
	seen := map[string]struct{}{}
	add := func(name string) {
		if name != "" {
			seen[name] = struct{}{}
		}
	}
	if persist.PostgreSQL != nil {
		add(persist.PostgreSQL.UsernameSecret.Name)
		add(persist.PostgreSQL.PasswordSecret.Name)
	}
	if persist.MySQL != nil {
		add(persist.MySQL.UsernameSecret.Name)
		add(persist.MySQL.PasswordSecret.Name)
	}
	names := make([]string, 0, len(seen))
	for name := range seen {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// InitWorkflowHydrator connects to Argo's offload database and installs the hydrator used by retry.
func InitWorkflowHydrator(ctx context.Context, kube kubernetes.Interface, persist *argoconfig.PersistConfig, secretsNamespace string) error {
	if persist == nil || !persist.NodeStatusOffload {
		SetWorkflowHydrator(hydratorfake.Noop)
		return nil
	}
	tableName, err := persistsqldb.GetTableName(persist)
	if err != nil {
		return err
	}
	loggerCtx := withArgoLogger(ctx)
	sessionProxy, err := argosqldb.NewSessionProxy(loggerCtx, argosqldb.SessionProxyConfig{
		KubectlConfig: kube,
		Namespace:     secretsNamespace,
		DBConfig:      persist.DBConfig,
	})
	if err != nil {
		return fmt.Errorf("failed to create Argo offload DB session: %w", err)
	}
	argosqldb.ConfigureDBSession(sessionProxy.Session(), persist.ConnectionPool)
	repo, err := persistsqldb.NewOffloadNodeStatusRepo(
		loggerCtx,
		argologging.RequireLoggerFromContext(ArgoContext()),
		sessionProxy,
		persist.GetClusterName(),
		tableName,
	)
	if err != nil {
		return fmt.Errorf("failed to create Argo offload node status repo: %w", err)
	}
	SetWorkflowHydrator(hydrator.New(repo))
	return nil
}

type offloadKey struct {
	uid     string
	version string
}

// MemoryOffloadNodeStatusRepo is an in-memory OffloadNodeStatusRepo for tests.
type MemoryOffloadNodeStatusRepo struct {
	mu    sync.Mutex
	nodes map[offloadKey]wfv1.Nodes
}

func NewMemoryOffloadNodeStatusRepo() *MemoryOffloadNodeStatusRepo {
	return &MemoryOffloadNodeStatusRepo{nodes: map[offloadKey]wfv1.Nodes{}}
}

func (r *MemoryOffloadNodeStatusRepo) Put(uid, version string, nodes wfv1.Nodes) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.nodes[offloadKey{uid: uid, version: version}] = nodes
}

func (r *MemoryOffloadNodeStatusRepo) Save(_ context.Context, uid, _ string, nodes wfv1.Nodes) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	version := fmt.Sprintf("mem:%d", len(r.nodes)+1)
	r.nodes[offloadKey{uid: uid, version: version}] = nodes
	return version, nil
}

func (r *MemoryOffloadNodeStatusRepo) Get(_ context.Context, uid, version string) (wfv1.Nodes, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	nodes, ok := r.nodes[offloadKey{uid: uid, version: version}]
	if !ok {
		return nil, fmt.Errorf("offloaded node status not found for uid %s version %s", uid, version)
	}
	return nodes, nil
}

func (r *MemoryOffloadNodeStatusRepo) List(_ context.Context, _ string) (map[persistsqldb.UUIDVersion]wfv1.Nodes, error) {
	return map[persistsqldb.UUIDVersion]wfv1.Nodes{}, nil
}

func (r *MemoryOffloadNodeStatusRepo) ListOldOffloads(_ context.Context, _ string) (map[string][]string, error) {
	return map[string][]string{}, nil
}

func (r *MemoryOffloadNodeStatusRepo) Delete(_ context.Context, uid, version string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.nodes, offloadKey{uid: uid, version: version})
	return nil
}

func (r *MemoryOffloadNodeStatusRepo) IsEnabled() bool {
	return true
}

// NewMemoryWorkflowHydrator returns an Argo hydrator backed by an in-memory offload repo.
func NewMemoryWorkflowHydrator(repo *MemoryOffloadNodeStatusRepo) hydrator.Interface {
	return hydrator.New(repo)
}

type alwaysOffloadHydrator struct {
	repo persistsqldb.OffloadNodeStatusRepo
}

// NewAlwaysOffloadWorkflowHydrator saves node status on every dehydrate, even when the
// workflow is small enough to compress in the CR. Used to test offload UID handling.
func NewAlwaysOffloadWorkflowHydrator(repo *MemoryOffloadNodeStatusRepo) hydrator.Interface {
	return alwaysOffloadHydrator{repo: repo}
}

func (h alwaysOffloadHydrator) IsHydrated(wf *wfv1.Workflow) bool {
	return wf.Status.CompressedNodes == "" && !wf.Status.IsOffloadNodeStatus()
}

func (h alwaysOffloadHydrator) Hydrate(ctx context.Context, wf *wfv1.Workflow) error {
	return hydrator.New(h.repo).Hydrate(ctx, wf)
}

func (h alwaysOffloadHydrator) HydrateWithNodes(wf *wfv1.Workflow, nodes wfv1.Nodes) {
	hydrator.New(h.repo).HydrateWithNodes(wf, nodes)
}

func (h alwaysOffloadHydrator) Dehydrate(ctx context.Context, wf *wfv1.Workflow) error {
	if !h.IsHydrated(wf) {
		return nil
	}
	version, err := h.repo.Save(ctx, string(wf.UID), wf.Namespace, wf.Status.Nodes)
	if err != nil {
		return err
	}
	wf.Status.Nodes = nil
	wf.Status.CompressedNodes = ""
	wf.Status.OffloadNodeStatusVersion = version
	return nil
}
