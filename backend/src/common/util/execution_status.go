// Copyright 2022 The Kubeflow Authors
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

package util

import (
	api "github.com/kubeflow/pipelines/backend/api/go_client"
	"github.com/kubeflow/pipelines/backend/src/common"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type RetrieveArtifact func(request *api.ReadArtifactRequest, user string) (*api.ReadArtifactResponse, error)

// Abstract interface to encapsulate the resources of the execution runtime specifically
// for status information. This interface is mainly to access the status related information
type ExecutionStatus interface {
	// FindObjectStoreArtifactKeyOrEmpty loops through all node running statuses and look up the first
	// S3 artifact with the specified nodeID and artifactName. Returns empty if nothing is found.
	FindObjectStoreArtifactKeyOrEmpty(nodeID string, artifactName string) string

	// Get information of current phase, high-level summary of where the Execution is in its lifecycle.
	Condition() common.ExecutionPhase

	// UNIX time the execution finished. If Execution is not finished, return 0
	FinishedAt() int64

	// FinishedAt in Time format
	FinishedAtTime() v1.Time

	// StartedAt in Time format
	StartedAtTime() v1.Time

	// IsInFinalState whether the workflow is in a final state.
	IsInFinalState() bool

	// details about the ExecutionSpec's current condition.
	Message() string

	// This function was in metrics_reporter.go. Moved to here because it
	// accesses the orchestration engine specific data struct. encapsulate the
	// specific data struct and provide a abstract function here.
	CollectionMetrics(retrieveArtifact RetrieveArtifact, user string) ([]*api.RunMetric, []error)

	// does ExecutionStatus contain any finished node or not
	HasMetrics() bool
}
