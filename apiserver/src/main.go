// Copyright 2018 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"flag"
	"fmt"

	"github.com/golang/glog"
	"github.com/kataras/iris"
)

var (
	portFlag   = flag.Int("port", 8888, "Port to listen on")
	configPath = flag.String("config", "", "Path to JSON file containing config")
)

func main() {
	flag.Parse()
	glog.Infof("starting web server")

	initConfig()

	clientManager := newClientManager()
	app := newApp(clientManager)

	app.Run(
		iris.Addr(fmt.Sprintf(":%d", *portFlag)),
		iris.WithOptimizations,
		// Post limit at 32MB.
		iris.WithPostMaxMemory(32<<20))

	clientManager.End()
}
