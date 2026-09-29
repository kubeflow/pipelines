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
	"sync"
	"sync/atomic"
	"testing"

	workflowapi "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/workflow/hydrator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
			}
			_ = lazy.IsHydrated(wf)
		}()
	}
	close(start)
	wg.Wait()
	assert.Equal(t, int32(1), initCalls.Load())
}
