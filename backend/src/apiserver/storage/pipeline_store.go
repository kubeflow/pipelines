package storage

import (
	"bytes"
	"fmt"

	"database/sql"

	"github.com/googleprivate/ml/backend/src/apiserver/model"
	"github.com/googleprivate/ml/backend/src/common/util"
)

const (
	insertPipelineQuery = `INSERT INTO pipeline_details(
					UUID,DisplayName,Name,Namespace,Description,
          PackageId,Enabled,Conditions,MaxConcurrency,
          CronScheduleStartTimeInSec,CronScheduleEndTimeInSec,Schedule,
          PeriodicScheduleStartTimeInSec,PeriodicScheduleEndTimeInSec,IntervalSecond,
          Parameters,CreatedAtInSec,UpdatedAtInSec,ScheduledWorkflow) 
          VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`
)

type PipelineStoreInterface interface {
	ListPipelines(pageToken string, pageSize int, sortByFieldName string, isDesc bool) ([]model.Pipeline, string, error)
	GetPipeline(id string) (*model.Pipeline, error)
	CreatePipeline(*model.Pipeline) (*model.Pipeline, error)
	DeletePipeline(id string) error
	EnablePipeline(id string, enabled bool) error
	UpdatePipeline(swf *util.ScheduledWorkflow) error
}

type PipelineStore struct {
	db   *sql.DB
	time util.TimeInterface
}

func (s *PipelineStore) ListPipelines(pageToken string, pageSize int, sortByFieldName string, isDesc bool) ([]model.Pipeline, string, error) {
	context, err := NewPaginationContext(pageToken, pageSize, sortByFieldName, model.GetPipelineTablePrimaryKeyColumn(), isDesc)
	if err != nil {
		return nil, "", err
	}
	models, pageToken, err := listModel(context, s.queryPipelineTable)
	if err != nil {
		return nil, "", util.Wrap(err, "List pipelines failed.")
	}
	return s.toPipelineMetadatas(models), pageToken, err
}

func (s *PipelineStore) queryPipelineTable(context *PaginationContext) ([]model.ListableDataModel, error) {
	var query bytes.Buffer
	query.WriteString(fmt.Sprintf("SELECT * FROM pipeline_details "))
	toPaginationQuery("WHERE", &query, context)
	query.WriteString(fmt.Sprintf(" LIMIT %v", context.pageSize))

	rows, err := s.db.Query(query.String())
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to list pipelines: %v",
			err.Error())
	}
	defer rows.Close()
	pipelines, err := s.scanRows(rows)
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to list pipelines: %v",
			err.Error())
	}
	return s.toListableModels(pipelines), nil
}

func (s *PipelineStore) GetPipeline(id string) (*model.Pipeline, error) {
	row, err := s.db.Query(`SELECT * FROM pipeline_details WHERE uuid=? LIMIT 1`, id)
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to get pipeline: %v",
			err.Error())
	}
	defer row.Close()
	pipelines, err := s.scanRows(row)
	if err != nil || len(pipelines) > 1 {
		return nil, util.NewInternalServerError(err, "Failed to get pipeline: %v", err.Error())
	}
	if len(pipelines) == 0 {
		return nil, util.NewResourceNotFoundError("Pipeline", fmt.Sprint(id))
	}
	return &pipelines[0].Pipeline, nil
}

