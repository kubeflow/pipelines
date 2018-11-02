package model

type Experiment struct {
	UUID           string `gorm:"column:UUID; not null; primary_key"`
	Name           string `gorm:"column:Name; not null; unique"`
	Description    string `gorm:"column:Description; not null"`
	CreatedAtInSec int64  `gorm:"column:CreatedAtInSec; not null"`
}

func (r Experiment) GetValueOfPrimaryKey() string {
	return r.UUID
}

func GetExperimentTablePrimaryKeyColumn() string {
	return "UUID"
}
