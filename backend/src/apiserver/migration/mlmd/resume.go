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

package mlmd

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"gorm.io/gorm"
)

// A record can map to several rows, as with a split metrics artifact.
type migrationRow struct {
	values []any
}

type migrationStream struct {
	table   string
	columns []string
	json    map[string]bool
	records [][]migrationRow
}

func (stream migrationStream) resume(ctx context.Context, db *gorm.DB) error {
	table := "`" + stream.table + "`"
	var predicates, placeholders []string
	for _, name := range stream.columns {
		column := "`" + name + "`"
		switch {
		case stream.json[name]:
			predicates = append(predicates, column+" <=> CAST(? AS JSON)")
			placeholders = append(placeholders, "CAST(? AS JSON)")
		case name == "Fingerprint":
			// Cache backfill owns this field and preserves pre-existing native keys.
			predicates = append(predicates, "(? IS NULL OR "+column+" IS NOT NULL)")
			placeholders = append(placeholders, "?")
		default:
			predicates = append(predicates, "BINARY "+column+" <=> BINARY ?")
			placeholders = append(placeholders, "?")
		}
	}
	checkSQL := "SELECT " + strings.Join(predicates, " AND ") + " FROM " + table + " WHERE UUID=?"
	insertSQL := "INSERT IGNORE INTO " + table + " (" + strings.Join(stream.columns, ",") + ") VALUES (" + strings.Join(placeholders, ",") + ")"
	check := func(tx *gorm.DB, record []migrationRow) (bool, error) {
		complete := true
		for _, row := range record {
			var matches bool
			args := append(append([]any(nil), row.values...), row.values[0])
			err := tx.Raw(checkSQL, args...).Row().Scan(&matches)
			switch {
			case errors.Is(err, sql.ErrNoRows):
				complete = false
			case err != nil:
				return false, err
			case !matches:
				return false, fmt.Errorf("%s UUID %s has conflicting migrated fields", stream.table, row.values[0])
			}
		}
		return complete, nil
	}
	db = db.WithContext(ctx)
	var probeErr error
	start := sort.Search(len(stream.records), func(index int) bool {
		if probeErr != nil {
			return true
		}
		var complete bool
		complete, probeErr = check(db, stream.records[index])
		return !complete
	})
	if probeErr != nil {
		return probeErr
	}
	// Check the presumed prefix so a restored or manually changed target
	// cannot hide an earlier hole or a conflicting row from a binary probe.
	for index, record := range stream.records[:start] {
		complete, err := check(db, record)
		if err != nil {
			return err
		}
		if !complete {
			start = index
			break
		}
	}
	for start < len(stream.records) {
		end := min(start+100, len(stream.records))
		err := db.Transaction(func(tx *gorm.DB) error {
			for _, record := range stream.records[start:end] {
				complete, err := check(tx, record)
				if err != nil {
					return err
				}
				if complete {
					continue
				}
				for _, row := range record {
					if err := tx.Exec(insertSQL, row.values...).Error; err != nil {
						return err
					}
					if err := rejectInsertWarnings(tx); err != nil {
						return err
					}
				}
				complete, err = check(tx, record)
				if err != nil {
					return err
				}
				if !complete {
					return fmt.Errorf("%s is missing rows after inserting source record %s", stream.table, record[0].values[0])
				}
			}
			return nil
		})
		if err != nil {
			return err
		}
		start = end
	}
	return nil
}

func encodeMigrationJSON(values ...any) ([]string, error) {
	encoded := make([]string, len(values))
	for i, value := range values {
		data, err := json.Marshal(value)
		if err != nil {
			return nil, err
		}
		encoded[i] = string(data)
	}
	return encoded, nil
}

func insertTasks(ctx context.Context, db *gorm.DB, table string, tasks []model.Task) error {
	stream := migrationStream{table: table,
		columns: strings.Split("UUID,Namespace,RunUUID,pods,CreatedAtInSec,StartedInSec,FinishedInSec,Fingerprint,CachedTaskUUID,Name,DisplayName,State,StateHistory,InputParameters,OutputParameters,Type,TypeAttrs,StatusMetadata", ","),
		json:    map[string]bool{"pods": true, "StateHistory": true, "InputParameters": true, "OutputParameters": true, "TypeAttrs": true, "StatusMetadata": true},
	}
	for _, task := range tasks {
		pods, attrs := task.Pods, task.TypeAttrs
		if pods == nil {
			pods = model.JSONSlice{}
		}
		if attrs == nil {
			attrs = model.JSONData{}
		}
		values, err := encodeMigrationJSON(pods, task.StateHistory, task.InputParameters, task.OutputParameters, attrs, task.StatusMetadata)
		if err != nil {
			return err
		}
		stream.records = append(stream.records, []migrationRow{{[]any{
			task.UUID, task.Namespace, task.RunUUID, values[0], task.CreatedAtInSec, task.StartedInSec,
			task.FinishedInSec, "", task.CachedTaskUUID, task.Name, task.DisplayName, int32(task.State),
			values[1], values[2], values[3], int32(task.Type), values[4], values[5],
		}}})
	}
	return stream.resume(ctx, db)
}

func insertArtifacts(ctx context.Context, db *gorm.DB, artifacts []model.Artifact, sizes []int) error {
	stream := migrationStream{table: "artifacts",
		columns: strings.Split("UUID,Namespace,Type,URI,URIHash,Name,Description,CreatedAtInSec,LastUpdateInSec,Metadata,NumberValue", ","),
		json:    map[string]bool{"Metadata": true},
	}
	for _, size := range sizes {
		record := make([]migrationRow, 0, size)
		for _, artifact := range artifacts[:size] {
			metadata, err := encodeMigrationJSON(artifact.Metadata)
			if err != nil {
				return err
			}
			record = append(record, migrationRow{[]any{artifact.UUID, artifact.Namespace, int32(artifact.Type), artifact.URI,
				artifact.URIHash, artifact.Name, artifact.Description, artifact.CreatedAtInSec, artifact.LastUpdateInSec, metadata[0], artifact.NumberValue}})
		}
		stream.records = append(stream.records, record)
		artifacts = artifacts[size:]
	}
	return stream.resume(ctx, db)
}

func insertLinks(ctx context.Context, db *gorm.DB, links []model.ArtifactTask) error {
	stream := migrationStream{table: "artifact_tasks",
		columns: strings.Split("UUID,ArtifactID,TaskID,Type,Iteration,RunUUID,Producer,ArtifactKey", ","),
		json:    map[string]bool{"Producer": true},
	}
	for _, link := range links {
		var producer any
		if link.Producer != nil {
			encoded, err := encodeMigrationJSON(link.Producer)
			if err != nil {
				return err
			}
			producer = encoded[0]
		}
		stream.records = append(stream.records, []migrationRow{{[]any{
			link.UUID, link.ArtifactID, link.TaskID, int32(link.Type), link.Iteration, link.RunUUID, producer, link.ArtifactKey,
		}}})
	}
	return stream.resume(ctx, db)
}
