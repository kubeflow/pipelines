package main

import (
	"os"
	"fmt"
	
	"github.com/kubeflow/pipelines/cmd/pipelines/commands"
)

func main() {
	if err := commands.NewCommand().Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
