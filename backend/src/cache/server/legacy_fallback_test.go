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
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/kubeflow/pipelines/backend/src/cache/model"
	"github.com/kubeflow/pipelines/backend/src/cache/storage"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/require"
)

func legacyCacheManager(t *testing.T) *FakeClientManager {
	t.Helper()
	m, err := NewFakeClientManager(util.NewFakeTime(time.Unix(100, 0)))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, m.Close()) })
	return m
}

func legacyCacheRow(t *testing.T) *model.ExecutionCache {
	t.Helper()
	template, ok := getArgoTemplate(fakePod)
	require.True(t, ok)
	key, err := generateLegacyCacheKeyFromTemplate(template)
	require.NoError(t, err)
	// Pin the pre-upgrade hash so the fallback cannot silently use the new format.
	require.Equal(t, "07f2c42567af4f141a52887e0a113c9aefdfdfd5e6b06b9908f7fdb0b43739af", key)
	output, err := json.Marshal(map[string]string{ArgoWorkflowOutputs: cachePod("tenant").Annotations[ArgoWorkflowOutputs]})
	require.NoError(t, err)
	return &model.ExecutionCache{
		ExecutionCacheKey: key,
		ExecutionTemplate: template,
		ExecutionOutput:   string(output),
		MaxCacheStaleness: -1,
		StartedAtInSec:    1,
		EndedAtInSec:      2,
	}
}

func TestLegacyCacheFallbackAdmission(t *testing.T) {
	for _, tc := range []struct {
		name         string
		setting      string
		namespace    string
		podTTL       string
		defaultTTL   string
		maximumTTL   string
		rowExpired   bool
		wantHit      bool
		wantRowCount int64
	}{
		{name: "unset by default", wantRowCount: 1},
		{name: "explicitly disabled", setting: "false", wantRowCount: 1},
		{name: "legacy hit", setting: "true", wantHit: true, wantRowCount: 1},
		{name: "another namespace excluded", setting: "true", namespace: "other", wantRowCount: 1},
		{name: "pod caching disabled", setting: "true", podTTL: "P0D", wantRowCount: 1},
		{name: "default caching disabled", setting: "true", defaultTTL: "P0D", wantRowCount: 1},
		{name: "pod TTL expired", setting: "true", podTTL: "PT1S", wantRowCount: 1},
		{name: "default TTL expired", setting: "true", defaultTTL: "PT1S", wantRowCount: 1},
		{name: "row TTL expired", setting: "true", rowExpired: true, wantRowCount: 1},
		{name: "maximum TTL expired", setting: "true", maximumTTL: "PT1S"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("ALLOW_LEGACY_CACHE_FALLBACK", tc.setting)
			if tc.setting == "" {
				require.NoError(t, os.Unsetenv("ALLOW_LEGACY_CACHE_FALLBACK"))
			}
			t.Setenv("DEFAULT_CACHE_STALENESS", tc.defaultTTL)
			t.Setenv("MAXIMUM_CACHE_STALENESS", tc.maximumTTL)
			// The fallback setting must be independent of this existing boolean option.
			t.Setenv("CACHE_NODE_RESTRICTIONS", "true")
			m := legacyCacheManager(t)
			row := legacyCacheRow(t)
			row.Namespace = tc.namespace
			if tc.rowExpired {
				row.MaxCacheStaleness = 1
			}
			require.NoError(t, m.DB().Create(row).Error)
			p := cachePod("tenant")
			if tc.podTTL != "" {
				p.Annotations[MaxCacheStalenessKey] = tc.podTTL
			}
			require.Equal(t, tc.wantHit, admitCachePod(t, p, m))
			if tc.wantHit {
				require.Equal(t, getValueFromSerializedMap(row.ExecutionOutput, ArgoWorkflowOutputs), p.Annotations[ArgoWorkflowOutputs])
				// A legacy hit is not promoted or given a fresh age by the watcher.
				require.NoError(t, cacheCompletedPod(context.Background(), p, m))
				var retained model.ExecutionCache
				require.NoError(t, m.DB().First(&retained, row.ID).Error)
				require.Equal(t, *row, retained)
			}
			var rows int64
			require.NoError(t, m.DB().Model(&model.ExecutionCache{}).Count(&rows).Error)
			require.Equal(t, tc.wantRowCount, rows)
		})
	}
}

