// Copyright 2025 The Kubeflow Authors
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

package proto_tests

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type caseOpts[T proto.Message] struct {
	message          T
	expectedPBPath   string
	expectedJSONPath string
}

func testOBJ[T proto.Message](t *testing.T, opts caseOpts[T]) {
	binaryPath := filepath.Join("testdata", opts.expectedPBPath)
	jsonPath := filepath.Join("testdata", opts.expectedJSONPath)

	// Serialize to binary
	binaryData, err := proto.Marshal(opts.message)
	if err != nil {
		t.Fatalf("Failed to marshal to binary: %v", err)
	}

	// Serialize to JSON
	jsonData, err := protojson.MarshalOptions{
		Multiline:       true,
		Indent:          "  ",
		EmitUnpopulated: true,
		// emulate marshaling options that we use
		// for grpc-gateway json marshaling
		UseProtoNames: true,
	}.Marshal(opts.message)
	if err != nil {
		t.Fatalf("Failed to marshal to JSON: %v", err)
	}

	if os.Getenv("UPDATE_EXPECTED") == "true" {
		// Create directories if they don't exist
		if err := os.MkdirAll(filepath.Dir(binaryPath), 0755); err != nil {
			t.Fatalf("Failed to create directory for binary file: %v", err)
		}
		if err := os.MkdirAll(filepath.Dir(jsonPath), 0755); err != nil {
			t.Fatalf("Failed to create directory for JSON file: %v", err)
		}
		if err := os.WriteFile(binaryPath, binaryData, 0644); err != nil {
			t.Fatalf("Failed to update expected binary file: %v", err)
		}
		if err := os.WriteFile(jsonPath, jsonData, 0644); err != nil {
			t.Fatalf("Failed to update expected JSON file: %v", err)
		}
		t.Log("expected files updated.")
		return
	}

	// Validate binary round-trip
	expectedBinary, err := os.ReadFile(binaryPath)
	if err != nil {
		t.Fatalf("Failed to read expected binary: %v", err)
	}
	// Create a new instance of the concrete type T to unmarshal into.
	// Proto.Clone allows us to get a new zero-value instance of a
	// proto.Message. It returns a proto.Message, which proto.Unmarshal
	// expects.
	expectedMsg := proto.Clone(opts.message)
	if err := proto.Unmarshal(expectedBinary, expectedMsg); err != nil {
		t.Fatalf("Failed to unmarshal expected binary: %v", err)
	}
	if !proto.Equal(opts.message, expectedMsg) {
		t.Errorf("Binary round-trip mismatch with expected file")
	}

	// Read expected JSON
	expectedJSON, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("Failed to read expected JSON file: %v", err)
	}

	// Unmarshal both to interface{} for comparison
	var actualObj, expectedObj interface{}
	if err := json.Unmarshal(jsonData, &actualObj); err != nil {
		t.Fatalf("Failed to unmarshal actual JSON: %v", err)
	}
	if err := json.Unmarshal(expectedJSON, &expectedObj); err != nil {
		t.Fatalf("Failed to unmarshal expected JSON: %v", err)
	}

	if diff := cmp.Diff(expectedObj, actualObj); diff != "" {
		t.Errorf("JSON output mismatch (-expected +current):\n%s", diff)
	}
}
