// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package history transfers completed native run history without executing it.
package history

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"unicode/utf8"

	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const Format = "kfp-native-run-history/v1"
const MaxRuns = 100

var sourcePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]{0,62}$`)

// Bundle is an administrative archive. It contains metadata, not artifact bytes.
// Recurring schedules are deliberately not imported; their original IDs remain
// in the archive's Run records for provenance.
type Bundle struct {
	Format       string
	Source       string
	Schema       string
	Experiments  []model.Experiment
	Pipelines    []model.Pipeline
	Versions     []model.PipelineVersion
	PipelineTags []model.PipelineTag
	VersionTags  []model.PipelineVersionTag
	Entries      []Entry
}

type Entry struct {
	Run       model.Run
	Tasks     []model.Task
	Artifacts []model.Artifact
	Links     []model.ArtifactTask
	Metrics   []model.RunMetricV1
}

// ImportOptions never grants permission to overwrite existing destination data.
type ImportOptions struct {
	// NamePrefix avoids namespace/name collisions for imported experiments and
	// pipelines. UUIDs and pipeline-version names remain unchanged.
	NamePrefix string
	// ExperimentID explicitly places every run into an existing experiment in
	// the same namespace. Without it, the source experiments are preserved.
	ExperimentID string
	DryRun       bool
}

type Result struct {
	Imported int
	Skipped  int
}

func models() []any {
	return []any{&model.Experiment{}, &model.Pipeline{}, &model.PipelineVersion{},
		&model.PipelineTag{}, &model.PipelineVersionTag{}, &model.Run{},
		&model.Task{}, &model.Artifact{}, &model.ArtifactTask{}, &model.RunMetricV1{}}
}

// schemaSignature rejects schema drift instead of silently discarding columns
// unknown to this binary. The importer does not run migrations or create tables.
func schemaSignature(db *gorm.DB) (string, error) {
	shape := []string{db.Name()}
	for _, value := range models() {
		stmt := &gorm.Statement{DB: db}
		if err := stmt.Parse(value); err != nil {
			return "", err
		}
		columns, err := db.Migrator().ColumnTypes(value)
		if err != nil {
			return "", fmt.Errorf("inspect %s: %w", stmt.Table, err)
		}
		actual := make([]string, 0, len(columns))
		for _, column := range columns {
			actual = append(actual, column.Name())
		}
		expected := append([]string(nil), stmt.Schema.DBNames...)
		sort.Strings(actual)
		sort.Strings(expected)
		if !reflect.DeepEqual(actual, expected) {
			return "", fmt.Errorf("schema mismatch for %s: use a history binary matching the database and upgrade the API server before importing", stmt.Table)
		}
		metadata := map[string]string{}
		for _, column := range columns {
			length, lengthOK := column.Length()
			precision, scale, precisionOK := column.DecimalSize()
			nullable, nullableOK := column.Nullable()
			metadata[column.Name()] = fmt.Sprintf("%s:%d:%t:%d:%d:%t:%t:%t", column.DatabaseTypeName(), length, lengthOK, precision, scale, precisionOK, nullable, nullableOK)
		}
		for _, name := range expected {
			shape = append(shape, fmt.Sprintf("%s.%s:%s", stmt.Table, name, metadata[name]))
		}
	}
	return digest(shape)
}

func digest(value any) (string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func find[T any](db *gorm.DB, column string, ids []string, out *[]T) error {
	if len(ids) == 0 {
		return nil
	}
	values := make([]any, len(ids))
	for i, id := range ids {
		values[i] = id
	}
	return readRows(db.Where(clause.IN{Column: clause.Column{Name: column}, Values: values}), out)
}

func createRecord(db *gorm.DB, row any) error {
	values, _, err := persistedValues(db, row)
	if err != nil {
		return err
	}
	return db.Model(row).Omit(clause.Associations).Create(values).Error
}

// readRows restores JSON columns using UseNumber: model JSON scanners serve
// API callers with float64 values, which would round large integers in an
// administrative transfer. Fetch JSON and primary keys in one additional query
// per collection, never one query per task/artifact.
func readRows[T any](query *gorm.DB, out *[]T) error {
	if err := query.Find(out).Error; err != nil {
		return err
	}
	if len(*out) == 0 {
		return nil
	}
	stmt := &gorm.Statement{DB: query}
	if err := stmt.Parse(new(T)); err != nil {
		return err
	}
	var jsonNames, primaryNames []string
	var columns []clause.Column
	for _, name := range stmt.Schema.DBNames {
		field := stmt.Schema.FieldsByDBName[name]
		if field.FieldType == reflect.TypeOf(model.JSONData{}) || field.FieldType == reflect.TypeOf(model.JSONSlice{}) {
			jsonNames = append(jsonNames, name)
			columns = append(columns, clause.Column{Name: name})
		}
	}
	if len(jsonNames) == 0 {
		return nil
	}
	for _, field := range stmt.Schema.PrimaryFields {
		primaryNames = append(primaryNames, field.DBName)
		columns = append(columns, clause.Column{Name: field.DBName})
	}
	var rawRows []map[string]any
	rows, err := query.Model(new(T)).Clauses(clause.Select{Columns: columns}).Rows()
	if err != nil {
		return err
	}
	names, err := rows.Columns()
	if err != nil {
		rows.Close()
		return err
	}
	for rows.Next() {
		values := make([]any, len(names))
		targets := make([]any, len(names))
		for i := range values {
			targets[i] = &values[i]
		}
		if err := rows.Scan(targets...); err != nil {
			rows.Close()
			return err
		}
		raw := map[string]any{}
		for i, name := range names {
			raw[name] = values[i]
		}
		rawRows = append(rawRows, raw)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	key := func(row map[string]any) string {
		values := make([]string, len(primaryNames))
		for i, name := range primaryNames {
			if b, ok := row[name].([]byte); ok {
				values[i] = string(b)
			} else {
				values[i] = fmt.Sprint(row[name])
			}
		}
		encoded, _ := json.Marshal(values)
		return string(encoded)
	}
	indexed := map[string]map[string]any{}
	for _, row := range rawRows {
		indexed[key(row)] = row
	}
	for i := range *out {
		value := &(*out)[i]
		persisted, _, err := persistedValues(query, value)
		if err != nil {
			return err
		}
		row, ok := indexed[key(persisted)]
		if !ok {
			return errors.New("JSON snapshot row missing")
		}
		rv := reflect.Indirect(reflect.ValueOf(value))
		for _, name := range jsonNames {
			raw := row[name]
			if raw == nil {
				continue
			}
			var data []byte
			switch v := raw.(type) {
			case string:
				data = []byte(v)
			case []byte:
				data = v
			default:
				return fmt.Errorf("unsupported JSON database value for %s", name)
			}
			decoder := json.NewDecoder(bytes.NewReader(data))
			decoder.UseNumber()
			field := stmt.Schema.FieldsByDBName[name]
			if field.FieldType == reflect.TypeOf(model.JSONData{}) {
				var parsed model.JSONData
				if err := decoder.Decode(&parsed); err != nil {
					return err
				}
				if err := field.Set(context.Background(), rv, parsed); err != nil {
					return err
				}
			} else {
				var parsed model.JSONSlice
				if err := decoder.Decode(&parsed); err != nil {
					return err
				}
				if err := field.Set(context.Background(), rv, parsed); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func unique(ids []string) []string {
	found := map[string]bool{}
	var result []string
	for _, id := range ids {
		if id != "" && !found[id] {
			found[id] = true
			result = append(result, id)
		}
	}
	sort.Strings(result)
	return result
}

func terminal(run model.Run) bool {
	state := run.State
	if state == "" || state == model.RuntimeStateUnspecified {
		state = model.RuntimeState(run.Conditions)
	}
	switch state.ToV2() {
	case model.RuntimeStateSucceeded, model.RuntimeStateFailed, model.RuntimeStateCanceled, model.RuntimeStateSkipped:
		return run.FinishedAtInSec > 0
	default:
		return false
	}
}

// Export reads a consistent snapshot of explicitly selected, completed runs.
// Read-only repeatable-read keeps task/artifact relationships in the same snapshot.
func Export(ctx context.Context, db *gorm.DB, source string, runIDs []string) (*Bundle, error) {
	if !sourcePattern.MatchString(source) {
		return nil, errors.New("source must be 1-63 letters, digits, dots, underscores or hyphens, starting with a letter or digit")
	}
	runIDs = unique(runIDs)
	if len(runIDs) == 0 || len(runIDs) > MaxRuns {
		return nil, fmt.Errorf("select between 1 and %d run IDs per export", MaxRuns)
	}
	schema, err := schemaSignature(db.WithContext(ctx))
	if err != nil {
		return nil, err
	}
	bundle := &Bundle{Format: Format, Source: source, Schema: schema}
	err = db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var experimentIDs, pipelineIDs, versionIDs []string
		for _, id := range runIDs {
			var entry Entry
			if err := tx.Where(clause.Eq{Column: "UUID", Value: id}).Take(&entry.Run).Error; err != nil {
				return fmt.Errorf("run %s: %w", id, err)
			}
			if !terminal(entry.Run) {
				return fmt.Errorf("run %s is not completed", id)
			}
			if entry.Run.ImportedFrom != "" {
				return fmt.Errorf("run %s is already imported; export from its original installation", id)
			}
			if entry.Run.PipelineRunContextId != 0 || entry.Run.PipelineContextId != 0 {
				return fmt.Errorf("run %s contains legacy MLMD context IDs; use the matching legacy migration tool", id)
			}
			if err := find(tx, "RunUUID", []string{id}, &entry.Tasks); err != nil {
				return err
			}
			if err := find(tx, "RunUUID", []string{id}, &entry.Links); err != nil {
				return err
			}
			if err := find(tx, "RunUUID", []string{id}, &entry.Metrics); err != nil {
				return err
			}
			var artifactIDs []string
			for _, link := range entry.Links {
				artifactIDs = append(artifactIDs, link.ArtifactID)
			}
			if err := find(tx, "UUID", unique(artifactIDs), &entry.Artifacts); err != nil {
				return err
			}
			sortEntry(&entry)
			bundle.Entries = append(bundle.Entries, entry)
			experimentIDs = append(experimentIDs, entry.Run.ExperimentId)
			pipelineIDs = append(pipelineIDs, entry.Run.PipelineId)
			versionIDs = append(versionIDs, entry.Run.PipelineVersionId)
		}
		if err := find(tx, "UUID", unique(experimentIDs), &bundle.Experiments); err != nil {
			return err
		}
		// Resolve version ownership even when the run stores only a version ID.
		if err := find(tx, "UUID", unique(versionIDs), &bundle.Versions); err != nil {
			return err
		}
		for _, version := range bundle.Versions {
			pipelineIDs = append(pipelineIDs, version.PipelineId)
		}
		if err := find(tx, "UUID", unique(pipelineIDs), &bundle.Pipelines); err != nil {
			return err
		}
		for _, pipeline := range bundle.Pipelines {
			versionIDs = append(versionIDs, pipeline.DefaultVersionId)
		}
		bundle.Versions = nil
		if err := find(tx, "UUID", unique(versionIDs), &bundle.Versions); err != nil {
			return err
		}
		if err := find(tx, "PipelineId", unique(pipelineIDs), &bundle.PipelineTags); err != nil {
			return err
		}
		if err := find(tx, "PipelineVersionId", unique(versionIDs), &bundle.VersionTags); err != nil {
			return err
		}
		return validate(bundle)
	}, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	return bundle, err
}

func sortEntry(e *Entry) {
	sort.Slice(e.Tasks, func(i, j int) bool { return e.Tasks[i].UUID < e.Tasks[j].UUID })
	sort.Slice(e.Artifacts, func(i, j int) bool { return e.Artifacts[i].UUID < e.Artifacts[j].UUID })
	sort.Slice(e.Links, func(i, j int) bool { return e.Links[i].UUID < e.Links[j].UUID })
	sort.Slice(e.Metrics, func(i, j int) bool {
		a, b := e.Metrics[i], e.Metrics[j]
		if a.NodeID != b.NodeID {
			return a.NodeID < b.NodeID
		}
		return a.Name < b.Name
	})
}

// persistedValues ignores GORM associations and derived fields when comparing
// dependencies. Import always disables association writes as well.
func persistedValues(db *gorm.DB, value any) (map[string]any, *gorm.Statement, error) {
	stmt := &gorm.Statement{DB: db}
	if err := stmt.Parse(value); err != nil {
		return nil, nil, err
	}
	rv := reflect.Indirect(reflect.ValueOf(value))
	values := map[string]any{}
	for _, name := range stmt.Schema.DBNames {
		field := stmt.Schema.FieldsByDBName[name]
		v, _ := field.ValueOf(context.Background(), rv)
		switch data := v.(type) {
		case model.JSONData:
			if data == nil {
				v = nil
			} else {
				raw, err := json.Marshal(data)
				if err != nil {
					return nil, nil, err
				}
				v = string(raw)
			}
		case model.JSONSlice:
			if data == nil {
				v = nil
			} else {
				raw, err := json.Marshal(data)
				if err != nil {
					return nil, nil, err
				}
				v = string(raw)
			}
		}
		values[name] = v
	}
	return values, stmt, nil
}

func insertOrCheck[T any](db *gorm.DB, row *T) error {
	wanted, stmt, err := persistedValues(db, row)
	if err != nil {
		return err
	}
	query := db
	for _, field := range stmt.Schema.PrimaryFields {
		query = query.Where(clause.Eq{Column: clause.Column{Name: field.DBName}, Value: wanted[field.DBName]})
	}
	var existingRows []T
	err = readRows(query.Limit(1), &existingRows)
	if err != nil {
		return err
	}
	if len(existingRows) == 0 {
		if err := createRecord(db, row); err != nil {
			return fmt.Errorf("insert %s: identity/name conflict or invalid dependency (destination unchanged): %w", stmt.Table, err)
		}
		return nil
	}
	if err != nil {
		return err
	}
	actual, _, err := persistedValues(db, &existingRows[0])
	if err != nil {
		return err
	}
	// Shared definitions can gain new runs, tags, or descriptions after the
	// first export. Never overwrite those destination-local changes.
	switch any(row).(type) {
	case *model.Experiment:
		return requireSame(stmt.Table, wanted, actual, "UUID", "Name", "Namespace")
	case *model.Pipeline:
		return requireSame(stmt.Table, wanted, actual, "UUID", "Name", "Namespace")
	case *model.PipelineVersion:
		return requireSame(stmt.Table, wanted, actual, "UUID", "Name", "PipelineId", "PipelineSpec", "PipelineSpecURI")
	default:
		if !reflect.DeepEqual(wanted, actual) {
			return fmt.Errorf("conflicting existing %s record; refusing to overwrite", stmt.Table)
		}
		return nil
	}
}

// insertHistoryArtifact never joins imported history to a destination-local artifact.
// Shared artifacts may already exist after an earlier run from the same source
// was imported, but matching contents alone do not establish that provenance.
func insertHistoryArtifact(db *gorm.DB, artifact *model.Artifact, source string) error {
	var existing int64
	if err := db.Model(&model.Artifact{}).Where(clause.Eq{Column: "UUID", Value: artifact.UUID}).Count(&existing).Error; err != nil {
		return err
	}
	if existing == 0 {
		return createRecord(db, artifact)
	}
	var owners []string
	if err := db.Model(&model.ArtifactTask{}).
		Where(clause.Eq{Column: "ArtifactID", Value: artifact.UUID}).
		Distinct("RunUUID").Pluck("RunUUID", &owners).Error; err != nil {
		return err
	}
	conflict := fmt.Errorf("artifact %s conflicts with destination lineage; existing artifacts must belong only to history imported from source %s", artifact.UUID, source)
	if len(owners) == 0 {
		return conflict
	}
	var runs []model.Run
	if err := find(db, "UUID", owners, &runs); err != nil {
		return err
	}
	if len(runs) != len(owners) {
		return conflict
	}
	for _, run := range runs {
		if run.ImportedFrom != source {
			return conflict
		}
	}
	return insertOrCheck(db, artifact)
}

func requireSame(table string, wanted, actual map[string]any, names ...string) error {
	for _, name := range names {
		if !reflect.DeepEqual(wanted[name], actual[name]) {
			return fmt.Errorf("conflicting existing %s record (%s); refusing to overwrite", table, name)
		}
	}
	return nil
}

func insertAll[T any](db *gorm.DB, values []T) error {
	for i := range values {
		if err := insertOrCheck(db, &values[i]); err != nil {
			return err
		}
	}
	return nil
}

var errDryRunRollback = errors.New("history dry-run rollback")

// Import atomically merges a bundle. All conflicts roll back the entire bundle.
// Its database credentials must be restricted to the intended destination.
func Import(ctx context.Context, db *gorm.DB, bundle *Bundle, opts ImportOptions) (Result, error) {
	var result Result
	if err := validate(bundle); err != nil {
		return result, err
	}
	schema, err := schemaSignature(db.WithContext(ctx))
	if err != nil {
		return result, err
	}
	if bundle.Schema != schema {
		return result, errors.New("archive schema does not match this history binary and destination")
	}
	err = db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var target model.Experiment
		if opts.ExperimentID != "" {
			if err := tx.Where(clause.Eq{Column: "UUID", Value: opts.ExperimentID}).Take(&target).Error; err != nil {
				return fmt.Errorf("target experiment: %w", err)
			}
			for _, entry := range bundle.Entries {
				if entry.Run.Namespace != target.Namespace {
					return errors.New("target experiment namespace must match every imported run")
				}
			}
		} else {
			for _, experiment := range bundle.Experiments {
				experiment.Name = opts.NamePrefix + experiment.Name
				if utf8.RuneCountInString(experiment.Name) > 128 {
					return errors.New("prefixed experiment name exceeds 128 characters")
				}
				if err := insertOrCheck(tx, &experiment); err != nil {
					return err
				}
			}
		}
		knownPipelines, knownVersions := map[string]bool{}, map[string]bool{}
		for _, pipeline := range bundle.Pipelines {
			knownPipelines[pipeline.UUID] = true
		}
		for _, version := range bundle.Versions {
			knownVersions[version.UUID] = true
		}
		for _, pipeline := range bundle.Pipelines {
			if !knownVersions[pipeline.DefaultVersionId] {
				pipeline.DefaultVersionId = ""
			}
			pipeline.Name = opts.NamePrefix + pipeline.Name
			if utf8.RuneCountInString(pipeline.Name) > 128 {
				return errors.New("prefixed pipeline name exceeds 128 characters")
			}
			if err := insertOrCheck(tx, &pipeline); err != nil {
				return err
			}
		}
		if err := insertAll(tx, bundle.Versions); err != nil {
			return err
		}
		if err := insertAll(tx, bundle.PipelineTags); err != nil {
			return err
		}
		if err := insertAll(tx, bundle.VersionTags); err != nil {
			return err
		}
		for _, entry := range bundle.Entries {
			sortEntry(&entry)
			checksum, err := digest(entry)
			if err != nil {
				return err
			}
			run := entry.Run
			if !knownPipelines[run.PipelineId] {
				run.PipelineId = ""
			}
			if !knownVersions[run.PipelineVersionId] {
				run.PipelineVersionId = ""
			}
			if opts.ExperimentID != "" {
				run.ExperimentId = opts.ExperimentID
			}
			var existing model.Run
			err = tx.Where(clause.Eq{Column: "UUID", Value: run.UUID}).Take(&existing).Error
			if err == nil {
				if existing.ImportedFrom != bundle.Source || existing.ImportDigest != checksum || existing.ExperimentId != run.ExperimentId {
					return fmt.Errorf("run %s conflicts with destination history; refusing to overwrite", run.UUID)
				}
				result.Skipped++
				continue
			}
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			run.ImportedFrom = bundle.Source
			run.ImportDigest = checksum
			// A schedule belongs to the source cluster. Keep its ID in the
			// archive, but never link history to a runnable destination job.
			run.RecurringRunId = ""
			if err := createRecord(tx, &run); err != nil {
				return err
			}
			// Insert parents before children without weakening foreign keys.
			pending := append([]model.Task(nil), entry.Tasks...)
			inserted := map[string]bool{}
			for len(pending) != 0 {
				remaining := make([]model.Task, 0, len(pending))
				for _, task := range pending {
					if task.ParentTaskUUID != nil && !inserted[*task.ParentTaskUUID] {
						remaining = append(remaining, task)
						continue
					}
					if err := insertOrCheck(tx, &task); err != nil {
						return err
					}
					inserted[task.UUID] = true
				}
				if len(remaining) == len(pending) {
					return fmt.Errorf("run %s has a cyclic or missing task parent", run.UUID)
				}
				pending = remaining
			}
			for _, artifact := range entry.Artifacts {
				// History must not register a reusable external-artifact identity
				// or collide with an independently registered destination object.
				artifact.IdentityKey = nil
				if err := insertHistoryArtifact(tx, &artifact, bundle.Source); err != nil {
					return err
				}
			}
			if err := insertAll(tx, entry.Links); err != nil {
				return err
			}
			if err := insertAll(tx, entry.Metrics); err != nil {
				return err
			}
			result.Imported++
		}
		if opts.DryRun {
			return errDryRunRollback
		}
		return nil
	})
	if errors.Is(err, errDryRunRollback) {
		return result, nil
	}
	if err != nil {
		return Result{}, err
	}
	return result, nil
}

func validate(b *Bundle) error {
	if b == nil || b.Format != Format || !sourcePattern.MatchString(b.Source) {
		return errors.New("unsupported history format or source installation ID")
	}
	if len(b.Entries) == 0 || len(b.Entries) > MaxRuns {
		return fmt.Errorf("archive must contain 1-%d runs", MaxRuns)
	}
	experiments := map[string]model.Experiment{}
	for _, e := range b.Experiments {
		if e.UUID == "" {
			return errors.New("empty experiment ID")
		}
		if _, ok := experiments[e.UUID]; ok {
			return errors.New("duplicate experiment ID")
		}
		experiments[e.UUID] = e
	}
	pipelines := map[string]model.Pipeline{}
	for _, p := range b.Pipelines {
		if p.UUID == "" {
			return errors.New("empty pipeline ID")
		}
		if _, ok := pipelines[p.UUID]; ok {
			return errors.New("duplicate pipeline ID")
		}
		pipelines[p.UUID] = p
	}
	versions := map[string]model.PipelineVersion{}
	for _, v := range b.Versions {
		if v.UUID == "" {
			return errors.New("empty pipeline version ID")
		}
		if _, ok := versions[v.UUID]; ok {
			return errors.New("duplicate pipeline version ID")
		}
		if _, ok := pipelines[v.PipelineId]; !ok {
			return errors.New("pipeline version has missing parent pipeline")
		}
		versions[v.UUID] = v
	}
	for _, p := range pipelines {
		if p.DefaultVersionId != "" {
			v, ok := versions[p.DefaultVersionId]
			if ok && v.PipelineId != p.UUID {
				return errors.New("inconsistent default pipeline version")
			}
		}
	}
	for _, tag := range b.PipelineTags {
		if _, ok := pipelines[tag.PipelineID]; !ok {
			return errors.New("tag has missing pipeline")
		}
	}
	for _, tag := range b.VersionTags {
		if _, ok := versions[tag.PipelineVersionID]; !ok {
			return errors.New("tag has missing pipeline version")
		}
	}
	seenRuns, seenTasks := map[string]bool{}, map[string]bool{}
	for _, e := range b.Entries {
		r := e.Run
		if r.UUID == "" || seenRuns[r.UUID] || !terminal(r) || r.ImportedFrom != "" || r.ImportDigest != "" {
			return errors.New("archive has duplicate, incomplete or already imported run")
		}
		if r.PipelineContextId != 0 || r.PipelineRunContextId != 0 {
			return errors.New("legacy MLMD history requires its matching migration tool")
		}
		seenRuns[r.UUID] = true
		experiment, ok := experiments[r.ExperimentId]
		if !ok || experiment.Namespace != r.Namespace {
			return fmt.Errorf("run %s has missing experiment or inconsistent namespace", r.UUID)
		}
		if r.PipelineId != "" {
			if _, ok := pipelines[r.PipelineId]; !ok && r.PipelineSpecManifest == "" {
				return fmt.Errorf("run %s has neither a catalog pipeline nor an embedded specification", r.UUID)
			}
		}
		if r.PipelineVersionId != "" {
			v, ok := versions[r.PipelineVersionId]
			if (!ok && r.PipelineSpecManifest == "") || (ok && r.PipelineId != "" && v.PipelineId != r.PipelineId) {
				return errors.New("missing or inconsistent run pipeline version")
			}
		}
		tasks := map[string]model.Task{}
		for _, t := range e.Tasks {
			if t.UUID == "" || seenTasks[t.UUID] || t.RunUUID != r.UUID || t.Namespace != r.Namespace {
				return errors.New("duplicate task or inconsistent task ownership")
			}
			seenTasks[t.UUID] = true
			tasks[t.UUID] = t
		}
		for _, t := range tasks {
			if t.ParentTaskUUID != nil {
				if _, ok := tasks[*t.ParentTaskUUID]; !ok {
					return errors.New("missing task parent")
				}
			}
		}
		artifacts := map[string]bool{}
		for _, a := range e.Artifacts {
			if a.UUID == "" || artifacts[a.UUID] || a.Namespace != r.Namespace {
				return errors.New("duplicate artifact or inconsistent artifact namespace")
			}
			artifacts[a.UUID] = true
		}
		links := map[string]bool{}
		for _, link := range e.Links {
			_, taskExists := tasks[link.TaskID]
			if link.UUID == "" || links[link.UUID] || link.RunUUID != r.UUID || !taskExists || !artifacts[link.ArtifactID] {
				return errors.New("invalid artifact/task relationship")
			}
			links[link.UUID] = true
		}
		for _, metric := range e.Metrics {
			if metric.RunUUID != r.UUID {
				return errors.New("metric belongs to another run")
			}
		}
	}
	return nil
}
