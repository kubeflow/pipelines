// Copyright 2021 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Main entry for launcher and driver command for Kubeflow Pipelines v2.
package main

import (
	"os"

	"github.com/spf13/cobra"
)

var (
	pipelineName      string
	runID             string
	mlmdServerAddress string
	mlmdServerPort    string
)

func main() {
	Execute()
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "runner",
	Short: "The single cmd tool for launcher and driver",
	Long: `Run command in the way as following:
	go run ./runner drive
	go run ./runner launch
	Check out drive.go and launch.go for corresponding subcommands' flags.
	`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&pipelineName, "pipeline_name", "", "pipeline context name")
	rootCmd.PersistentFlags().StringVar(&runID, "run_id", "", "pipeline run uid")
	rootCmd.PersistentFlags().StringVar(&mlmdServerAddress, "mlmd_server_address", "", "The MLMD gRPC server address.")
	rootCmd.PersistentFlags().StringVar(&mlmdServerPort, "mlmd_server_port", "8080", "The MLMD gRPC server port.")
}
