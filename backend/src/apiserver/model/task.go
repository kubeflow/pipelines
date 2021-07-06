package model

type Task struct {
	UUID              string `gorm:"column:UUID; not null; primary_key"`
	Namespace         string `gorm:"column:Namespace; not null;"`
	PipelineName      string `gorm:"column:PipelineName; not null;"`
	RunUUID           string `gorm:"column:RunUUID; not null;"`
	MLMDExecutionID   string `gorm:"column:MLMDExecutionID; not null;"`
	CreatedTimestamp  int64  `gorm:"column:CreatedTimestamp; not null"`
	FinishedTimestamp int64  `gorm:"column:FinishedTimestamp"`
	Fingerprint       string `gorm:"column:Fingerprint; not null;"`
}

func (t Task) PrimaryKeyColumnName() string {
	return "UUID"
}

func (t Task) DefaultSortField() string {
	return "CreatedTimestamp"
}

func (t Task) APIToModelFieldMap() map[string]string {
	return taskAPIToModelFieldMap
}

func (t Task) GetModelName() string {
	return "tasks"
}

func (t Task) GetSortByFieldPrefix(s string) string {
	return "tasks."
}

func (t Task) GetKeyFieldPrefix() string {
	return "tasks."
}

var taskAPIToModelFieldMap = map[string]string{
	"id":              "UUID",
	"namespace":       "Namespace",
	"pipelineName":    "PipelineName",
	"runId":           "RunUUID ",
	"mlmdExecutionID": "MLMDExecutionID",
	"created_at":      "CreatedTimestamp",
	"finished_at":     "FinishedTimestamp",
	"fingerprint":     "Fingerprint",
}

func (t Task) GetField(name string) (string, bool) {
	if field, ok := taskAPIToModelFieldMap[name]; ok {
		return field, true
	}
	return "", false
}

func (t Task) GetFieldValue(name string) interface{} {
	switch name {
	case "UUID":
		return t.UUID
	case "Namespace":
		return t.Namespace
	case "PipelineName":
		return t.PipelineName
	case "RunUUID":
		return t.RunUUID
	case "MLMDExecutionID":
		return t.MLMDExecutionID
	case "CreatedTimestamp":
		return t.CreatedTimestamp
	case "FinishedTimestamp":
		return t.FinishedTimestamp
	case "Fingerprint":
		return t.Fingerprint
	default:
		return nil
	}
}
