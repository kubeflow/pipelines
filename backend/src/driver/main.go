// Copyright 2021-2023 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"flag"
	"os/signal"
	"syscall"

	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/backend/src/driver/executorplugin"
)

func main() {
	logLevel := flag.String("log_level", "1", "The verbosity level to log.")
	serverPort := flag.String("server_port", ":8080", "Server port")
	// Use WARNING default logging level to facilitate troubleshooting.
	flag.Set("logtostderr", "true")
	flag.Set("stderrthreshold", "WARNING")
	flag.Parse()

	glog.Infof("Setting log level to: '%s'", *logLevel)
	if err := flag.Set("v", *logLevel); err != nil {
		glog.Warningf("Failed to set log level: %s", err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()
	if err := executorplugin.Serve(ctx, *serverPort, executorplugin.DefaultTokenPath); err != nil {
		glog.Exitf("Driver service failed: %v", err)
	}
}
