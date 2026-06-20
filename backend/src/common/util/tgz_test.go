// Copyright 2018 The Kubeflow Authors
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
	"archive/tar"
	"bytes"
	"compress/gzip"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testTgzEntry struct {
	name     string
	content  string
	typeflag byte
}

func createTestTgz(t *testing.T, entries []testTgzEntry) []byte {
	t.Helper()

	var buf bytes.Buffer
	gzipWriter := gzip.NewWriter(&buf)
	tarWriter := tar.NewWriter(gzipWriter)
	for _, entry := range entries {
		err := tarWriter.WriteHeader(&tar.Header{
			Typeflag: entry.typeflag,
			Name:     entry.name,
			Size:     int64(len(entry.content)),
		})
		require.NoError(t, err)
		_, err = tarWriter.Write([]byte(entry.content))
		require.NoError(t, err)
	}
	require.NoError(t, tarWriter.Close())
	require.NoError(t, gzipWriter.Close())

	return buf.Bytes()
}

func TestArchiveTgzAndReadSingleFileFromTgz_Roundtrip(t *testing.T) {
	tgzContent, err := ArchiveTgz(map[string]string{"metrics.json": "content"})
	require.NoError(t, err)

	var content []byte
	err = readSingleFileFromTgz([]byte(tgzContent), 7, func(reader io.Reader) error {
		content, err = io.ReadAll(reader)
		return err
	})

	require.NoError(t, err)
	assert.Equal(t, "content", string(content))
}

func TestReadSingleFileFromTgz_RejectsInvalidArchive(t *testing.T) {
	err := readSingleFileFromTgz([]byte("not a valid tgz"), 1024, func(io.Reader) error {
		return nil
	})

	assert.Error(t, err)
}

func TestReadSingleFileFromTgz_RequiresOneEntry(t *testing.T) {
	testCases := []struct {
		name          string
		entries       []testTgzEntry
		errorContains string
	}{
		{
			name:          "empty archive",
			entries:       nil,
			errorContains: "metrics archive must contain exactly one regular file",
		},
		{
			name: "multiple files",
			entries: []testTgzEntry{
				{name: "first.json", content: "first"},
				{name: "second.json", content: "second"},
			},
			errorContains: "metrics archive must contain exactly one regular file",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			tgzContent := createTestTgz(t, testCase.entries)
			err := readSingleFileFromTgz(tgzContent, 1024, func(reader io.Reader) error {
				_, err := io.Copy(io.Discard, reader)
				return err
			})

			assert.ErrorContains(t, err, testCase.errorContains)
		})
	}
}

func TestReadSingleFileFromTgz_RejectsNonRegularEntry(t *testing.T) {
	tgzContent := createTestTgz(t, []testTgzEntry{{name: "metrics", typeflag: tar.TypeDir}})

	err := readSingleFileFromTgz(tgzContent, 1024, func(io.Reader) error {
		return nil
	})

	assert.ErrorContains(t, err, `metrics archive entry "metrics" must be a regular file`)
}

func TestReadSingleFileFromTgz_EnforcesConfigurableByteLimit(t *testing.T) {
	testCases := []struct {
		name          string
		maxBytes      int64
		errorContains string
	}{
		{name: "exact limit", maxBytes: 7},
		{name: "over limit", maxBytes: 6, errorContains: `metrics archive entry "metrics.json" exceeds maximum size of 6 bytes`},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			tgzContent := createTestTgz(t, []testTgzEntry{{name: "metrics.json", content: "content"}})
			err := readSingleFileFromTgz(tgzContent, testCase.maxBytes, func(reader io.Reader) error {
				_, err := io.Copy(io.Discard, reader)
				return err
			})

			if testCase.errorContains == "" {
				assert.NoError(t, err)
			} else {
				assert.ErrorContains(t, err, testCase.errorContains)
			}
		})
	}
}

func TestReadSingleFileFromTgz_PropagatesConsumerError(t *testing.T) {
	tgzContent := createTestTgz(t, []testTgzEntry{{name: "metrics.json", content: "content"}})
	expectedError := errors.New("decode failed")

	err := readSingleFileFromTgz(tgzContent, 1024, func(io.Reader) error {
		return expectedError
	})

	assert.ErrorIs(t, err, expectedError)
}
