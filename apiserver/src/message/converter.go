package message

import (
	"ml/apiserver/src/message/argo"
	"ml/apiserver/src/message/pipelinemanager"
)

func Convert(workflow argo.Workflow) pipelinemanager.Job {
	return pipelinemanager.Job{
		Name:     workflow.ObjectMeta.Name,
		CreateAt: &workflow.ObjectMeta.CreationTimestamp.Time,
		StartAt:  &workflow.Status.StartedAt.Time,
		FinishAt: &workflow.Status.FinishedAt.Time,
		Status:   string(workflow.Status.Phase)}
}
