package pipelinemanager

import "github.com/jinzhu/gorm"

type Pipeline struct {
	gorm.Model
	Name        string      `json:"name" gorm:"not null"`
	Description string      `json:"description,omitempty"`
	PackageId   string      `json:"packageId" gorm:"not null"`
	Parameters  []Parameter `json:"parameters" gorm:"polymorphic:Owner;"`
}
