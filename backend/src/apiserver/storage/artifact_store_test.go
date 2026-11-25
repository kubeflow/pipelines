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

	apiv2 "github.com/kubeflow/pipelines/backend/api/v2/go_client"
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
		Type:      model.ArtifactType(apiv2.Artifact_Artifact),
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
	assert.Equal(t, model.ArtifactType(apiv2.Artifact_Artifact), created.Type)
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

func TestGetArtifactsByIDs_Success(t *testing.T) {
	db, store := initializeArtifactStore()
	defer db.Close()

	store.uuid = util.NewFakeUUIDGeneratorOrFatal(artifactUUID1, nil)
	art1, err := store.CreateArtifact(&model.Artifact{
		Namespace: "ns1",
		Type:      model.ArtifactType(apiv2.Artifact_Artifact),
		URI:       strPTR("s3://bucket/path/art1"),
		Name:      "art1",
	})
	assert.NoError(t, err)

	store.uuid = util.NewFakeUUIDGeneratorOrFatal(artifactUUID2, nil)
	art2, err := store.CreateArtifact(&model.Artifact{
		Namespace: "ns1",
		Type:      model.ArtifactType(apiv2.Artifact_Artifact),
		URI:       strPTR("s3://bucket/path/art2"),
		Name:      "art2",
	})
	assert.NoError(t, err)

	// Fetch batch by IDs including duplicates, empty strings, and non-existent IDs
	batch, err := store.GetArtifactsByIDs([]string{art1.UUID, art2.UUID, art1.UUID, "", "non-existent-id"})
	assert.NoError(t, err)
	assert.Equal(t, 2, len(batch))
	assert.Equal(t, art1.UUID, batch[art1.UUID].UUID)
	assert.Equal(t, "art1", batch[art1.UUID].Name)
	assert.Equal(t, art2.UUID, batch[art2.UUID].UUID)
	assert.Equal(t, "art2", batch[art2.UUID].Name)

	// Empty input returns empty map
	emptyBatch, err := store.GetArtifactsByIDs(nil)
	assert.NoError(t, err)
	assert.Empty(t, emptyBatch)
}

