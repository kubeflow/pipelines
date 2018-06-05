package model

type JobV2 struct {
	UUID             string `gorm:"column:UUID; not null; primary_key"`
	Name             string `gorm:"column:Name; not null;"`
	Namespace        string `gorm:"column:Namespace; not null;"`
	PipelineID       uint32 `gorm:"column:PipelineID; not null"`
	CreatedAtInSec   int64  `gorm:"column:CreatedAtInSec; not null"`
	ScheduledAtInSec int64  `gorm:"column:ScheduledAtInSec; not null"`
	Condition        string `gorm:"column:Condition; not null"`
	/* Argo CRD Status. Set size to 65535 so it will be stored as longtext. https://dev.mysql.com/doc/refman/8.0/en/column-count-limit.html */
	Status string `gorm:"column:StatusDetails; size:65535"`
	/* Argo CRD Spec. Set size to 65535 so it will be stored as longtext. https://dev.mysql.com/doc/refman/8.0/en/column-count-limit.html */
	Spec string `gorm:"column:Spec; size:65535"`
}
