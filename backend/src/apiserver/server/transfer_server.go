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
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"strconv"
	"unicode/utf8"

	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/backend/src/apiserver/transfer"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
)

// TransferService keeps HTTP parsing separate from authorized resource transfer.
type TransferService interface {
	AuthorizeTransfer(context.Context, string, bool) error
	ExportTransfer(context.Context, string, transfer.ExportOptions) ([]byte, error)
	ImportTransfer(context.Context, string, []byte, transfer.ImportOptions) (transfer.Summary, error)
}

// TransferServer serves metadata downloads and uploads through the existing
// authenticated HTTP surface, without exposing database credentials to users.
type TransferServer struct {
	service         TransferService
	active          chan struct{}
	maxArchiveBytes int64
}

func NewTransferServer(service TransferService) *TransferServer {
	return &TransferServer{service: service, active: make(chan struct{}, 1), maxArchiveBytes: transfer.MaxArchiveBytes}
}

func transferContext(r *http.Request) context.Context {
	md := metadata.MD{}
	for key, values := range r.Header {
		md.Set(key, values...)
	}
	return metadata.NewIncomingContext(r.Context(), md)
}

func (s *TransferServer) begin(w http.ResponseWriter, r *http.Request, importing bool) (context.Context, bool) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeTransferError(w, http.StatusMethodNotAllowed, "Use POST for export and import")
		return nil, false
	}
	ctx := transferContext(r)
	if err := s.service.AuthorizeTransfer(ctx, r.URL.Query().Get("namespace"), importing); err != nil {
		transferServiceError(w, err)
		return nil, false
	}
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		writeTransferError(w, http.StatusUnsupportedMediaType, "Send an application/json archive or export request")
		return nil, false
	}
	select {
	case s.active <- struct{}{}:
		return ctx, true
	default:
		w.Header().Set("Retry-After", "5")
		writeTransferError(w, http.StatusTooManyRequests, "Another transfer is running on this server. Retry shortly")
		return nil, false
	}
}

// Export downloads a namespace archive; the optional time range applies only
// to completed history, never to the full catalog, experiments or schedules.
func (s *TransferServer) Export(w http.ResponseWriter, r *http.Request) {
	ctx, ok := s.begin(w, r, false)
	if !ok {
		return
	}
	defer func() { <-s.active }()
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096))
	decoder.DisallowUnknownFields()
	var options *transfer.ExportOptions
	if err := decoder.Decode(&options); err != nil || options == nil {
		writeTransferError(w, http.StatusBadRequest, "Export options must be one JSON object")
		return
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		writeTransferError(w, http.StatusBadRequest, "Export options must contain exactly one JSON object")
		return
	}
	if options.CompletedAfter < 0 || options.CompletedBefore < 0 ||
		(options.CompletedBefore != 0 && options.CompletedBefore <= options.CompletedAfter) {
		writeTransferError(w, http.StatusBadRequest, "Choose a completed-history end time after its start time")
		return
	}
	data, err := s.service.ExportTransfer(ctx, r.URL.Query().Get("namespace"), *options)
	if err != nil {
		transferServiceError(w, err)
		return
	}
	if int64(len(data)) > s.maxArchiveBytes {
		writeTransferError(w, http.StatusRequestEntityTooLarge, "Metadata exceeds the archive limit. Export a smaller completed-history date range")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", `attachment; filename="kfp-transfer.json"`)
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(data); err != nil {
		glog.Warningf("Transfer download interrupted: %v", err)
	}
}

// Import validates by default. Publication requires the explicit dry_run=false
// query argument and repeats all authorization and archive checks.
func (s *TransferServer) Import(w http.ResponseWriter, r *http.Request) {
	ctx, ok := s.begin(w, r, true)
	if !ok {
		return
	}
	defer func() { <-s.active }()
	options := transfer.ImportOptions{DryRun: true, NamePrefix: r.URL.Query().Get("name_prefix")}
	if raw := r.URL.Query().Get("dry_run"); raw != "" {
		if raw != "true" && raw != "false" {
			writeTransferError(w, http.StatusBadRequest, "dry_run must be true or false")
			return
		}
		options.DryRun, _ = strconv.ParseBool(raw)
	}
	if !utf8.ValidString(options.NamePrefix) || utf8.RuneCountInString(options.NamePrefix) > 63 {
		writeTransferError(w, http.StatusBadRequest, "Name prefix must contain at most 63 characters")
		return
	}
	data, err := io.ReadAll(http.MaxBytesReader(w, r.Body, s.maxArchiveBytes))
	if err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			writeTransferError(w, http.StatusRequestEntityTooLarge, "Archive exceeds 256 MiB. Export a smaller completed-history date range")
		} else {
			writeTransferError(w, http.StatusBadRequest, "Could not read the archive. Upload the complete file and retry")
		}
		return
	}
	if len(data) == 0 {
		writeTransferError(w, http.StatusBadRequest, "Select a nonempty transfer archive")
		return
	}
	summary, err := s.service.ImportTransfer(ctx, r.URL.Query().Get("namespace"), data, options)
	if err != nil {
		transferServiceError(w, err)
		return
	}
	summary.DryRun = options.DryRun
	if summary.Warnings == nil {
		summary.Warnings = []string{}
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(summary); err != nil {
		glog.Warningf("Transfer response interrupted: %v", err)
	}
}

func transferServiceError(w http.ResponseWriter, err error) {
	var userError *util.UserError
	if errors.As(err, &userError) {
		code := http.StatusInternalServerError
		switch userError.ExternalStatusCode() {
		case codes.InvalidArgument:
			code = http.StatusBadRequest
		case codes.AlreadyExists, codes.Aborted, codes.FailedPrecondition:
			code = http.StatusConflict
		case codes.Unauthenticated:
			code = http.StatusUnauthorized
		case codes.PermissionDenied:
			code = http.StatusForbidden
		case codes.NotFound:
			code = http.StatusNotFound
		case codes.ResourceExhausted:
			code = http.StatusRequestEntityTooLarge
		case codes.Unavailable:
			code = http.StatusServiceUnavailable
		}
		if code == http.StatusServiceUnavailable {
			writeTransferError(w, code, "Transfer service is temporarily unavailable. Retry the same archive")
			return
		}
		if code < 500 {
			writeTransferError(w, code, userError.ExternalMessage())
			return
		}
	}
	// SQL errors and runtime details can contain source metadata or credentials.
	glog.Errorf("Namespace transfer failed (error type %T)", err)
	writeTransferError(w, http.StatusInternalServerError, "Transfer did not complete. Retry the same archive; contact your administrator if the problem persists")
}

func writeTransferError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}
