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
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"

	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"go.uber.org/zap/zapcore"
)

const (
	// This regex expresses the following constraints:
	// * Allows lowercase letters and numbers at the beginning
	// * Allows lowercase letters, numbers, and "-" in everywhere else
	// * Sets max length to be 128 characters
	pipelineNamePattern = "^[a-z0-9][a-z0-9-]{0,127}$"
	// PipelineIDResourceNameKey identifies pipeline IDs returned by ParseResourceIdsFromFullName.
	PipelineIDResourceNameKey = "PipelineId"
	// PipelineVersionIDResourceNameKey identifies version IDs returned by ParseResourceIdsFromFullName.
	PipelineVersionIDResourceNameKey = "PipelineVersionId"
)

// CreateArtifactPath creates artifact resource path.
func CreateArtifactPath(runID string, nodeID string, artifactName string) string {
	return fmt.Sprintf("runs/%s/nodes/%s/artifacts/%s", runID, nodeID, artifactName)
}

// Fetches parent resource ids from a resource name.
// Supports namespaces, pipelines, pipeline versions, experiments, runs, recurring runs.
//
// Example:
//
//	namespaces/ns1/pipelines/p1/versions/p1.1 results in
//	  Namespace         : ns1
//	  PipelineId        : p1
//	  PipelineVersionId : p1.1
func ParseResourceIdsFromFullName(p string) map[string]string {
	p = strings.TrimPrefix(strings.TrimSuffix(p, "/"), "/")
	results := map[string]string{
		"Namespace":                      "",
		"ExperimentId":                   "",
		PipelineIDResourceNameKey:        "",
		PipelineVersionIDResourceNameKey: "",
		"RunID":                          "",
		"RecurringRunId":                 "",
		"ArtifactId":                     "",
		"ExecutionId":                    "",
	}
	names := strings.Split(p, "/")
	i := 0
	for i < len(names) {
		if i+1 < len(names) {
			switch strings.ToLower(names[i]) {
			case "namespaces", "namespace":
				results["Namespace"] = names[i+1]
			case "pipelines", "pipeline":
				results[PipelineIDResourceNameKey] = names[i+1]
			case "versions", "version", "pipelineversions", "pipelineversion", "pipeline_versions", "pipeline_version":
				results[PipelineVersionIDResourceNameKey] = names[i+1]
			case "experiments", "experiment":
				results["ExperimentId"] = names[i+1]
			case "runs", "run":
				results["RunID"] = names[i+1]
			case "jobs", "job", "recurringruns", "recurringrun", "recurring_runs", "recurring_run":
				results["RecurringRunId"] = names[i+1]
			case "artifacts", "artifact":
				results["ArtifactId"] = names[i+1]
			case "executions", "execution":
				results["ExecutionId"] = names[i+1]
			}
		}
		i = i + 2
	}
	return results
}

// ValidatePipelineName Validates a pipeline name.
func ValidatePipelineName(pipelineName string) error {
	if pipelineName == "" {
		return util.NewInvalidInputError("pipeline's name cannot be empty")
	}
	if len(pipelineName) > 128 {
		return util.NewInvalidInputError("pipeline's name must contain no more than 128 characters")
	}
	if matched, err := regexp.MatchString(pipelineNamePattern, pipelineName); err != nil {
		return util.NewInternalServerError(
			err, "failed to compile pattern '%s'", pipelineNamePattern)
	} else if !matched {
		return util.NewInvalidInputError("pipeline's name must contain only lowercase alphanumeric characters or '-' and must start with alphanumeric characters")
	}
	return nil
}

func ParseLogLevel(logLevel string) (zapcore.Level, error) {
	switch logLevel {
	case "debug":
		return zapcore.DebugLevel, nil
	case "info":
		return zapcore.InfoLevel, nil
	case "warn":
		return zapcore.WarnLevel, nil
	case "error":
		return zapcore.ErrorLevel, nil
	case "panic":
		return zapcore.PanicLevel, nil
	case "fatal":
		return zapcore.FatalLevel, nil
	default:
		return zapcore.Level(0), fmt.Errorf("could not translate log level to ZAP levels: %s", logLevel)
	}
}

func FileExists(filePath string) bool {
	_, err := os.Stat(filePath)
	return !os.IsNotExist(err)
}

// CustomMarshaler will create a custom marshaler to use snake_case
// for proto field names, and allow the api server to error on
// invalid fields.
func CustomMarshaler() *runtime.JSONPb {
	return &runtime.JSONPb{
		MarshalOptions: protojson.MarshalOptions{
			// This allows us to use proto field names which are
			// in snake_case format
			UseProtoNames:   true,
			EmitUnpopulated: false,
		},
		UnmarshalOptions: protojson.UnmarshalOptions{
			// We want to allow the api server to error on
			// invalid fields
			DiscardUnknown: false,
		},
	}
}

type contextKey string

const (
	AuthoritativeFinalStatusContextKey contextKey = "kfp-authoritative-final-status"
)

// WithAuthoritativeRequirement returns a context indicating that authoritative final status is required,
// setting both the in-process context value and outgoing gRPC metadata.
func WithAuthoritativeRequirement(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	ctx = context.WithValue(ctx, AuthoritativeFinalStatusContextKey, true)
	return metadata.AppendToOutgoingContext(ctx, AuthoritativeFinalStatusMetadataKey, "true")
}

// IsAuthoritativeFinalStatusRequired reports whether the context requires authoritative final status.
func IsAuthoritativeFinalStatusRequired(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	if v, ok := ctx.Value(AuthoritativeFinalStatusContextKey).(bool); ok && v {
		return true
	}
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if isHeaderTrue(md, AuthoritativeFinalStatusMetadataKey) ||
			isHeaderTrue(md, "x-kfp-authoritative") ||
			isHeaderTrue(md, "x-kfp-authoritative-sync") {
			return true
		}
	}
	if md, ok := metadata.FromOutgoingContext(ctx); ok {
		if isHeaderTrue(md, AuthoritativeFinalStatusMetadataKey) ||
			isHeaderTrue(md, "x-kfp-authoritative") ||
			isHeaderTrue(md, "x-kfp-authoritative-sync") {
			return true
		}
	}
	return false
}

func isHeaderTrue(md metadata.MD, key string) bool {
	for _, val := range md.Get(key) {
		if strings.EqualFold(val, "true") || val == "1" {
			return true
		}
	}
	return false
}
