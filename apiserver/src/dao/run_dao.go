package dao

import (
	"encoding/json"

	"ml/apiserver/src/message"
	"ml/apiserver/src/message/argo"
	"ml/apiserver/src/message/pipelinemanager"

	"github.com/golang/glog"
)

type RunDaoInterface interface {
	ListRuns() ([]pipelinemanager.Run, error)
}

type RunDao struct {
	argoClient ArgoClientInterface
}

func (dao *RunDao) ListRuns() ([]pipelinemanager.Run, error) {
	var runs []pipelinemanager.Run

	bodyBytes, _ := dao.argoClient.Request("GET", "workflows")

	var workflows argo.Workflows
	if err := json.Unmarshal(bodyBytes, &workflows); err != nil {
		glog.Fatalf("Failed to parse workflows: %v", err)
	}

	for _, workflow := range workflows.Items {
		run := message.Convert(workflow)
		runs = append(runs, run)
	}

	return runs, nil
}

// factory function for template DAO
func NewRunDao(argoClient ArgoClientInterface) *RunDao {
	return &RunDao{argoClient}
}
