package util

import (
	"net/http"

	"github.com/golang/glog"
)

// CheckErr checks if err is null and exit if not.
func CheckErr(err error) {
	if err != nil {
		glog.Fatalf("%v", err)
	}
}

// HandleError handles error statuses for external requests.
func HandleError(w http.ResponseWriter, err error) {
	// TODO(yangpa): Error handling by status enum.
	http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
}
