// Copyright 2018 Google LLC
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

package storage

import (
	"ml/apiserver/src/util"

	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	workflowclient "github.com/argoproj/argo/pkg/client/clientset/versioned/typed/workflow/v1alpha1"
	k8sclient "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type JobStoreInterface interface {
	GetJob(name string) (*v1alpha1.Workflow, error)
	ListJobs() ([]v1alpha1.Workflow, error)
	CreateJob(*v1alpha1.Workflow) (*v1alpha1.Workflow, error)
}

type JobStore struct {
	wfClient workflowclient.WorkflowInterface
}

func (s *JobStore) ListJobs() ([]v1alpha1.Workflow, error) {
	jobs, err := s.wfClient.List(k8sclient.ListOptions{})
	if err != nil {
		return nil, util.NewInternalError("Failed to list jobs",
			"Failed to list workflows from K8s CRD. Error: %s", err.Error())
	}
	return jobs.Items, nil
}

func (s *JobStore) CreateJob(wf *v1alpha1.Workflow) (*v1alpha1.Workflow, error) {
	job, err := s.wfClient.Create(wf)
	if err != nil {
		return job, util.NewInternalError("Failed to create job",
			"Failed to create workflow . Error: %s", err.Error())
	}
	return job, nil
}

func (s *JobStore) GetJob(name string) (*v1alpha1.Workflow, error) {
	job, err := s.wfClient.Get(name, k8sclient.GetOptions{})
	if err != nil {
		return job, util.NewInternalError("Failed to get a job",
			"Failed to get workflow %s from K8s CRD. Error: %s", name, err.Error())
	}
	return job, nil
}

// factory function for package store
func NewJobStore(wfClient workflowclient.WorkflowInterface) *JobStore {
	return &JobStore{wfClient}
}
