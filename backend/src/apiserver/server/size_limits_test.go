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
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/stretchr/testify/require"
)

func sizeLimitedPackage(t *testing.T, format, name string, data []byte) []byte {
	t.Helper()
	var buffer bytes.Buffer
	switch format {
	case "zip":
		writer := zip.NewWriter(&buffer)
		entry, err := writer.Create(name)
		require.NoError(t, err)
		_, err = entry.Write(data)
		require.NoError(t, err)
		require.NoError(t, writer.Close())
	case "tgz":
		compressed := gzip.NewWriter(&buffer)
		writer := tar.NewWriter(compressed)
		require.NoError(t, writer.WriteHeader(&tar.Header{Name: name, Mode: 0600, Size: int64(len(data))}))
		_, err := writer.Write(data)
		require.NoError(t, err)
		require.NoError(t, writer.Close())
		require.NoError(t, compressed.Close())
	default:
		return data
	}
	return buffer.Bytes()
}

func TestConfiguredPipelineReadLimits(t *testing.T) {
	for _, format := range []string{"yaml", "zip", "tgz"} {
		t.Run(format, func(t *testing.T) {
			payload := []byte(strings.Repeat("x", 1024))
			packed := sizeLimitedPackage(t, format, "pipeline.yaml", payload)
			for _, tc := range []struct {
				name        string
				input, spec int
				setting     string
			}{
				{"exact limits", len(packed), len(payload), ""},
				{"input exceeded", len(packed) - 1, len(payload), common.MaxPipelineUploadBytesEnv},
				{"spec exceeded", len(packed), len(payload) - 1, common.MaxPipelineSpecBytesEnv},
			} {
				t.Run(tc.name, func(t *testing.T) {
					t.Setenv(common.MaxPipelineUploadBytesEnv, strconv.Itoa(tc.input))
					t.Setenv(common.MaxPipelineSpecBytesEnv, strconv.Itoa(tc.spec))
					t.Setenv(common.MaxPipelineUpdateBodyBytesEnv, "")
					got, err := ReadPipelineFileWithConfiguredLimits("package."+format, bytes.NewReader(packed))
					if tc.setting == "" {
						require.NoError(t, err)
						require.Equal(t, payload, got)
					} else {
						var limitErr *common.SizeLimitError
						require.ErrorAs(t, err, &limitErr)
						require.Equal(t, tc.setting, limitErr.Setting)
						limit := tc.spec
						if tc.setting == common.MaxPipelineUploadBytesEnv {
							limit = tc.input
						}
						require.Equal(t, int64(limit), limitErr.Limit)
						require.Contains(t, limitErr.Error(), "exceeds maximum")
					}
				})
			}
		})
	}
}

func TestConfiguredPipelineReadAboveOriginalDefault(t *testing.T) {
	payload := bytes.Repeat([]byte("x"), common.MaxFileLength+1)
	t.Setenv(common.MaxPipelineUploadBytesEnv, strconv.Itoa(len(payload)))
	t.Setenv(common.MaxPipelineSpecBytesEnv, strconv.Itoa(len(payload)))
	t.Setenv(common.MaxPipelineUpdateBodyBytesEnv, "")
	got, err := ReadPipelineFileWithConfiguredLimits("pipeline.yaml", bytes.NewReader(payload))
	require.NoError(t, err)
	require.Equal(t, payload, got)
}

func TestConfiguredUploadHTTPSizeErrors(t *testing.T) {
	const secret = "private-payload-marker"
	for _, apiVersion := range []string{"v1beta1", "v2beta1"} {
		for _, version := range []bool{false, true} {
			for _, tc := range []struct {
				name, format, entry, setting string
				input, spec                  int
				status                       int
			}{
				{"input", "yaml", "pipeline.yaml", common.MaxPipelineUploadBytesEnv, 8, 1024, http.StatusRequestEntityTooLarge},
				{"raw spec", "yaml", "pipeline.yaml", common.MaxPipelineSpecBytesEnv, 4096, 8, http.StatusRequestEntityTooLarge},
				{"zip spec", "zip", "pipeline.yaml", common.MaxPipelineSpecBytesEnv, 4096, 8, http.StatusRequestEntityTooLarge},
				{"tgz spec", "tgz", "pipeline.yaml", common.MaxPipelineSpecBytesEnv, 4096, 8, http.StatusRequestEntityTooLarge},
				{"malformed archive", "zip", secret + ".txt", "", 4096, 1024, http.StatusBadRequest},
			} {
				t.Run(fmt.Sprintf("%s/version=%t/%s", apiVersion, version, tc.name), func(t *testing.T) {
					t.Setenv(common.MaxPipelineUploadBytesEnv, strconv.Itoa(tc.input))
					t.Setenv(common.MaxPipelineSpecBytesEnv, strconv.Itoa(tc.spec))
					t.Setenv(common.MaxPipelineUpdateBodyBytesEnv, "")
					clients, server := setupClientManagerAndServer()
					t.Cleanup(func() { clients.Close() })
					buffer, writer := setupWriter("")
					packed := sizeLimitedPackage(t, tc.format, tc.entry, []byte(secret))
					setWriterWithBuffer("uploadfile", "package."+tc.format, string(packed), writer)
					endpoint := "/apis/" + apiVersion + "/pipelines/upload?name=test"
					handler := server.UploadPipeline
					if apiVersion == "v1beta1" {
						handler = server.UploadPipelineV1
					}
					if version {
						endpoint = "/apis/" + apiVersion + "/pipelines/upload_version?pipelineid=" + DefaultFakeUUID
						handler = server.UploadPipelineVersion
						if apiVersion == "v1beta1" {
							handler = server.UploadPipelineVersionV1
						}
					}
					response := uploadPipeline(endpoint, bytes.NewReader(buffer.Bytes()), writer, handler)
					require.Equal(t, tc.status, response.Code, response.Body.String())
					require.NotContains(t, response.Body.String(), secret)
					if tc.setting != "" {
						require.Contains(t, response.Body.String(), tc.setting)
						require.Contains(t, response.Body.String(), "8 bytes")
						require.Contains(t, response.Body.String(), "administrator")
					} else {
						require.Contains(t, response.Body.String(), "Failed to")
						require.NotContains(t, response.Body.String(), "Expecting a pipeline.yaml")
					}
				})
			}
		}
	}
}

func TestConfiguredPipelineTraversalLimit(t *testing.T) {
	// A tiny compressed package can spend the extraction budget before the
	// pipeline entry is found. Raising the input limit must not relax traversal.
	packed := sizeLimitedPackage(t, "tgz", "unrelated.txt", bytes.Repeat([]byte("x"), (1<<20)+1024))
	t.Setenv(common.MaxPipelineUploadBytesEnv, "32768")
	t.Setenv(common.MaxPipelineSpecBytesEnv, "8")
	t.Setenv(common.MaxPipelineUpdateBodyBytesEnv, "")
	_, err := ReadPipelineFileWithConfiguredLimits("package.tgz", bytes.NewReader(packed))
	var limitErr *common.SizeLimitError
	require.ErrorAs(t, err, &limitErr)
	require.Equal(t, "pipeline_archive_traversal", limitErr.Control)
	require.Equal(t, common.MaxPipelineSpecBytesEnv, limitErr.Setting)
	require.Equal(t, int64((1<<20)+8), limitErr.Limit)
}
