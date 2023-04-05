// Copyright 2021-2023 The Kubeflow Authors
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

package util

import (
	"encoding/json"

	structpb "github.com/golang/protobuf/ptypes/struct"
	"google.golang.org/protobuf/proto"
	"sigs.k8s.io/yaml"
)

// stablyMarshalJSON makes sure result is stable, so we can use it for snapshot
// testing.
func StablyMarshalJSON(msg proto.Message) (string, error) {
	unstableJSON, err := yaml.Marshal(msg)
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

// Converts a protobuf struct into a given protobuf message.
// Note, incompatible fields will be missing in the output protobuf message.
func ProtoStructToProtoMessage(s *structpb.Struct, m *proto.Message) error {
	structYaml, err := yaml.Marshal(s)
	if err != nil {
		return err
	}
	if err := yaml.Unmarshal(structYaml, *m); err != nil {
		return err
	}
	return nil
}

// Converts a protobuf message into a protobuf struct.
// Note, empty fields will be missing in the output struct.
func ProtoMessageToProtoStruct(m proto.Message) (*structpb.Struct, error) {
	messageYaml, err := yaml.Marshal(m)
	if err != nil {
		return nil, err
	}
	s := &structpb.Struct{}
	if err := yaml.Unmarshal(messageYaml, s); err != nil {
		return nil, err
	}
	return s, nil
}
