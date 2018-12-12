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

func (e *Experiment) PrimaryKeyValue() string {
	return e.UUID
}

func (e *Experiment) PrimaryKeyColumnName() string {
	return "UUID"
}

func (e *Experiment) DefaultSortField() string {
	return "CreatedAtInSec"
}

var experimentAPIToModelFieldMap = map[string]string{
	"id":          "UUID",
	"name":        "Name",
	"created_at":  "CreatedAtInSec",
	"description": "Description",
}

func (e *Experiment) APIToModelFieldMap() map[string]string {
	return experimentAPIToModelFieldMap
}
