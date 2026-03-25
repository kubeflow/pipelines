// Copyright 2026 The Kubeflow Authors
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

package common

import (
	"strings"
	"unicode/utf8"

	"github.com/kubeflow/pipelines/backend/src/common/util"
)

// MaxTagKeyLength is the maximum allowed length (in characters) for a tag key.
// Consistent with Kubernetes label value length limit (63 characters).
const MaxTagKeyLength = 63

// MaxTagValueLength is the maximum allowed length (in characters) for a tag value.
// Consistent with Kubernetes label value length limit (63 characters).
const MaxTagValueLength = 63

// MaxTagsPerEntity is the maximum number of tags allowed on a single entity.
const MaxTagsPerEntity = 20

// ValidateTags checks that tags conform to all constraints:
// - maximum number of tags per entity
// - no empty keys
// - keys must not contain "." (conflicts with tag filter prefix "tags.")
// - key and value character lengths within limits
func ValidateTags(tags map[string]string) error {
	if len(tags) > MaxTagsPerEntity {
		return util.NewInvalidInputError("number of tags (%d) exceeds maximum of %d per entity", len(tags), MaxTagsPerEntity)
	}
	for key, value := range tags {
		if key == "" {
			return util.NewInvalidInputError("tag key cannot be empty")
		}
		if strings.Contains(key, ".") {
			return util.NewInvalidInputError("tag key %q must not contain '.' character", key)
		}
		if utf8.RuneCountInString(key) > MaxTagKeyLength {
			return util.NewInvalidInputError("tag key %q exceeds maximum length of %d characters", key, MaxTagKeyLength)
		}
		if utf8.RuneCountInString(value) > MaxTagValueLength {
			return util.NewInvalidInputError("tag value %q for key %q exceeds maximum length of %d characters", value, key, MaxTagValueLength)
		}
	}
	return nil
}
