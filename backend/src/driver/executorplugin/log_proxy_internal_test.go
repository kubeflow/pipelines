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

package executorplugin

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/v2/objectstore"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes/fake"
)

func TestProductionLogUploaderUsesInvocationProxy(t *testing.T) {
	t.Setenv("AWS_ACCESS_KEY_ID", "test-access")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "test-secret")
	t.Setenv("AWS_EC2_METADATA_DISABLED", "true")
	var uploads atomic.Int32
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		uploads.Add(1)
		if r.URL.Host != "storage.invalid" || r.URL.Path != "/bucket/driver/test-logs" {
			t.Errorf("unexpected upload URL %s", r.URL)
		}
		data, _ := io.ReadAll(r.Body)
		if string(data) != "driver diagnostics" {
			t.Errorf("unexpected body %q", data)
		}
		w.Header().Set("ETag", `"logs"`)
	}))
	defer proxy.Close()
	proxyURL, err := url.Parse(proxy.URL)
	require.NoError(t, err)
	ctx := util.WithHTTPProxy(context.Background(), http.ProxyURL(proxyURL))
	path := filepath.Join(t.TempDir(), "logs")
	require.NoError(t, os.WriteFile(path, []byte("driver diagnostics"), 0600))
	session, err := json.Marshal(objectstore.SessionInfo{Provider: "s3", Params: map[string]string{
		"fromEnv": "true", "endpoint": "storage.invalid", "region": "us-east-1", "forcePathStyle": "true", "disableSSL": "true",
	}})
	require.NoError(t, err)
	uploader := logUploader{openBucket: objectstore.OpenBucket, uploadBlob: objectstore.UploadBlob}
	require.NoError(t, uploader.uploadDriverLogArtifact(ctx, &driverLogArtifactContext{
		LocalPath: path, PipelineRoot: "s3://bucket", Namespace: "ns", LogID: "test", StoreSessionInfo: string(session),
	}, fake.NewClientset()))
	require.EqualValues(t, 1, uploads.Load())
}
