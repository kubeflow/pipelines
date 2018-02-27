package storage

import (
	"encoding/json"
	"ml/apiserver/src/message/argo"
	"ml/apiserver/src/message/pipelinemanager"
	"reflect"
	"testing"
	"time"

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

func (ac *FakeArgoClient) Request(method string, api string) ([]byte, error) {

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

	if !reflect.DeepEqual(job, jobExpect) {
		t.Errorf("Unexpecte Job parsed. Expect %v. Got %v", string(job), string(jobExpect))
	}
}
