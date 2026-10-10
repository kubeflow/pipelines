// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package util_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithLogger_ContextIsolationAndWrites(t *testing.T) {
	parent, cancel := context.WithCancel(context.Background())
	defer cancel()
	assert.Nil(t, util.GetLoggerFrom(parent))

	firstPath := filepath.Join(t.TempDir(), "first.log")
	first, firstFile, err := util.WithLogger(parent, firstPath)
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, firstFile.Close()) })
	secondPath := filepath.Join(t.TempDir(), "second.log")
	second, secondFile, err := util.WithLogger(parent, secondPath)
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, secondFile.Close()) })

	firstLogger := util.GetLoggerFrom(first)
	secondLogger := util.GetLoggerFrom(second)
	require.NotNil(t, firstLogger)
	require.NotNil(t, secondLogger)
	assert.NotSame(t, firstLogger, secondLogger)
	assert.Nil(t, util.GetLoggerFrom(parent))
	child, childCancel := context.WithCancel(first)
	defer childCancel()
	assert.Same(t, firstLogger, util.GetLoggerFrom(child))

	firstLogger.Info("first request only")
	secondLogger.Info("second request only")
	firstContents, err := os.ReadFile(firstPath)
	require.NoError(t, err)
	assert.Contains(t, string(firstContents), "first request only")
	assert.NotContains(t, string(firstContents), "second request only")
	secondContents, err := os.ReadFile(secondPath)
	require.NoError(t, err)
	assert.Contains(t, string(secondContents), "second request only")
	assert.NotContains(t, string(secondContents), "first request only")

	cancel()
	assert.ErrorIs(t, first.Err(), context.Canceled)
	assert.ErrorIs(t, second.Err(), context.Canceled)
}

func TestWithLogger_ExistingLogger(t *testing.T) {
	path := filepath.Join(t.TempDir(), "driver.log")
	ctx, logFile, err := util.WithLogger(context.Background(), path)
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, logFile.Close()) })
	logger := util.GetLoggerFrom(ctx)
	logger.Info("keep existing log")

	duplicate, duplicateFile, err := util.WithLogger(ctx, path)
	require.ErrorContains(t, err, "reuse it with GetLoggerFrom")
	assert.Nil(t, duplicate)
	assert.Nil(t, duplicateFile)
	assert.Same(t, logger, util.GetLoggerFrom(ctx))
	contents, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(contents), "keep existing log")
}

func TestWithLogger_NilContext(t *testing.T) {
	path := filepath.Join(t.TempDir(), "driver.log")
	ctx, logFile, err := util.WithLogger(nil, path)
	require.ErrorContains(t, err, "provide a non-nil context")
	assert.Nil(t, ctx)
	assert.Nil(t, logFile)
	_, err = os.Stat(path)
	assert.ErrorIs(t, err, os.ErrNotExist)
}

func TestWithLogger_CreationFailures(t *testing.T) {
	for _, test := range []struct {
		name string
		path string
	}{
		{name: "missing parent directory", path: filepath.Join(t.TempDir(), "missing", "driver.log")},
		{name: "path is directory", path: t.TempDir()},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx, logFile, err := util.WithLogger(context.Background(), test.path)
			require.ErrorContains(t, err, "choose a writable file path with an existing parent directory")
			assert.Contains(t, err.Error(), test.path)
			assert.Nil(t, ctx)
			assert.Nil(t, logFile)
			var pathErr *os.PathError
			assert.ErrorAs(t, err, &pathErr)
			if test.name == "missing parent directory" {
				assert.ErrorIs(t, err, os.ErrNotExist)
			}
		})
	}
}
