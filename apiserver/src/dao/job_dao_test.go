package dao

import (
	"reflect"
	"testing"

	"ml/apiserver/src/message/pipelinemanager"
	"ml/apiserver/src/message/argo"
	"encoding/json"
)

type FakeArgoClient struct {
}

func (ac *FakeArgoClient) Request(method string, api string) ([]byte, error) {
	workflow := &argo.Workflows{
		Items: []argo.Workflow{
			{Metadata: argo.WorkflowMetadata{Name: "artifact-passing-5sd2d", CreationTimestamp: "2018-02-08T02:19:01Z"},
				Status: argo.WorkflowStatus{StartTimestamp: "2018-02-08T02:19:01Z", FinishTimestamp: "2018-02-08T02:19:04Z", Status: "Failed"}}}}
	body, _ := json.Marshal(workflow)
	return []byte(body), nil
}

func TestListJobs(t *testing.T) {
	dao := &JobDao{
		argoClient: &FakeArgoClient{},
	}
	jobs, err := dao.ListJobs()

	if err != nil {
		t.Errorf("Something wrong. Error %v", err)
	}
	if len(jobs) != 1 {
		t.Errorf("Error parsing jobs. Get %d jobs", len(jobs))
	}
	job := jobs[0]
	jobExpect := pipelinemanager.Job{Name: "artifact-passing-5sd2d", CreationTimestamp: "2018-02-08T02:19:01Z",
		StartTimestamp: "2018-02-08T02:19:01Z", FinishTimestamp: "2018-02-08T02:19:04Z", Status: "Failed"}
	if !reflect.DeepEqual(job, jobExpect) {
		t.Errorf("Unexpecte Job parsed. Expect %v. Got %v", job, jobExpect)
	}
}
