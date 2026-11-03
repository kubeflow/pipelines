// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package clientmanager

import (
	"fmt"
	"testing"

	"github.com/kubeflow/pipelines/backend/src/apiserver/storage"
	"github.com/stretchr/testify/require"
)

type migrationInventoryStore struct {
	afterIDs           []string
	fail               bool
	failAfterFirstPage bool
}

func (s *migrationInventoryStore) ListJobsWithoutRecurringRunState(afterID string, limit uint64) ([]storage.RecurringRunMigrationCandidate, error) {
	s.afterIDs = append(s.afterIDs, afterID)
	if s.fail || (s.failAfterFirstPage && afterID != "") {
		return nil, fmt.Errorf("database unavailable")
	}
	if afterID != "" {
		return []storage.RecurringRunMigrationCandidate{{ID: "job-100", Namespace: "n1", Name: "swf-100"}}, nil
	}
	candidates := make([]storage.RecurringRunMigrationCandidate, limit)
	for i := range candidates {
		candidates[i] = storage.RecurringRunMigrationCandidate{ID: fmt.Sprintf("job-%03d", i), Namespace: "n1", Name: "swf", Enabled: true}
	}
	return candidates, nil
}

func TestReportRecurringRunMigration(t *testing.T) {
	store := &migrationInventoryStore{}
	var warnings []string
	err := reportRecurringRunMigration(store, func(format string, args ...any) { warnings = append(warnings, fmt.Sprintf(format, args...)) })
	require.NoError(t, err)
	require.Equal(t, []string{"", "job-099"}, store.afterIDs)
	require.Len(t, warnings, 102)
	require.Contains(t, warnings[0], `id="job-000"`)
	require.Contains(t, warnings[100], `id="job-100"`)
	require.Contains(t, warnings[100], "enabled=false")
	require.Contains(t, warnings[101], "affected_jobs=101")
	store.fail = true
	err = reportRecurringRunMigration(store, func(string, ...any) { t.Fatal("incomplete inventory must not report success") })
	require.ErrorContains(t, err, "database unavailable")
}

func TestReportRecurringRunMigrationPartialInventoryFailure(t *testing.T) {
	store := &migrationInventoryStore{failAfterFirstPage: true}
	var warnings []string
	err := reportRecurringRunMigration(store, func(format string, args ...any) { warnings = append(warnings, fmt.Sprintf(format, args...)) })
	require.ErrorContains(t, err, "database unavailable")
	require.Len(t, warnings, 100)
	for _, warning := range warnings {
		require.NotContains(t, warning, "affected_jobs=")
	}
}
