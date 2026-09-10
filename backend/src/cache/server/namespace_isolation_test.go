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

	"github.com/kubeflow/pipelines/backend/src/cache/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

func namespaceCacheManager(t *testing.T) *FakeClientManager {
	t.Helper()
	m := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	t.Cleanup(func() { require.NoError(t, m.Close()) })
	return m
}

func cachePod(namespace string) *corev1.Pod {
	p := fakePod.DeepCopy()
	p.Namespace = namespace
	p.Name = "cache-step"
	p.Labels[CacheIDLabelKey] = ""
	p.Status.Phase = corev1.PodSucceeded
	p.Annotations[ArgoWorkflowOutputs] = `{"parameters":[{"name":"result","value":"tenant-output"}]}`
	return p
}

// admitCachePod uses the production admission handler and applies the metadata patches.
func admitCachePod(t *testing.T, p *corev1.Pod, m *FakeClientManager) bool {
	t.Helper()
	req := GetFakeRequestFromPod(p)
	req.Namespace = p.Namespace
	patches, err := MutatePodIfCached(req, m)
	require.NoError(t, err)
	hit := false
	for _, patch := range patches {
		switch patch.Path {
		case AnnotationPath:
			p.Annotations = patch.Value.(map[string]string)
		case LabelPath:
			p.Labels = patch.Value.(map[string]string)
		}
		if patch.Op == OperationTypeReplace {
			hit = true
		}
	}
	return hit
}

func TestCacheNamespaceIsolationThroughWatcherAndAdmission(t *testing.T) {
	m := namespaceCacheManager(t)
	attacker := cachePod("attacker")
	victim := cachePod("victim")
	require.False(t, admitCachePod(t, attacker, m))
	require.False(t, admitCachePod(t, victim, m))
	attackerKey, victimKey := attacker.Annotations[ExecutionKey], victim.Annotations[ExecutionKey]
	require.NotEmpty(t, attackerKey)
	require.NotEqual(t, attackerKey, victimKey)

	// A tenant changes its mutable key to the victim's deterministic hash.
	attacker.Annotations[ExecutionKey] = victimKey
	require.ErrorContains(t, cacheCompletedPod(context.Background(), attacker, m), "does not match")
	var rows int64
	require.NoError(t, m.DB().Model(&model.ExecutionCache{}).Count(&rows).Error)
	require.Zero(t, rows)
	require.False(t, admitCachePod(t, cachePod("victim"), m))

	// Normal completion may populate only the attacker's namespace.
	attacker.Annotations[ExecutionKey] = attackerKey
	require.NoError(t, cacheCompletedPod(context.Background(), attacker, m))
	require.NotEmpty(t, attacker.Labels[CacheIDLabelKey])
	require.True(t, admitCachePod(t, cachePod("attacker"), m))
	require.False(t, admitCachePod(t, cachePod("victim"), m))

	// Victim completion becomes reusable within the victim namespace.
	victim.Annotations[ArgoWorkflowOutputs] = `{"parameters":[{"name":"result","value":"victim-output"}]}`
	require.NoError(t, cacheCompletedPod(context.Background(), victim, m))
	nextVictim := cachePod("victim")
	require.True(t, admitCachePod(t, nextVictim, m))
	require.Equal(t, victim.Annotations[ArgoWorkflowOutputs], nextVictim.Annotations[ArgoWorkflowOutputs])
}

func TestCacheWatcherSkipsInvalidIdentityAndExcludedPods(t *testing.T) {
	for _, tc := range []struct {
		name      string
		mutate    func(*corev1.Pod)
		wantError bool
	}{
		{"missing namespace", func(p *corev1.Pod) { p.Namespace = "" }, true},
		{"missing key", func(p *corev1.Pod) { delete(p.Annotations, ExecutionKey) }, true},
		{"malformed template", func(p *corev1.Pod) { p.Spec.Containers[0].Env[0].Value = "{" }, true},
		{"missing template", func(p *corev1.Pod) { p.Spec.Containers[0].Env = nil }, false},
		{"no containers", func(p *corev1.Pod) { p.Spec.Containers = nil }, false},
		{"cache disabled", func(p *corev1.Pod) { delete(p.Labels, KFPCacheEnabledLabelKey) }, false},
		{"v2 pod", func(p *corev1.Pod) { p.Annotations[V2ComponentAnnotationKey] = V2ComponentAnnotationValue }, false},
		{"tfx pod", func(p *corev1.Pod) { p.Labels[SdkTypeLabel] = TfxSdkTypeLabel }, false},
		{"unfinished pod", func(p *corev1.Pod) { p.Status.Phase = corev1.PodRunning }, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := namespaceCacheManager(t)
			p := cachePod("tenant")
			require.False(t, admitCachePod(t, p, m))
			tc.mutate(p)
			err := cacheCompletedPod(context.Background(), p, m)
			if tc.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			var rows int64
			require.NoError(t, m.DB().Model(&model.ExecutionCache{}).Count(&rows).Error)
			require.Zero(t, rows)
		})
	}
}

func TestCacheAdmissionSkipsUnknownNamespaceAndMalformedTemplate(t *testing.T) {
	for _, tc := range []struct{ name, requestNamespace, podNamespace, template string }{
		{"missing admission namespace", "", "tenant", "{}"},
		{"namespace mismatch", "victim", "attacker", "{}"},
		{"malformed template", "tenant", "tenant", "{"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := namespaceCacheManager(t)
			p := cachePod(tc.podNamespace)
			p.Spec.Containers[0].Env[0].Value = tc.template
			req := GetFakeRequestFromPod(p)
			req.Namespace = tc.requestNamespace
			patches, err := MutatePodIfCached(req, m)
			require.NoError(t, err)
			require.Empty(t, patches)
		})
	}
}

func TestCacheRejectsWrongShapedTemplates(t *testing.T) {
	for _, template := range []string{
		`null`, `[]`, `"template"`,
		`{"container":null}`, `{"container":[]}`, `{"container":"image"}`,
		`{"container":42}`, `{"container":true}`,
	} {
		t.Run(template, func(t *testing.T) {
			m := namespaceCacheManager(t)
			p := cachePod("tenant")
			p.Spec.Containers[0].Env[0].Value = template
			key, err := generateCacheKeyFromTemplate(template, p.Namespace)
			require.Error(t, err)
			require.Empty(t, key)
			req := GetFakeRequestFromPod(p)
			req.Namespace = p.Namespace
			patches, err := MutatePodIfCached(req, m)
			require.NoError(t, err)
			require.Empty(t, patches)
			require.Error(t, cacheCompletedPod(context.Background(), p, m))
			var rows int64
			require.NoError(t, m.DB().Model(&model.ExecutionCache{}).Count(&rows).Error)
			require.Zero(t, rows)
		})
	}
}
