// Copyright 2023 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tektoncompiler

import (
	"encoding/json"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// stablyMarshalJSON makes sure result is stable, so we can use it for snapshot
// testing.
func stablyMarshalJSON(msg proto.Message) (string, error) {
	unstableJSON, err := protojson.Marshal(msg)
	if err != nil {
		return "", err
	}
	// This json unmarshal and marshal is to use encoding/json formatter to format the bytes[] returned by protojson
	// Do the json formatter because of https://developers.google.com/protocol-buffers/docs/reference/go/faq#unstable-json
	var v interface{}
	if err := json.Unmarshal(unstableJSON, &v); err != nil {
		return "", err
	}
	stableJSON, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(stableJSON), err
}
