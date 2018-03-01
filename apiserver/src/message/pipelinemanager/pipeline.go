package pipelinemanager

type Pipeline struct {
	*Metadata               `json:",omitempty"`
	Name        string      `json:"name" gorm:"not null"`
	Description string      `json:"description,omitempty"`
	PackageId   uint        `json:"packageId" gorm:"not null"`
	Parameters  []Parameter `json:"parameters,omitempty" gorm:"polymorphic:Owner;"`
}
