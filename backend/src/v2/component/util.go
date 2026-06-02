// Copyright 2025 The Kubeflow Authors
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

package component

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	apiV2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"google.golang.org/protobuf/types/known/structpb"
)

// CopyThisBinary copies the running binary into destination path.
func CopyThisBinary(destination string) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("copy this binary to %s: %w", destination, err)
		}
	}()

	path, err := findThisBinary()
	if err != nil {
		return err
	}
	src, err := os.Open(path)
	if err != nil {
		return err
	}
	defer src.Close()
	dst, err := os.OpenFile(destination, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o555) // 0o555 -> readable and executable by all
	if err != nil {
		return err
	}
	defer dst.Close()
	if _, err = io.Copy(dst, src); err != nil {
		return err
	}
	return dst.Close()
}

func findThisBinary() (string, error) {
	path, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("findThisBinary failed: %w", err)
	}
	return path, nil
}

func textToPbValue(text string, t pipelinespec.ParameterType_ParameterTypeEnum) (*structpb.Value, error) {
	msg := func(err error) error {
		return fmt.Errorf("TextToPbValue(text=%q, type=%q) failed: %w", text, t, err)
	}
	switch t {
	case pipelinespec.ParameterType_STRING:
		return structpb.NewStringValue(text), nil
	case pipelinespec.ParameterType_NUMBER_INTEGER:
		i, err := strconv.ParseInt(strings.TrimSpace(text), 10, 0)
		if err != nil {
			return nil, msg(err)
		}
		return structpb.NewNumberValue(float64(i)), nil
	case pipelinespec.ParameterType_NUMBER_DOUBLE:
		f, err := strconv.ParseFloat(strings.TrimSpace(text), 64)
		if err != nil {
			return nil, msg(err)
		}
		return structpb.NewNumberValue(f), nil
	case pipelinespec.ParameterType_BOOLEAN:
		v, err := strconv.ParseBool(strings.TrimSpace(text))
		if err != nil {
			return nil, msg(err)
		}
		return structpb.NewBoolValue(v), nil
	case pipelinespec.ParameterType_LIST:
		v := &structpb.Value{}
		if err := v.UnmarshalJSON([]byte(text)); err != nil {
			return nil, msg(err)
		}
		if _, ok := v.GetKind().(*structpb.Value_ListValue); !ok {
			return nil, msg(fmt.Errorf("unexpected type"))
		}
		return v, nil
	case pipelinespec.ParameterType_STRUCT:
		v := &structpb.Value{}
		if err := v.UnmarshalJSON([]byte(text)); err != nil {
			return nil, msg(err)
		}
		if _, ok := v.GetKind().(*structpb.Value_StructValue); !ok {
			return nil, msg(fmt.Errorf("unexpected type"))
		}
		return v, nil
	default:
		return nil, msg(fmt.Errorf("unknown type. Expected STRING, NUMBER_INTEGER, NUMBER_DOUBLE, BOOLEAN, LIST or STRUCT"))
	}
}

var artifactTypeSchemaToArtifactTypeMap = map[string]apiV2beta1.Artifact_ArtifactType{
	"system.Artifact":                    apiV2beta1.Artifact_Artifact,
	"system.Dataset":                     apiV2beta1.Artifact_Dataset,
	"system.Model":                       apiV2beta1.Artifact_Model,
	"system.Metrics":                     apiV2beta1.Artifact_Metric,
	"system.ClassificationMetrics":       apiV2beta1.Artifact_ClassificationMetric,
	"system.SlicedClassificationMetrics": apiV2beta1.Artifact_SlicedClassificationMetric,
	"system.HTML":                        apiV2beta1.Artifact_HTML,
	"system.Markdown":                    apiV2beta1.Artifact_Markdown,
}

func artifactTypeSchemaToArtifactType(typeSchema string) (apiV2beta1.Artifact_ArtifactType, error) {
	if artifactType, ok := artifactTypeSchemaToArtifactTypeMap[typeSchema]; ok {
		return artifactType, nil
	}
	return apiV2beta1.Artifact_TYPE_UNSPECIFIED, fmt.Errorf("unknown artifact type: %s", typeSchema)
}
