// Copyright 2025 The Kubeflow Authors
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
	"fmt"
	"sync"
	"testing"

	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common/sql/dialect"
	"github.com/kubeflow/pipelines/backend/src/apiserver/filter"
	"github.com/kubeflow/pipelines/backend/src/apiserver/list"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"path/filepath"
)

func strPTR(s string) *string { return &s }

const (
	artifactUUID1 = "a23e4567-e89b-12d3-a456-426655441011"
	artifactUUID2 = "a23e4567-e89b-12d3-a456-426655441012"
	artifactUUID3 = "a23e4567-e89b-12d3-a456-426655441013"
)

// initializeArtifactStore sets up a fake DB and returns an ArtifactStore ready for testing.
func initializeArtifactStore() (*sql.DB, *ArtifactStore) {
	db, testDialect := NewFakeDBOrFatal()
	fakeTime := util.NewFakeTimeForEpoch()
	store := NewArtifactStore(db, fakeTime, util.NewFakeUUIDGeneratorOrFatal(artifactUUID1, nil), testDialect)
	return db, store
}

func TestArtifactAPIFieldMap(t *testing.T) {
	for _, modelField := range (&model.Artifact{}).APIToModelFieldMap() {
		assert.Contains(t, artifactColumns, modelField)
	}
}

func TestCreateArtifact_Success(t *testing.T) {
	db, store := initializeArtifactStore()
	defer db.Close()

	art := &model.Artifact{
		Namespace: "ns1",
		Type:      model.ArtifactType(apiv2beta1.Artifact_Artifact),
		URI:       strPTR("s3://bucket/path/file"),
		Name:      "model.pt",
		Metadata:  model.JSONData(map[string]interface{}{"k": "v"}),
	}

	created, err := store.CreateArtifact(art)
	assert.NoError(t, err)
	assert.Equal(t, artifactUUID1, created.UUID)
	assert.Greater(t, created.CreatedAtInSec, int64(0))
	assert.Equal(t, created.CreatedAtInSec, created.LastUpdateInSec)
	assert.Equal(t, "ns1", created.Namespace)
	assert.Equal(t, model.ArtifactType(apiv2beta1.Artifact_Artifact), created.Type)
	assert.Equal(t, "s3://bucket/path/file", *created.URI)
	assert.Equal(t, artifactURIHash("s3://bucket/path/file"), created.URIHash)
	assert.Equal(t, "model.pt", created.Name)
	assert.Equal(t, "v", created.Metadata["k"])

	// fetch back
	fetched, err := store.GetArtifact(created.UUID)
	assert.NoError(t, err)
	assert.Equal(t, created.UUID, fetched.UUID)
	assert.Equal(t, created.CreatedAtInSec, fetched.CreatedAtInSec)
	assert.Equal(t, created.LastUpdateInSec, fetched.LastUpdateInSec)
	assert.Equal(t, created.Namespace, fetched.Namespace)
	assert.Equal(t, created.Type, fetched.Type)
	assert.Equal(t, created.URI, fetched.URI)
	assert.Equal(t, created.URIHash, fetched.URIHash)
	assert.Equal(t, created.Name, fetched.Name)
	assert.Equal(t, created.Metadata, fetched.Metadata)
}

func TestGetArtifact_NotFound(t *testing.T) {
	db, store := initializeArtifactStore()
	defer db.Close()
	_, err := store.GetArtifact(artifactUUID1)
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())
}

func TestGetArtifactsByURI_ExactNamespaceAndURIMatch(t *testing.T) {
	db, store := initializeArtifactStore()
	defer db.Close()

	sharedURI := "s3://bucket/path/to/artifact"
	store.uuid = util.NewFakeUUIDGeneratorOrFatal(artifactUUID1, nil)
	_, err := store.CreateArtifact(&model.Artifact{
		Namespace: "ns1",
		Type:      1,
		URI:       strPTR(sharedURI),
		Name:      "match-1",
		Metadata:  map[string]interface{}{},
	})
	assert.NoError(t, err)

	store.uuid = util.NewFakeUUIDGeneratorOrFatal(artifactUUID2, nil)
	_, err = store.CreateArtifact(&model.Artifact{
		Namespace: "ns1",
		Type:      1,
		URI:       strPTR("s3://bucket/other"),
		Name:      "other-uri",
		Metadata:  map[string]interface{}{},
	})
	assert.NoError(t, err)

	store.uuid = util.NewFakeUUIDGeneratorOrFatal(artifactUUID3, nil)
	_, err = store.CreateArtifact(&model.Artifact{
		Namespace: "ns2",
		Type:      1,
		URI:       strPTR(sharedURI),
		Name:      "other-ns",
		Metadata:  map[string]interface{}{},
	})
	assert.NoError(t, err)

	matched, err := store.GetArtifactsByURI("ns1", sharedURI)
	assert.NoError(t, err)
	assert.Len(t, matched, 1)
	assert.Equal(t, "match-1", matched[0].Name)
	assert.Equal(t, "ns1", matched[0].Namespace)
	assert.Equal(t, sharedURI, *matched[0].URI)
	assert.Equal(t, artifactURIHash(sharedURI), matched[0].URIHash)
}

func TestGetArtifactsByURI_EmptyNamespaceSingleUser(t *testing.T) {
	db, store := initializeArtifactStore()
	defer db.Close()

	sharedURI := "s3://bucket/path/to/artifact"
	store.uuid = util.NewFakeUUIDGeneratorOrFatal(artifactUUID1, nil)
	_, err := store.CreateArtifact(&model.Artifact{
		Namespace: "",
		Type:      1,
		URI:       strPTR(sharedURI),
		Name:      "single-user-match",
		Metadata:  map[string]interface{}{},
	})
	assert.NoError(t, err)

	store.uuid = util.NewFakeUUIDGeneratorOrFatal(artifactUUID2, nil)
	_, err = store.CreateArtifact(&model.Artifact{
		Namespace: "ns1",
		Type:      1,
		URI:       strPTR(sharedURI),
		Name:      "namespaced-same-uri",
		Metadata:  map[string]interface{}{},
	})
	assert.NoError(t, err)

	matched, err := store.GetArtifactsByURI("", sharedURI)
	assert.NoError(t, err)
	assert.Len(t, matched, 1)
	assert.Equal(t, "single-user-match", matched[0].Name)
	assert.Equal(t, "", matched[0].Namespace)
	assert.Equal(t, sharedURI, *matched[0].URI)
}

