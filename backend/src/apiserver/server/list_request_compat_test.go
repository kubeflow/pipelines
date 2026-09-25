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

package server

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"strings"
	"testing"

	sq "github.com/Masterminds/squirrel"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/stretchr/testify/require"
)

type legacyFilterToken struct {
	Name    string          `json:"name"`
	Version string          `json:"version"`
	Filter  json.RawMessage `json:"filter"`
	Token   string          `json:"token"`
}

func legacyFilterTokens(t *testing.T) []legacyFilterToken {
	t.Helper()
	data, err := os.ReadFile("testdata/filtered_tokens_2_17_2.json")
	require.NoError(t, err)
	var fixtures []legacyFilterToken
	require.NoError(t, json.Unmarshal(data, &fixtures))
	v2Fixtures := make([]legacyFilterToken, 0, len(fixtures))
	for _, fixture := range fixtures {
		if fixture.Version == "v2" {
			v2Fixtures = append(v2Fixtures, fixture)
		}
	}
	require.NotEmpty(t, v2Fixtures)
	return v2Fixtures
}

func TestValidatedListOptions_LegacyFilteredTokens(t *testing.T) {
	for _, fixture := range legacyFilterTokens(t) {
		t.Run(fixture.Name, func(t *testing.T) {
			fresh, err := validatedListOptions(&model.Experiment{}, "", 10, "", string(fixture.Filter))
			require.NoError(t, err)
			for _, repeat := range []bool{false, true} {
				// Numeric filter comparison already fails on 2.17.2 (JSON float64 vs
				// protobuf int64). This change deliberately preserves strict comparison.
				if repeat && strings.HasSuffix(fixture.Name, "GREATER_THAN") {
					continue
				}
				spec := ""
				if repeat {
					spec = string(fixture.Filter)
				}
				restored, err := validatedListOptions(&model.Experiment{}, fixture.Token, 10, "", spec)
				require.NoError(t, err, "repeat criteria: %v", repeat)
				// Both token-only and repeated-filter requests use current, server-derived
				// filtering semantics. Cursor values are preserved independently.
				quote := func(s string) string { return "`" + s + "`" }
				wantSQL, wantArgs, err := fresh.AddFilterToSelect(sq.Select("*").From("experiments"), quote).ToSql()
				require.NoError(t, err)
				gotSQL, gotArgs, err := restored.AddFilterToSelect(sq.Select("*").From("experiments"), quote).ToSql()
				require.NoError(t, err)
				require.Equal(t, wantSQL, gotSQL)
				wantJSON, err := json.Marshal(wantArgs)
				require.NoError(t, err)
				gotJSON, err := json.Marshal(gotArgs)
				require.NoError(t, err)
				require.JSONEq(t, string(wantJSON), string(gotJSON))
				require.Equal(t, "experiment-next-page", restored.KeyFieldValue)
				require.Equal(t, float64(1700000100), restored.GetSortByFieldValue())
			}
		})
	}
}

func TestValidatedListOptions_LegacyFilterRejectsChangedCriteria(t *testing.T) {
	fixture := legacyFilterTokens(t)[0]
	for _, spec := range []string{
		`{"predicates":[{"key":"name","operation":"EQUALS","string_value":"other"}]}`,
		`{"predicates":[{"key":"name","operation":"NOT_EQUALS","string_value":"alpha"}]}`,
		`{"predicates":[{"key":"description","operation":"EQUALS","string_value":"alpha"}]}`,
	} {
		_, err := validatedListOptions(&model.Experiment{}, fixture.Token, 10, "", spec)
		require.ErrorContains(t, err, "does not match")
	}
	_, err := validatedListOptions(&model.Experiment{}, fixture.Token, 10, "created_at desc", string(fixture.Filter))
	require.ErrorContains(t, err, "does not match")
	_, err = validatedListOptions(nil, fixture.Token, 10, "", "")
	require.ErrorContains(t, err, "valid type")
}

func TestValidatedListOptions_TokenFilterMetadataIsNotAuthoritative(t *testing.T) {
	fixture := legacyFilterTokens(t)[0]
	decoded, err := base64.StdEncoding.DecodeString(fixture.Token)
	require.NoError(t, err)
	var token map[string]interface{}
	require.NoError(t, json.Unmarshal(decoded, &token))
	tokenFilter := token["Filter"].(map[string]interface{})
	tokenFilter["CaseInsensitiveKeys"] = map[string]interface{}{"experiments.UUID": map[string]interface{}{}}
	encode := func() string {
		data, err := json.Marshal(token)
		require.NoError(t, err)
		return base64.StdEncoding.EncodeToString(data)
	}
	for _, spec := range []string{"", string(fixture.Filter)} {
		opts, err := validatedListOptions(&model.Experiment{}, encode(), 10, "", spec)
		require.NoError(t, err)
		encoded, err := json.Marshal(opts.Filter)
		require.NoError(t, err)
		require.Contains(t, string(encoded), `"experiments.Name":{}`)
		require.NotContains(t, string(encoded), `"experiments.UUID":{}`)
	}
	tokenFilter["EQ"] = map[string]interface{}{"experiments.Name;DROP TABLE experiments": []string{"alpha"}}
	_, err = validatedListOptions(&model.Experiment{}, encode(), 10, "", "")
	require.ErrorContains(t, err, "filter key")
}
