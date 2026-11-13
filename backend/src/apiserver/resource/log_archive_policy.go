// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package resource

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	kvalidation "k8s.io/apimachinery/pkg/util/validation"
)

const defaultLogArchiveKeyFormat = "private-artifacts/{{workflow.namespace}}/{{workflow.name}}/{{workflow.creationTimestamp.Y}}/{{workflow.creationTimestamp.m}}/{{workflow.creationTimestamp.d}}/{{pod.name}}"

// RunLogArchiveContext contains authorized run scope and untrusted archive hints.
// Callers must authorize readLog before exposing or using this context.
type RunLogArchiveContext struct {
	RunID        string `json:"runId"`
	Namespace    string `json:"namespace"`
	WorkflowName string `json:"workflowName"`
	// Imported history must never select live resources in the destination cluster.
	ArchiveOnly bool `json:"archiveOnly"`
	// These optional hints come from historical status and are NOT authorized
	// storage locations. Consumers must apply their operator-owned storage policy.
	ArchiveKey                string `json:"archiveKey,omitempty"`
	ArchiveBucket             string `json:"archiveBucket,omitempty"`
	ArchiveEndpoint           string `json:"archiveEndpoint,omitempty"`
	ArchiveInsecure           *bool  `json:"archiveInsecure,omitempty"`
	WorkflowCreationTimestamp string `json:"workflowCreationTimestamp,omitempty"`
}

// GetRunLogArchiveContext resolves persisted run scope and optional archive hints.
// The caller must authorize readLog for runID before using the result.
func (r *ResourceManager) GetRunLogArchiveContext(runID, nodeID string) (*RunLogArchiveContext, error) {
	run, err := r.GetRun(runID)
	if err != nil {
		return nil, err
	}
	// Standalone native requests keep the existing direct reader. Only the
	// explicit provenance flag is needed; legacy rows can lack a namespace.
	if !common.IsMultiUserMode() && run.ImportedFrom == nil {
		return &RunLogArchiveContext{RunID: run.UUID, Namespace: run.Namespace, WorkflowName: run.K8SName}, nil
	}
	identity, err := r.runLogArchiveContext(run)
	if err != nil {
		return nil, err
	}
	if len(kvalidation.IsDNS1123Subdomain(nodeID)) != 0 {
		return nil, util.NewInvalidInputError("Invalid log node name")
	}
	addArchiveHints(identity, string(run.WorkflowRuntimeManifest), nodeID)
	return identity, nil
}

func (r *ResourceManager) runLogArchiveContext(run *model.Run) (*RunLogArchiveContext, error) {
	namespace := run.Namespace
	if r.IsEmptyNamespace(namespace) {
		var err error
		namespace, err = r.GetNamespaceFromExperimentId(run.ExperimentId)
		if err != nil {
			return nil, err
		}
	}
	// Standalone transfer requires the source runtime namespace to match the
	// destination configuration. Legacy standalone rows can store an empty
	// logical namespace; use that operator-owned runtime scope, never metadata.
	if r.IsEmptyNamespace(namespace) && !common.IsMultiUserMode() {
		namespace = common.GetPodNamespace()
	}
	if len(kvalidation.IsDNS1123Label(namespace)) != 0 || len(kvalidation.IsDNS1123Subdomain(run.K8SName)) != 0 {
		return nil, util.NewFailedPreconditionError(fmt.Errorf("run namespace or workflow name is missing or invalid"), "Archived logs require a valid persisted namespace and workflow name")
	}
	return &RunLogArchiveContext{RunID: run.UUID, Namespace: namespace, WorkflowName: run.K8SName, ArchiveOnly: run.ImportedFrom != nil}, nil
}

