package main

import (
	"flag"
	"os"
	"os/exec"

	"github.com/golang/glog"
	publisherCmd "github.com/kubeflow/pipelines/backend/src/v2/publisher/cmd"
)

func main() {
	publisherCmd.InitFlags()
	publisherCmd.ValidateFlagsOrFatal()
	glog.Infof("Command and arguments: %v", flag.Args())
	cmd := exec.Command(flag.Args()[0], flag.Args()[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		publisherCmd.Publish()
		glog.Fatal(err)
	} else {
		publisherCmd.Publish()
	}
	glog.Flush()
}
