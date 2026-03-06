package mlflow

import (
	"context"
	"encoding/json"
)

type TaskInfo struct {
	TrackingUri string
	Workspace   string
	ParentRunId string
	AuthType    string
	Creds       map[string]string
	RunId       string
}

type TaskStartResult struct {
	RunId string `json:"runId"`
}

type PluginConfig struct {
	Endpoint string          `json:"endpoint"`
	Timeout  string          `json:"timeout,omitempty"` // e.g. "30s"; defaults to "30s"
	TLS      *TLSConfig      `json:"tls,omitempty"`
	Settings json.RawMessage `json:"settings,omitempty"`
}

type TLSConfig struct {
	InsecureSkipVerify bool   `json:"insecureSkipVerify,omitempty"` // Skip TLS certificate verification
	CABundlePath       string `json:"caBundlePath,omitempty"`       // Path to a custom CA bundle
}

type Run struct {
	ExperimentID string `json:"experiment_id"`
	RunName      string `json:"run_name"`
	StartTime    int64  `json:"start_time"`
	//ToDo: the below may not be necessary
	Tags map[string]string `json:"tags"`
}

type RunInfo struct {
	RunID        string    `json:"run_id"`
	RunName      string    `json:"run_name"`
	ExperimentID string    `json:"experiment_id"`
	Status       RunStatus `json:"status"`
	StartTime    int64     `json:"start_time"`
	EndTime      int64     `json:"end_time"`
	//todo: more fields may be necessary here. see https://mlflow.org/docs/latest/api_reference/rest-api.html#mlflowruninfo
}

type RunResponse struct {
	Info RunInfo `json:"info"`
	//todo: more fields may be necessary here. see https://mlflow.org/docs/latest/api_reference/rest-api.html#mlflowrun
}

type RunUpdate struct {
	RunID   string    `json:"run_id"`
	Status  RunStatus `json:"status"`
	EndTime int64     `json:"end_time"`
	RunName string    `json:"run_name"`
}

type RunStatus int

const (
	RunStatus_STATE_UNSPECIFIED RunStatus = iota
	RunStatus_OK
	RunStatus_ERROR
	RunStatus_IN_PROGRESS
)

func OnTaskStart(ctx context.Context, taskInfo TaskInfo) (*TaskStartResult, error) {
	client, err := NewClient(taskInfo)
	if err != nil {
		return nil, err
	}
	return &TaskStartResult{RunId: client.CreateRun()}, nil
}

// Mark run as complete or failed.
func OnTaskEnd(ctx context.Context, taskInfo TaskInfo, metrics map[string]float64, params map[string]string) error {
	client, err := NewClient(taskInfo)
	if err != nil {
		return err
	}

	if err = client.UpdateRun(RunUpdate{
		RunID:   params["run_id"],
		Status:  RunStatus(metrics["status"]),
		EndTime: int64(metrics["end_time"]),
		RunName: params["run_name"],
	}); err != nil {
		return err
	}
	return nil
}
