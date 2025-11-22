// Copyright 2025 The Kubeflow Authors
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

package artifactclient

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/kubeflow/pipelines/backend/src/agent/persistence/client/tokenrefresher"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockFileReader struct {
	token string
}

func (m *mockFileReader) ReadFile(string) ([]byte, error) {
	return []byte(m.token), nil
}

func newTestTokenRefresher(t *testing.T) *tokenrefresher.TokenRefresher {
	refresher := tokenrefresher.NewTokenRefresher(time.Hour, &mockFileReader{token: "test-token"})
	require.NoError(t, refresher.RefreshToken(), "Failed to refresh token in test setup")
	return refresher
}

// createTarGz creates a tar.gz archive containing the given files
func createTarGz(files map[string]string) ([]byte, error) {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	for name, content := range files {
		hdr := &tar.Header{
			Typeflag: tar.TypeReg,
			Name:     name,
			Size:     int64(len(content)),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return nil, err
		}
		if _, err := tw.Write([]byte(content)); err != nil {
			return nil, err
		}
	}

	if err := tw.Close(); err != nil {
		return nil, err
	}
	if err := gw.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// extractTarGz extracts files from a tar.gz archive (like util.ExtractTgz but works with []byte)
func extractTarGz(t *testing.T, tgzData []byte) map[string]string {
	gr, err := gzip.NewReader(bytes.NewReader(tgzData))
	require.NoError(t, err, "Failed to create gzip reader")
	defer func() {
		require.NoError(t, gr.Close(), "Failed to close gzip reader")
	}()

	tr := tar.NewReader(gr)
	files := make(map[string]string)

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		require.NoError(t, err, "Failed to read tar header")
		if hdr == nil {
			continue
		}

		fileContent, err := io.ReadAll(tr)
		require.NoError(t, err, "Failed to read tar file content")
		files[hdr.Name] = string(fileContent)
	}

	return files
}

func TestReadArtifact_Success_ReturnsDecodedTarGz(t *testing.T) {
	expectedFiles := map[string]string{
		"metrics.json": `{"accuracy": 0.95, "loss": 0.05}`,
	}

	tgzData, err := createTarGz(expectedFiles)
	require.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/apis/v1beta1/runs/run-123/nodes/node-456/artifacts/metrics:read", r.URL.Path)
		assert.Equal(t, "GET", r.Method)
		assert.Contains(t, r.Header.Get("Authorization"), "Bearer")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		base64Data := base64.StdEncoding.EncodeToString(tgzData)
		response := map[string]string{"data": base64Data}
		jsonBytes, err := json.Marshal(response)
		require.NoError(t, err)
		_, err = w.Write(jsonBytes)
		require.NoError(t, err)
	}))
	defer server.Close()

	tokenRefresher := newTestTokenRefresher(t)
	client := NewClient(server.URL, server.Client(), tokenRefresher)

	request := &ReadArtifactRequest{
		RunID:        "run-123",
		NodeID:       "node-456",
		ArtifactName: "metrics",
	}

	response, err := client.ReadArtifact(request)
	require.NoError(t, err)
	require.NotNil(t, response)
	require.NotNil(t, response.Data)

	extractedFiles := extractTarGz(t, response.Data)
	assert.Equal(t, expectedFiles, extractedFiles, "Extracted files should match original files")
}

func TestReadArtifact_NotFound_ReturnsNil(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	tokenRefresher := newTestTokenRefresher(t)
	client := NewClient(server.URL, server.Client(), tokenRefresher)

	request := &ReadArtifactRequest{
		RunID:        "run-123",
		NodeID:       "node-456",
		ArtifactName: "missing",
	}

	response, err := client.ReadArtifact(request)
	assert.NoError(t, err)
	assert.Nil(t, response)
}

func TestReadArtifact_Unauthorized_ReturnsTransientError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	tokenRefresher := newTestTokenRefresher(t)
	client := NewClient(server.URL, server.Client(), tokenRefresher)

	request := &ReadArtifactRequest{
		RunID:        "run-123",
		NodeID:       "node-456",
		ArtifactName: "metrics",
	}

	response, err := client.ReadArtifact(request)
	assert.Nil(t, response)
	require.Error(t, err)

	var artifactErr *Error
	require.ErrorAs(t, err, &artifactErr)
	assert.Equal(t, ErrorCodeTransient, artifactErr.Code)
}

func TestReadArtifact_Forbidden_ReturnsPermanentError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	tokenRefresher := newTestTokenRefresher(t)
	client := NewClient(server.URL, server.Client(), tokenRefresher)

	request := &ReadArtifactRequest{
		RunID:        "run-123",
		NodeID:       "node-456",
		ArtifactName: "metrics",
	}

	response, err := client.ReadArtifact(request)
	assert.Nil(t, response)
	require.Error(t, err)

	var artifactErr *Error
	require.ErrorAs(t, err, &artifactErr)
	assert.Equal(t, ErrorCodePermanent, artifactErr.Code)
}

func TestReadArtifact_BadRequest_ReturnsPermanentError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	tokenRefresher := newTestTokenRefresher(t)
	client := NewClient(server.URL, server.Client(), tokenRefresher)

	request := &ReadArtifactRequest{
		RunID:        "run-123",
		NodeID:       "node-456",
		ArtifactName: "metrics",
	}

	response, err := client.ReadArtifact(request)
	assert.Nil(t, response)
	require.Error(t, err)

	var artifactErr *Error
	require.ErrorAs(t, err, &artifactErr)
	assert.Equal(t, ErrorCodePermanent, artifactErr.Code)
}

