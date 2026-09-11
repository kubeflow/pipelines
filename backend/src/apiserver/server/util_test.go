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

package server

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createZipArchive(t *testing.T, content []byte) []byte {
	t.Helper()

	var buffer bytes.Buffer
	zipWriter := zip.NewWriter(&buffer)
	fileWriter, err := zipWriter.Create("pipeline.yaml")
	require.NoError(t, err)
	_, err = fileWriter.Write(content)
	require.NoError(t, err)
	require.NoError(t, zipWriter.Close())
	return buffer.Bytes()
}

func createTarGzArchive(t *testing.T, content []byte) []byte {
	t.Helper()

	var buffer bytes.Buffer
	gzipWriter := gzip.NewWriter(&buffer)
	tarWriter := tar.NewWriter(gzipWriter)
	require.NoError(t, tarWriter.WriteHeader(&tar.Header{
		Name: "pipeline.yaml",
		Mode: 0600,
		Size: int64(len(content)),
	}))
	_, err := tarWriter.Write(content)
	require.NoError(t, err)
	require.NoError(t, tarWriter.Close())
	require.NoError(t, gzipWriter.Close())
	return buffer.Bytes()
}

func TestLoadFile(t *testing.T) {
	file := "12345"
	bytes, err := loadFile(strings.NewReader(file), 5)
	assert.Nil(t, err)
	assert.Equal(t, []byte(file), bytes)
}

func TestLoadFile_ExceedSizeLimit(t *testing.T) {
	file := "12345"
	_, err := loadFile(strings.NewReader(file), 4)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "File size too large")
}

func TestLoadFile_LargeDoc(t *testing.T) {
	bytes, _ := os.ReadFile("test/xgboost_sample_pipeline.yaml")
	file := string(bytes)
	readBytes, err := loadFile(strings.NewReader(file), common.MaxFileLength)
	assert.Nil(t, err)
	assert.Equal(t, bytes, readBytes)
}

func TestDecompressPipelineTarball(t *testing.T) {
	tarballByte, _ := os.ReadFile("test/arguments_tarball/arguments.tar.gz")
	pipelineFile, err := DecompressPipelineTarball(tarballByte)
	assert.Nil(t, err)

	expectedPipelineFile, _ := os.ReadFile("test/arguments-parameters.yaml")
	assert.Equal(t, expectedPipelineFile, pipelineFile)
}

func TestDecompressPipelineTarball_MalformattedTarball(t *testing.T) {
	tarballByte, _ := os.ReadFile("test/malformatted_tarball.tar.gz")
	_, err := DecompressPipelineTarball(tarballByte)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Not a valid tarball file")
}

func TestDecompressPipelineTarball_NonYamlTarball(t *testing.T) {
	tarballByte, _ := os.ReadFile("test/non_yaml_tarball/non_yaml_tarball.tar.gz")
	_, err := DecompressPipelineTarball(tarballByte)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Expecting a pipeline.yaml file inside the tarball")
}

func TestDecompressPipelineTarball_EmptyTarball(t *testing.T) {
	tarballByte, _ := os.ReadFile("test/empty_tarball/empty.tar.gz")
	_, err := DecompressPipelineTarball(tarballByte)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Not a valid tarball file")
}

func TestDecompressPipelineZip(t *testing.T) {
	zipByte, _ := os.ReadFile("test/arguments_zip/arguments-parameters.zip")
	pipelineFile, err := DecompressPipelineZip(zipByte)
	assert.Nil(t, err)

	expectedPipelineFile, _ := os.ReadFile("test/arguments-parameters.yaml")
	assert.Equal(t, expectedPipelineFile, pipelineFile)
}

func TestDecompressPipelineZip_MalformattedZip(t *testing.T) {
	zipByte, _ := os.ReadFile("test/malformatted_zip.zip")
	_, err := DecompressPipelineZip(zipByte)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Not a valid zip file")
}

func TestDecompressPipelineZip_MalformedZip2(t *testing.T) {
	zipByte, _ := os.ReadFile("test/malformed_zip2.zip")
	_, err := DecompressPipelineZip(zipByte)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Not a valid zip file")
}

func TestDecompressPipelineZip_NonYamlZip(t *testing.T) {
	zipByte, _ := os.ReadFile("test/non_yaml_zip/non_yaml_file.zip")
	_, err := DecompressPipelineZip(zipByte)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Expecting a pipeline.yaml file inside the zip")
}

func TestDecompressPipelineZip_EmptyZip(t *testing.T) {
	zipByte, _ := os.ReadFile("test/empty_tarball/empty.zip")
	_, err := DecompressPipelineZip(zipByte)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Not a valid zip file")
}

