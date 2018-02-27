package pipelinemanager

type Job struct {
	Name              string `json:"name"`
	CreationTimestamp string `json:"createdAt"`
	StartTimestamp    string `json:"startedAt"`
	FinishTimestamp   string `json:"finishedAt"`
	Status            string `json:"status"`
}
