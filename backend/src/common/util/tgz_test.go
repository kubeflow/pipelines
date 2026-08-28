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
	"fmt"
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

func rewriteFirstTarEntryTypeflag(t *testing.T, tgzContent []byte, typeflag byte) []byte {
	t.Helper()

	gzipReader, err := gzip.NewReader(bytes.NewReader(tgzContent))
	require.NoError(t, err)
	tarContent, err := io.ReadAll(gzipReader)
	require.NoError(t, err)
	require.NoError(t, gzipReader.Close())

	const (
		tarBlockSize   = 512
		checksumOffset = 148
		checksumLength = 8
		typeflagOffset = 156
	)
	require.GreaterOrEqual(t, len(tarContent), tarBlockSize)
	tarContent[typeflagOffset] = typeflag
	for index := checksumOffset; index < checksumOffset+checksumLength; index++ {
		tarContent[index] = ' '
	}
	checksum := 0
	for _, value := range tarContent[:tarBlockSize] {
		checksum += int(value)
	}
	copy(tarContent[checksumOffset:checksumOffset+checksumLength], fmt.Sprintf("%06o\x00 ", checksum))

	var rewritten bytes.Buffer
	gzipWriter := gzip.NewWriter(&rewritten)
	_, err = gzipWriter.Write(tarContent)
	require.NoError(t, err)
	require.NoError(t, gzipWriter.Close())
	return rewritten.Bytes()
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

func TestReadSingleFileFromTgz_AcceptsLegacyRegularEntry(t *testing.T) {
	const legacyRegularFileTypeflag byte = '\x00'

	tgzContent := createTestTgz(t, []testTgzEntry{{
		name:     "metrics.json",
		content:  "content",
		typeflag: tar.TypeReg,
	}})
	tgzContent = rewriteFirstTarEntryTypeflag(t, tgzContent, legacyRegularFileTypeflag)

	var content []byte
	err := readSingleFileFromTgz(tgzContent, 7, func(reader io.Reader) error {
		var readError error
		content, readError = io.ReadAll(reader)
		return readError
	})

	require.NoError(t, err)
	assert.Equal(t, "content", string(content))
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

// createPAXBombTgz returns a tgz whose first entry is a small regular file
// followed by a PAX extended-header entry with a value field that exceeds
// budgetBytes. tar.Reader.Next() fully decompresses PAX blocks internally,
// so the total decompressed work exceeds the traversal budget even though
// the compressed archive is tiny — proving compression amplification.
func createPAXBombTgz(t *testing.T, metricsContent string, budgetBytes int64) []byte {
	t.Helper()
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	require.NoError(t, tw.WriteHeader(&tar.Header{
		Typeflag: tar.TypeReg,
		Name:     "metrics.json",
		Mode:     0600,
		Size:     int64(len(metricsContent)),
	}))
	_, err := tw.Write([]byte(metricsContent))
	require.NoError(t, err)

	// A PAX extended-header whose "comment" value exceeds the traversal budget.
	// tar.Reader.Next() must decompress the entire PAX block to parse it.
	paxValue := string(bytes.Repeat([]byte("x"), int(budgetBytes+1)))
	require.NoError(t, tw.WriteHeader(&tar.Header{
		Typeflag: tar.TypeXHeader,
		Name:     "pax-bomb",
		Size:     int64(len(paxValue)),
		PAXRecords: map[string]string{
			"comment": paxValue,
		},
	}))
	_, err = tw.Write([]byte(paxValue))
	require.NoError(t, err)

	require.NoError(t, tw.Close())
	require.NoError(t, gw.Close())
	return buf.Bytes()
}

// TestReadSingleFileFromTgz_TraversalBudgetExhaustion verifies that a
// tar.gz whose PAX metadata causes attacker-controlled unbounded work inside
// tar.Reader.Next() is rejected before the persistence agent exhausts memory.
func TestReadSingleFileFromTgz_TraversalBudgetExhaustion(t *testing.T) {
	const maxFileSize int64 = 1024
	budget := ArchiveTraversalBudget(maxFileSize)

	tgz := createPAXBombTgz(t, "content", budget)

	// Confirm compression amplification: the archive must be smaller than
	// maxFileSize even though it decompresses to more than the budget.
	require.Less(t, int64(len(tgz)), maxFileSize,
		"bomb fixture must compress below maxFileSize to prove amplification")

	err := readSingleFileFromTgz(tgz, maxFileSize, func(r io.Reader) error {
		_, err := io.Copy(io.Discard, r)
		return err
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "traversal exceeded budget")
}

// TestReadSingleFileFromTgz_TraversalBudgetBoundary verifies exact-budget
// acceptance and budget+1 rejection so the sentinel byte cannot be silently
// removed without breaking tests.
func TestReadSingleFileFromTgz_TraversalBudgetBoundary(t *testing.T) {
	const maxFileSize int64 = 1024
	budget := ArchiveTraversalBudget(maxFileSize)

	t.Run("exact file size accepted", func(t *testing.T) {
		content := string(bytes.Repeat([]byte("a"), int(maxFileSize)))
		tgz := createTestTgz(t, []testTgzEntry{{name: "metrics.json", content: content}})
		err := readSingleFileFromTgz(tgz, maxFileSize, func(r io.Reader) error {
			_, err := io.Copy(io.Discard, r)
			return err
		})
		assert.NoError(t, err)
	})

	t.Run("budget+1 PAX metadata rejected", func(t *testing.T) {
		tgz := createPAXBombTgz(t, "small", budget)
		err := readSingleFileFromTgz(tgz, maxFileSize, func(r io.Reader) error {
			_, err := io.Copy(io.Discard, r)
			return err
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "traversal exceeded budget")
	})
}
