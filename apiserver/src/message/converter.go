package message

import (
	"ml/apiserver/src/message/argo"
	"ml/apiserver/src/message/pipelinemanager"
)

func Convert(workflow argo.Workflow) pipelinemanager.Job {
	return pipelinemanager.Job{Name: workflow.Metadata.Name,
		CreationTimestamp: workflow.Metadata.CreationTimestamp,
		StartTimestamp: workflow.Status.StartTimestamp,
		FinishTimestamp: workflow.Status.FinishTimestamp,
		Status: workflow.Status.Status}
}
