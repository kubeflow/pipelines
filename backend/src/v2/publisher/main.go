package main

import (
	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/backend/src/v2/publisher/cmd"
)

func main() {
	cmd.InitFlags()
	cmd.ValidateFlagsOrFatal()
	err := cmd.Publish()
	if err != nil {
		glog.Fatal(err)
	}
	glog.Flush()
}
