package message

import (
	"ml/apiserver/src/message/argo"
	"ml/apiserver/src/message/pipelinemanager"
)

func Convert(workflow argo.Workflow) pipelinemanager.Job {
	return pipelinemanager.Job{Name: workflow.ObjectMeta.Name,
		CreationTimestamp: workflow.ObjectMeta.CreationTimestamp.Time,
		StartTimestamp: workflow.Status.StartedAt.Time,
		FinishTimestamp: workflow.Status.FinishedAt.Time,
		Status: string(workflow.Status.Phase)}
}
