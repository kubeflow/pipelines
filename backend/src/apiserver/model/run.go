package model

type Run struct {
	UUID             string `gorm:"column:UUID; not null; primary_key"`
	Name             string `gorm:"column:Name; not null"`
	Namespace        string `gorm:"column:Namespace; not null"`
	JobID            string `gorm:"column:JobID; not null"`
	CreatedAtInSec   int64  `gorm:"column:CreatedAtInSec; not null"`
	ScheduledAtInSec int64  `gorm:"column:ScheduledAtInSec; not null"`
	Conditions       string `gorm:"column:Conditions; not null"`
}

type RunDetail struct {
	Run
	/* Argo CRD. Set size to 65535 so it will be stored as longtext. https://dev.mysql.com/doc/refman/8.0/en/column-count-limit.html */
	Workflow string `gorm:"column:Workflow; not null; size:65535"`
}

func (r Run) GetValueOfPrimaryKey() string {
	return r.UUID
}

func GetRunTablePrimaryKeyColumn() string {
	return "UUID"
}
