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
