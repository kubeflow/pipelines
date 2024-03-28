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

package common

// used this package to hold data types that are needed by
// multiple components to avoid import cycle issue.

// ExecutionPhase the phase for ExecutionStatus under common/util
type ExecutionPhase string

// borrow from Workflow.Status.Phase:
// https://pkg.go.dev/github.com/argoproj/argo-workflows/v3@v3.4.16/pkg/apis/workflow/v1alpha1#WorkflowPhase
const (
	ExecutionUnknown   ExecutionPhase = ""
	ExecutionPending   ExecutionPhase = "Pending" // pending some set-up - rarely used
	ExecutionRunning   ExecutionPhase = "Running" // any node has started; pods might not be running yet, the workflow maybe suspended too
	ExecutionSucceeded ExecutionPhase = "Succeeded"
	ExecutionFailed    ExecutionPhase = "Failed" // it maybe that the the workflow was terminated
	ExecutionError     ExecutionPhase = "Error"
)
