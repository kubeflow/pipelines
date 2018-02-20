package argo

type Workflows struct {
	Items []Workflow `json:"items"`
}

type Workflow struct {
	Metadata WorkflowMetadata `json:"Metadata"`
	Status   WorkflowStatus   `json:"Status"`
}

type WorkflowMetadata struct {
	Name              string `json:"Name"`
	CreationTimestamp string `json:"CreationTimestamp"`
}

type WorkflowStatus struct {
	StartTimestamp  string `json:"startedAt"`
	FinishTimestamp string `json:"finishedAt"`
	Status          string `json:"phase"`
}