func TestReadPipelineFile_YAML(t *testing.T) {
	file, _ := os.Open("test/arguments-parameters.yaml")
	fileBytes, err := ReadPipelineFile("arguments-parameters.yaml", file, common.MaxFileLength)
	assert.Nil(t, err)

	expectedFileBytes, _ := os.ReadFile("test/arguments-parameters.yaml")
	assert.Equal(t, expectedFileBytes, fileBytes)
}

func TestReadPipelineFile_JSON(t *testing.T) {
	file, _ := os.Open("test/v2-hello-world.json")
	fileBytes, err := ReadPipelineFile("v2-hello-world.json", file, common.MaxFileLength)
	assert.Nil(t, err)

	expectedFileBytes, _ := os.ReadFile("test/v2-hello-world.json")
	assert.Equal(t, expectedFileBytes, fileBytes)
}

func TestReadPipelineFile_Zip(t *testing.T) {
	file, _ := os.Open("test/arguments_zip/arguments-parameters.zip")
	pipelineFile, err := ReadPipelineFile("arguments-parameters.zip", file, common.MaxFileLength)
	assert.Nil(t, err)

	expectedPipelineFile, _ := os.ReadFile("test/arguments-parameters.yaml")
	assert.Equal(t, expectedPipelineFile, pipelineFile)
}

func TestReadPipelineFile_Zip_AnyExtension(t *testing.T) {
	file, _ := os.Open("test/arguments_zip/arguments-parameters.zip")
	pipelineFile, err := ReadPipelineFile("arguments-parameters.pipeline", file, common.MaxFileLength)
	assert.Nil(t, err)

	expectedPipelineFile, _ := os.ReadFile("test/arguments-parameters.yaml")
	assert.Equal(t, expectedPipelineFile, pipelineFile)
}

func TestReadPipelineFile_MultifileZip(t *testing.T) {
	file, _ := os.Open("test/pipeline_plus_component/pipeline_plus_component.zip")
	pipelineFile, err := ReadPipelineFile("pipeline_plus_component.ai-hub-package", file, common.MaxFileLength)
	assert.Nil(t, err)

	expectedPipelineFile, _ := os.ReadFile("test/pipeline_plus_component/pipeline.yaml")
	assert.Equal(t, expectedPipelineFile, pipelineFile)
}

func TestReadPipelineFile_Tarball(t *testing.T) {
	file, _ := os.Open("test/arguments_tarball/arguments.tar.gz")
	pipelineFile, err := ReadPipelineFile("arguments.tar.gz", file, common.MaxFileLength)
	assert.Nil(t, err)

	expectedPipelineFile, _ := os.ReadFile("test/arguments-parameters.yaml")
	assert.Equal(t, expectedPipelineFile, pipelineFile)
}

func TestReadPipelineFile_Tarball_AnyExtension(t *testing.T) {
	file, _ := os.Open("test/arguments_tarball/arguments.tar.gz")
	pipelineFile, err := ReadPipelineFile("arguments.pipeline", file, common.MaxFileLength)
	assert.Nil(t, err)

	expectedPipelineFile, _ := os.ReadFile("test/arguments-parameters.yaml")
	assert.Equal(t, expectedPipelineFile, pipelineFile)
}

func TestReadPipelineFile_CompressedFileSizeLimit(t *testing.T) {
	const maxFileLength = 1024

	formats := []struct {
		name          string
		fileName      string
		createArchive func(*testing.T, []byte) []byte
	}{
		{
			name:          "zip",
			fileName:      "pipeline.zip",
			createArchive: createZipArchive,
		},
		{
			name:          "tar.gz",
			fileName:      "pipeline.tar.gz",
			createArchive: createTarGzArchive,
		},
	}
	testCases := []struct {
		name        string
		contentSize int
		wantError   bool
	}{
		{
			name:        "at limit",
			contentSize: maxFileLength,
		},
		{
			name:        "over limit",
			contentSize: maxFileLength + 1,
			wantError:   true,
		},
	}

	for _, format := range formats {
		t.Run(format.name, func(t *testing.T) {
			for _, testCase := range testCases {
				t.Run(testCase.name, func(t *testing.T) {
					content := bytes.Repeat([]byte("a"), testCase.contentSize)
					archive := format.createArchive(t, content)
					require.Less(t, len(archive), maxFileLength)

					pipelineFile, err := ReadPipelineFile(
						format.fileName,
						bytes.NewReader(archive),
						maxFileLength,
					)
					if testCase.wantError {
						require.Error(t, err)
						assert.Nil(t, pipelineFile)
						assert.Contains(t, err.Error(), "Decompressed file size too large. Maximum supported size: 1024 bytes")
						return
					}

					require.NoError(t, err)
					assert.Equal(t, content, pipelineFile)
				})
			}
		})
	}
}

