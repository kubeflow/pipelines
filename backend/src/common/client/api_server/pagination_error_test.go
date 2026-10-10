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

package api_server //nolint:staticcheck // ST1003: package name matches existing convention in this directory

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/kubeflow/pipelines/backend/src/common/util"
)

func TestPaginationRestartClassification(t *testing.T) {
	for _, tc := range []struct {
		name, details string
		code          int32
		want          bool
	}{
		{"valid", `[{"@type":"type.googleapis.com/google.rpc.ErrorInfo","reason":"PAGINATION_RESTART_REQUIRED","domain":"kubeflow.org"}]`, 9, true},
		{"wrong type", `[{"@type":"other","reason":"PAGINATION_RESTART_REQUIRED","domain":"kubeflow.org"}]`, 9, false},
		{"wrong domain", `[{"@type":"type.googleapis.com/google.rpc.ErrorInfo","reason":"PAGINATION_RESTART_REQUIRED","domain":"other"}]`, 9, false},
		{"wrong reason", `[{"@type":"type.googleapis.com/google.rpc.ErrorInfo","reason":"OTHER","domain":"kubeflow.org"}]`, 9, false},
		{"wrong code", `[{"@type":"type.googleapis.com/google.rpc.ErrorInfo","reason":"PAGINATION_RESTART_REQUIRED","domain":"kubeflow.org"}]`, 3, false},
		{"message alone", `[]`, 9, false},
		{"malformed", `[{"reason":7}]`, 9, false},
		{"null", `null`, 9, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := CreateErrorFromAPIStatusWithDetails("PAGINATION_RESTART_REQUIRED", tc.code, json.RawMessage(tc.details))
			err = util.NewUserError(fmt.Errorf("context: %w", err), "list failed", "list failed")
			if got := IsPaginationRestartRequired(err); got != tc.want {
				t.Fatalf("classification=%v want %v: %v", got, tc.want, err)
			}
		})
	}
	if IsPaginationRestartRequired(nil) {
		t.Fatal("nil is not a restart error")
	}
}