func TestLegacyCacheFallbackPreservesScopedWritesAndHitPrecedence(t *testing.T) {
	m := legacyCacheManager(t)
	legacy := legacyCacheRow(t)
	require.NoError(t, m.DB().Create(legacy).Error)
	t.Setenv("ALLOW_LEGACY_CACHE_FALLBACK", "false")
	p := cachePod("tenant")
	require.False(t, admitCachePod(t, p, m))

	t.Setenv("ALLOW_LEGACY_CACHE_FALLBACK", "true")
	t.Setenv("CACHE_NODE_RESTRICTIONS", "false")
	p.Annotations[ArgoWorkflowOutputs] = `{"parameters":[{"name":"result","value":"scoped-output"}]}`
	require.NoError(t, cacheCompletedPod(context.Background(), p, m))
	next := cachePod("tenant")
	require.True(t, admitCachePod(t, next, m))
	require.Equal(t, p.Annotations[ArgoWorkflowOutputs], next.Annotations[ArgoWorkflowOutputs])
	var rows []model.ExecutionCache
	require.NoError(t, m.DB().Order("ID").Find(&rows).Error)
	require.Len(t, rows, 2)
	require.Equal(t, *legacy, rows[0])
	require.Equal(t, "tenant", rows[1].Namespace)
	require.NotEqual(t, legacy.ExecutionCacheKey, rows[1].ExecutionCacheKey)
}

func TestLegacyCacheFallbackInvalidConfiguration(t *testing.T) {
	t.Setenv("ALLOW_LEGACY_CACHE_FALLBACK", "invalid")
	m := legacyCacheManager(t)
	require.NoError(t, m.DB().Create(legacyCacheRow(t)).Error)
	p := cachePod("tenant")
	req := GetFakeRequestFromPod(p)
	req.Namespace = p.Namespace
	patches, err := MutatePodIfCached(req, m)
	require.ErrorContains(t, err, "ALLOW_LEGACY_CACHE_FALLBACK")
	require.Empty(t, patches)
}

type scopedLookupErrorStore struct {
	storage.ExecutionCacheStoreInterface
	lookupErr   error
	legacyReads int
}

func (s *scopedLookupErrorStore) GetExecutionCache(string, string, int64, int64) (*model.ExecutionCache, error) {
	return nil, s.lookupErr
}

func (s *scopedLookupErrorStore) GetLegacyExecutionCache(key string, staleness, maximumStaleness int64) (*model.ExecutionCache, error) {
	s.legacyReads++
	return s.ExecutionCacheStoreInterface.GetLegacyExecutionCache(key, staleness, maximumStaleness)
}

func TestLegacyCacheFallbackOnlyAfterCacheMiss(t *testing.T) {
	t.Setenv("ALLOW_LEGACY_CACHE_FALLBACK", "true")
	for _, tc := range []struct {
		name    string
		err     error
		wantHit bool
	}{
		{"database read error", errors.New("database read failed"), false},
		{"database cleanup error", errors.New("database cleanup failed"), false},
		{"wrapped cache miss", fmt.Errorf("lookup: %w", storage.ErrExecutionCacheNotFound), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := legacyCacheManager(t)
			require.NoError(t, m.DB().Create(legacyCacheRow(t)).Error)
			s := &scopedLookupErrorStore{ExecutionCacheStoreInterface: m.CacheStore(), lookupErr: tc.err}
			m.cacheStore = s
			require.Equal(t, tc.wantHit, admitCachePod(t, cachePod("tenant"), m))
			if tc.wantHit {
				require.Equal(t, 1, s.legacyReads)
			} else {
				require.Zero(t, s.legacyReads)
			}
		})
	}
}