func TestReadArtifact_InternalServerError_ReturnsTransientError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	tokenRefresher := newTestTokenRefresher(t)
	client := NewClient(server.URL, server.Client(), tokenRefresher)

	request := &ReadArtifactRequest{
		RunID:        "run-123",
		NodeID:       "node-456",
		ArtifactName: "metrics",
	}

	response, err := client.ReadArtifact(request)
	assert.Nil(t, response)
	require.Error(t, err)

	var artifactErr *Error
	require.ErrorAs(t, err, &artifactErr)
	assert.Equal(t, ErrorCodeTransient, artifactErr.Code)
}

func TestReadArtifact_LargeArtifact_Success(t *testing.T) {
	largeContent := string(make([]byte, 10*1024*1024))
	expectedFiles := map[string]string{
		"large-file.bin": largeContent,
	}

	tgzData, err := createTarGz(expectedFiles)
	require.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		base64Data := base64.StdEncoding.EncodeToString(tgzData)
		response := map[string]string{"data": base64Data}
		jsonBytes, err := json.Marshal(response)
		require.NoError(t, err)
		_, err = w.Write(jsonBytes)
		require.NoError(t, err)
	}))
	defer server.Close()

	tokenRefresher := newTestTokenRefresher(t)
	client := NewClient(server.URL, server.Client(), tokenRefresher)

	request := &ReadArtifactRequest{
		RunID:        "run-123",
		NodeID:       "node-456",
		ArtifactName: "large-artifact",
	}

	response, err := client.ReadArtifact(request)
	require.NoError(t, err)
	require.NotNil(t, response)

	extractedFiles := extractTarGz(t, response.Data)
	assert.Equal(t, len(expectedFiles), len(extractedFiles))
}

func TestReadArtifact_MultipleFiles_Success(t *testing.T) {
	expectedFiles := map[string]string{
		"metrics.json":  `{"accuracy": 0.95}`,
		"metadata.json": `{"model": "v1"}`,
		"config.yaml":   "batch_size: 32\nepochs: 10",
	}

	tgzData, err := createTarGz(expectedFiles)
	require.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		base64Data := base64.StdEncoding.EncodeToString(tgzData)
		response := map[string]string{"data": base64Data}
		jsonBytes, err := json.Marshal(response)
		require.NoError(t, err)
		_, err = w.Write(jsonBytes)
		require.NoError(t, err)
	}))
	defer server.Close()

	tokenRefresher := newTestTokenRefresher(t)
	client := NewClient(server.URL, server.Client(), tokenRefresher)

	request := &ReadArtifactRequest{
		RunID:        "run-123",
		NodeID:       "node-456",
		ArtifactName: "multi-file-artifact",
	}

	response, err := client.ReadArtifact(request)
	require.NoError(t, err)
	require.NotNil(t, response)

	extractedFiles := extractTarGz(t, response.Data)
	assert.Equal(t, expectedFiles, extractedFiles)
}

func TestReadArtifact_InvalidURL_ReturnsError(t *testing.T) {
	tokenRefresher := newTestTokenRefresher(t)
	client := NewClient("http://[invalid-url", &http.Client{}, tokenRefresher)

	request := &ReadArtifactRequest{
		RunID:        "run-123",
		NodeID:       "node-456",
		ArtifactName: "metrics",
	}

	response, err := client.ReadArtifact(request)
	assert.Nil(t, response)
	require.Error(t, err)

	var artifactErr *Error
	require.ErrorAs(t, err, &artifactErr)
	assert.Equal(t, ErrorCodePermanent, artifactErr.Code)
}

func TestReadArtifact_UnexpectedStatusCode_ReturnsPermanentError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))
	defer server.Close()

	tokenRefresher := newTestTokenRefresher(t)
	client := NewClient(server.URL, server.Client(), tokenRefresher)

	request := &ReadArtifactRequest{
		RunID:        "run-123",
		NodeID:       "node-456",
		ArtifactName: "metrics",
	}

	response, err := client.ReadArtifact(request)
	assert.Nil(t, response)
	require.Error(t, err)

	var artifactErr *Error
	require.ErrorAs(t, err, &artifactErr)
	assert.Equal(t, ErrorCodePermanent, artifactErr.Code)
	assert.Contains(t, err.Error(), "418")
}

func TestReadArtifactRequest_String(t *testing.T) {
	request := &ReadArtifactRequest{
		RunID:        "run-123",
		NodeID:       "node-456",
		ArtifactName: "metrics",
	}

	expected := "run-123/node-456/metrics"
	assert.Equal(t, expected, request.String())
}

func TestError_Error(t *testing.T) {
	t.Run("with cause", func(t *testing.T) {
		cause := fmt.Errorf("underlying error")
		err := NewError(ErrorCodePermanent, cause, "operation failed: %s", "reason")

		assert.Contains(t, err.Error(), "operation failed: reason")
		assert.Contains(t, err.Error(), "underlying error")
	})

	t.Run("without cause", func(t *testing.T) {
		err := NewError(ErrorCodePermanent, nil, "simple error")
		assert.Equal(t, "simple error", err.Error())
	})
}
