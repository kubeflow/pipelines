// Copyright 2021 Google LLC
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

package component

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"google.golang.org/protobuf/testing/protocmp"
)

func Test_parseRuntimeInfo(t *testing.T) {
	tests := []struct {
		name        string
		jsonEncoded string
		want        *runtimeInfo
		wantErr     bool
	}{
		{
			name: "Parses InputParameters Correctly",
			jsonEncoded: `{
				"inputParameters": {
					"my_param": {
						"parameterType": "INT",
						"parameterValue": "123"
					}
				}
			}`,
			want: &runtimeInfo{
				InputParameters: map[string]*inputParameter{
					"my_param": {
						ParameterType:  "INT",
						ParameterValue: "123",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Parses OutputParameters Correctly",
			jsonEncoded: `{
				"outputParameters": {
					"my_param": {
						"parameterType": "INT",
						"fileOutputPath": "/tmp/outputs/my_param/data"
					}
				}
			}`,
			want: &runtimeInfo{
				OutputParameters: map[string]*outputParameter{
					"my_param": {
						ParameterType:  "INT",
						FileOutputPath: "/tmp/outputs/my_param/data",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Parses OutputArtifacts Correctly",
			jsonEncoded: `{
				"outputArtifacts": {
					"my_artifact": {
						"artifactSchema": "properties:\ntitle: kfp.Dataset\ntype: object\n",
					  "fileOutputPath": "/tmp/outputs/my_artifact/data"
					}
				}
			}`,
			want: &runtimeInfo{
				OutputArtifacts: map[string]*outputArtifact{
					"my_artifact": {
						ArtifactSchema: "properties:\ntitle: kfp.Dataset\ntype: object\n",
						FileOutputPath: "/tmp/outputs/my_artifact/data",
					},
				},
			},
			wantErr: false,
		},
		// TODO add tests for input params, artifacts.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseRuntimeInfo(tt.jsonEncoded)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseRuntimeInfo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if diff := cmp.Diff(tt.want, got, cmpopts.EquateEmpty(), protocmp.Transform()); diff != "" {
				t.Errorf("parseRuntimeInfo() = %+v, want %+v\nDiff (-want, +got)\n%s", got, tt.want, diff)
				s, _ := json.MarshalIndent(tt.want, "", "  ")
				fmt.Printf("Want %s", s)
			}

		})
	}
}
