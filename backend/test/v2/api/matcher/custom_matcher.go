// Copyright 2018-2023 The Kubeflow Authors
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
package matcher

import (
	"fmt"

	"github.com/kubeflow/pipelines/backend/test/logger"

	"github.com/google/go-cmp/cmp"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

// MatchMaps - Iterate over 2 maps and compare if they are same or not
//
// param mapType - string value to append to the assertion error message
func MatchMaps(actual interface{}, expected interface{}, mapType string) {
	ginkgo.GinkgoHelper()
	logger.Log("Comparing maps of %s", mapType)
	diff := cmp.Diff(actual, expected)
	gomega.Expect(diff).To(gomega.BeEmpty(), fmt.Sprintf("%s maps are not same, actual diff:\n%s", mapType, diff))
}
