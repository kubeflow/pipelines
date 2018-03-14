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
	"encoding/json"
	"ml/apiserver/src/message/argo"
	"ml/apiserver/src/message/pipelinemanager"
	"ml/apiserver/src/util"
)

type JobStoreInterface interface {
	ListJobs() ([]pipelinemanager.Job, error)
	ListJobs2() (argo.WorkflowList, error)
	CreateJob(workflow []byte) (pipelinemanager.Job, error)
}

type JobStore struct {
	argoClient ArgoClientInterface
}

func (s *JobStore) ListJobs() ([]pipelinemanager.Job, error) {
	var jobs []pipelinemanager.Job

	bodyBytes, _ := s.argoClient.Request("GET", "workflows", nil)

	var workflows argo.WorkflowList
	if err := json.Unmarshal(bodyBytes, &workflows); err != nil {
		return jobs, util.NewInternalError("Failed to get jobs", "Failed to parse the workflows returned from K8s CRD. Error: %s", err.Error())
	}

	for _, workflow := range workflows.Items {
		job := pipelinemanager.ToJob(workflow)
		jobs = append(jobs, job)
	}

	return jobs, nil
}

func (s *JobStore) ListJobs2() (argo.WorkflowList, error) {

	bodyBytes, _ := s.argoClient.Request("GET", "workflows", nil)

	var workflows argo.WorkflowList
	if err := json.Unmarshal(bodyBytes, &workflows); err != nil {
		return workflows, util.NewInternalError("Failed to get jobs", "Failed to parse the workflows returned from K8s CRD. Error: %s", err.Error())
	}

	return workflows, nil
}

func (s *JobStore) CreateJob(workflow []byte) (pipelinemanager.Job, error) {
	var job pipelinemanager.Job

	bodyBytes, _ := s.argoClient.Request("POST", "workflows", workflow)

	var wf argo.Workflow
	if err := json.Unmarshal(bodyBytes, &wf); err != nil {
		return job, util.NewInternalError("Failed to create job", "Failed to parse the workflow returned from K8s CRD. Error: %s", err.Error())
	}
	job = pipelinemanager.ToJob(wf)
	return job, nil
}

// factory function for package store
func NewJobStore(argoClient ArgoClientInterface) *JobStore {
	return &JobStore{argoClient}
}
