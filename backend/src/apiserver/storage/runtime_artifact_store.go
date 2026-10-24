// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package storage

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	sq "github.com/Masterminds/squirrel"
	apiv2 "github.com/kubeflow/pipelines/backend/api/v2/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common/sql/dialect"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
)

// checkpointArtifacts writes output metadata and relationships while the public
// task and runtime row are locked. Identity depends on the attempt and logical
// output, never on a fresh random UUID or publication delivery count.
func (s *runtimeStore) checkpointArtifacts(tx sqlTransaction, runtime *model.RuntimeTask, task *model.Task, outputs []model.TaskArtifactHydrated) error {
	seen := map[string]bool{}
	for _, output := range outputs {
		if output.Value == nil || output.Key == "" {
			return errors.New("runtime output artifact requires metadata and an output key")
		}
		if output.Type != 0 && output.Type != apiv2.IOType_OUTPUT {
			return errors.New("runtime output artifact relationship must have OUTPUT type")
		}
		if output.Value.Namespace != "" && output.Value.Namespace != task.Namespace {
			return errors.New("runtime output artifact cannot change the task namespace")
		}
		var producer model.JSONData
		if output.Producer != nil {
			var err error
			producer, err = model.ProtoMessageToJSONData(&apiv2.IOProducer{
				TaskName: output.Producer.TaskName, Iteration: output.Producer.Iteration,
			})
			if err != nil {
				return fmt.Errorf("encode runtime output producer: %w", err)
			}
		}
		link := &model.ArtifactTask{
			TaskID: task.UUID, RunUUID: task.RunUUID, Type: model.IOType(apiv2.IOType_OUTPUT),
			ArtifactKey: output.Key, Producer: producer,
		}
		if err := link.SyncIterationFromProducer(); err != nil {
			return err
		}
		identity, err := json.Marshal([]any{"runtime-output", task.UUID, runtimeAttempt(runtime), output.Key, link.Iteration})
		if err != nil {
			return err
		}
		link.ArtifactID = util.NewDeterministicUUID(string(identity))
		link.UUID = util.NewDeterministicUUID("runtime-output-link:" + link.ArtifactID)
		if seen[link.ArtifactID] {
			return fmt.Errorf("duplicate runtime output key %q at iteration %d", output.Key, link.Iteration)
		}
		seen[link.ArtifactID] = true
		if err := s.checkpointArtifact(tx, task.Namespace, link.ArtifactID, output.Value); err != nil {
			return err
		}
		if err := s.checkpointArtifactLink(tx, link); err != nil {
			return err
		}
	}
	return nil
}

func (s *runtimeStore) checkpointArtifact(tx sqlTransaction, namespace, id string, artifact *model.Artifact) error {
	q := s.tasks.dbDialect.QuoteIdentifier
	qb := s.tasks.dbDialect.QueryBuilder()
	query, args, err := qb.Select(dialect.QuoteAll(q, artifactColumns)...).From(q("artifacts")).Where(sq.Eq{q("UUID"): id}).ToSql()
	if err != nil {
		return err
	}
	rows, err := tx.Query(query, args...)
	if err != nil {
		return fmt.Errorf("read checkpoint artifact: %w", err)
	}
	store := NewArtifactStore(s.tasks.db, s.tasks.time, s.tasks.uuid, s.tasks.dbDialect)
	existing, err := store.scanRows(rows)
	closeErr := rows.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return fmt.Errorf("close checkpoint artifact query: %w", closeErr)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate checkpoint artifact query: %w", err)
	}
	if len(existing) > 0 {
		if existing[0].Namespace != namespace || !modelArtifactsEqualForReuse(existing[0], artifact) ||
			!reflect.DeepEqual(existing[0].NumberValue, artifact.NumberValue) {
			return fmt.Errorf("runtime artifact UUID %q already identifies different metadata; use attempt-scoped identities", id)
		}
		return nil
	}
	created := *artifact
	created.UUID, created.Namespace = id, namespace
	if created.CreatedAtInSec == 0 {
		created.CreatedAtInSec = s.tasks.time.Now().Unix()
	}
	if created.LastUpdateInSec == 0 {
		created.LastUpdateInSec = created.CreatedAtInSec
	}
	_, err = store.insertArtifactWithExecutor(tx.Exec, &created)
	return err
}

func (s *runtimeStore) checkpointArtifactLink(tx sqlTransaction, link *model.ArtifactTask) error {
	q := s.tasks.dbDialect.QuoteIdentifier
	qb := s.tasks.dbDialect.QueryBuilder()
	query, args, err := qb.Select(q("ArtifactID"), q("TaskID"), q("RunUUID"), q("Type"), q("Iteration"), q("ArtifactKey")).
		From(q("artifact_tasks")).Where(sq.Eq{q("UUID"): link.UUID}).ToSql()
	if err != nil {
		return err
	}
	var existing model.ArtifactTask
	err = tx.QueryRow(query, args...).Scan(&existing.ArtifactID, &existing.TaskID, &existing.RunUUID, &existing.Type, &existing.Iteration, &existing.ArtifactKey)
	if err == nil {
		if existing.ArtifactID != link.ArtifactID || existing.TaskID != link.TaskID || existing.RunUUID != link.RunUUID ||
			existing.Type != link.Type || existing.Iteration != link.Iteration || existing.ArtifactKey != link.ArtifactKey {
			return fmt.Errorf("runtime artifact relationship UUID %q already identifies a different output", link.UUID)
		}
		return nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("read checkpoint artifact relationship: %w", err)
	}
	_, err = insertArtifactTaskWithExecutor(tx.Exec, link, s.tasks.dbDialect)
	return err
}

func (s *runtimeStore) checkpointSuppliedArtifacts(tx sqlTransaction, task *model.Task, artifacts []*model.Artifact, links []*model.ArtifactTask) error {
	byID := make(map[string]*model.Artifact, len(artifacts))
	for _, artifact := range artifacts {
		if artifact == nil || artifact.UUID == "" {
			return errors.New("runtime artifact checkpoint requires deterministic artifact UUIDs")
		}
		if artifact.Namespace != "" && artifact.Namespace != task.Namespace {
			return errors.New("runtime artifact checkpoint cannot change the task namespace")
		}
		if byID[artifact.UUID] != nil {
			return fmt.Errorf("duplicate runtime artifact UUID %q", artifact.UUID)
		}
		byID[artifact.UUID] = artifact
	}
	linked := make(map[string]bool, len(links))
	for _, link := range links {
		if link == nil || link.UUID == "" || link.ArtifactKey == "" {
			return errors.New("runtime artifact checkpoint requires deterministic link UUIDs and output keys")
		}
		if link.TaskID != task.UUID || link.RunUUID != task.RunUUID || link.Type != model.IOType(apiv2.IOType_OUTPUT) {
			return errors.New("runtime artifact checkpoint requires OUTPUT links for the current task and run")
		}
		if byID[link.ArtifactID] == nil {
			return fmt.Errorf("runtime output link %q has no matching artifact metadata", link.UUID)
		}
		linked[link.ArtifactID] = true
	}
	for _, artifact := range artifacts {
		if !linked[artifact.UUID] {
			return fmt.Errorf("runtime artifact %q has no output relationship", artifact.UUID)
		}
		if err := s.checkpointArtifact(tx, task.Namespace, artifact.UUID, artifact); err != nil {
			return err
		}
	}
	for _, link := range links {
		if err := s.checkpointArtifactLink(tx, link); err != nil {
			return err
		}
	}
	return nil
}