func TestGetArtifactsByIDs_CaseSensitivityAndNOCASE(t *testing.T) {
	// 1. Standard SQLite: case-sensitive matching by default
	dbStandard, storeStandard := initializeArtifactStore()
	defer dbStandard.Close()

	art, err := storeStandard.CreateArtifact(&model.Artifact{
		Namespace: "ns1",
		Type:      model.ArtifactType(apiv2.Artifact_Artifact),
		URI:       strPTR("s3://bucket/test"),
		Name:      "test-artifact",
	})
	require.NoError(t, err)

	// Exact match succeeds
	exactBatch, err := storeStandard.GetArtifactsByIDs([]string{art.UUID})
	require.NoError(t, err)
	assert.Equal(t, 1, len(exactBatch))

	// Mixed-case ID fails on exact-matching database
	upperID := "A23E4567-E89B-12D3-A456-426655441011"
	upperBatch, err := storeStandard.GetArtifactsByIDs([]string{upperID})
	require.NoError(t, err)
	assert.Empty(t, upperBatch, "expected no match for mixed-case ID on case-sensitive database")

	// Trailing whitespace fails on exact-matching database
	spaceBatch, err := storeStandard.GetArtifactsByIDs([]string{art.UUID + "   "})
	require.NoError(t, err)
	assert.Empty(t, spaceBatch, "expected no match for trailing-space ID on exact-matching database")

	// 2. Case-insensitive database: SQLite table defined with COLLATE NOCASE
	dbNocase, testDialect := NewFakeDBOrFatal()
	defer dbNocase.Close()
	_, err = dbNocase.Exec("DROP TABLE IF EXISTS " + artifactTableName)
	require.NoError(t, err)
	_, err = dbNocase.Exec(`CREATE TABLE ` + artifactTableName + ` (
		UUID VARCHAR(255) COLLATE NOCASE NOT NULL PRIMARY KEY,
		Namespace VARCHAR(255),
		Name VARCHAR(255),
		Type INTEGER,
		URI VARCHAR(255),
		URIHash VARCHAR(64),
		Description TEXT,
		CreatedAtInSec BIGINT,
		LastUpdateInSec BIGINT,
		Metadata TEXT,
		IdentityKey VARCHAR(255),
		NumberValue DOUBLE PRECISION
	)`)
	require.NoError(t, err)

	storeNocase := NewArtifactStore(dbNocase, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(artifactUUID1, nil), testDialect)
	artNocase, err := storeNocase.CreateArtifact(&model.Artifact{
		Namespace: "ns1",
		Type:      model.ArtifactType(apiv2.Artifact_Artifact),
		URI:       strPTR("s3://bucket/test-nocase"),
		Name:      "test-artifact-nocase",
	})
	require.NoError(t, err)

	// On case-insensitive database, mixed-case ID matches
	upperNocaseID := "A23E4567-E89B-12D3-A456-426655441011"
	nocaseBatch, err := storeNocase.GetArtifactsByIDs([]string{upperNocaseID})
	require.NoError(t, err)
	assert.NotEmpty(t, nocaseBatch, "expected mixed-case ID to match on case-insensitive database")
	assert.NotNil(t, nocaseBatch[upperNocaseID])
	assert.Equal(t, artNocase.UUID, nocaseBatch[upperNocaseID].UUID)

	// 3. Mixed-column table: UUID is case-sensitive (BINARY), but Name uses COLLATE NOCASE.
	// UUID comparison is governed by UUID column collation, unaffected by NOCASE on Name.
	dbMixed, testDialectMixed := NewFakeDBOrFatal()
	defer dbMixed.Close()
	_, err = dbMixed.Exec("DROP TABLE IF EXISTS " + artifactTableName)
	require.NoError(t, err)
	_, err = dbMixed.Exec(`CREATE TABLE ` + artifactTableName + ` (
		UUID VARCHAR(255) NOT NULL PRIMARY KEY,
		Namespace VARCHAR(255),
		Name VARCHAR(255) COLLATE NOCASE,
		Type INTEGER,
		URI VARCHAR(255),
		URIHash VARCHAR(64),
		Description TEXT,
		CreatedAtInSec BIGINT,
		LastUpdateInSec BIGINT,
		Metadata TEXT,
		IdentityKey VARCHAR(255),
		NumberValue DOUBLE PRECISION
	)`)
	require.NoError(t, err)

	storeMixed := NewArtifactStore(dbMixed, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(artifactUUID1, nil), testDialectMixed)
	artMixed, err := storeMixed.CreateArtifact(&model.Artifact{
		Namespace: "ns1",
		Type:      model.ArtifactType(apiv2.Artifact_Artifact),
		URI:       strPTR("s3://bucket/test-mixed"),
		Name:      "test-artifact-mixed",
	})
	require.NoError(t, err)

	// In mixed-column table, batch containing existing lowercase ID and nonexistent uppercase ID:
	// exact match succeeds, uppercase ID does NOT match.
	mixedBatch, err := storeMixed.GetArtifactsByIDs([]string{artMixed.UUID, upperID})
	require.NoError(t, err)
	assert.Equal(t, 1, len(mixedBatch))
	assert.NotNil(t, mixedBatch[artMixed.UUID])
	assert.Nil(t, mixedBatch[upperID], "expected nonexistent uppercase ID not to resolve when only Name has NOCASE")

	// 4. Multiline SQLite column definition with COLLATE NOCASE.
	// SQLite allows newlines in column definitions, e.g. between VARCHAR(255) and COLLATE NOCASE.
	dbMultiline, testDialectMultiline := NewFakeDBOrFatal()
	defer dbMultiline.Close()
	_, err = dbMultiline.Exec("DROP TABLE IF EXISTS " + artifactTableName)
	require.NoError(t, err)
	_, err = dbMultiline.Exec(`CREATE TABLE ` + artifactTableName + ` (
		UUID VARCHAR(255)
			COLLATE NOCASE NOT NULL PRIMARY KEY,
		Namespace VARCHAR(255),
		Name VARCHAR(255),
		Type INTEGER,
		URI VARCHAR(255),
		URIHash VARCHAR(64),
		Description TEXT,
		CreatedAtInSec BIGINT,
		LastUpdateInSec BIGINT,
		Metadata TEXT,
		IdentityKey VARCHAR(255),
		NumberValue DOUBLE PRECISION
	)`)
	require.NoError(t, err)

	storeMultiline := NewArtifactStore(dbMultiline, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(artifactUUID1, nil), testDialectMultiline)
	artMultiline, err := storeMultiline.CreateArtifact(&model.Artifact{
		Namespace: "ns1",
		Type:      model.ArtifactType(apiv2.Artifact_Artifact),
		URI:       strPTR("s3://bucket/test-multiline"),
		Name:      "test-artifact-multiline",
	})
	require.NoError(t, err)

	upperMultilineID := "A23E4567-E89B-12D3-A456-426655441011"
	multilineBatch, err := storeMultiline.GetArtifactsByIDs([]string{upperMultilineID})
	require.NoError(t, err)
	assert.NotEmpty(t, multilineBatch, "expected mixed-case ID to match on multiline NOCASE column definition")
	assert.NotNil(t, multilineBatch[upperMultilineID])
	assert.Equal(t, artMultiline.UUID, multilineBatch[upperMultilineID].UUID)

	// 5. Quoted SQLite column definition with COLLATE "NOCASE".
	dbQuoted, testDialectQuoted := NewFakeDBOrFatal()
	defer dbQuoted.Close()
	_, err = dbQuoted.Exec("DROP TABLE IF EXISTS " + artifactTableName)
	require.NoError(t, err)
	_, err = dbQuoted.Exec(`CREATE TABLE ` + artifactTableName + ` (
		UUID VARCHAR(255) COLLATE "NOCASE" NOT NULL PRIMARY KEY,
		Namespace VARCHAR(255),
		Name VARCHAR(255),
		Type INTEGER,
		URI VARCHAR(255),
		URIHash VARCHAR(64),
		Description TEXT,
		CreatedAtInSec BIGINT,
		LastUpdateInSec BIGINT,
		Metadata TEXT,
		IdentityKey VARCHAR(255),
		NumberValue DOUBLE PRECISION
	)`)
	require.NoError(t, err)

	storeQuoted := NewArtifactStore(dbQuoted, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(artifactUUID1, nil), testDialectQuoted)
	artQuoted, err := storeQuoted.CreateArtifact(&model.Artifact{
		Namespace: "ns1",
		Type:      model.ArtifactType(apiv2.Artifact_Artifact),
		URI:       strPTR("s3://bucket/test-quoted"),
		Name:      "test-artifact-quoted",
	})
	require.NoError(t, err)

	upperQuotedID := "A23E4567-E89B-12D3-A456-426655441011"
	quotedBatch, err := storeQuoted.GetArtifactsByIDs([]string{upperQuotedID})
	require.NoError(t, err)
	assert.NotEmpty(t, quotedBatch, "expected mixed-case ID to match on quoted NOCASE column definition")
	assert.NotNil(t, quotedBatch[upperQuotedID])
	assert.Equal(t, artQuoted.UUID, quotedBatch[upperQuotedID].UUID)

	// 6. Collation comparison semantics (e.g. SQLite COLLATE RTRIM):
	// When table column has COLLATE RTRIM, trailing whitespace in request ID matches the stored UUID.
	// Both single-row lookup and batch lookup must succeed and return the map keyed by the requested spelling.
	dbRtrim, testDialectRtrim := NewFakeDBOrFatal()
	defer dbRtrim.Close()
	_, err = dbRtrim.Exec("DROP TABLE IF EXISTS " + artifactTableName)
	require.NoError(t, err)
	_, err = dbRtrim.Exec(`CREATE TABLE ` + artifactTableName + ` (
		UUID VARCHAR(255) COLLATE RTRIM NOT NULL PRIMARY KEY,
		Namespace VARCHAR(255),
		Name VARCHAR(255),
		Type INTEGER,
		URI VARCHAR(255),
		URIHash VARCHAR(64),
		Description TEXT,
		CreatedAtInSec BIGINT,
		LastUpdateInSec BIGINT,
		Metadata TEXT,
		IdentityKey VARCHAR(255),
		NumberValue DOUBLE PRECISION
	)`)
	require.NoError(t, err)

	storeRtrim := NewArtifactStore(dbRtrim, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(artifactUUID1, nil), testDialectRtrim)
	artRtrim, err := storeRtrim.CreateArtifact(&model.Artifact{
		Namespace: "ns1",
		Type:      model.ArtifactType(apiv2.Artifact_Artifact),
		URI:       strPTR("s3://bucket/test-rtrim"),
		Name:      "test-artifact-rtrim",
	})
	require.NoError(t, err)

	// Single lookup with trailing spaces matches under RTRIM
	var singleUUID string
	err = dbRtrim.QueryRow("SELECT UUID FROM "+artifactTableName+" WHERE UUID = ?", artRtrim.UUID+"   ").Scan(&singleUUID)
	require.NoError(t, err)
	assert.Equal(t, artRtrim.UUID, singleUUID)

	// Batch lookup with trailing spaces matches under RTRIM and keys result by requested spelling
	rtrimReqID := artRtrim.UUID + "   "
	rtrimBatch, err := storeRtrim.GetArtifactsByIDs([]string{rtrimReqID})
	require.NoError(t, err)
	assert.NotEmpty(t, rtrimBatch, "expected trailing-space ID to match under COLLATE RTRIM")
	assert.NotNil(t, rtrimBatch[rtrimReqID])
	assert.Equal(t, artRtrim.UUID, rtrimBatch[rtrimReqID].UUID)
}