func TestReadPipelineFile_MultifileTarball(t *testing.T) {
	file, _ := os.Open("test/pipeline_plus_component/pipeline_plus_component.tar.gz")
	pipelineFile, err := ReadPipelineFile("pipeline_plus_component.ai-hub-package", file, common.MaxFileLength)
	assert.Nil(t, err)

	expectedPipelineFile, _ := os.ReadFile("test/pipeline_plus_component/pipeline.yaml")
	assert.Equal(t, expectedPipelineFile, pipelineFile)
}

func TestReadPipelineFile_UnknownFileFormat(t *testing.T) {
	file, _ := os.Open("test/unknown_format.foo")
	_, err := ReadPipelineFile("unknown_format.foo", file, common.MaxFileLength)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Unexpected pipeline file format")
}

func TestReadPipelineFile_SizeTooLarge_RecommendationIncluded(t *testing.T) {
	big := strings.Repeat("X", 1024)
	_, err := ReadPipelineFile("large.yaml", strings.NewReader(big), 10)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "File size too large")
	assert.Contains(t, err.Error(), "Consider moving large embedded artifacts or notebooks")
}

func TestDecompressPipelineZip_ValidEmptyZip(t *testing.T) {
	var buffer bytes.Buffer
	zipWriter := zip.NewWriter(&buffer)
	require.NoError(t, zipWriter.Close())
	_, err := DecompressPipelineZip(buffer.Bytes())
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Empty zip file")
}

// createPreTargetPaxBomb creates a tar.gz with a large-PAX-metadata decoy entry
// placed BEFORE targetName. tar.Reader.Next() fully decompresses PAX block data
// internally; making the PAX payload exceed the traversal budget exhausts the
// limiter during Next() before the target entry is ever reached.
func createPreTargetPaxBomb(t *testing.T, targetName string, targetContent string, paxPayloadBytes int64) []byte {
	t.Helper()
	require.True(t, paxPayloadBytes > 0, "paxPayloadBytes must be positive")

	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	// archive/tar has a 1 MiB limit for PAX special headers.
	// Split the payload across multiple decoys to safely exceed the cumulative budget.
	const maxPaxPerDecoy = 700 * 1024 // ~700 KiB
	numDecoys := int((paxPayloadBytes + maxPaxPerDecoy - 1) / maxPaxPerDecoy)
	remainingPayload := paxPayloadBytes

	decoyContent := "x"

	for i := 0; i < numDecoys; i++ {
		payloadForThisDecoy := remainingPayload
		if payloadForThisDecoy > maxPaxPerDecoy {
			payloadForThisDecoy = maxPaxPerDecoy
		}
		remainingPayload -= payloadForThisDecoy

		// Size the PAX value so the serialized record body is exactly payloadForThisDecoy.
		var paxValue string
		for valLen := int(payloadForThisDecoy) - 20; valLen <= int(payloadForThisDecoy); valLen++ {
			if valLen < 0 {
				continue
			}
			s := fmt.Sprintf("%d comment=%s\n", payloadForThisDecoy, bytes.Repeat([]byte("x"), valLen))
			if len(s) == int(payloadForThisDecoy) {
				paxValue = string(bytes.Repeat([]byte("x"), valLen))
				break
			}
		}
		require.NotEmpty(t, paxValue, "could not size PAX payload to exactly %d bytes", payloadForThisDecoy)

		require.NoError(t, tw.WriteHeader(&tar.Header{
			Typeflag:   tar.TypeReg,
			Name:       fmt.Sprintf("decoy-%d.txt", i),
			Mode:       0600,
			Size:       int64(len(decoyContent)),
			PAXRecords: map[string]string{"comment": paxValue},
		}))
		_, err := tw.Write([]byte(decoyContent))
		require.NoError(t, err)
	}

	// Write the real target after the decoys.
	require.NoError(t, tw.WriteHeader(&tar.Header{
		Typeflag: tar.TypeReg,
		Name:     targetName,
		Mode:     0600,
		Size:     int64(len(targetContent)),
	}))
	_, err := tw.Write([]byte(targetContent))
	require.NoError(t, err)
	require.NoError(t, tw.Close())

	var gzBuf bytes.Buffer
	gw := gzip.NewWriter(&gzBuf)
	_, err = gw.Write(buf.Bytes())
	require.NoError(t, err)
	require.NoError(t, gw.Close())
	return gzBuf.Bytes()
}

