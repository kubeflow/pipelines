package util

import (
	"encoding/json"

	"github.com/golang/glog"
)

func UnmarshalOrFail(data string, v interface{}) {
	err := json.Unmarshal([]byte(data), v)
	if err != nil {
		glog.Fatalf("Failed to unmarshal the object: %v", data)
	}
}

func MarshalOrFail(v interface{}) []byte {
	bytes, err := json.Marshal(v)
	if err != nil {
		glog.Fatalf("Failed to marshal the object: %+v", v)
	}
	return bytes
}
