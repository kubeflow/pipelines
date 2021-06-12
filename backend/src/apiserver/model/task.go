package model

type Task struct {
	UUID             string `gorm:"column:UUID; not null; primary_key"`
	Namespace        string `gorm:"column:Namespace; not null;"`
	PipelineName     string `gorm:"column:PipelineName; not null;"`
	RunUUID          string `gorm:"column:ExperimentUUID; not null;"`
	MLMDExecutionID  string `gorm:"column:MLMDExecutionID; not null;"`
	CreatedTimestamp int64  `gorm:"column:CreatedTimestamp; not null"`
	EndedTimestamp   int64  `gorm:"column:EndedTimestamp"`
	Fingerprint      string `gorm:"column:Fingerprint; not null;"`
}
