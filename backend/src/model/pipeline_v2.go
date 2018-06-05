package model

type PipelineV2 struct {
	UUID           string `gorm:"column:UUID; primary_key"`
	Name           string `gorm:"column:Name; not null"`
	Namespace      string `gorm:"column:Namespace; not null;"`
	CreatedAtInSec int64  `gorm:"column:CreatedAtInSec; not null"` /* The time this record is stored in DB*/
	StartedAtInSec int64  `gorm:"column:CreatedAtInSec; not null"` /* The time that pipeline is started */
	UpdateAtInSec  int64  `gorm:"column:UpdateAtInSec; not null"`
	Description    string `gorm:"column:Description"`
	PackageId      uint32 `gorm:"column:PackageId; not null"`
	Enabled        bool   `gorm:"column:Enabled; not null"`
	Condition      string `gorm:"column:Condition; not null"`
	/* Argo CRD Status. Set size to 65535 so it will be stored as longtext. https://dev.mysql.com/doc/refman/8.0/en/column-count-limit.html */
	Status string `gorm:"column:Status; size:65535"`
	/* Argo CRD Spec. Set size to 65535 so it will be stored as longtext. https://dev.mysql.com/doc/refman/8.0/en/column-count-limit.html */
	Spec string `gorm:"column:Spec; size:65535"`
}
