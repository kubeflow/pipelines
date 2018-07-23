package model

type Job struct {
	UUID             string `gorm:"column:UUID; not null; primary_key"`
	Name             string `gorm:"column:Name; not null"`
	Namespace        string `gorm:"column:Namespace; not null"`
	PipelineID       string `gorm:"column:PipelineID; not null"`
	CreatedAtInSec   int64  `gorm:"column:CreatedAtInSec; not null"`
	ScheduledAtInSec int64  `gorm:"column:ScheduledAtInSec; not null"`
	Conditions       string `gorm:"column:Conditions; not null"`
}

type JobDetail struct {
	Job
	/* Argo CRD. Set size to 65535 so it will be stored as longtext. https://dev.mysql.com/doc/refman/8.0/en/column-count-limit.html */
	Workflow string `gorm:"column:Workflow; not null; size:65535"`
}

func (j Job) GetValueOfPrimaryKey() string {
	return j.UUID
}

func GetJobTablePrimaryKeyColumn() string {
	return "UUID"
}
