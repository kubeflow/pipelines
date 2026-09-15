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

// createExactSizePaxBomb creates a valid tar.gz archive whose uncompressed size is exactly exactTarSizeBytes.
// It uses a valid local TypeXHeader (PAX) record which archive/tar.Reader.Next() consumes internally.
func createExactSizePaxBomb(t *testing.T, targetName string, targetContent string, exactTarSizeBytes int64) []byte {
	t.Helper()
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	targetContentLen := int64(len(targetContent))
	targetBlocks := (targetContentLen + 511) / 512

	// Base tar size: Target Header (1 block) + Target Body + 2 EOF blocks
	baseBlocks := 1 + targetBlocks + 2
	baseBytes := baseBlocks * 512

	var paxRecords map[string]string
	if exactTarSizeBytes > 0 {
		require.True(t, exactTarSizeBytes >= baseBytes+512, "exactTarSizeBytes too small for PAX header")
		require.Equal(t, int64(0), exactTarSizeBytes%512, "exactTarSizeBytes must be multiple of 512")

		paxBytesNeeded := exactTarSizeBytes - baseBytes
		// PAX Header is 1 block. The rest is PAX Body.
		paxPayloadBlocks := (paxBytesNeeded / 512) - 1
		paxPayloadBytes := paxPayloadBlocks * 512

		var paxValue string
		for valLen := int(paxPayloadBytes) - 20; valLen <= int(paxPayloadBytes); valLen++ {
			s := fmt.Sprintf("%d comment=%s\n", paxPayloadBytes, bytes.Repeat([]byte("x"), valLen))
			if len(s) == int(paxPayloadBytes) {
				paxValue = string(bytes.Repeat([]byte("x"), valLen))
				break
			}
		}
		require.NotEmpty(t, paxValue, "Could not perfectly size the PAX payload")
		paxRecords = map[string]string{"comment": paxValue}
	}

	require.NoError(t, tw.WriteHeader(&tar.Header{
		Typeflag:   tar.TypeReg,
		Name:       targetName,
		Mode:       0600,
		Size:       targetContentLen,
		PAXRecords: paxRecords,
	}))
	_, err := tw.Write([]byte(targetContent))
	require.NoError(t, err)
	require.NoError(t, tw.Close())

	if exactTarSizeBytes > 0 {
		require.Equal(t, exactTarSizeBytes, int64(buf.Len()), "Tarball uncompressed size must be exact")
	}

	var gzBuf bytes.Buffer
	gw := gzip.NewWriter(&gzBuf)
	_, err = gw.Write(buf.Bytes())
	require.NoError(t, err)
	require.NoError(t, gw.Close())
	return gzBuf.Bytes()
}

// TestReadSingleFileFromTgz_TraversalBudgetExhaustion verifies that a
// tar.gz whose PAX metadata causes attacker-controlled unbounded work inside
// tar.Reader.Next() is rejected before the persistence agent exhausts memory.
func TestReadSingleFileFromTgz_TraversalBudgetExhaustion(t *testing.T) {
	const maxFileSize int64 = 1024
	budget := ArchiveTraversalBudget(maxFileSize)

	tgz := createExactSizePaxBomb(t, "metrics.json", "content", budget+512)

	err := readSingleFileFromTgz(tgz, maxFileSize, func(r io.Reader) error {
		return nil
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "traversal exceeded budget")
}

// TestReadSingleFileFromTgz_TraversalBudgetBoundary verifies that the exact boundary
// (budget bytes consumed) passes, but budget+1 fails. The +1 check ensures the
// sentinel logic correctly differentiates exact budget EOF from exhaustion, so
// exactly-sized edge cases (such as exact uncompressed limits) aren't falsely
// removed without breaking tests.
func TestReadSingleFileFromTgz_TraversalBudgetBoundary(t *testing.T) {
	const maxFileSize int64 = 1024
	budget := ArchiveTraversalBudget(maxFileSize)

	t.Run("exact file size accepted", func(t *testing.T) {
		content := string(bytes.Repeat([]byte("a"), int(maxFileSize)))
		tgz := createExactSizePaxBomb(t, "metrics.json", content, budget)
		err := readSingleFileFromTgz(tgz, maxFileSize, func(r io.Reader) error {
			return nil
		})
		require.NoError(t, err)
	})

	t.Run("budget+1 PAX metadata rejected", func(t *testing.T) {
		tgz := createExactSizePaxBomb(t, "metrics.json", "content", budget+512)
		err := readSingleFileFromTgz(tgz, maxFileSize, func(r io.Reader) error {
			return nil
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "traversal exceeded budget")
	})
}

// TestArchiveWireResponseBudget_GzipFlushRegression demonstrates that pathological
// valid gzip streams containing excessive empty flushes can exceed the independent
// finite wire limit calculated by ArchiveWireResponseBudget. The persistence agent
// will correctly truncate such pathological responses.
func TestArchiveWireResponseBudget_GzipFlushRegression(t *testing.T) {
	const maxFileSize = 4096
	budget := ArchiveWireResponseBudget(maxFileSize)

	var buf bytes.Buffer
	gw, err := gzip.NewWriterLevel(&buf, gzip.NoCompression)
	require.NoError(t, err)
	tw := tar.NewWriter(gw)

	content := bytes.Repeat([]byte("a"), maxFileSize)
	require.NoError(t, tw.WriteHeader(&tar.Header{
		Typeflag: tar.TypeReg,
		Name:     "pipeline.yaml",
		Size:     int64(len(content)),
	}))
	_, err = tw.Write(content)
	require.NoError(t, err)
	require.NoError(t, tw.Close())

	// Write 300,000 empty DEFLATE blocks via excessive flushes.
	// This reliably exceeds the conservative wire budget (which is ~1.5MB for a 4KB file).
	for i := 0; i < 300000; i++ {
		require.NoError(t, gw.Flush())
	}
	require.NoError(t, gw.Close())

	// Compute the simulated wire response size (base64 + JSON framing).
	maxBase64 := ((int64(buf.Len()) + 2) / 3) * 4
	jsonBodyLen := maxBase64 + 11 // {"data":""}

	// Assert that this pathological gzip stream legitimately exceeds our independent wire cap.
	assert.Greater(t, jsonBodyLen, budget, "Pathological gzip stream should exceed the independent wire budget")
}
