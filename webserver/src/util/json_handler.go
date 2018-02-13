package util

import (
	"encoding/json"
	"net/http"
)

// HandleJSONPayload writes payload to http response with JSON format.
func HandleJSONPayload(w http.ResponseWriter, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		HandleError(w, err)
		return
	}
}
