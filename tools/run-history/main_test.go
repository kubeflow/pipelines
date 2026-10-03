// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/kubeflow/pipelines/backend/src/apiserver/history"
	"github.com/stretchr/testify/require"
)

func TestArchiveFileBoundary(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.json")
	original := &history.Bundle{Format: history.Format, Source: "old"}
	require.NoError(t, writeArchive(path, original))
	stat, err := os.Stat(path)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0600), stat.Mode().Perm())
	require.Error(t, writeArchive(path, &history.Bundle{Source: "overwrite"}))
	got, err := readArchive(path)
	require.NoError(t, err)
	require.Equal(t, original.Source, got.Source)
	for _, input := range []string{`{"Unknown":true}`, `{} {}`, `null trailing`, `{"Entries":`} {
		require.NoError(t, os.WriteFile(path, []byte(input), 0600))
		_, err := readArchive(path)
		require.Error(t, err)
	}
}

func TestHelpDoesNotRequireDatabase(t *testing.T) {
	t.Setenv("KFP_HISTORY_DSN", "")
	require.NoError(t, run(context.Background(), []string{"export", "--help"}, io.Discard))
	require.NoError(t, run(context.Background(), []string{"import", "--help"}, io.Discard))
	require.Error(t, run(context.Background(), []string{"import", "--file", "history.json"}, io.Discard))
}
