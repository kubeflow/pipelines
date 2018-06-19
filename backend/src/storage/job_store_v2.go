package storage

import (
	"fmt"

	"bytes"

	"github.com/googleprivate/ml/backend/src/model"
	"github.com/googleprivate/ml/backend/src/util"
	"github.com/jinzhu/gorm"
)

type JobStoreV2Interface interface {
	GetJob(pipelineId string, jobId string) (*model.JobDetailV2, error)
	ListJobs(pipelineId string, pageToken string, pageSize int, sortByFieldName string) ([]model.JobV2, string, error)
}

type JobStoreV2 struct {
	db   *gorm.DB
	time util.TimeInterface
}

// ListJobs list the job metadata for a pipeline from DB
func (s *JobStoreV2) ListJobs(pipelineId string, pageToken string, pageSize int, sortByFieldName string) ([]model.JobV2, string, error) {
	paginationContext, err := NewPaginationContext(pageToken, pageSize, sortByFieldName, model.GetJobV2TablePrimaryKeyColumn())
	if err != nil {
		return nil, "", err
	}
	queryJobTable := func(request *PaginationContext) ([]model.ListableDataModel, error) {
		return s.queryJobTable(pipelineId, request)
	}
	models, pageToken, err := listModel(paginationContext, queryJobTable)
	if err != nil {
		return nil, "", util.Wrap(err, "List jobs failed.")
	}
	return s.toJobMetadatas(models), pageToken, err
}

// GetJob Get the job manifest from Workflow CRD
func (s *JobStoreV2) GetJob(pipelineId string, jobId string) (*model.JobDetailV2, error) {
	var job model.JobDetailV2
	r := s.db.Raw(`SELECT * FROM job_detail_v2 WHERE PipelineId=? AND UUID=? LIMIT 1`, pipelineId, jobId).Scan(&job)
	if r.RecordNotFound() {
		return nil, util.NewResourceNotFoundError("Job", fmt.Sprint(jobId))
	}
	if r.Error != nil {
		return nil, util.NewInternalServerError(r.Error, "Failed to get job: %v", r.Error.Error())
	}
	return &job, nil
}

func (s *JobStoreV2) queryJobTable(pipelineId string, context *PaginationContext) ([]model.ListableDataModel, error) {
	var jobs []model.JobDetailV2
	var query bytes.Buffer
	query.WriteString(fmt.Sprintf("SELECT * FROM job_detail_v2 WHERE PipelineID = '%s'", pipelineId))
	toPaginationQuery("AND", &query, context)
	query.WriteString(fmt.Sprintf(" LIMIT %v", context.pageSize))
	if r := s.db.Raw(query.String()).Scan(&jobs); r.Error != nil {
		return nil, util.NewInternalServerError(r.Error, "Failed to list jobs: %v", r.Error.Error())
	}
	return s.toListableModels(jobs), nil
}

func (s *JobStoreV2) toListableModels(jobs []model.JobDetailV2) []model.ListableDataModel {
	models := make([]model.ListableDataModel, len(jobs))
	for i := range models {
		models[i] = jobs[i].JobV2
	}
	return models
}

func (s *JobStoreV2) toJobMetadatas(models []model.ListableDataModel) []model.JobV2 {
	jobMetadatas := make([]model.JobV2, len(models))
	for i := range models {
		jobMetadatas[i] = models[i].(model.JobV2)
	}
	return jobMetadatas
}

// factory function for job store
func NewJobStoreV2(db *gorm.DB, time util.TimeInterface) *JobStoreV2 {
	return &JobStoreV2{db: db, time: time}
}
