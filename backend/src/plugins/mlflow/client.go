package mlflow

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/backend/src/common/util"
)

type ExecutionType string

const (
	ContainerExecutionTypeName ExecutionType = "system.ContainerExecution"
	DagExecutionTypeName       ExecutionType = "system.DAGExecution"
)

const (
	basePath = "/api/2.0/mlflow/"
)

type Client struct {
	client  *http.Client
	baseURL string
}

const (
	BearerAuth     = "bearer"
	BasicAuth      = "basic-auth"
	KubernetesAuth = "kubernetes"
)

func NewClient(info TaskInfo) (*Client, error) {

	scheme := "http"
	var httpClient *http.Client
	httpClient = http.DefaultClient
	//todo: is this health check necessary?
	healthURL := "placeholder-url-for-now"

	placeholderTimeout := 10 * time.Second
	err := util.WaitForAPIAvailable(placeholderTimeout, healthURL, httpClient)
	if err != nil {
		fmt.Println("Failed to initialize mlflow client. Error: %s", err.Error())
	}
	httpBaseURL := fmt.Sprintf("%s://%s:%s", scheme, info.TrackingUri+"/"+info.Workspace, basePath)

	return &Client{
		client:  httpClient,
		baseURL: httpBaseURL,
	}, nil
}

func (c *Client) CreateRun() string {
	run := Run{
		RunName:   "",
		StartTime: 0,
	}
	jsonData, err := json.Marshal(run)
	if err != nil {
		glog.Errorf("Failed to marshal mlflow run: %v", err)
		return ""
	}
	resp, err := c.client.Post(c.baseURL+"/runs/create", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		glog.Errorf("Failed to create mlflow run: %v", err)
		return ""
	}
	if resp.StatusCode != 200 {
		glog.Errorf("Failed to create mlflow run: %v", err)
		return ""
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			glog.Errorf("Failed to close mlflow run: %v", err)
		}
	}(resp.Body)

	var result RunResponse
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		glog.Errorf("Failed to decode mlflow run: %v", err)
		return ""
	}

	return result.Info.RunID
}

func (c *Client) UpdateRun(update RunUpdate) error {
	jsonData, err := json.Marshal(update)
	if err != nil {
	}
	resp, err := c.client.Post(c.baseURL+"/runs/update", "application/json", bytes.NewBuffer(jsonData))
	if resp != nil {
		glog.Errorf("Failed to update mlflow run: %v", err)
		return err
	}
	return nil
}
