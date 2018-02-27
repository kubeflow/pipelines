package argo

type Workflows struct {
	Items []Workflow `json:"items"`
}

type Workflow struct {
	Metadata WorkflowMetadata `json:"metadata"`
	Status   WorkflowStatus   `json:"status"`
}

type WorkflowMetadata struct {
	Name              string `json:"name"`
	CreationTimestamp string `json:"creationTimestamp"`
}

type WorkflowStatus struct {
	StartTimestamp  string `json:"startedAt"`
	FinishTimestamp string `json:"finishedAt"`
	Status          string `json:"phase"`
}