func (s *PipelineStore) scanRows(r *sql.Rows) ([]model.PipelineDetail, error) {
	var pipelines []model.PipelineDetail
	for r.Next() {
		var uuid, displayName, name, namespace, packageId, conditions,
			scheduledWorkflow, description, parameters string
		var cronScheduleStartTimeInSec, cronScheduleEndTimeInSec,
			periodicScheduleStartTimeInSec, periodicScheduleEndTimeInSec, intervalSecond sql.NullInt64
		var cron sql.NullString
		var enabled bool
		var createdAtInSec, updatedAtInSec, maxConcurrency int64
		err := r.Scan(
			&uuid, &displayName, &name, &namespace, &description,
			&packageId, &enabled, &conditions, &maxConcurrency,
			&cronScheduleStartTimeInSec, &cronScheduleEndTimeInSec, &cron,
			&periodicScheduleStartTimeInSec, &periodicScheduleEndTimeInSec, &intervalSecond,
			&parameters, &createdAtInSec, &updatedAtInSec, &scheduledWorkflow)

		if err != nil {
			return nil, err
		}
		pipelines = append(pipelines, model.PipelineDetail{Pipeline: model.Pipeline{
			UUID:           uuid,
			DisplayName:    displayName,
			Name:           name,
			Namespace:      namespace,
			Description:    description,
			PackageId:      packageId,
			Enabled:        enabled,
			Conditions:     conditions,
			MaxConcurrency: maxConcurrency,
			Trigger: model.Trigger{
				CronSchedule: model.CronSchedule{
					CronScheduleStartTimeInSec: NullInt64ToPointer(cronScheduleStartTimeInSec),
					CronScheduleEndTimeInSec:   NullInt64ToPointer(cronScheduleEndTimeInSec),
					Cron:                       NullStringToPointer(cron),
				},
				PeriodicSchedule: model.PeriodicSchedule{
					PeriodicScheduleStartTimeInSec: NullInt64ToPointer(periodicScheduleStartTimeInSec),
					PeriodicScheduleEndTimeInSec:   NullInt64ToPointer(periodicScheduleEndTimeInSec),
					IntervalSecond:                 NullInt64ToPointer(intervalSecond),
				},
			},
			Parameters:     parameters,
			CreatedAtInSec: createdAtInSec,
			UpdatedAtInSec: updatedAtInSec,
		}, ScheduledWorkflow: scheduledWorkflow})
	}
	return pipelines, nil
}

func (s *PipelineStore) DeletePipeline(id string) error {
	_, err := s.db.Exec(`DELETE FROM pipeline_details WHERE UUID=?`, id)
	if err != nil {
		return util.NewInternalServerError(err, "Failed to delete pipeline: %v", err.Error())
	}
	return nil
}

func (s *PipelineStore) CreatePipeline(p *model.Pipeline) (*model.Pipeline, error) {
	var newPipeline model.PipelineDetail
	newPipeline.Pipeline = *p
	now := s.time.Now().Unix()
	newPipeline.CreatedAtInSec = now
	newPipeline.UpdatedAtInSec = now
	stmt, err := s.db.Prepare(insertPipelineQuery)
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to add pipeline to pipeline table: %v",
			err.Error())
	}
	defer stmt.Close()
	if _, err := stmt.Exec(
		newPipeline.UUID, newPipeline.DisplayName, newPipeline.Name, newPipeline.Namespace, newPipeline.Description,
		newPipeline.PackageId, newPipeline.Enabled, newPipeline.Conditions, newPipeline.MaxConcurrency,
		PointerToNullInt64(newPipeline.CronScheduleStartTimeInSec), PointerToNullInt64(newPipeline.CronScheduleEndTimeInSec), PointerToNullString(newPipeline.Cron),
		PointerToNullInt64(newPipeline.PeriodicScheduleStartTimeInSec), PointerToNullInt64(newPipeline.PeriodicScheduleEndTimeInSec), PointerToNullInt64(newPipeline.IntervalSecond),
		newPipeline.Parameters, newPipeline.CreatedAtInSec, newPipeline.UpdatedAtInSec, newPipeline.ScheduledWorkflow); err != nil {
		return nil, util.NewInternalServerError(err, "Failed to add pipeline to pipeline table: %v",
			err.Error())
	}
	return &newPipeline.Pipeline, nil
}

