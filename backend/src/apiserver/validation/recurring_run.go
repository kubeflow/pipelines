// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package validation

import (
	"unicode/utf8"

	"github.com/kubeflow/pipelines/backend/src/common/util"
)

// ValidateRecurringRunRequestKey ensures a claimed key can be stored as the run's display name.
func ValidateRecurringRunRequestKey(requestKey string) error {
	if requestKey == "" {
		return util.NewInvalidInputError("Provide an idempotency key when claiming a recurring run")
	}
	if !utf8.ValidString(requestKey) {
		return util.NewInvalidInputError("Use valid UTF-8 for the recurring run idempotency key")
	}
	// MySQL stores run display names in VARCHAR(255), whose limit is characters, not bytes.
	if utf8.RuneCountInString(requestKey) > 255 {
		return util.NewInvalidInputError("Recurring run idempotency key cannot exceed 255 characters; shorten the run display name")
	}
	return nil
}
