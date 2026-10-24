// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package storage

import (
	"database/sql"
	"errors"

	"github.com/kubeflow/pipelines/backend/src/common/util"
	"google.golang.org/grpc/codes"
)

// IsNotFound recognizes wrapped SQL and KFP storage errors. util.IsNotFound
// classifies Kubernetes API errors, not the stores' util.UserError values.
func IsNotFound(err error) bool {
	if errors.Is(err, sql.ErrNoRows) {
		return true
	}
	var userError *util.UserError
	if errors.As(err, &userError) && util.IsUserErrorCodeMatch(userError, codes.NotFound) {
		return true
	}
	for cause := err; cause != nil; cause = errors.Unwrap(cause) {
		if util.IsNotFound(cause) {
			return true
		}
	}
	return false
}
