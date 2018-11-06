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
	rootCmd, _ := GetRealRootCommand()
	rootCmd.Execute()
}

func GetRealRootCommand() (*cmd.RootCommand, *cmd.ClientFactory) {
	clientFactory := cmd.NewClientFactory()
	rootCmd := cmd.NewRootCmd(clientFactory)
	rootCmd = cmd.CreateSubCommands(rootCmd, defaultPageSize)
	return rootCmd, clientFactory
}
