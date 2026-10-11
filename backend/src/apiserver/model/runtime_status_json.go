// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0

package model

import (
	"bytes"
	"encoding/json"

	statuspb "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
)

type runtimeStatusJSON struct {
	UpdateTimeInSec int64           `json:"UpdateTimeInSec,omitempty"`
	State           RuntimeState    `json:"State,omitempty"`
	Error           json.RawMessage `json:"Error,omitempty"`
}

// MarshalJSON persists the RPC status instead of the unexported fields of a Go
// error. The regular protobuf JSON tags retain opaque Any detail bytes even when
// their message types are not linked into the API server.
func (s RuntimeStatus) MarshalJSON() ([]byte, error) {
	value := runtimeStatusJSON{UpdateTimeInSec: s.UpdateTimeInSec, State: s.State}
	if s.Error != nil {
		data, err := json.Marshal(status.Convert(s.Error).Proto())
		if err != nil {
			return nil, err
		}
		value.Error = data
	}
	return json.Marshal(value)
}

// UnmarshalJSON makes persisted history readable without decoding JSON directly
// into an error interface, while preserving RPC codes, messages and details.
func (s *RuntimeStatus) UnmarshalJSON(data []byte) error {
	var value runtimeStatusJSON
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&value); err != nil {
		return err
	}
	result := RuntimeStatus{UpdateTimeInSec: value.UpdateTimeInSec, State: value.State}
	raw := bytes.TrimSpace(value.Error)
	if len(raw) != 0 && !bytes.Equal(raw, []byte("null")) {
		rpc := &statuspb.Status{}
		decoder = json.NewDecoder(bytes.NewReader(raw))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(rpc); err != nil {
			// Also accept the standard protobuf JSON representation of known Any types.
			if err = protojson.Unmarshal(raw, rpc); err != nil {
				return err
			}
		}
		// Older error implementations serialized as {} and retained no information.
		if rpc.Code != 0 || rpc.Message != "" || len(rpc.Details) > 0 {
			result.Error = &persistedRuntimeError{value: rpc}
		}
	}
	*s = result
	return nil
}

// A wrapper also preserves a nonempty persisted status with code OK. grpc's
// Status.Err returns nil for that code and would otherwise discard its fields.
type persistedRuntimeError struct{ value *statuspb.Status }

func (e *persistedRuntimeError) Error() string              { return e.GRPCStatus().String() }
func (e *persistedRuntimeError) GRPCStatus() *status.Status { return status.FromProto(e.value) }
