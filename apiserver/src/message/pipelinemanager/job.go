package pipelinemanager

import "time"

type Job struct {
	Name     string     `json:"name" db:"name"`
	CreateAt *time.Time `json:"createdAt,omitempty" db:"createAt"`
	StartAt  *time.Time `json:"startedAt,omitempty" db:"startedAt"`
	FinishAt *time.Time `json:"finishedAt,omitempty" db:"finishedAt"`
	Status   string     `json:"status,omitempty" db:"status"`
}
