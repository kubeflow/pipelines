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

package storage

import (
	"encoding/json"
	"ml/apiserver/src/message/argo"
	"ml/apiserver/src/message/pipelinemanager"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
)

var ct, st, ft time.Time

var body []byte

type FakeArgoClient struct {
}

func init() {
	ct, _ = time.Parse(time.RFC1123Z, "2018-02-08T02:19:01-08:00")
	st, _ = time.Parse(time.RFC1123Z, "2018-02-08T02:19:01-08:00")
	ft, _ = time.Parse(time.RFC1123Z, "2018-02-08T02:19:01-08:00")
}

func (ac *FakeArgoClient) Request(method string, api string, requestBody []byte) ([]byte, error) {

	workflow := &argo.WorkflowList{
		Items: []argo.Workflow{
			{ObjectMeta: v1.ObjectMeta{
				Name:              "artifact-passing-5sd2d",
				CreationTimestamp: v1.Time{Time: ct}},
				Status: argo.WorkflowStatus{
					StartedAt:  v1.Time{Time: st},
					FinishedAt: v1.Time{Time: ft},
					Phase:      "Failed"}}}}
	body, _ = json.Marshal(workflow)
	return []byte(body), nil
}

func TestListJobs(t *testing.T) {
	store := &JobStore{
		argoClient: &FakeArgoClient{},
	}
	jobs, err := store.ListJobs()

	if err != nil {
		t.Errorf("Something wrong. Error %v", err)
	}
	if len(jobs) != 1 {
		t.Errorf("Error parsing jobs. Get %d jobs", len(jobs))
	}
	job, _ := json.Marshal(jobs[0])
	jobExpect, _ := json.Marshal(pipelinemanager.Job{
		Name:     "artifact-passing-5sd2d",
		CreateAt: &ct,
		StartAt:  &st,
		FinishAt: &ft,
		Status:   "Failed"})

	assert.Equal(t, job, jobExpect, "Unexpected Job parsed. Expect %v. Got %v", string(job), string(jobExpect))
}
