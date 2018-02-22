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

func TestListRuns(t *testing.T) {
	dao := &RunDao{
		argoClient: &FakeArgoClient{},
	}
	runs, err := dao.ListRuns()

	if err != nil {
		t.Errorf("Something wrong. Error %v", err)
	}
	if len(runs) != 1 {
		t.Errorf("Error parsing runs. Get %d runs", len(runs))
	}
	run := runs[0]
	runExpect := pipelinemanager.Run{Name: "artifact-passing-5sd2d", CreationTimestamp: "2018-02-08T02:19:01Z",
		StartTimestamp: "2018-02-08T02:19:01Z", FinishTimestamp: "2018-02-08T02:19:04Z", Status: "Failed"}
	if !reflect.DeepEqual(run, runExpect) {
		t.Errorf("Unexpecte Run parsed. Expect %v. Got %v", run, runExpect)
	}
}
