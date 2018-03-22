// Copyright 2018 The Kubeflow Authors
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

package storage

import (
	"fmt"
	"ml/src/message"
	"ml/src/util"

	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/google/uuid"
)

type jobMapWrapper struct {
	// Maps an Argo workflow name to job details.
	jobMap map[string]message.JobDetail
}

type FakePersistentJobStore struct {
	// Maps a pipeline ID to a map of Argo jobs
	pipelineMap map[uint]jobMapWrapper
}

func NewFakePersistentJobStore() *FakePersistentJobStore {
	result := &FakePersistentJobStore{
		pipelineMap: make(map[uint]jobMapWrapper),
	}
	return result
}

func (s *FakePersistentJobStore) GetJob(pipelineId uint, name string) (message.JobDetail, error) {
	if jobMapWrapper, pipelineFound := s.pipelineMap[pipelineId]; pipelineFound {
		// If the pipeline is found.
		if jobDetail, workflowFound := jobMapWrapper.jobMap[name]; workflowFound {
			// If the job is found, we return the job details.
			return jobDetail, nil
		} else {
			// If the job is not found, we return an error.
			return message.JobDetail{}, util.NewResourceNotFoundError("workflow", name)
		}
	} else {
		// If the pipeline is not found, we return an error.
		return message.JobDetail{}, util.NewResourceNotFoundError(
			"pipeline", fmt.Sprintf("%v", pipelineId))
	}
}

func (s *FakePersistentJobStore) ListJobs(pipelineId uint) ([]message.Job, error) {
	return nil, util.NewInternalError("not implemented", "")
}

func (s *FakePersistentJobStore) CreateJob(pipelineId uint, workflow *v1alpha1.Workflow) (message.JobDetail, error) {

	// Creating the job details.
	workflow.Name = uuid.New().String()
	jobDetails := message.JobDetail{
		Workflow: workflow,
	}

	if jobMap, ok := s.pipelineMap[pipelineId]; ok {
		// If we find the pipeline ID, we add the new job to the job map.
		jobMap.jobMap[workflow.Name] = jobDetails
	} else {
		// If we do not find the pipeline ID, we first create the job map.
		jobMap := make(map[string]message.JobDetail)
		jobMap[workflow.Name] = jobDetails

		s.pipelineMap[pipelineId] = jobMapWrapper{
			jobMap: jobMap,
		}
	}

	return jobDetails, nil
}

func (s *FakePersistentJobStore) GetJobCountForPipeline(pipelineId uint) int {
	if jobMap, ok := s.pipelineMap[pipelineId]; ok {
		// If a pipeline is found, we return the number of jobs in the map.
		return len(jobMap.jobMap)
	} else {
		// If there is no pipeline found, we return that no jobs are found.
		return 0
	}
}
