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
	"testing"
	"time"

	"github.com/golang/glog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

var namespace = flag.String("namespace", "ml-pipeline-test", "The namespace ml pipeline deployed to")
var initializeTimeout = flag.Duration("initializeTimeout", 5*time.Minute, "Duration to wait for test initialization")

type PackageSuite struct {
	suite.Suite
	namespace string
}

// Check the namespace have ML pipeline installed and ready
func (suite *PackageSuite) SetupTest() {
	if err := initTest(*namespace, *initializeTimeout); err != nil {
		glog.Exit("Failed to initialize test. Error: %s", err.Error())
	}
	suite.namespace = *namespace
}

func (suite *PackageSuite) TearDownTest() {
}

// TODO(yangpa) Implement the tests
func (suite *PackageSuite) TestListPipelines() {
	t := suite.T()
	clientSet, err := getKubernetesClient()
	if err != nil {
		t.Fatalf("Can't initialize a Kubernete client. Error: %s", err.Error())
	}

	response := clientSet.RESTClient().
		Get().AbsPath(fmt.Sprintf(mlPipelineBase, suite.namespace, "pipelines")).Do()
	var code int
	response.StatusCode(&code)
	assert.Equal(t, 200, code)
}

func TestPackageApis(t *testing.T) {
	suite.Run(t, new(PackageSuite))
}
