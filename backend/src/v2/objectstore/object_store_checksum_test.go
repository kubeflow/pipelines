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

package objectstore

import (
	"context"
	"encoding/pem"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// newNoChunkedEncodingS3Server starts an HTTPS server that behaves like an
// S3-compatible store without aws-chunked support, such as OCI Object Storage.
// It points AWS_CA_BUNDLE at the server certificate so the SDK trusts it.
func newNoChunkedEncodingS3Server(t *testing.T) *httptest.Server {
	t.Helper()
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		if strings.Contains(r.Header.Get("Content-Encoding"), "aws-chunked") ||
			strings.HasPrefix(r.Header.Get("X-Amz-Content-Sha256"), "STREAMING-") {
			w.Header().Set("Content-Type", "application/xml")
			w.WriteHeader(http.StatusNotImplemented)
			_, _ = io.WriteString(w, `<?xml version="1.0" encoding="UTF-8"?><Error><Code>NotImplemented</Code><Message>AWS chunked encoding not supported.</Message></Error>`)
			return
		}
		w.Header().Set("ETag", `"etag"`)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	caBundle := filepath.Join(t.TempDir(), "ca.pem")
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: server.Certificate().Raw})
	require.NoError(t, os.WriteFile(caBundle, certPEM, 0o600))
	t.Setenv("AWS_CA_BUNDLE", caBundle)
	t.Setenv("AWS_ACCESS_KEY_ID", "test-access-key")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "test-secret-key")
	t.Setenv("AWS_REGION", "us-east-1")
	return server
}

// Regression test for https://github.com/kubeflow/pipelines/issues/15349.
func TestOpenBucketUploadsWithoutChunkedEncoding(t *testing.T) {
	server := newNoChunkedEncodingS3Server(t)

	bucket, err := OpenBucket(context.Background(), nil, "", &Config{
		Scheme:     "s3://",
		BucketName: "test-bucket",
		Prefix:     "artifacts/",
	}, &SessionInfo{
		Provider: "s3",
		Params: map[string]string{
			"region":         "us-east-1",
			"endpoint":       server.URL,
			"fromEnv":        "true",
			"forcePathStyle": "true",
		},
	})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, bucket.Close()) })

	require.NoError(t, bucket.WriteAll(context.Background(), "executor-logs", []byte("hello"), nil))
}
