package pipelinemanager

import "github.com/jinzhu/gorm"

type Package struct {
	gorm.Model
	Name        string `json:"name" `
	Description string `json:"description" `
}
