package main

import (
	"flag"

	"github.com/kubeflow/pipelines/backend/src/cmd/ml/cmd"
)

const (
	defaultPageSize = int32(10)
)

func main() {
	flag.Parse()
	clientFactory := cmd.NewClientFactory()
	rootCmd := cmd.NewRootCmd(clientFactory)
	rootCmd = cmd.CreateSubCommands(rootCmd, defaultPageSize)
	rootCmd.Execute()
}
