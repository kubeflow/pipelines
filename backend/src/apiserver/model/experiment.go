package model

type Experiment struct {
	UUID           string `gorm:"column:UUID; not null; primary_key"`
	Name           string `gorm:"column:Name; not null; unique_index:idx_name_namespace"`
	Description    string `gorm:"column:Description; not null"`
	CreatedAtInSec int64  `gorm:"column:CreatedAtInSec; not null"`
	Namespace      string `gorm:"column:Namespace; not null; unique_index:idx_name_namespace"`
	StorageState   string `gorm:"column:StorageState; not null;"`
}

func (e Experiment) GetValueOfPrimaryKey() string {
	return e.UUID
}

func GetExperimentTablePrimaryKeyColumn() string {
	return "UUID"
}

// PrimaryKeyColumnName returns the primary key for model Experiment.
func (e *Experiment) PrimaryKeyColumnName() string {
	return "UUID"
}

// DefaultSortField returns the default sorting field for model Experiment.
func (e *Experiment) DefaultSortField() string {
	return "CreatedAtInSec"
}

var experimentAPIToModelFieldMap = map[string]string{
	"id":            "UUID",
	"name":          "Name",
	"created_at":    "CreatedAtInSec",
	"description":   "Description",
	"namespace":     "Namespace",
	"storage_state": "StorageState",
}

// APIToModelFieldMap returns a map from API names to field names for model
// Experiment.
func (e *Experiment) APIToModelFieldMap() map[string]string {
	return experimentAPIToModelFieldMap
}

// GetModelName returns table name used as sort field prefix
func (e *Experiment) GetModelName() string {
	return "experiments"
}

func (e *Experiment) GetField(name string) (string, bool) {
	if field, ok := experimentAPIToModelFieldMap[name]; ok {
		return field, true
	}
	return "", false
}

func (e *Experiment) GetFieldValue(name string) interface{} {
	switch name {
	case "UUID":
		return e.UUID
	case "Name":
		return e.Name
	case "CreatedAtInSec":
		return e.CreatedAtInSec
	case "Description":
		return e.Description
	case "Namespace":
		return e.Namespace
	case "StorageState":
		return e.StorageState
	default:
		return nil
	}
}

func (e *Experiment) GetSortByFieldPrefix(name string) string {
	return "experiments."
}

func (e *Experiment) GetKeyFieldPrefix() string {
	return "experiments."
}
