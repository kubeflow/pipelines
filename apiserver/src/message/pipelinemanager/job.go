package pipelinemanager

import (
	"ml/apiserver/src/message/argo"
	"time"
)

type Job struct {
	Name     string     `json:"name"`
	CreateAt *time.Time `json:"createdAt,omitempty"`
	StartAt  *time.Time `json:"startedAt,omitempty"`
	FinishAt *time.Time `json:"finishedAt,omitempty"`
	Status   string     `json:"status,omitempty"`
}

func ToJob(workflow argo.Workflow) Job {
	return Job{
		Name:     workflow.ObjectMeta.Name,
		CreateAt: &workflow.ObjectMeta.CreationTimestamp.Time,
		StartAt:  &workflow.Status.StartedAt.Time,
		FinishAt: &workflow.Status.FinishedAt.Time,
		Status:   string(workflow.Status.Phase)}
}
