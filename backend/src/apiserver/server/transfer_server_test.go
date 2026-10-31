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

package server

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kubeflow/pipelines/backend/src/apiserver/transfer"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"
)

type fakeTransferService struct {
	authorize func(context.Context, string, bool) error
	export    func(context.Context, string, transfer.ExportOptions) ([]byte, error)
	restore   func(context.Context, string, []byte, transfer.ImportOptions) (transfer.Summary, error)
}

func (f fakeTransferService) AuthorizeTransfer(ctx context.Context, ns string, importing bool) error {
	if f.authorize != nil {
		return f.authorize(ctx, ns, importing)
	}
	return nil
}
func (f fakeTransferService) ExportTransfer(ctx context.Context, ns string, opts transfer.ExportOptions) ([]byte, error) {
	return f.export(ctx, ns, opts)
}
func (f fakeTransferService) ImportTransfer(ctx context.Context, ns string, data []byte, opts transfer.ImportOptions) (transfer.Summary, error) {
	return f.restore(ctx, ns, data, opts)
}
func transferRequest(path, body string) *http.Request {
	r := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	return r
}
func TestTransferDownloadPreservesArchiveAndIdentity(t *testing.T) {
	archive := []byte(`{"large_id":9007199254740993}`)
	service := fakeTransferService{
		authorize: func(ctx context.Context, ns string, importing bool) error {
			require.Equal(t, "team", ns)
			require.False(t, importing)
			md, _ := metadata.FromIncomingContext(ctx)
			require.Equal(t, []string{"user@example.org"}, md.Get("kubeflow-userid"))
			return nil
		},
		export: func(_ context.Context, ns string, opts transfer.ExportOptions) ([]byte, error) {
			require.Equal(t, "team", ns)
			require.EqualValues(t, 100, opts.CompletedAfter)
			return archive, nil
		},
	}
	r := transferRequest("/apis/v2/transfer/export?namespace=team", `{"completed_after":100}`)
	r.Header.Set("Kubeflow-Userid", "user@example.org")
	w := httptest.NewRecorder()
	NewTransferServer(service).Export(w, r)
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, archive, w.Body.Bytes())
	require.Contains(t, w.Header().Get("Content-Disposition"), "attachment")
	require.Equal(t, "no-store", w.Header().Get("Cache-Control"))
}
func TestTransferImportDefaultsToPreviewAndPreservesBytes(t *testing.T) {
	archive := `{"large_id":9007199254740993}`
	for _, suffix := range []string{"", "&dry_run=true", "&dry_run=false"} {
		t.Run(suffix, func(t *testing.T) {
			s := NewTransferServer(fakeTransferService{restore: func(_ context.Context, ns string, data []byte, opts transfer.ImportOptions) (transfer.Summary, error) {
				require.Equal(t, "team", ns)
				require.Equal(t, archive, string(data))
				require.Equal(t, "old-", opts.NamePrefix)
				require.Equal(t, suffix != "&dry_run=false", opts.DryRun)
				return transfer.Summary{Counts: transfer.Counts{Experiments: 2, Schedules: 1}}, nil
			}})
			w := httptest.NewRecorder()
			s.Import(w, transferRequest("/apis/v2/transfer/import?namespace=team&name_prefix=old-"+suffix, archive))
			require.Equal(t, http.StatusOK, w.Code)
			require.Contains(t, w.Body.String(), `"schedules":1`)
			require.Contains(t, w.Body.String(), `"warnings":[]`)
		})
	}
}

type unreadableTransferBody struct{ read bool }

