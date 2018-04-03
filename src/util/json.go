package util

import (
	"encoding/json"

	"github.com/golang/glog"
)

func MarshalOrFail(data string, v interface{}) {
	err := json.Unmarshal([]byte(data), v)
	if err != nil {
		glog.Fatalf("Failed to marshal the object.")
	}
}
