package pipelinemanager

import "time"

type Job struct {
	Name     string     `json:"name"`
	CreateAt *time.Time `json:"createdAt,omitempty"`
	StartAt  *time.Time `json:"startedAt,omitempty"`
	FinishAt *time.Time `json:"finishedAt,omitempty"`
	Status   string     `json:"status,omitempty"`
}
