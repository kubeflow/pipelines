package model

import "fmt"

// PipelineVersionStatus a label for the status of the Pipeline.
// This is intend to make pipeline creation and deletion atomic.
type PipelineVersionStatus string

const (
	PipelineVersionCreating PipelineVersionStatus = "CREATING"
	PipelineVersionReady    PipelineVersionStatus = "READY"
	PipelineVersionDeleting PipelineVersionStatus = "DELETING"
)

type PipelineVersion struct {
	UUID           string `gorm:"column:VersionUUID; not null; primary_key"`
	CreatedAtInSec int64  `gorm:"column:VersionCreatedAtInSec; not null"`
	Name           string `gorm:"column:VersionName; not null; unique"`
	// Set size to 65535 so it will be stored as longtext.
	// https://dev.mysql.com/doc/refman/8.0/en/column-count-limit.html
	Parameters string                `gorm:"column:VersionParameters; not null; size:65535"`
	// PipelineVersion belongs to Pipeline. If a pipeline with a specific UUID
	// is deleted from Pipeline table, all this pipeline's versions will be
	// deleted from PipelineVersion table.
	Pipeline Pipeline `gorm:"foreignkey:PipelineID;"`
	PipelineId string                `gorm:"column:PipelineId; not null;"`
	Status     PipelineVersionStatus `gorm:"column:VersionStatus; not null"`
	CodeSource
}

type CodeSource struct {
	// PipelineVersion can be based on a git commit or a package stored at some
	// url.
	// All fields below are optional.
	RepoName  string `gorm:"column:RepoName"`
	CommitSHA string `gorm:"column:CommitSHA"`

	URL string `gorm:"column:URL`
}

func (p PipelineVersion) GetValueOfPrimaryKey() string {
	return fmt.Sprint(p.UUID)
}

func GetPipelineVersionTablePrimaryKeyColumn() string {
	return "VersionUUID"
}

// PrimaryKeyColumnName returns the primary key for model PipelineVersion.
func (p *PipelineVersion) PrimaryKeyColumnName() string {
	return "VersionUUID"
}

// DefaultSortField returns the default sorting field for model Pipeline.
func (p *PipelineVersion) DefaultSortField() string {
	return "VersionCreatedAtInSec"
}

// APIToModelFieldMap returns a map from API names to field names for model
// PipelineVersion.
func (p *PipelineVersion) APIToModelFieldMap() map[string]string {
	return map[string]string{
		"id":          "VersionUUID",
		"name":        "VersionName",
		"created_at":  "VersionCreatedAtInSec",
		"pipeline_id": "PipelineId",
		"status":      "VersionStatus",
	}
}