func TestGetArtifactsByIDs_MoreThan500IDs(t *testing.T) {
	dbRtrim, testDialectRtrim := NewFakeDBOrFatal()
	defer dbRtrim.Close()
	_, err := dbRtrim.Exec("DROP TABLE IF EXISTS " + artifactTableName)
	require.NoError(t, err)
	_, err = dbRtrim.Exec(`CREATE TABLE ` + artifactTableName + ` (
		UUID VARCHAR(255) COLLATE RTRIM NOT NULL PRIMARY KEY,
		Namespace VARCHAR(255),
		Name VARCHAR(255),
		Type INTEGER,
		URI VARCHAR(255),
		URIHash VARCHAR(64),
		Description TEXT,
		CreatedAtInSec BIGINT,
		LastUpdateInSec BIGINT,
		Metadata TEXT,
		IdentityKey VARCHAR(255),
		NumberValue DOUBLE PRECISION
	)`)
	require.NoError(t, err)

	storeRtrim := NewArtifactStore(dbRtrim, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(artifactUUID1, nil), testDialectRtrim)
	artRtrim, err := storeRtrim.CreateArtifact(&model.Artifact{
		Namespace: "ns1",
		Type:      model.ArtifactType(apiv2.Artifact_Artifact),
		URI:       strPTR("s3://bucket/test-rtrim-501"),
		Name:      "test-artifact-rtrim-501",
	})
	require.NoError(t, err)

	// Construct at least 501 distinct IDs: 500 non-existent IDs, followed by the requested spelling past the 500th ID.
	// On SQLite, compound SELECT without batching fails at 501 terms with "too many terms in compound SELECT".
	requestedSpelling := artRtrim.UUID + "   "
	ids := make([]string, 505)
	for i := 0; i < 500; i++ {
		ids[i] = fmt.Sprintf("non-existent-artifact-%d", i)
	}
	ids[500] = requestedSpelling
	for i := 501; i < 505; i++ {
		ids[i] = fmt.Sprintf("non-existent-artifact-%d", i)
	}

	batch, err := storeRtrim.GetArtifactsByIDs(ids)
	require.NoError(t, err)
	assert.Equal(t, 1, len(batch))
	assert.NotNil(t, batch[requestedSpelling])
	assert.Equal(t, artRtrim.UUID, batch[requestedSpelling].UUID)
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
	fProto := &apiv2.Filter{Predicates: []*apiv2.Predicate{
		{Key: "type", Operation: apiv2.Predicate_EQUALS, Value: &apiv2.Predicate_IntValue{IntValue: 1}},
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
			Type:      model.ArtifactType(apiv2.Artifact_Model),
			Name:      "should-rollback",
		},
		&model.ArtifactTask{
			TaskID:      "task-id",
			RunUUID:     "run-id",
			Type:        model.IOType(apiv2.IOType_OUTPUT),
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
		return createArtifactTaskWithExecutor(tx.Exec, store.uuid, artifactTask, store.dbDialect)
	}

	_, _, err := store.CreateArtifactsWithTasks(
		[]*model.Artifact{
			{
				Namespace: "ns1",
				Type:      model.ArtifactType(apiv2.Artifact_Model),
				Name:      "first-artifact",
			},
			{
				Namespace: "ns1",
				Type:      model.ArtifactType(apiv2.Artifact_Model),
				Name:      "second-artifact",
			},
		},
		[]*model.ArtifactTask{
			{
				TaskID:      "task-id-1",
				RunUUID:     "run-id-1",
				Type:        model.IOType(apiv2.IOType_OUTPUT),
				ArtifactKey: "first-output",
				Producer: model.JSONData{
					"taskName": "task-1",
				},
			},
			{
				TaskID:      "task-id-2",
				RunUUID:     "run-id-2",
				Type:        model.IOType(apiv2.IOType_OUTPUT),
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
					Type:      model.ArtifactType(apiv2.Artifact_Model),
					URI:       strPTR(sharedURI),
					Name:      "shared-model",
					Metadata:  model.JSONData{"source": "importer"},
				},
				&model.ArtifactTask{
					TaskID:      fmt.Sprintf("task-%d", taskIndex),
					RunUUID:     fmt.Sprintf("run-%d", taskIndex),
					Type:        model.IOType(apiv2.IOType_OUTPUT),
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
			Type:      model.ArtifactType(apiv2.Artifact_Model),
			URI:       strPTR(sharedURI),
			Name:      "shared-model",
		},
		&model.ArtifactTask{
			TaskID:      "task-1",
			RunUUID:     "run-1",
			Type:        model.IOType(apiv2.IOType_OUTPUT),
			ArtifactKey: "artifact",
			Producer:    model.JSONData{"taskName": "importer-1"},
		},
	)
	require.NoError(t, err)

	store.uuid = util.NewFakeUUIDGeneratorOrFatal(artifactUUID2, nil)
	second, _, err := store.CreateArtifactWithTask(
		&model.Artifact{
			Namespace: "ns1",
			Type:      model.ArtifactType(apiv2.Artifact_Model),
			URI:       strPTR(sharedURI),
			Name:      "shared-model",
		},
		&model.ArtifactTask{
			TaskID:      "task-2",
			RunUUID:     "run-2",
			Type:        model.IOType(apiv2.IOType_OUTPUT),
			ArtifactKey: "artifact",
			Producer:    model.JSONData{"taskName": "importer-2"},
		},
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
		Type:      model.ArtifactType(apiv2.Artifact_Model),
		URI:       strPTR(sharedURI),
		Name:      "shared-model",
		Metadata:  model.JSONData{"source": "importer"},
	}
	link := &model.ArtifactTask{
		TaskID:      "task-1",
		RunUUID:     "run-1",
		Type:        model.IOType(apiv2.IOType_OUTPUT),
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
		Type:      model.ArtifactType(apiv2.Artifact_Model),
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
			Type:        model.IOType(apiv2.IOType_OUTPUT),
			ArtifactKey: "artifact",
			Producer:    model.JSONData{"taskName": "importer-1"},
		},
	)
	require.NoError(t, err)

	store.uuid = util.NewFakeUUIDGeneratorOrFatal(artifactUUID2, nil)
	reused, reusedLink, err := store.FindOrCreateArtifactWithTask(
		&model.Artifact{
			Namespace: "ns1",
			Type:      model.ArtifactType(apiv2.Artifact_Model),
			URI:       strPTR(sharedURI),
			Name:      "shared-model",
			Metadata:  model.JSONData{"source": "importer"},
		},
		&model.ArtifactTask{
			TaskID:      "task-2",
			RunUUID:     "run-2",
			Type:        model.IOType(apiv2.IOType_OUTPUT),
			ArtifactKey: "artifact",
			Producer:    model.JSONData{"taskName": "importer-2"},
		},
	)
	require.NoError(t, err)
	assert.Equal(t, existing.UUID, reused.UUID)
	assert.Equal(t, existing.UUID, reusedLink.ArtifactID)

	var artifactCount int
	row := db.QueryRow("SELECT count(*) FROM artifacts")
	require.NoError(t, row.Scan(&artifactCount))
	assert.Equal(t, 1, artifactCount)
}
