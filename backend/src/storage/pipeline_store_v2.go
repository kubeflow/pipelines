package storage

import (
	"fmt"

	"bytes"

	"github.com/googleprivate/ml/backend/src/model"
	"github.com/googleprivate/ml/backend/src/util"
	"github.com/jinzhu/gorm"
)

type PipelineStoreV2Interface interface {
	ListPipelines(pageToken string, pageSize int, sortByFieldName string) ([]model.PipelineV2, string, error)
	GetPipeline(id string) (*model.PipelineV2, error)
	CreatePipeline(*model.PipelineV2) (*model.PipelineV2, error)
	DeletePipeline(id string) error
	EnablePipeline(id string, enabled bool) error
}

type PipelineStoreV2 struct {
	db   *gorm.DB
	time util.TimeInterface
}

func (s *PipelineStoreV2) ListPipelines(pageToken string, pageSize int, sortByFieldName string) ([]model.PipelineV2, string, error) {
	context, err := NewPaginationContext(pageToken, pageSize, sortByFieldName, model.GetPipelineV2TablePrimaryKeyColumn())
	if err != nil {
		return nil, "", err
	}
	models, pageToken, err := listModel(context, s.queryPipelineTable)
	if err != nil {
		return nil, "", util.Wrap(err, "List pipelines failed.")
	}
	return s.toPipelineMetadatas(models), pageToken, err
}

func (s *PipelineStoreV2) queryPipelineTable(context *PaginationContext) ([]model.ListableDataModel, error) {
	var pipelines []model.PipelineDetailV2
	var query bytes.Buffer
	query.WriteString(fmt.Sprintf("SELECT * FROM pipeline_detail_v2 "))
	toPaginationQuery("WHERE", &query, context)
	query.WriteString(fmt.Sprintf(" LIMIT %v", context.pageSize))

	if r := s.db.Raw(query.String()).Scan(&pipelines); r.Error != nil {
		return nil, util.NewInternalServerError(r.Error, "Failed to list pipelines: %v",
			r.Error.Error())
	}
	return s.toListableModels(pipelines), nil
}

func (s *PipelineStoreV2) GetPipeline(id string) (*model.PipelineV2, error) {
	var pipeline model.PipelineDetailV2
	// Get the pipeline as well as its parameter.
	r := s.db.Raw(`SELECT * FROM pipeline_detail_v2 WHERE UUID=? LIMIT 1`, id).Scan(&pipeline)
	if r.RecordNotFound() {
		return nil, util.NewResourceNotFoundError("Pipeline", fmt.Sprint(id))
	}
	if r.Error != nil {
		return nil, util.NewInternalServerError(r.Error, "Failed to get pipeline: %v", r.Error.Error())
	}
	return &pipeline.PipelineV2, nil
}

func (s *PipelineStoreV2) DeletePipeline(id string) error {
	r := s.db.Exec(`DELETE FROM pipeline_detail_v2 WHERE UUID=?`, id)
	if r.Error != nil {
		return util.NewInternalServerError(r.Error, "Failed to delete pipeline: %v", r.Error.Error())
	}
	return nil
}

func (s *PipelineStoreV2) CreatePipeline(p *model.PipelineV2) (*model.PipelineV2, error) {
	var newPipeline model.PipelineDetailV2
	newPipeline.PipelineV2 = *p
	now := s.time.Now().Unix()
	newPipeline.CreatedAtInSec = now
	newPipeline.UpdatedAtInSec = now

	if r := s.db.Create(&newPipeline); r.Error != nil {
		println(r.Error.Error())
		return nil, util.NewInternalServerError(r.Error, "Failed to add pipeline to pipeline table: %v",
			r.Error.Error())
	}
	return &newPipeline.PipelineV2, nil
}

func (s *PipelineStoreV2) EnablePipeline(id string, enabled bool) error {
	now := s.time.Now().Unix()
	r := s.db.Exec(`UPDATE pipeline_detail_v2 SET Enabled = ?, UpdatedAtInSec = ? WHERE UUID = ? and Enabled = ?`, enabled, now, id, !enabled)
	if r.Error != nil {
		return util.NewInternalServerError(r.Error, "Error when enabling pipeline %v to %v", id, enabled)
	}
	return nil
}

func (s *PipelineStoreV2) toListableModels(pipelines []model.PipelineDetailV2) []model.ListableDataModel {
	models := make([]model.ListableDataModel, len(pipelines))
	for i := range models {
		models[i] = pipelines[i].PipelineV2
	}
	return models
}

func (s *PipelineStoreV2) toPipelineMetadatas(models []model.ListableDataModel) []model.PipelineV2 {
	pipelines := make([]model.PipelineV2, len(models))
	for i := range models {
		pipelines[i] = models[i].(model.PipelineV2)
	}
	return pipelines
}

// factory function for pipeline store
func NewPipelineStoreV2(db *gorm.DB, time util.TimeInterface) *PipelineStoreV2 {
	return &PipelineStoreV2{
		db:   db,
		time: time,
	}
}