func (b *unreadableTransferBody) Read([]byte) (int, error) {
	b.read = true
	return 0, errors.New("must not read")
}
func (*unreadableTransferBody) Close() error { return nil }
func TestTransferDenialDoesNotReadArchive(t *testing.T) {
	b := &unreadableTransferBody{}
	r := transferRequest("/apis/v2/transfer/import?namespace=other", "{}")
	r.Body = b
	s := NewTransferServer(fakeTransferService{authorize: func(context.Context, string, bool) error {
		return util.NewPermissionDeniedError(errors.New("denied"), "No access to destination")
	}})
	w := httptest.NewRecorder()
	s.Import(w, r)
	require.Equal(t, http.StatusForbidden, w.Code)
	require.False(t, b.read)
}
func TestTransferRejectsInvalidRequestsBeforeEngine(t *testing.T) {
	for _, tc := range []struct {
		name, path, body string
		export           bool
		want             int
	}{
		{"unknown option", "/export", `{"unknown":1}`, true, 400},
		{"trailing JSON", "/export", `{} {}`, true, 400},
		{"null export", "/export", `null`, true, 400},
		{"reversed interval", "/export", `{"completed_after":20,"completed_before":10}`, true, 400},
		{"invalid dry run", "/import?dry_run=no", "{}", false, 400},
		{"empty archive", "/import", "", false, 400},
		{"large archive", "/import", strings.Repeat("x", 33), false, 413},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := NewTransferServer(fakeTransferService{})
			s.maxArchiveBytes = 32
			w := httptest.NewRecorder()
			r := transferRequest(tc.path, tc.body)
			if tc.export {
				s.Export(w, r)
			} else {
				s.Import(w, r)
			}
			require.Equal(t, tc.want, w.Code)
		})
	}
}
func TestTransferErrorsDoNotExposeStorageDetails(t *testing.T) {
	for _, tc := range []struct {
		err     error
		code    int
		message string
	}{
		{errors.New("password=private database query"), 500, "Transfer did not complete"},
		{util.NewUnavailableServerError(errors.New("password=private"), "password=private"), 503, "temporarily unavailable"},
		{util.NewAlreadyExistError("Pipeline name conflicts; choose a name prefix"), 409, "choose a name prefix"},
	} {
		s := NewTransferServer(fakeTransferService{restore: func(context.Context, string, []byte, transfer.ImportOptions) (transfer.Summary, error) {
			return transfer.Summary{}, tc.err
		}})
		w := httptest.NewRecorder()
		s.Import(w, transferRequest("/import", "{}"))
		require.Equal(t, tc.code, w.Code)
		require.Contains(t, w.Body.String(), tc.message)
		require.NotContains(t, w.Body.String(), "password=")
	}
}
func TestTransferLimitsConcurrentWorkAndReleasesSlot(t *testing.T) {
	started, release := make(chan struct{}), make(chan struct{})
	s := NewTransferServer(fakeTransferService{export: func(context.Context, string, transfer.ExportOptions) ([]byte, error) {
		close(started)
		<-release
		return []byte("{}"), nil
	}, restore: func(context.Context, string, []byte, transfer.ImportOptions) (transfer.Summary, error) {
		return transfer.Summary{}, nil
	}})
	done := make(chan struct{})
	go func() { defer close(done); s.Export(httptest.NewRecorder(), transferRequest("/export", "{}")) }()
	<-started
	w := httptest.NewRecorder()
	s.Import(w, transferRequest("/import", "{}"))
	require.Equal(t, 429, w.Code)
	close(release)
	<-done
	w = httptest.NewRecorder()
	s.Import(w, transferRequest("/import", "{}"))
	require.Equal(t, 200, w.Code)
}
func TestTransferRejectsMethodsAndMediaTypes(t *testing.T) {
	s := NewTransferServer(fakeTransferService{})
	r := httptest.NewRequest(http.MethodGet, "/export", nil)
	w := httptest.NewRecorder()
	s.Export(w, r)
	require.Equal(t, 405, w.Code)
	r = httptest.NewRequest(http.MethodPost, "/import", io.NopCloser(strings.NewReader("{}")))
	w = httptest.NewRecorder()
	s.Import(w, r)
	require.Equal(t, 415, w.Code)
}
