package storage

import (
	"encoding/json"
	"ml/apiserver/src/message"
	"ml/apiserver/src/message/argo"
	"ml/apiserver/src/message/pipelinemanager"
	"ml/apiserver/src/util"
)

type JobStoreInterface interface {
	ListJobs() ([]pipelinemanager.Job, error)
}

type JobStore struct {
	argoClient ArgoClientInterface
}

func (s *JobStore) ListJobs() ([]pipelinemanager.Job, error) {
	var jobs []pipelinemanager.Job

	bodyBytes, _ := s.argoClient.Request("GET", "workflows")

	var workflows argo.WorkflowList
	if err := json.Unmarshal(bodyBytes, &workflows); err != nil {
		return jobs, util.NewInternalError("Failed to get jobs", "Failed to parse the workflows returned from K8s CRD.", err.Error())
	}

	for _, workflow := range workflows.Items {
		job := message.Convert(workflow)
		jobs = append(jobs, job)
	}

	return jobs, nil
}

// factory function for package store
func NewJobStore(argoClient ArgoClientInterface) *JobStore {
	return &JobStore{argoClient}
}
