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
	"testing"

	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/filter"
	"github.com/kubeflow/pipelines/backend/src/apiserver/list"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
)

func strPTR(s string) *string { return &s }

const (
	artifactUUID1 = "a23e4567-e89b-12d3-a456-426655441011"
	artifactUUID2 = "a23e4567-e89b-12d3-a456-426655441012"
	artifactUUID3 = "a23e4567-e89b-12d3-a456-426655441013"
)

// initializeArtifactStore sets up a fake DB and returns an ArtifactStore ready for testing.
func initializeArtifactStore() (*DB, *ArtifactStore) {
	db := NewFakeDBOrFatal()
	fakeTime := util.NewFakeTimeForEpoch()
	store := NewArtifactStore(db, fakeTime, util.NewFakeUUIDGeneratorOrFatal(artifactUUID1, nil))
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
	)
	require.Error(t, err)

	var artifactCount int
	row := db.QueryRow("SELECT count(*) FROM artifacts")
	require.NoError(t, row.Scan(&artifactCount))
	assert.Equal(t, 0, artifactCount)
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
		return createArtifactTaskWithExecutor(tx.Exec, store.uuid, artifactTask)
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
	)
	require.Error(t, err)

	var artifactCount int
	row := db.QueryRow("SELECT count(*) FROM artifacts")
	require.NoError(t, row.Scan(&artifactCount))
	assert.Equal(t, 0, artifactCount)
}
