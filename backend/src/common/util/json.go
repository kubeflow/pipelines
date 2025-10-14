// Copyright 2018 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package util

import (
	"google.golang.org/protobuf/encoding/protojson"

	"encoding/json"

	"github.com/golang/glog"
	"google.golang.org/protobuf/proto"
)

func UnmarshalJsonOrFail(data string, v interface{}) {
	err := json.Unmarshal([]byte(data), v)
	if err != nil {
		glog.Fatalf("Failed to unmarshal the object: %v", data)
	}
}

func MarshalJsonOrFail(v interface{}) []byte {
	bytes, err := json.Marshal(v)
	if err != nil {
		glog.Fatalf("Failed to marshal the object: %+v", v)
	}
	return bytes
}

// Converts an object into []byte array
func MarshalJsonWithError(v interface{}) ([]byte, error) {
	bytes, err := json.Marshal(v)
	if err != nil {
		return nil, Wrapf(err, "Failed to marshal the object: %+v", v)
	}
	return bytes, nil
}

// Converts a []byte array into an interface
func UnmarshalJsonWithError(data interface{}, v *interface{}) error {
	var bytes []byte
	switch data := data.(type) {
	case string:
		bytes = []byte(data)
	case []byte:
		bytes = data
	case *[]byte:
		bytes = *data
	default:
		return NewInvalidInputError("Unmarshalling %T is not implemented.", data)
	}
	err := json.Unmarshal(bytes, v)
	if err != nil {
		return Wrapf(err, "Failed to unmarshal the object: %+v", v)
	}
	return nil
}

// UnmarshalString unmarshals a JSON object from s into m.
// Allows unknown fields
func UnmarshalString(s string, m proto.Message) error {
	unmarshaler := protojson.UnmarshalOptions{
		DiscardUnknown: true,
	}
	return unmarshaler.Unmarshal([]byte(s), m)
}