// authorizeArchiveKey interprets only an operator-owned format. Tenant history
// nominates a key, but cannot expand the namespace/name scope authorized by SAR.
func authorizeArchiveKey(key, pod string, identity *RunLogArchiveContext) error {
	deny := func() error {
		return util.NewFailedPreconditionError(fmt.Errorf("archive key is outside the configured run scope"), "Archived log location is not authorized for this run")
	}
	if identity == nil || len(kvalidation.IsDNS1123Label(identity.Namespace)) != 0 || len(kvalidation.IsDNS1123Subdomain(identity.WorkflowName)) != 0 || len(kvalidation.IsDNS1123Subdomain(pod)) != 0 {
		return deny()
	}
	format := os.Getenv("KFP_LOG_ARCHIVE_KEY_FORMAT")
	if format == "" {
		format = defaultLogArchiveKeyFormat
	}
	expected, actual := strings.Split(format+"/main.log", "/"), strings.Split(key, "/")
	if len(expected) != len(actual) {
		return deny()
	}
	seen := map[string]bool{}
	date := map[string]int{}
	archivePod := ""
	for i, segment := range expected {
		value := actual[i]
		if value == "" || value == "." || value == ".." || strings.ContainsAny(value, "%\\?#") {
			return deny()
		}
		for _, c := range value {
			if c <= 0x20 || c == 0x7f {
				return deny()
			}
		}
		switch segment {
		case "{{workflow.namespace}}":
			if value != identity.Namespace {
				return deny()
			}
			seen[segment] = true
		case "{{workflow.name}}":
			if value != identity.WorkflowName {
				return deny()
			}
			seen[segment] = true
		case "{{pod.name}}":
			if len(kvalidation.IsDNS1123Subdomain(value)) != 0 || (archivePod != "" && archivePod != value) {
				return deny()
			}
			archivePod = value
			seen[segment] = true
		case "{{workflow.creationTimestamp.Y}}", "{{workflow.creationTimestamp.m}}", "{{workflow.creationTimestamp.d}}":
			width := 2
			if segment == "{{workflow.creationTimestamp.Y}}" {
				width = 4
			}
			if len(value) != width {
				return deny()
			}
			for _, c := range value {
				if c < '0' || c > '9' {
					return deny()
				}
			}
			number, err := strconv.Atoi(value)
			if err != nil || (seen[segment] && date[segment] != number) {
				return deny()
			}
			date[segment], seen[segment] = number, true
		default:
			if !archiveLiteralSegment(segment) || value != segment {
				return deny()
			}
		}
	}
	for _, placeholder := range []string{"{{workflow.namespace}}", "{{workflow.name}}", "{{pod.name}}"} {
		if !seen[placeholder] {
			return deny()
		}
	}
	// Custom formats may contain only some date components. Validate each
	// range, and the real calendar date when all three are present. Historical
	// workflow timestamps need not equal SQL creation time.
	y, m, d := date["{{workflow.creationTimestamp.Y}}"], date["{{workflow.creationTimestamp.m}}"], date["{{workflow.creationTimestamp.d}}"]
	if (seen["{{workflow.creationTimestamp.Y}}"] && y == 0) ||
		(seen["{{workflow.creationTimestamp.m}}"] && (m < 1 || m > 12)) ||
		(seen["{{workflow.creationTimestamp.d}}"] && (d < 1 || d > 31)) {
		return deny()
	}
	if len(date) == 3 {
		parsed := time.Date(y, time.Month(m), d, 0, 0, 0, 0, time.UTC)
		if parsed.Year() != y || int(parsed.Month()) != m || parsed.Day() != d {
			return deny()
		}
	}
	return nil
}

// Archived history is advisory input only, including after its Workflow is GC'd.
func addArchiveHints(identity *RunLogArchiveContext, manifest, nodeID string) {
	var history struct {
		Metadata struct {
			CreationTimestamp string `json:"creationTimestamp"`
		} `json:"metadata"`
		Status struct {
			ArtifactRepositoryRef struct {
				ArtifactRepository struct {
					S3 struct {
						Bucket   string `json:"bucket"`
						Endpoint string `json:"endpoint"`
						Insecure *bool  `json:"insecure"`
					} `json:"s3"`
				} `json:"artifactRepository"`
			} `json:"artifactRepositoryRef"`
		} `json:"status"`
	}
	if json.Unmarshal([]byte(manifest), &history) != nil {
		return
	}
	identity.WorkflowCreationTimestamp = history.Metadata.CreationTimestamp
	repository := history.Status.ArtifactRepositoryRef.ArtifactRepository.S3
	identity.ArchiveBucket, identity.ArchiveEndpoint, identity.ArchiveInsecure = repository.Bucket, repository.Endpoint, repository.Insecure
	if execution, err := util.NewExecutionSpecJSON(util.CurrentExecutionType(), []byte(manifest)); err == nil {
		identity.ArchiveKey = execution.ExecutionStatus().FindObjectStoreArtifactKeyOrEmpty(nodeID, "main-logs")
	}
}

func archiveLiteralSegment(segment string) bool {
	if segment == "" || segment == "." || segment == ".." {
		return false
	}
	for _, c := range segment {
		if (c < 'a' || c > 'z') && (c < 'A' || c > 'Z') && (c < '0' || c > '9') && c != '_' && c != '-' && c != '.' {
			return false
		}
	}
	return true
}
