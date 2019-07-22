package server

import (
	"github.com/kubeflow/pipelines/backend/api/go_client"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestValidateCreateVisualizationRequest(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	server := NewVisualizationServer(manager)
	apiVisualization := &go_client.Visualization{
		Type:      go_client.Visualization_ROC_CURVE,
		InputPath: "gs://ml-pipeline/roc/data.csv",
		Arguments: "{}",
	}
	err := server.ValidateCreateVisualizationRequest(apiVisualization)
	assert.Nil(t, err)
}

func TestValidateCreateVisualizationRequest_ArgumentsAreEmpty(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	server := NewVisualizationServer(manager)
	apiVisualization := &go_client.Visualization{
		Type:      go_client.Visualization_ROC_CURVE,
		InputPath: "gs://ml-pipeline/roc/data.csv",
		Arguments: "",
	}
	err := server.ValidateCreateVisualizationRequest(apiVisualization)
	assert.Nil(t, err)
}

func TestValidateCreateVisualizationRequest_InputPathIsEmpty(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	server := NewVisualizationServer(manager)
	apiVisualization := &go_client.Visualization{
		Type:      go_client.Visualization_ROC_CURVE,
		InputPath: "",
		Arguments: "{}",
	}
	err := server.ValidateCreateVisualizationRequest(apiVisualization)
	assert.Contains(t, err.Error(), "A visualization requires an InputPath to be provided. Received")
}

func TestValidateCreateVisualizationRequest_ArgumentsNotValidJSON(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	server := NewVisualizationServer(manager)
	apiVisualization := &go_client.Visualization{
		Type:      go_client.Visualization_ROC_CURVE,
		InputPath: "gs://ml-pipeline/roc/data.csv",
		Arguments: "{",
	}
	err := server.ValidateCreateVisualizationRequest(apiVisualization)
	assert.Contains(t, err.Error(), "A visualization requires valid JSON to be provided as Arguments. Received {")
}

func TestGetArgumentsAsJSONFromVisualization(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	server := NewVisualizationServer(manager)
	apiVisualization := &go_client.Visualization{
		Type:      go_client.Visualization_ROC_CURVE,
		InputPath: "gs://ml-pipeline/roc/data.csv",
		Arguments: "{}",
	}
	arguments, err := server.GetArgumentsAsJSONFromVisualization(apiVisualization)
	assert.Equal(t, []byte("{\"input_path\":\"gs://ml-pipeline/roc/data.csv\"}"), arguments)
	assert.Nil(t, err)
}

func TestGetArgumentsAsJSONFromVisualization_ArgumentsNotValidJSON(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	server := NewVisualizationServer(manager)
	apiVisualization := &go_client.Visualization{
		Type:      go_client.Visualization_ROC_CURVE,
		InputPath: "gs://ml-pipeline/roc/data.csv",
		Arguments: "{",
	}
	arguments, err := server.GetArgumentsAsJSONFromVisualization(apiVisualization)
	assert.Nil(t, arguments)
	assert.Contains(t, err.Error(), "Unable to parse provided JSON.")
}

func TestCreatePythonArgumentsFromTypeAndJSON(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	server := NewVisualizationServer(manager)
	apiVisualization := &go_client.Visualization{
		Type:      go_client.Visualization_ROC_CURVE,
		InputPath: "gs://ml-pipeline/roc/data.csv",
		Arguments: "{}",
	}
	arguments := []byte("{\"input_path\": \"gs://ml-pipeline/roc/data.csv\"}")
	pythonArguments := server.CreatePythonArgumentsFromTypeAndJSON(apiVisualization.Type, arguments)
	assert.Equal(t, "--type roc_curve --arguments '{\"input_path\": \"gs://ml-pipeline/roc/data.csv\"}'", pythonArguments)
}

func TestGenerateVisualization(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	httpServer := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		assert.Equal(t, "/", req.URL.String())
		rw.Write([]byte("roc_curve"))
	}))
	defer httpServer.Close()
	server := &VisualizationServer{resourceManager: manager, serviceURL: httpServer.URL}
	body, err := server.GenerateVisualization("--type roc_curve --arguments '{\"input_path\": \"gs://ml-pipeline/roc/data.csv\"}'")
	assert.Equal(t, []byte("roc_curve"), body)
	assert.Nil(t, err)
}

func TestGenerateVisualization_ServerError(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	httpServer := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		assert.Equal(t, "/", req.URL.String())
		rw.WriteHeader(500)
	}))
	defer httpServer.Close()
	server := &VisualizationServer{resourceManager: manager, serviceURL: httpServer.URL}
	body, err := server.GenerateVisualization("--type roc_curve --arguments '{\"input_path\": \"gs://ml-pipeline/roc/data.csv\"}'")
	assert.Nil(t, body)
	assert.Equal(t, "500 Internal Server Error", err.Error())
}
