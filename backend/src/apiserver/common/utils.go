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
	"regexp"
	"strings"

	"github.com/kubeflow/pipelines/backend/src/common/util"
	"go.uber.org/zap/zapcore"
)

const (
	// This regex expresses the following constraints:
	// * Allows lowercase letters and numbers at the beginning
	// * Allows lowercase letters, numbers, and "-" in everywhere else
	// * Sets max length to be 128 characters
	pipelineNamePattern = "^[a-z0-9][a-z0-9-]{0,127}$"
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
		"Namespace":         "",
		"ExperimentId":      "",
		"PipelineId":        "",
		"PipelineVersionId": "",
		"RunId":             "",
		"RecurringRunId":    "",
		"ArtifactId":        "",
		"ExecutionId":       "",
	}
	names := strings.Split(p, "/")
	i := 0
	for i < len(names) {
		if i+1 < len(names) {
			switch strings.ToLower(names[i]) {
			case "namespaces", "namespace":
				results["Namespace"] = names[i+1]
			case "pipelines", "pipeline":
				results["PipelineId"] = names[i+1]
			case "versions", "version", "pipelineversions", "pipelineversion", "pipeline_versions", "pipeline_version":
				results["PipelineVersionId"] = names[i+1]
			case "experiments", "experiment":
				results["ExperimentId"] = names[i+1]
			case "runs", "run":
				results["RunId"] = names[i+1]
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

// Mutate default values of specified pipeline spec.
// Args:
//
//	text: (part of) pipeline file in string.
func PatchPipelineDefaultParameter(text string) (string, error) {
	defaultBucket := GetStringConfig(DefaultBucketNameEnvVar)
	projectId := GetStringConfig(ProjectIDEnvVar)
	toPatch := map[string]string{
		"{{kfp-default-bucket}}": defaultBucket,
		"{{kfp-project-id}}":     projectId,
	}
	for key, value := range toPatch {
		text = strings.Replace(text, key, value, -1)
	}
	return text, nil
}

// Validates a pipeline name to match MLMD requirements.
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
