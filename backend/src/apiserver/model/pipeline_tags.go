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

package model

import (
	"encoding/json"
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

// MaxTagsPerEntity is the maximum number of tags allowed on a single pipeline or pipeline version.
const MaxTagsPerEntity = 20

// ValidateTags checks that tags conform to all supported constraints.
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

// ParseTagsJSON parses the upload tags JSON and validates the resulting map.
func ParseTagsJSON(tagsJSON *string) (map[string]string, error) {
	if tagsJSON == nil || *tagsJSON == "" {
		return nil, nil
	}

	var tags map[string]string
	if err := json.Unmarshal([]byte(*tagsJSON), &tags); err != nil {
		return nil, util.NewInvalidInputError("failed to parse tags JSON: %v", err)
	}

	if tags == nil {
		return nil, nil
	}

	if err := ValidateTags(tags); err != nil {
		return nil, err
	}

	return tags, nil
}
