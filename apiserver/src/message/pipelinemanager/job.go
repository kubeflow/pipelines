package pipelinemanager

import (
	"time"

	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

type Job struct {
	Name     string     `json:"name"`
	CreateAt *time.Time `json:"createdAt,omitempty"`
	StartAt  *time.Time `json:"startedAt,omitempty"`
	FinishAt *time.Time `json:"finishedAt,omitempty"`
	Status   string     `json:"status,omitempty"`
}

func ToJob(workflow v1alpha1.Workflow) Job {
	return Job{
		Name:     workflow.ObjectMeta.Name,
		CreateAt: &workflow.ObjectMeta.CreationTimestamp.Time,
		StartAt:  &workflow.Status.StartedAt.Time,
		FinishAt: &workflow.Status.FinishedAt.Time,
		Status:   string(workflow.Status.Phase)}
}