func TestGetArtifactsByURI_FiltersHashCollisions(t *testing.T) {
	db, store := initializeArtifactStore()
	defer db.Close()

	lookupURI := "s3://bucket/path/to/artifact"
	lookupHash := artifactURIHash(lookupURI)

	store.uuid = util.NewFakeUUIDGeneratorOrFatal(artifactUUID1, nil)
	_, err := store.CreateArtifact(&model.Artifact{
		Namespace: "ns1",
		Type:      1,
		URI:       strPTR(lookupURI),
		Name:      "real-match",
		Metadata:  map[string]interface{}{},
	})
	assert.NoError(t, err)

	// Simulate a hash collision: same URIHash, different URI.
	_, err = db.Exec(
		`INSERT INTO artifacts (UUID, Namespace, Type, URI, URIHash, Name, Description, CreatedAtInSec, LastUpdateInSec, Metadata, NumberValue)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		artifactUUID2,
		"ns1",
		1,
		"s3://bucket/different-uri",
		lookupHash,
		"collision",
		"",
		1,
		1,
		"{}",
		nil,
	)
	require.NoError(t, err)

	matched, err := store.GetArtifactsByURI("ns1", lookupURI)
	assert.NoError(t, err)
	assert.Len(t, matched, 1)
	assert.Equal(t, "real-match", matched[0].Name)
	assert.Equal(t, lookupURI, *matched[0].URI)
}

func TestArtifactURIHash_EmptyURI(t *testing.T) {
	assert.Equal(t, "", artifactURIHash(""))
}

func TestListArtifacts_BasicFiltersAndPagination(t *testing.T) {
	db, store := initializeArtifactStore()
	defer db.Close()

	// Seed 3 artifacts across 2 namespaces and types
	store.uuid = util.NewFakeUUIDGeneratorOrFatal(artifactUUID1, nil)
	_, err := store.CreateArtifact(&model.Artifact{
		Namespace: "ns1",
		Type:      1,
		URI:       strPTR("u1"),
		Name:      "a1",
		Metadata:  map[string]interface{}{"m": 1},
	})
	assert.NoError(t, err)

	store.uuid = util.NewFakeUUIDGeneratorOrFatal(artifactUUID2, nil)
	_, err = store.CreateArtifact(&model.Artifact{
		Namespace: "ns1",
		Type:      2,
		URI:       strPTR("u2"),
		Name:      "a2",
		Metadata:  map[string]interface{}{"m": 2},
	})
	assert.NoError(t, err)

	store.uuid = util.NewFakeUUIDGeneratorOrFatal(artifactUUID3, nil)
	_, err = store.CreateArtifact(&model.Artifact{
		Namespace: "ns2",
		Type:      1,
		URI:       strPTR("u3"),
		Name:      "a3",
		Metadata:  map[string]interface{}{"m": 3},
	})
	assert.NoError(t, err)

	// List all
	opts, _ := list.NewOptions(&model.Artifact{}, 10, "", nil)
	all, total, npt, err := store.ListArtifacts(&model.FilterContext{}, opts)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(all))
	assert.Equal(t, 3, total)
	assert.Equal(t, "", npt)

	// Filter by Namespace
	opts2, _ := list.NewOptions(&model.Artifact{}, 10, "", nil)
	nsFiltered, total2, _, err := store.ListArtifacts(&model.FilterContext{ReferenceKey: &model.ReferenceKey{Type: model.NamespaceResourceType, ID: "ns1"}}, opts2)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(nsFiltered))
	assert.Equal(t, 2, total2)

	// Filter predicate on Type equals 1
	fProto := &apiv2beta1.Filter{Predicates: []*apiv2beta1.Predicate{
		{Key: "type", Operation: apiv2beta1.Predicate_EQUALS, Value: &apiv2beta1.Predicate_IntValue{IntValue: 1}},
	}}
	f, err := filter.New(fProto)
	assert.NoError(t, err)
	opts3, err := list.NewOptions(&model.Artifact{}, 10, "", f)
	assert.NoError(t, err)
	filtered, total3, _, err := store.ListArtifacts(&model.FilterContext{}, opts3)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(filtered))
	assert.Equal(t, 2, total3)

	// Pagination page size 2 with token
	opts4, _ := list.NewOptions(&model.Artifact{}, 2, "", nil)
	page1, total4, token, err := store.ListArtifacts(&model.FilterContext{}, opts4)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(page1))
	assert.Equal(t, 3, total4)
	assert.NotEqual(t, "", token)

	opts5, err := list.NewOptionsFromToken(token, 2)
	assert.NoError(t, err)
	page2, total5, token2, err := store.ListArtifacts(&model.FilterContext{}, opts5)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(page2))
	assert.Equal(t, 3, total5)
	assert.Equal(t, "", token2)
}

func TestCreateArtifactWithTask_RollsBackArtifactOnLinkFailure(t *testing.T) {
	db, store := initializeArtifactStore()
	defer db.Close()

	store.createArtifactTaskInTransaction = func(tx *sql.Tx, artifactTask *model.ArtifactTask) (*model.ArtifactTask, error) {
		return nil, errors.New("injected artifact-task failure")
	}

	_, _, err := store.CreateArtifactWithTask(
		&model.Artifact{
			Namespace: "ns1",
			Type:      model.ArtifactType(apiv2beta1.Artifact_Model),
			Name:      "should-rollback",
		},
		&model.ArtifactTask{
			TaskID:      "task-id",
			RunUUID:     "run-id",
			Type:        model.IOType(apiv2beta1.IOType_OUTPUT),
			ArtifactKey: "output",
			Producer: model.JSONData{
				"taskName": "task-name",
			},
		},
		0,
	)
	require.Error(t, err)

	var artifactCount, linkCount, identityCount int

	require.NoError(
		t,
		db.QueryRow("SELECT count(*) FROM artifacts").Scan(&artifactCount),
	)

	require.NoError(
		t,
		db.QueryRow("SELECT count(*) FROM artifact_tasks").Scan(&linkCount),
	)

	require.NoError(
		t,
		db.QueryRow("SELECT count(*) FROM artifact_write_identities").Scan(&identityCount),
	)

	assert.Equal(t, 0, artifactCount)
	assert.Equal(t, 0, linkCount)
	assert.Equal(t, 0, identityCount)
}

func TestCreateArtifactWithTask_RollsBackOnIdentityFailure(t *testing.T) {
	db, store := initializeArtifactStore()
	defer db.Close()

	store.createArtifactWriteIdentity = func(*sql.Tx, *model.ArtifactWriteIdentity) error {
		return errors.New("injected artifact-write-identity failure")
	}

	_, _, err := store.CreateArtifactWithTask(
		&model.Artifact{
			Namespace: "ns1",
			Type:      model.ArtifactType(apiv2beta1.Artifact_Model),
			Name:      "should-rollback",
		},
		&model.ArtifactTask{
			TaskID:      "task-id",
			RunUUID:     "run-id",
			Type:        model.IOType(apiv2beta1.IOType_OUTPUT),
			ArtifactKey: "output",
			Producer: model.JSONData{
				"taskName": "task-name",
			},
		},
		0,
	)
	require.Error(t, err)

	var artifactCount, linkCount, identityCount int

	require.NoError(
		t,
		db.QueryRow("SELECT count(*) FROM artifacts").Scan(&artifactCount),
	)

	require.NoError(
		t,
		db.QueryRow("SELECT count(*) FROM artifact_tasks").Scan(&linkCount),
	)

	require.NoError(
		t,
		db.QueryRow("SELECT count(*) FROM artifact_write_identities").Scan(&identityCount),
	)

	assert.Equal(t, 0, artifactCount)
	assert.Equal(t, 0, linkCount)
	assert.Equal(t, 0, identityCount)
}

func TestCreateArtifactsWithTasks_RollsBackWholeBatchOnFailure(t *testing.T) {
	db, store := initializeArtifactStore()
	defer db.Close()

	callCount := 0
	store.createArtifactTaskInTransaction = func(tx *sql.Tx, artifactTask *model.ArtifactTask) (*model.ArtifactTask, error) {
		callCount++
		if callCount == 2 {
			return nil, errors.New("injected artifact-task failure")
		}
		return createArtifactTaskWithExecutor(tx.Exec, store.uuid, artifactTask, store.dbDialect)
	}

	_, _, err := store.CreateArtifactsWithTasks(
		[]*model.Artifact{
			{
				Namespace: "ns1",
				Type:      model.ArtifactType(apiv2beta1.Artifact_Model),
				Name:      "first-artifact",
			},
			{
				Namespace: "ns1",
				Type:      model.ArtifactType(apiv2beta1.Artifact_Model),
				Name:      "second-artifact",
			},
		},
		[]*model.ArtifactTask{
			{
				TaskID:      "task-id-1",
				RunUUID:     "run-id-1",
				Type:        model.IOType(apiv2beta1.IOType_OUTPUT),
				ArtifactKey: "first-output",
				Producer: model.JSONData{
					"taskName": "task-1",
				},
			},
			{
				TaskID:      "task-id-2",
				RunUUID:     "run-id-2",
				Type:        model.IOType(apiv2beta1.IOType_OUTPUT),
				ArtifactKey: "second-output",
				Producer: model.JSONData{
					"taskName": "task-2",
				},
			},
		},
		make([]int64, 2),
	)
	require.Error(t, err)

	var artifactCount int
	row := db.QueryRow("SELECT count(*) FROM artifacts")
	require.NoError(t, row.Scan(&artifactCount))
	assert.Equal(t, 0, artifactCount)
}

func TestFindOrCreateArtifactWithTask_ConcurrentReuseCreatesOneArtifact(t *testing.T) {
	db, testDialect := NewFakeDBOrFatal()
	defer db.Close()
	db.SetMaxOpenConns(1)

	firstStore := NewArtifactStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(artifactUUID1, nil), testDialect)
	secondStore := NewArtifactStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(artifactUUID2, nil), testDialect)
	stores := []*ArtifactStore{firstStore, secondStore}

	sharedURI := "s3://bucket/shared-model"
	start := make(chan struct{})
	results := make(chan *model.Artifact, len(stores))
	errors := make(chan error, len(stores))
	var waitGroup sync.WaitGroup
	for index, store := range stores {
		waitGroup.Add(1)
		go func(taskStore *ArtifactStore, taskIndex int) {
			defer waitGroup.Done()
			<-start
			createdArtifact, _, err := taskStore.FindOrCreateArtifactWithTask(
				&model.Artifact{
					Namespace: "ns1",
					Type:      model.ArtifactType(apiv2beta1.Artifact_Model),
					URI:       strPTR(sharedURI),
					Name:      "shared-model",
					Metadata:  model.JSONData{"source": "importer"},
				},
				&model.ArtifactTask{
					TaskID:      fmt.Sprintf("task-%d", taskIndex),
					RunUUID:     fmt.Sprintf("run-%d", taskIndex),
					Type:        model.IOType(apiv2beta1.IOType_OUTPUT),
					ArtifactKey: "artifact",
					Producer: model.JSONData{
						"taskName": fmt.Sprintf("importer-%d", taskIndex),
					},
				},
			)
			results <- createdArtifact
			errors <- err
		}(store, index)
	}
	close(start)
	waitGroup.Wait()
	close(results)
	close(errors)

	for err := range errors {
		require.NoError(t, err)
	}
	var artifactID string
	for createdArtifact := range results {
		require.NotNil(t, createdArtifact)
		if artifactID == "" {
			artifactID = createdArtifact.UUID
		} else {
			assert.Equal(t, artifactID, createdArtifact.UUID)
		}
	}

	var artifactCount int
	row := db.QueryRow("SELECT count(*) FROM artifacts")
	require.NoError(t, row.Scan(&artifactCount))
	assert.Equal(t, 1, artifactCount)

	var linkCount int
	row = db.QueryRow("SELECT count(*) FROM artifact_tasks")
	require.NoError(t, row.Scan(&linkCount))
	assert.Equal(t, 2, linkCount)
}

func TestFindOrCreateArtifactWithTask_UnconditionalCreateAllowsDuplicates(t *testing.T) {
	db, store := initializeArtifactStore()
	defer db.Close()

	sharedURI := "s3://bucket/shared-model"
	first, _, err := store.CreateArtifactWithTask(
		&model.Artifact{
			Namespace: "ns1",
			Type:      model.ArtifactType(apiv2beta1.Artifact_Model),
			URI:       strPTR(sharedURI),
			Name:      "shared-model",
		},
		&model.ArtifactTask{
			TaskID:      "task-1",
			RunUUID:     "run-1",
			Type:        model.IOType(apiv2beta1.IOType_OUTPUT),
			ArtifactKey: "artifact",
			Producer:    model.JSONData{"taskName": "importer-1"},
		},
		0,
	)
	require.NoError(t, err)

	store.uuid = util.NewFakeUUIDGeneratorOrFatal(artifactUUID2, nil)
	second, _, err := store.CreateArtifactWithTask(
		&model.Artifact{
			Namespace: "ns1",
			Type:      model.ArtifactType(apiv2beta1.Artifact_Model),
			URI:       strPTR(sharedURI),
			Name:      "shared-model",
		},
		&model.ArtifactTask{
			TaskID:      "task-2",
			RunUUID:     "run-2",
			Type:        model.IOType(apiv2beta1.IOType_OUTPUT),
			ArtifactKey: "artifact",
			Producer:    model.JSONData{"taskName": "importer-2"},
		},
		0,
	)
	require.NoError(t, err)
	assert.NotEqual(t, first.UUID, second.UUID)

	var artifactCount int
	row := db.QueryRow("SELECT count(*) FROM artifacts")
	require.NoError(t, row.Scan(&artifactCount))
	assert.Equal(t, 2, artifactCount)
}

func TestFindOrCreateArtifactWithTask_ReplaysSameTaskLinkIdempotently(t *testing.T) {
	db, store := initializeArtifactStore()
	defer db.Close()

	sharedURI := "s3://bucket/shared-model"
	artifact := &model.Artifact{
		Namespace: "ns1",
		Type:      model.ArtifactType(apiv2beta1.Artifact_Model),
		URI:       strPTR(sharedURI),
		Name:      "shared-model",
		Metadata:  model.JSONData{"source": "importer"},
	}
	link := &model.ArtifactTask{
		TaskID:      "task-1",
		RunUUID:     "run-1",
		Type:        model.IOType(apiv2beta1.IOType_OUTPUT),
		ArtifactKey: "artifact",
		Producer:    model.JSONData{"taskName": "importer"},
	}

	firstArtifact, firstLink, err := store.FindOrCreateArtifactWithTask(artifact, link)
	require.NoError(t, err)

	secondArtifact, secondLink, err := store.FindOrCreateArtifactWithTask(artifact, link)
	require.NoError(t, err)
	assert.Equal(t, firstArtifact.UUID, secondArtifact.UUID)
	assert.Equal(t, firstLink.UUID, secondLink.UUID)

	var linkCount int
	row := db.QueryRow("SELECT count(*) FROM artifact_tasks")
	require.NoError(t, row.Scan(&linkCount))
	assert.Equal(t, 1, linkCount)
}

func TestFindOrCreateArtifactWithTask_RecoversAfterIdentityKeyConflict(t *testing.T) {
	db, store := initializeArtifactStore()
	defer db.Close()

	sharedURI := "s3://bucket/shared-model"
	artifact := &model.Artifact{
		Namespace: "ns1",
		Type:      model.ArtifactType(apiv2beta1.Artifact_Model),
		URI:       strPTR(sharedURI),
		Name:      "shared-model",
		Metadata:  model.JSONData{"source": "importer"},
	}
	identityKey, err := computeArtifactIdentityKey(artifact)
	require.NoError(t, err)
	artifact.IdentityKey = &identityKey

	existing, _, err := store.CreateArtifactWithTask(
		artifact,
		&model.ArtifactTask{
			TaskID:      "task-1",
			RunUUID:     "run-1",
			Type:        model.IOType(apiv2beta1.IOType_OUTPUT),
			ArtifactKey: "artifact",
			Producer:    model.JSONData{"taskName": "importer-1"},
		},
		0,
	)
	require.NoError(t, err)

	store.uuid = util.NewFakeUUIDGeneratorOrFatal(artifactUUID2, nil)
	reused, reusedLink, err := store.FindOrCreateArtifactWithTask(
		&model.Artifact{
			Namespace: "ns1",
			Type:      model.ArtifactType(apiv2beta1.Artifact_Model),
			URI:       strPTR(sharedURI),
			Name:      "shared-model",
			Metadata:  model.JSONData{"source": "importer"},
		},
		&model.ArtifactTask{
			TaskID:      "task-2",
			RunUUID:     "run-2",
			Type:        model.IOType(apiv2beta1.IOType_OUTPUT),
			ArtifactKey: "artifact",
			Producer:    model.JSONData{"taskName": "importer-2"},
		},
	)
	require.NoError(t, err)
	assert.Equal(t, existing.UUID, reused.UUID, "reused artifact should have the same UUID as the existing artifact")
	assert.Equal(t, existing.UUID, reusedLink.ArtifactID, "reused link should have the same artifact ID as the existing artifact")
}

func TestComputeArtifactWritePayloadHash(t *testing.T) {
	uri := "s3://bucket/output"

	artifact := &model.Artifact{
		Type:        model.ArtifactType(apiv2beta1.Artifact_Model),
		URI:         &uri,
		Name:        "model",
		Description: "trained model",
		Metadata: model.JSONData{
			"z": "last",
			"a": "first",
		},
	}

	first, err := computeArtifactWritePayloadHash(artifact)
	require.NoError(t, err)

	reordered := &model.Artifact{
		Type:        artifact.Type,
		URI:         artifact.URI,
		Name:        artifact.Name,
		Description: artifact.Description,
		Metadata: model.JSONData{
			"a": "first",
			"z": "last",
		},
	}

	second, err := computeArtifactWritePayloadHash(reordered)
	require.NoError(t, err)

	assert.Equal(t, first, second)
}

func TestComputeArtifactWritePayloadHash_ChangesWithPayload(t *testing.T) {
	uri := "s3://bucket/output"

	artifact := &model.Artifact{
		Type: model.ArtifactType(apiv2beta1.Artifact_Model),
		URI:  &uri,
		Name: "model",
	}

	first, err := computeArtifactWritePayloadHash(artifact)
	require.NoError(t, err)

	artifact.Name = "different-model"

	second, err := computeArtifactWritePayloadHash(artifact)
	require.NoError(t, err)

	assert.NotEqual(t, first, second)
}

func TestComputeArtifactWritePayloadHash_NumberValuePresenceMatters(t *testing.T) {
	withoutValue := &model.Artifact{
		Type: model.ArtifactType(apiv2beta1.Artifact_Model),
	}

	zero := float64(0)
	withZero := &model.Artifact{
		Type:        model.ArtifactType(apiv2beta1.Artifact_Model),
		NumberValue: &zero,
	}

	first, err := computeArtifactWritePayloadHash(withoutValue)
	require.NoError(t, err)

	second, err := computeArtifactWritePayloadHash(withZero)
	require.NoError(t, err)

	assert.NotEqual(t, first, second)
}

func TestCreateArtifactWithTask_ReplayOfCommittedWriteDoesNotDuplicate(t *testing.T) {
	db, store := initializeArtifactStore()
	defer db.Close()

	newArtifact := func() *model.Artifact {
		return &model.Artifact{
			Namespace: "ns1",
			Type:      model.ArtifactType(apiv2beta1.Artifact_Model),
			URI:       strPTR("s3://bucket/output-model"),
			Name:      "output-model",
		}
	}
	newLink := func() *model.ArtifactTask {
		return &model.ArtifactTask{
			TaskID:      "task-1",
			RunUUID:     "run-1",
			Type:        model.IOType(apiv2beta1.IOType_OUTPUT),
			ArtifactKey: "output",
			Producer:    model.JSONData{"taskName": "train"},
		}
	}

	first, firstLink, err := store.CreateArtifactWithTask(newArtifact(), newLink(), 0)
	require.NoError(t, err)
	// Simulate a lost response: the caller retries the same request.
	// In production every call draws a fresh random UUID, so use a different one here.
	store.uuid = util.NewFakeUUIDGeneratorOrFatal(artifactUUID2, nil)
	second, secondLink, err := store.CreateArtifactWithTask(newArtifact(), newLink(), 0)
	require.NoError(t, err)

	assert.Equal(t, first.UUID, second.UUID, "replay should return the original artifact")
	assert.Equal(t, firstLink.UUID, secondLink.UUID, "replay should return the original link")

	var artifactCount, linkCount int
	require.NoError(t, db.QueryRow("SELECT count(*) FROM artifacts").Scan(&artifactCount))
	require.NoError(t, db.QueryRow("SELECT count(*) FROM artifact_tasks").Scan(&linkCount))
	assert.Equal(t, 1, artifactCount)
	assert.Equal(t, 1, linkCount)
}

func TestCreateArtifactWithTask_ConcurrentSameIdentityConverges(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "artifact-store.db")

	gormDB, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	require.NoError(t, err)

	require.NoError(t, gormDB.AutoMigrate(model.AllModels()...))

	db, err := gormDB.DB()
	require.NoError(t, err)
	defer db.Close()

	db.SetMaxOpenConns(4)

	testDialect := dialect.NewDBDialect("sqlite")
	firstStore := NewArtifactStore(
		db,
		util.NewFakeTimeForEpoch(),
		util.NewFakeUUIDGeneratorOrFatal(artifactUUID1, nil),
		testDialect,
	)
	secondStore := NewArtifactStore(
		db,
		util.NewFakeTimeForEpoch(),
		util.NewFakeUUIDGeneratorOrFatal(artifactUUID2, nil),
		testDialect,
	)

	artifact := &model.Artifact{
		Namespace: "ns1",
		Type:      model.ArtifactType(apiv2beta1.Artifact_Model),
		Name:      "concurrent-write",
	}

	artifactTask := &model.ArtifactTask{
		TaskID:      "task-id",
		RunUUID:     "run-id",
		Type:        model.IOType(apiv2beta1.IOType_OUTPUT),
		ArtifactKey: "output",
		Producer: model.JSONData{
			"taskName": "task-name",
		},
	}

	type result struct {
		artifact     *model.Artifact
		artifactTask *model.ArtifactTask
		err          error
	}

	start := make(chan struct{})
	results := make(chan result, 2)

	var waitGroup sync.WaitGroup
	for _, store := range []*ArtifactStore{firstStore, secondStore} {
		waitGroup.Add(1)

		go func(store *ArtifactStore) {
			defer waitGroup.Done()

			<-start

			createdArtifact, createdArtifactTask, err :=
				store.CreateArtifactWithTask(artifact, artifactTask, 0)

			results <- result{
				artifact:     createdArtifact,
				artifactTask: createdArtifactTask,
				err:          err,
			}
		}(store)
	}

	close(start)
	waitGroup.Wait()
	close(results)

	var resultsList []result
	for result := range results {
		resultsList = append(resultsList, result)
	}

	require.Len(t, resultsList, 2)

	for _, result := range resultsList {
		require.NoError(t, result.err)
		require.NotNil(t, result.artifact)
		require.NotNil(t, result.artifactTask)
	}

	assert.Equal(t, resultsList[0].artifact.UUID, resultsList[1].artifact.UUID)
	assert.Equal(t, resultsList[0].artifactTask.UUID, resultsList[1].artifactTask.UUID)

	var artifactCount, linkCount, identityCount int

	require.NoError(
		t,
		db.QueryRow("SELECT count(*) FROM artifacts").Scan(&artifactCount),
	)
	require.NoError(
		t,
		db.QueryRow("SELECT count(*) FROM artifact_tasks").Scan(&linkCount),
	)
	require.NoError(
		t,
		db.QueryRow("SELECT count(*) FROM artifact_write_identities").Scan(&identityCount),
	)

	assert.Equal(t, 1, artifactCount)
	assert.Equal(t, 1, linkCount)
	assert.Equal(t, 1, identityCount)
}

func TestCreateArtifactWithTask_ReplayWithDifferentPayloadReturnsConflict(t *testing.T) {
	db, store := initializeArtifactStore()
	defer db.Close()

	firstArtifact := &model.Artifact{
		Namespace: "ns1",
		Type:      model.ArtifactType(apiv2beta1.Artifact_Model),
		URI:       strPTR("s3://bucket/output-model"),
		Name:      "output-model",
	}

	firstLink := &model.ArtifactTask{
		TaskID:      "task-1",
		RunUUID:     "run-1",
		Type:        model.IOType(apiv2beta1.IOType_OUTPUT),
		ArtifactKey: "output",
		Producer:    model.JSONData{"taskName": "train"},
	}

	first, _, err := store.CreateArtifactWithTask(firstArtifact, firstLink, 0)
	require.NoError(t, err)

	secondArtifact := &model.Artifact{
		Namespace: "ns1",
		Type:      model.ArtifactType(apiv2beta1.Artifact_Model),
		URI:       strPTR("s3://bucket/output-model"),
		Name:      "different-output-model",
	}

	secondLink := &model.ArtifactTask{
		TaskID:      "task-1",
		RunUUID:     "run-1",
		Type:        model.IOType(apiv2beta1.IOType_OUTPUT),
		ArtifactKey: "output",
		Producer:    model.JSONData{"taskName": "train"},
	}

	_, _, err = store.CreateArtifactWithTask(secondArtifact, secondLink, 0)
	require.Error(t, err)

	assert.Contains(t, err.Error(), "already exists with a different payload")

	var artifactCount, linkCount, identityCount int
	require.NoError(t, db.QueryRow("SELECT count(*) FROM artifacts").Scan(&artifactCount))
	require.NoError(t, db.QueryRow("SELECT count(*) FROM artifact_tasks").Scan(&linkCount))
	require.NoError(t, db.QueryRow("SELECT count(*) FROM artifact_write_identities").Scan(&identityCount))

	assert.Equal(t, 1, artifactCount)
	assert.Equal(t, 1, linkCount)
	assert.Equal(t, 1, identityCount)
	assert.NotEmpty(t, first.UUID)
}

func TestCreateArtifactWithTask_DifferentRetryGenerationCreatesDistinctWrite(t *testing.T) {
	db, store := initializeArtifactStore()
	defer db.Close()

	newArtifact := func() *model.Artifact {
		return &model.Artifact{
			Namespace: "ns1",
			Type:      model.ArtifactType(apiv2beta1.Artifact_Model),
			URI:       strPTR("s3://bucket/output-model"),
			Name:      "output-model",
		}
	}

	newLink := func() *model.ArtifactTask {
		return &model.ArtifactTask{
			TaskID:      "task-1",
			RunUUID:     "run-1",
			Type:        model.IOType(apiv2beta1.IOType_OUTPUT),
			ArtifactKey: "output",
			Producer:    model.JSONData{"taskName": "train"},
		}
	}

	first, firstLink, err := store.CreateArtifactWithTask(
		newArtifact(),
		newLink(),
		0,
	)
	require.NoError(t, err)

	// Use a different generated UUID for the second physical artifact.
	// RetryGeneration is what makes the two writes logically distinct.
	store.uuid = util.NewFakeUUIDGeneratorOrFatal(artifactUUID2, nil)

	second, secondLink, err := store.CreateArtifactWithTask(
		newArtifact(),
		newLink(),
		1,
	)
	require.NoError(t, err)

	assert.NotEqual(t, first.UUID, second.UUID)
	assert.NotEqual(t, firstLink.UUID, secondLink.UUID)

	var artifactCount, linkCount, identityCount int

	require.NoError(
		t,
		db.QueryRow("SELECT count(*) FROM artifacts").Scan(&artifactCount),
	)

	require.NoError(
		t,
		db.QueryRow("SELECT count(*) FROM artifact_tasks").Scan(&linkCount),
	)

	require.NoError(
		t,
		db.QueryRow("SELECT count(*) FROM artifact_write_identities").Scan(&identityCount),
	)

	assert.Equal(t, 2, artifactCount)
	assert.Equal(t, 2, linkCount)
	assert.Equal(t, 2, identityCount)
}

func TestCreateArtifactWithTask_DifferentTaskIDCreatesDistinctWrite(t *testing.T) {
	db, store := initializeArtifactStore()
	defer db.Close()

	newArtifact := func() *model.Artifact {
		return &model.Artifact{
			Namespace: "ns1",
			Type:      model.ArtifactType(apiv2beta1.Artifact_Model),
			URI:       strPTR("s3://bucket/output-model"),
			Name:      "output-model",
		}
	}

	firstLink := &model.ArtifactTask{
		TaskID:      "task-1",
		RunUUID:     "run-1",
		Type:        model.IOType(apiv2beta1.IOType_OUTPUT),
		ArtifactKey: "output",
		Producer:    model.JSONData{"taskName": "train"},
	}

	secondLink := &model.ArtifactTask{
		TaskID:      "task-2",
		RunUUID:     "run-1",
		Type:        model.IOType(apiv2beta1.IOType_OUTPUT),
		ArtifactKey: "output",
		Producer:    model.JSONData{"taskName": "train"},
	}

	first, firstArtifactTask, err := store.CreateArtifactWithTask(
		newArtifact(),
		firstLink,
		0,
	)
	require.NoError(t, err)

	// Use a different generated UUID for the second physical artifact.
	store.uuid = util.NewFakeUUIDGeneratorOrFatal(artifactUUID2, nil)

	second, secondArtifactTask, err := store.CreateArtifactWithTask(
		newArtifact(),
		secondLink,
		0,
	)
	require.NoError(t, err)

	assert.NotEqual(t, first.UUID, second.UUID)
	assert.NotEqual(t, firstArtifactTask.UUID, secondArtifactTask.UUID)

	var artifactCount, linkCount, identityCount int

	require.NoError(
		t,
		db.QueryRow("SELECT count(*) FROM artifacts").Scan(&artifactCount),
	)

	require.NoError(
		t,
		db.QueryRow("SELECT count(*) FROM artifact_tasks").Scan(&linkCount),
	)

	require.NoError(
		t,
		db.QueryRow("SELECT count(*) FROM artifact_write_identities").Scan(&identityCount),
	)

	assert.Equal(t, 2, artifactCount)
	assert.Equal(t, 2, linkCount)
	assert.Equal(t, 2, identityCount)
}

func TestCreateArtifactWithTask_DifferentRunUUIDCreatesDistinctWrite(t *testing.T) {
	db, store := initializeArtifactStore()
	defer db.Close()

	newArtifact := func() *model.Artifact {
		return &model.Artifact{
			Namespace: "ns1",
			Type:      model.ArtifactType(apiv2beta1.Artifact_Model),
			URI:       strPTR("s3://bucket/output-model"),
			Name:      "output-model",
		}
	}

	firstLink := &model.ArtifactTask{
		TaskID:      "task-1",
		RunUUID:     "run-1",
		Type:        model.IOType(apiv2beta1.IOType_OUTPUT),
		ArtifactKey: "output",
		Producer:    model.JSONData{"taskName": "train"},
	}

	secondLink := &model.ArtifactTask{
		TaskID:      "task-1",
		RunUUID:     "run-2",
		Type:        model.IOType(apiv2beta1.IOType_OUTPUT),
		ArtifactKey: "output",
		Producer:    model.JSONData{"taskName": "train"},
	}

	first, firstArtifactTask, err := store.CreateArtifactWithTask(
		newArtifact(),
		firstLink,
		0,
	)
	require.NoError(t, err)

	// Use a different generated UUID for the second physical artifact.
	store.uuid = util.NewFakeUUIDGeneratorOrFatal(artifactUUID2, nil)

	second, secondArtifactTask, err := store.CreateArtifactWithTask(
		newArtifact(),
		secondLink,
		0,
	)
	require.NoError(t, err)

	assert.NotEqual(t, first.UUID, second.UUID)
	assert.NotEqual(t, firstArtifactTask.UUID, secondArtifactTask.UUID)

	var artifactCount, linkCount, identityCount int

	require.NoError(
		t,
		db.QueryRow("SELECT count(*) FROM artifacts").Scan(&artifactCount),
	)

	require.NoError(
		t,
		db.QueryRow("SELECT count(*) FROM artifact_tasks").Scan(&linkCount),
	)

	require.NoError(
		t,
		db.QueryRow("SELECT count(*) FROM artifact_write_identities").Scan(&identityCount),
	)

	assert.Equal(t, 2, artifactCount)
	assert.Equal(t, 2, linkCount)
	assert.Equal(t, 2, identityCount)
}

func TestCreateArtifactWithTask_DifferentNamespaceCreatesDistinctWrite(t *testing.T) {
	db, store := initializeArtifactStore()
	defer db.Close()

	firstArtifact := &model.Artifact{
		Namespace: "ns1",
		Type:      model.ArtifactType(apiv2beta1.Artifact_Model),
		URI:       strPTR("s3://bucket/output-model"),
		Name:      "output-model",
	}

	secondArtifact := &model.Artifact{
		Namespace: "ns2",
		Type:      model.ArtifactType(apiv2beta1.Artifact_Model),
		URI:       strPTR("s3://bucket/output-model"),
		Name:      "output-model",
	}

	newLink := func() *model.ArtifactTask {
		return &model.ArtifactTask{
			TaskID:      "task-1",
			RunUUID:     "run-1",
			Type:        model.IOType(apiv2beta1.IOType_OUTPUT),
			ArtifactKey: "output",
			Producer:    model.JSONData{"taskName": "train"},
		}
	}

	first, firstArtifactTask, err := store.CreateArtifactWithTask(
		firstArtifact,
		newLink(),
		0,
	)
	require.NoError(t, err)

	// Use a different generated UUID for the second physical artifact.
	store.uuid = util.NewFakeUUIDGeneratorOrFatal(artifactUUID2, nil)

	second, secondArtifactTask, err := store.CreateArtifactWithTask(
		secondArtifact,
		newLink(),
		0,
	)
	require.NoError(t, err)

	assert.NotEqual(t, first.UUID, second.UUID)
	assert.NotEqual(t, firstArtifactTask.UUID, secondArtifactTask.UUID)

	var artifactCount, linkCount, identityCount int

	require.NoError(
		t,
		db.QueryRow("SELECT count(*) FROM artifacts").Scan(&artifactCount),
	)

	require.NoError(
		t,
		db.QueryRow("SELECT count(*) FROM artifact_tasks").Scan(&linkCount),
	)

	require.NoError(
		t,
		db.QueryRow("SELECT count(*) FROM artifact_write_identities").Scan(&identityCount),
	)

	assert.Equal(t, 2, artifactCount)
	assert.Equal(t, 2, linkCount)
	assert.Equal(t, 2, identityCount)
}

func TestCreateArtifactWithTask_DifferentArtifactKeyCreatesDistinctWrite(t *testing.T) {
	db, store := initializeArtifactStore()
	defer db.Close()

	newArtifact := func() *model.Artifact {
		return &model.Artifact{
			Namespace: "ns1",
			Type:      model.ArtifactType(apiv2beta1.Artifact_Model),
			URI:       strPTR("s3://bucket/output-model"),
			Name:      "output-model",
		}
	}

	firstLink := &model.ArtifactTask{
		TaskID:      "task-1",
		RunUUID:     "run-1",
		Type:        model.IOType(apiv2beta1.IOType_OUTPUT),
		ArtifactKey: "output-a",
		Producer:    model.JSONData{"taskName": "train"},
	}

	secondLink := &model.ArtifactTask{
		TaskID:      "task-1",
		RunUUID:     "run-1",
		Type:        model.IOType(apiv2beta1.IOType_OUTPUT),
		ArtifactKey: "output-b",
		Producer:    model.JSONData{"taskName": "train"},
	}

	first, firstArtifactTask, err := store.CreateArtifactWithTask(
		newArtifact(),
		firstLink,
		0,
	)
	require.NoError(t, err)

	// Use a different generated UUID for the second physical artifact.
	store.uuid = util.NewFakeUUIDGeneratorOrFatal(artifactUUID2, nil)

	second, secondArtifactTask, err := store.CreateArtifactWithTask(
		newArtifact(),
		secondLink,
		0,
	)
	require.NoError(t, err)

	assert.NotEqual(t, first.UUID, second.UUID)
	assert.NotEqual(t, firstArtifactTask.UUID, secondArtifactTask.UUID)

	var artifactCount, linkCount, identityCount int

	require.NoError(
		t,
		db.QueryRow("SELECT count(*) FROM artifacts").Scan(&artifactCount),
	)

	require.NoError(
		t,
		db.QueryRow("SELECT count(*) FROM artifact_tasks").Scan(&linkCount),
	)

	require.NoError(
		t,
		db.QueryRow("SELECT count(*) FROM artifact_write_identities").Scan(&identityCount),
	)

	assert.Equal(t, 2, artifactCount)
	assert.Equal(t, 2, linkCount)
	assert.Equal(t, 2, identityCount)
}

func TestCreateArtifactWithTask_DifferentIterationCreatesDistinctWrite(t *testing.T) {
	db, store := initializeArtifactStore()
	defer db.Close()

	newArtifact := func() *model.Artifact {
		return &model.Artifact{
			Namespace: "ns1",
			Type:      model.ArtifactType(apiv2beta1.Artifact_Model),
			URI:       strPTR("s3://bucket/output-model"),
			Name:      "output-model",
		}
	}

	firstLink := &model.ArtifactTask{
		TaskID:      "task-1",
		RunUUID:     "run-1",
		Type:        model.IOType(apiv2beta1.IOType_OUTPUT),
		ArtifactKey: "output",
		Producer: model.JSONData{
			"taskName":  "train",
			"iteration": float64(0),
		},
	}

	secondLink := &model.ArtifactTask{
		TaskID:      "task-1",
		RunUUID:     "run-1",
		Type:        model.IOType(apiv2beta1.IOType_OUTPUT),
		ArtifactKey: "output",
		Producer: model.JSONData{
			"taskName":  "train",
			"iteration": float64(1),
		},
	}

	first, firstArtifactTask, err := store.CreateArtifactWithTask(
		newArtifact(),
		firstLink,
		0,
	)
	require.NoError(t, err)

	// Use a different generated UUID for the second physical artifact.
	store.uuid = util.NewFakeUUIDGeneratorOrFatal(artifactUUID2, nil)

	second, secondArtifactTask, err := store.CreateArtifactWithTask(
		newArtifact(),
		secondLink,
		0,
	)
	require.NoError(t, err)

	assert.NotEqual(t, first.UUID, second.UUID)
	assert.NotEqual(t, firstArtifactTask.UUID, secondArtifactTask.UUID)
	assert.NotEqual(t, firstArtifactTask.Iteration, secondArtifactTask.Iteration)

	var artifactCount, linkCount, identityCount int

	require.NoError(
		t,
		db.QueryRow("SELECT count(*) FROM artifacts").Scan(&artifactCount),
	)

	require.NoError(
		t,
		db.QueryRow("SELECT count(*) FROM artifact_tasks").Scan(&linkCount),
	)

	require.NoError(
		t,
		db.QueryRow("SELECT count(*) FROM artifact_write_identities").Scan(&identityCount),
	)

	assert.Equal(t, 2, artifactCount)
	assert.Equal(t, 2, linkCount)
	assert.Equal(t, 2, identityCount)
}
