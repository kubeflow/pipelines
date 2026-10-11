// Copyright 2018-2023 The Kubeflow Authors
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

// Package api_server provides shared configuration and errors for API clients.
package api_server //nolint:staticcheck // ST1003: package name matches existing convention in this directory

import (
	"encoding/json"
	"errors"

	"google.golang.org/grpc/codes"
)

// PaginationRestartRequiredError means the caller must discard its page token
// and any accumulated results before starting the listing again. It is never
// safe to append a restarted first page to an earlier traversal.
type PaginationRestartRequiredError struct {
	cause error
}

func (e *PaginationRestartRequiredError) Error() string { return e.cause.Error() }
func (e *PaginationRestartRequiredError) Unwrap() error { return e.cause }

// IsPaginationRestartRequired recognizes the structured server condition even
// after a client adds context to the error. It does not match message text.
func IsPaginationRestartRequired(err error) bool {
	var restart *PaginationRestartRequiredError
	return errors.As(err, &restart)
}

// CreateErrorFromAPIStatusWithDetails preserves the restart condition from the
// generated HTTP clients' distinct status/detail types. Their JSON contracts
// are identical; using that contract avoids dependencies on every generated
// model package. Other errors retain the existing message/code representation.
func CreateErrorFromAPIStatusWithDetails(message string, code int32, details any) error {
	cause := CreateErrorFromAPIStatus(message, code)
	if code != int32(codes.FailedPrecondition) {
		return cause
	}
	data, err := json.Marshal(details)
	if err != nil {
		return cause
	}
	var entries []json.RawMessage
	if json.Unmarshal(data, &entries) != nil {
		return cause
	}
	for _, entry := range entries {
		var info struct {
			Type   string `json:"@type"`
			Reason string `json:"reason"`
			Domain string `json:"domain"`
		}
		if json.Unmarshal(entry, &info) == nil &&
			info.Type == "type.googleapis.com/google.rpc.ErrorInfo" &&
			info.Reason == "PAGINATION_RESTART_REQUIRED" && info.Domain == "kubeflow.org" {
			return &PaginationRestartRequiredError{cause: cause}
		}
	}
	return cause
}