func (s *PipelineStore) EnablePipeline(id string, enabled bool) error {
	now := s.time.Now().Unix()
	stmt, err := s.db.Prepare(`UPDATE pipeline_details SET Enabled=?, UpdatedAtInSec=? WHERE UUID=? and Enabled=?`)
	if err != nil {
		return util.NewInternalServerError(err, "Error when enabling pipeline %v to %v", id, enabled)
	}
	defer stmt.Close()
	if _, err := stmt.Exec(enabled, now, id, !enabled); err != nil {
		return util.NewInternalServerError(err, "Error when enabling pipeline %v to %v", id, enabled)
	}
	return nil
}

func (s *PipelineStore) UpdatePipeline(swf *util.ScheduledWorkflow) error {
	now := s.time.Now().Unix()

	if swf.Name == "" {
		return util.NewInvalidInputError("The resource must have a name: %+v", swf.ScheduledWorkflow)
	}
	if swf.Namespace == "" {
		return util.NewInvalidInputError("The resource must have a namespace: %+v", swf.ScheduledWorkflow)
	}

	if swf.UID == "" {
		return util.NewInvalidInputError("The resource must have a UID: %+v", swf.UID)
	}

	parameters, err := swf.ParametersAsString()
	if err != nil {
		return err
	}
	stmt, err := s.db.Prepare(`UPDATE pipeline_details SET 
		Name = ?,
		Namespace = ?,
		Enabled = ?,
		Conditions = ?,
		MaxConcurrency = ?,
		Parameters = ?,
		UpdatedAtInSec = ?,
		CronScheduleStartTimeInSec = ?,
		CronScheduleEndTimeInSec = ?,
		Schedule = ?,
		PeriodicScheduleStartTimeInSec = ?,
		PeriodicScheduleEndTimeInSec = ?,
		IntervalSecond = ? 
		WHERE UUID = ?`)
	if err != nil {
		return util.NewInternalServerError(err,
			"Error while updating pipeline with scheduled workflow: %v: %+v",
			err, swf.ScheduledWorkflow)
	}
	defer stmt.Close()
	r, err := stmt.Exec(
		swf.Name,
		swf.Namespace,
		swf.Spec.Enabled,
		swf.ConditionSummary(),
		swf.MaxConcurrencyOr0(),
		parameters,
		now,
		PointerToNullInt64(swf.CronScheduleStartTimeInSecOrNull()),
		PointerToNullInt64(swf.CronScheduleEndTimeInSecOrNull()),
		swf.CronOrEmpty(),
		PointerToNullInt64(swf.PeriodicScheduleStartTimeInSecOrNull()),
		PointerToNullInt64(swf.PeriodicScheduleEndTimeInSecOrNull()),
		swf.IntervalSecondOr0(),
		string(swf.UID))

	if err != nil {
		return util.NewInternalServerError(err,
			"Error while updating pipeline with scheduled workflow: %v: %+v",
			err, swf.ScheduledWorkflow)
	}

	rowsAffected, err := r.RowsAffected()
	if err != nil {
		return util.NewInternalServerError(err,
			"Error getting affected rows while updating pipeline with scheduled workflow: %v: %+v",
			err, swf.ScheduledWorkflow)
	}
	if rowsAffected <= 0 {
		return util.NewInvalidInputError(
			"There is no pipeline corresponding to this scheduled workflow: %v/%v/%v",
			swf.UID, swf.Namespace, swf.Name)
	}

	return nil
}

func (s *PipelineStore) toListableModels(pipelines []model.PipelineDetail) []model.ListableDataModel {
	models := make([]model.ListableDataModel, len(pipelines))
	for i := range models {
		models[i] = pipelines[i].Pipeline
	}
	return models
}

func (s *PipelineStore) toPipelineMetadatas(models []model.ListableDataModel) []model.Pipeline {
	pipelines := make([]model.Pipeline, len(models))
	for i := range models {
		pipelines[i] = models[i].(model.Pipeline)
	}
	return pipelines
}

// factory function for pipeline store
func NewPipelineStore(db *sql.DB, time util.TimeInterface) *PipelineStore {
	return &PipelineStore{
		db:   db,
		time: time,
	}
}
