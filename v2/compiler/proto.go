package compiler

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