// TestReadPipelineFile_TraversalBudgetExhaustion verifies that PAX metadata
// placed BEFORE pipeline.yaml exhausts the traversal budget inside
// tar.Reader.Next() before the target entry is ever located.
func TestReadPipelineFile_TraversalBudgetExhaustion(t *testing.T) {
	const maxFileLength = 4096
	budget := util.ArchiveTraversalBudget(int64(maxFileLength))

	// Round up to 512-byte boundary; keeps payload below archive/tar 1 MiB limit.
	paxPayload := ((budget + 511) / 512) * 512
	tgz := createPreTargetPaxBomb(t, "pipeline.yaml", "foo: bar\n", paxPayload)

	_, err := ReadPipelineFile("pipeline.tar.gz", bytes.NewReader(tgz), maxFileLength)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Archive extraction exceeded traversal budget")
}

// TestReadPipelineFile_TraversalBudgetBoundary verifies two distinct boundary
// conditions: a small within-budget archive is accepted, and a pre-target PAX
// bomb exceeding the budget is rejected.
// createExactSkippedBytesBomb creates a tar.gz that forces tar.Reader.Next() to consume
// exactly `skippedBytes` before yielding `targetName`. It achieves this cleanly without
// PAX overhead by writing a decoy file whose header and padded body sum to `skippedBytes - 512`.
// When Next() advances to the target file, it consumes the decoy + target header (512 bytes),
// totaling exactly `skippedBytes` consumed.
func createExactSkippedBytesBomb(t *testing.T, targetName string, targetContent string, skippedBytes int64) []byte {
	t.Helper()
	require.True(t, skippedBytes >= 1024 && skippedBytes%512 == 0, "skippedBytes must be a multiple of 512 and >= 1024")

	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	// 1. Write the decoy entry.
	// To consume exactly (skippedBytes - 512) bytes for the decoy, we subtract 512 bytes for
	// its own header. The remaining (skippedBytes - 1024) is the exact body size.
	decoyBodySize := skippedBytes - 1024
	require.NoError(t, tw.WriteHeader(&tar.Header{
		Typeflag: tar.TypeReg,
		Name:     "decoy.txt",
		Mode:     0600,
		Size:     decoyBodySize,
	}))

	// Write the decoy body efficiently (chunks of zeros).
	chunk := make([]byte, 32*1024)
	var written int64
	for written < decoyBodySize {
		w := int64(len(chunk))
		if decoyBodySize-written < w {
			w = decoyBodySize - written
		}
		n, err := tw.Write(chunk[:w])
		require.NoError(t, err)
		written += int64(n)
	}

	// 2. Write the target entry.
	// When Next() yields this header, it will have consumed the 512-byte header.
	require.NoError(t, tw.WriteHeader(&tar.Header{
		Typeflag: tar.TypeReg,
		Name:     targetName,
		Mode:     0600,
		Size:     int64(len(targetContent)),
	}))
	_, err := tw.Write([]byte(targetContent))
	require.NoError(t, err)
	require.NoError(t, tw.Close())

	var gzBuf bytes.Buffer
	gw := gzip.NewWriter(&gzBuf)
	_, err = gw.Write(buf.Bytes())
	require.NoError(t, err)
	require.NoError(t, gw.Close())
	return gzBuf.Bytes()
}

// TestReadPipelineFile_TraversalBudgetBoundary verifies two distinct boundary
// conditions using pre-target decoys to achieve exact byte consumption bounds.
func TestReadPipelineFile_TraversalBudgetBoundary(t *testing.T) {
	const maxFileLength = 4096
	budget := util.ArchiveTraversalBudget(int64(maxFileLength))

	t.Run("exact budget bytes consumed is accepted", func(t *testing.T) {
		tgz := createExactSkippedBytesBomb(t, "pipeline.yaml", "", budget)
		_, err := ReadPipelineFile("pipeline.tar.gz", bytes.NewReader(tgz), maxFileLength)
		require.NoError(t, err)
	})

	t.Run("budget+1 bytes consumed is rejected", func(t *testing.T) {
		tgz := createExactSkippedBytesBomb(t, "pipeline.yaml", "a", budget)
		_, err := ReadPipelineFile("pipeline.tar.gz", bytes.NewReader(tgz), maxFileLength)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Archive extraction exceeded traversal budget")
	})
}
