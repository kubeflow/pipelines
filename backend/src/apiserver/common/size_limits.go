// Copyright 2018 The Kubeflow Authors
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
	"fmt"
	"os"
	"strconv"

	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/backend/src/common/util"
)

const (
	// MaxPipelineUploadBytesEnv limits the uploaded or downloaded pipeline package.
	MaxPipelineUploadBytesEnv = "MAX_PIPELINE_UPLOAD_BYTES"
	// MaxPipelineSpecBytesEnv limits extracted specifications and object-store spec reads.
	MaxPipelineSpecBytesEnv = "MAX_PIPELINE_SPEC_BYTES"
	// MaxPipelineUpdateBodyBytesEnv limits pipeline PUT/PATCH request bodies.
	MaxPipelineUpdateBodyBytesEnv = "MAX_PIPELINE_UPDATE_BODY_BYTES"
	// MaximumPipelineSizeBytes bounds administrator overrides; requests remain buffered in memory.
	MaximumPipelineSizeBytes = 128 << 20
)

// PipelineSizeLimits contains independent, finite pipeline byte ceilings.
type PipelineSizeLimits struct {
	UploadBytes     int
	SpecBytes       int
	UpdateBodyBytes int
}

// GetPipelineSizeLimits validates operator environment settings. Unset or empty
// settings retain the 32 MiB defaults; invalid settings never disable a ceiling.
func GetPipelineSizeLimits() (PipelineSizeLimits, error) {
	limits := PipelineSizeLimits{MaxFileLength, MaxFileLength, MaxFileLength}
	for _, setting := range []struct {
		name  string
		value *int
	}{
		{MaxPipelineUploadBytesEnv, &limits.UploadBytes},
		{MaxPipelineSpecBytesEnv, &limits.SpecBytes},
		{MaxPipelineUpdateBodyBytesEnv, &limits.UpdateBodyBytes},
	} {
		raw := os.Getenv(setting.name)
		if raw == "" {
			continue
		}
		value, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || value < 1 || value > MaximumPipelineSizeBytes {
			return PipelineSizeLimits{}, fmt.Errorf("%s must be an integer from 1 to %d bytes; unset it to use the %d-byte default", setting.name, MaximumPipelineSizeBytes, MaxFileLength)
		}
		*setting.value = int(value)
	}
	return limits, nil
}

// SizeLimitError contains only safe, operator-controlled rejection details.
type SizeLimitError struct {
	Control string
	Limit   int64
	Setting string
}

func (e *SizeLimitError) Error() string {
	return SizeLimitErrorMessage(e.Control, e.Limit, e.Setting)
}

// SizeLimitErrorMessage describes a finite ceiling without claiming the truncated
// read measured the entire payload. Controls must be static, not user input.
func SizeLimitErrorMessage(control string, limit int64, setting string) string {
	label := control
	advice := "Reduce the payload"
	switch control {
	case "pipeline_upload":
		label = "File"
		advice = "Move large embedded artifacts, notebooks, or Python code into a container image or object store, or reduce the package"
	case "pipeline_spec":
		label = "Pipeline spec file"
	case "pipeline_decompressed_spec":
		label = "Decompressed file"
	case "pipeline_archive_traversal":
		label = "Archive extraction traversal budget"
		advice = "Reduce archive entries and metadata; the traversal budget is derived from the pipeline spec limit"
	case "pipeline_update_body":
		label = "Request body"
	}
	return fmt.Sprintf("%s size too large: exceeds maximum %d bytes (%.2f MiB). %s or ask an administrator to adjust %s within its supported range.", label, limit, float64(limit)/(1<<20), advice, setting)
}

// NewSizeLimitError logs a bounded rejection and preserves InvalidArgument for
// API callers while allowing upload handlers to expose only the safe message.
func NewSizeLimitError(control string, limit int64, setting string) error {
	err := &SizeLimitError{Control: control, Limit: limit, Setting: setting}
	glog.Warningf("size_limit_exceeded control=%s limit_bytes=%d setting=%s: %s", control, limit, setting, err.Error())
	return util.NewInvalidInputErrorWithDetails(err, err.Error())
}
