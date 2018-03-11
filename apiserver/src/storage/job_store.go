package storage

import (
	"encoding/json"
	"ml/apiserver/src/message/pipelinemanager"
	"ml/apiserver/src/util"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/pkg/client/clientset/versioned/typed/workflow/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type JobStoreInterface interface {
	ListJobs() ([]pipelinemanager.Job, error)
	CreateJob(workflow []byte) (pipelinemanager.Job, error)
}

type JobStore struct {
	wfClient v1alpha1.WorkflowInterface
}

func (s *JobStore) ListJobs() ([]pipelinemanager.Job, error) {
	var jobs []pipelinemanager.Job
	wfList, err := s.wfClient.List(metav1.ListOptions{})
	if err != nil {
		return jobs, util.NewInternalError("Failed to get jobs", "Failed to get workflows from K8s CRD. Error: %s", err.Error())
	}
	for _, workflow := range wfList.Items {
		job := pipelinemanager.ToJob(workflow)
		jobs = append(jobs, job)
	}

	return jobs, nil
}

func (s *JobStore) CreateJob(workflow []byte) (pipelinemanager.Job, error) {
	var job pipelinemanager.Job
	var wf wfv1.Workflow
	json.Unmarshal(workflow, &wf)
	created, err := s.wfClient.Create(&wf)
	if err != nil {
		return job, util.NewInternalError("Failed to create job", "Failed to create workflow . Error: %s", err.Error())
	}
	job = pipelinemanager.ToJob(*created)
	return job, nil
}

// factory function for package store
func NewJobStore(wfClient v1alpha1.WorkflowInterface) *JobStore {
	return &JobStore{wfClient}
}
