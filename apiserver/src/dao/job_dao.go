package dao

import (
	"encoding/json"
	"ml/apiserver/src/message"
	"ml/apiserver/src/message/argo"
	"ml/apiserver/src/message/pipelinemanager"
	"ml/apiserver/src/util"
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

	var workflows argo.Workflows
	if err := json.Unmarshal(bodyBytes, &workflows); err != nil {
		return jobs, &util.InternalError{Message: "Failed to parse the workflows returned from K8s CRD"}
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
