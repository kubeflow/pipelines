package pipelinemanager

import "time"

// Metadata Common metadata for resources.
// This will be filled by gorm when the resource is stored to database.
type Metadata struct {
	ID        uint       `json:"id" gorm:"primary_key"`
	CreatedAt time.Time  `json:"createAt"`
	UpdatedAt time.Time  `json:"-"`
	DeletedAt *time.Time `json:"-" sql:"index"`
}
