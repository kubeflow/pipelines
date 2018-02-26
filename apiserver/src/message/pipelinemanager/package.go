package pipelinemanager

type Package struct {
	Id          string `json:"id" db:"id"`
	Name        string `json:"name" db:"name"`
	Description string `json:"description,omitempty" db:"description"`
}
