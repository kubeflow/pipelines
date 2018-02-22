package dao

import (
	"reflect"
	"testing"

	"ml/apiserver/src/message/pipelinemanager"
)

type FakeArgoClient struct {
}

func (ac *FakeArgoClient) Request(method string, api string) ([]byte, error) {
	body := "{\"items\":[{\"metadata\":{\"creationTimestamp\":\"2018-02-08T02:19:01Z\",\"name\":\"artifact-passing-5sd2d\"},\"status\":{\"finishedAt\":\"2018-02-08T02:19:04Z\",\"message\":\"step group artifact-passing-5sd2d[0] was unsuccessful\",\"nodes\":{\"artifact-passing-5sd2d\":{\"children\":[\"artifact-passing-5sd2d-4291870024\"],\"finishedAt\":\"2018-02-08T02:19:04Z\",\"id\":\"artifact-passing-5sd2d\",\"message\":\"step group artifact-passing-5sd2d[0] was unsuccessful\",\"name\":\"artifact-passing-5sd2d\",\"phase\":\"Failed\",\"startedAt\":\"2018-02-08T02:19:01Z\"}},\"phase\":\"Failed\",\"startedAt\":\"2018-02-08T02:19:01Z\"}}]}"
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
