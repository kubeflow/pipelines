package pipelinemanager

import "time"

type Job struct {
	Name              string `json:"name"`
	CreationTimestamp time.Time `json:"createdAt"`
	StartTimestamp    time.Time `json:"startedAt"`
	FinishTimestamp   time.Time `json:"finishedAt"`
	Status            string `json:"status"`
}
