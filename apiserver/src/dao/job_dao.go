package dao

import (
	"encoding/json"

	"ml/apiserver/src/message"
	"ml/apiserver/src/message/argo"
	"ml/apiserver/src/message/pipelinemanager"

	"github.com/golang/glog"
)

type JobDaoInterface interface {
	ListJobs() ([]pipelinemanager.Job, error)
}

type JobDao struct {
	argoClient ArgoClientInterface
}

func (dao *JobDao) ListJobs() ([]pipelinemanager.Job, error) {
	var jobs []pipelinemanager.Job

	bodyBytes, _ := dao.argoClient.Request("GET", "workflows")

	var workflows argo.WorkflowList
	if err := json.Unmarshal(bodyBytes, &workflows); err != nil {
		glog.Fatalf("Failed to parse workflows: %v", err)
	}

	for _, workflow := range workflows.Items {
		job := message.Convert(workflow)
		jobs = append(jobs, job)
	}

	return jobs, nil
}

// factory function for package DAO
func NewJobDao(argoClient ArgoClientInterface) *JobDao {
	return &JobDao{argoClient}
}
