package pipelinemanager

type Package struct {
	*Metadata               `json:",omitempty"`
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	Parameters  []Parameter `json:"parameters,omitempty" gorm:"polymorphic:Owner;"`
}
