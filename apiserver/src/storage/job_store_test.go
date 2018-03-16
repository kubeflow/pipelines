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
	"ml/apiserver/src/util"
	"testing"
	"time"

	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/kataras/iris/core/errors"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
)

var ct, st, ft time.Time

var body []byte

type FakeWorkflowClient struct {
}

func (FakeWorkflowClient) List(opts v1.ListOptions) (*v1alpha1.WorkflowList, error) {
	return &v1alpha1.WorkflowList{
		Items: []v1alpha1.Workflow{
			{ObjectMeta: v1.ObjectMeta{
				Name:              "artifact-passing-5sd2d",
				CreationTimestamp: v1.Time{Time: ct}},
				Status: v1alpha1.WorkflowStatus{
					StartedAt:  v1.Time{Time: st},
					FinishedAt: v1.Time{Time: ft},
					Phase:      "Failed"}}}}, nil
}

func (FakeWorkflowClient) Create(*v1alpha1.Workflow) (*v1alpha1.Workflow, error) {
	return &v1alpha1.Workflow{ObjectMeta: v1.ObjectMeta{
		Name:              "artifact-passing-abcd",
		CreationTimestamp: v1.Time{Time: ct}},
		Status: v1alpha1.WorkflowStatus{
			StartedAt:  v1.Time{Time: st},
			FinishedAt: v1.Time{Time: ft},
			Phase:      "Pending"}}, nil
}

func (FakeWorkflowClient) Get(name string, options v1.GetOptions) (*v1alpha1.Workflow, error) {
	return &v1alpha1.Workflow{ObjectMeta: v1.ObjectMeta{
		Name:              "artifact-passing-xyz",
		CreationTimestamp: v1.Time{Time: ct}},
		Status: v1alpha1.WorkflowStatus{
			StartedAt:  v1.Time{Time: st},
			FinishedAt: v1.Time{Time: ft},
			Phase:      "Pending"}}, nil
}

func (FakeWorkflowClient) Update(*v1alpha1.Workflow) (*v1alpha1.Workflow, error) {
	panic("implement me")
}

func (FakeWorkflowClient) Delete(name string, options *v1.DeleteOptions) error {
	panic("implement me")
}

func (FakeWorkflowClient) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	panic("implement me")
}

func (FakeWorkflowClient) Watch(opts v1.ListOptions) (watch.Interface, error) {
	panic("implement me")
}

func (FakeWorkflowClient) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.Workflow, err error) {
	panic("implement me")
}

type FakeBadWorkflowClient struct {
	FakeWorkflowClient
}

func (FakeBadWorkflowClient) List(opts v1.ListOptions) (*v1alpha1.WorkflowList, error) {
	return nil, errors.New("some error")
}

func (FakeBadWorkflowClient) Create(*v1alpha1.Workflow) (*v1alpha1.Workflow, error) {
	return nil, errors.New("some error")
}

func (FakeBadWorkflowClient) Get(name string, options v1.GetOptions) (*v1alpha1.Workflow, error) {
	return nil, errors.New("some error")
}

func init() {
	ct, _ = time.Parse(time.RFC1123Z, "2018-02-08T02:19:01-08:00")
	st, _ = time.Parse(time.RFC1123Z, "2018-02-08T02:19:01-08:00")
	ft, _ = time.Parse(time.RFC1123Z, "2018-02-08T02:19:01-08:00")
}

func TestListJobs(t *testing.T) {
	store := &JobStore{
		wfClient: &FakeWorkflowClient{},
	}
	jobs, err := store.ListJobs()

	if err != nil {
		t.Errorf("Something wrong. Error %v", err)
	}
	if len(jobs) != 1 {
		t.Errorf("Error parsing jobs. Get %d jobs", len(jobs))
	}
	job := &jobs[0]
	jobExpect := &v1alpha1.Workflow{ObjectMeta: v1.ObjectMeta{
		Name:              "artifact-passing-5sd2d",
		CreationTimestamp: v1.Time{Time: ct}},
		Status: v1alpha1.WorkflowStatus{
			StartedAt:  v1.Time{Time: st},
			FinishedAt: v1.Time{Time: ft},
			Phase:      "Failed"}}

	assert.Equal(t, job, jobExpect, "Unexpected Job parsed. Expect %v. Got %v", job, jobExpect)
}

func TestListJobsError(t *testing.T) {
	store := &JobStore{
		wfClient: &FakeBadWorkflowClient{},
	}
	_, err := store.ListJobs()
	assert.IsType(t, new(util.InternalError), err, "expect to throw an internal error")
}

func TestCreateJob(t *testing.T) {
	store := &JobStore{
		wfClient: &FakeWorkflowClient{},
	}
	job, err := store.CreateJob(&v1alpha1.Workflow{})

	if err != nil {
		t.Errorf("Something wrong. Error %v", err)
	}
	jobExpect := &v1alpha1.Workflow{ObjectMeta: v1.ObjectMeta{
		Name:              "artifact-passing-abcd",
		CreationTimestamp: v1.Time{Time: ct}},
		Status: v1alpha1.WorkflowStatus{
			StartedAt:  v1.Time{Time: st},
			FinishedAt: v1.Time{Time: ft},
			Phase:      "Pending"}}

	assert.Equal(t, job, jobExpect, "Unexpected Job parsed. Expect %v. Got %v", job, jobExpect)
}

func TestCreateJobError(t *testing.T) {
	store := &JobStore{
		wfClient: &FakeBadWorkflowClient{},
	}
	_, err := store.CreateJob(&v1alpha1.Workflow{})
	assert.IsType(t, new(util.InternalError), err, "expect to throw an internal error")
}

func TestGetJob(t *testing.T) {
	store := &JobStore{
		wfClient: &FakeWorkflowClient{},
	}
	job, err := store.GetJob("job1")

	if err != nil {
		t.Errorf("Something wrong. Error %v", err)
	}
	jobExpect := &v1alpha1.Workflow{ObjectMeta: v1.ObjectMeta{
		Name:              "artifact-passing-xyz",
		CreationTimestamp: v1.Time{Time: ct}},
		Status: v1alpha1.WorkflowStatus{
			StartedAt:  v1.Time{Time: st},
			FinishedAt: v1.Time{Time: ft},
			Phase:      "Pending"}}

	assert.Equal(t, job, jobExpect, "Unexpected Job parsed. Expect %v. Got %v", job, jobExpect)
}

func TestGetJobError(t *testing.T) {
	store := &JobStore{
		wfClient: &FakeBadWorkflowClient{},
	}
	_, err := store.GetJob("job1")
	assert.IsType(t, new(util.InternalError), err, "expect to throw an internal error")
}
