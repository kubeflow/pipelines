package pipelinemanager

import "time"

type Job struct {
	Name              string    `json:"name"`
	CreationTimestamp *time.Time `json:"createdAt,omitempty"`
	StartTimestamp    *time.Time `json:"startedAt,omitempty"`
	FinishTimestamp   *time.Time `json:"finishedAt,omitempty"`
	Status            string    `json:"status,omitempty"`
}
