//go:build ignore

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

package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"

	sq "github.com/Masterminds/squirrel"
	v1 "github.com/kubeflow/pipelines/backend/api/v1beta1/go_client"
	v2 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/filter"
	"github.com/kubeflow/pipelines/backend/src/apiserver/list"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"google.golang.org/protobuf/encoding/protojson"
)

type Fixture struct {
	Name    string          `json:"name"`
	Version string          `json:"version"`
	Filter  json.RawMessage `json:"filter"`
	Token   string          `json:"token"`
	Decoded json.RawMessage `json:"decoded"`
	Matches bool            `json:"matches"`
	SQL     string          `json:"sql"`
	Args    []interface{}   `json:"args"`
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
func opts(version string, raw []byte) *list.Options {
	var p interface{}
	if version == "v1" {
		f := &v1.Filter{}
		must(protojson.Unmarshal(raw, f))
		p = f
	} else {
		f := &v2.Filter{}
		must(protojson.Unmarshal(raw, f))
		p = f
	}
	e := &model.Experiment{}
	f, err := filter.New(p)
	must(err)
	o, err := list.NewOptions(e, 10, "", f)
	must(err)
	return o
}
func main() {
	var fixtures []Fixture
	if len(os.Args) > 1 {
		b, err := os.ReadFile(os.Args[1])
		must(err)
		must(json.Unmarshal(b, &fixtures))
	} else {
		for _, version := range []string{"v1", "v2"} {
			key := "name"
			if version == "v2" {
				key = "display_name"
			}
			for _, op := range []string{"EQUALS", "NOT_EQUALS", "IN", "IS_SUBSTRING", "GREATER_THAN"} {
				value := `"string_value":"alpha"`
				k := key
				if op == "IN" {
					value = `"string_values":{"values":["alpha","beta"]}`
				}
				if op == "GREATER_THAN" {
					k = "created_at"
					value = `"long_value":"1700000000"`
				}
				opKey := "op"
				if version == "v2" {
					opKey = "operation"
				}
				raw := []byte(fmt.Sprintf(`{"predicates":[{"key":%q,%q:%q,%s}]}`, k, opKey, op, value))
				o := opts(version, raw)
				token, err := o.NextPageToken(&model.Experiment{UUID: "experiment-next-page", Name: "alpha", CreatedAtInSec: 1700000100})
				must(err)
				decoded, err := base64.StdEncoding.DecodeString(token)
				must(err)
				fixtures = append(fixtures, Fixture{Name: version + "_" + op, Version: version, Filter: raw, Token: token, Decoded: decoded})
			}
		}
	}
	for i := range fixtures {
		f := &fixtures[i]
		o := opts(f.Version, f.Filter)
		token, err := list.NewOptionsFromToken(f.Token, 10)
		must(err)
		f.Matches = token.Matches(o)
		f.SQL, f.Args, err = token.AddFilterToSelect(token.AddPaginationToSelect(token.AddSortingToSelect(sq.Select("*").From("experiments")))).ToSql()
		must(err)
	}
	out, err := json.MarshalIndent(fixtures, "", "  ")
	must(err)
	fmt.Println(string(out))
}
