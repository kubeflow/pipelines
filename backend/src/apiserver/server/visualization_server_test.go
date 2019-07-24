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
	visualization := &go_client.Visualization{
		Type:      go_client.Visualization_ROC_CURVE,
		InputPath: "gs://ml-pipeline/roc/data.csv",
		Arguments: "{}",
	}
	request := &go_client.CreateVisualizationRequest{
		Visualization: visualization,
	}
	err := server.validateCreateVisualizationRequest(request)
	assert.Nil(t, err)
}

func TestValidateCreateVisualizationRequest_ArgumentsAreEmpty(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	server := NewVisualizationServer(manager)
	visualization := &go_client.Visualization{
		Type:      go_client.Visualization_ROC_CURVE,
		InputPath: "gs://ml-pipeline/roc/data.csv",
		Arguments: "",
	}
	request := &go_client.CreateVisualizationRequest{
		Visualization: visualization,
	}
	err := server.validateCreateVisualizationRequest(request)
	assert.Nil(t, err)
}

func TestValidateCreateVisualizationRequest_InputPathIsEmpty(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	server := NewVisualizationServer(manager)
	visualization := &go_client.Visualization{
		Type:      go_client.Visualization_ROC_CURVE,
		InputPath: "",
		Arguments: "{}",
	}
	request := &go_client.CreateVisualizationRequest{
		Visualization: visualization,
	}
	err := server.validateCreateVisualizationRequest(request)
	assert.Contains(t, err.Error(), "A visualization requires an InputPath to be provided. Received")
}

func TestValidateCreateVisualizationRequest_ArgumentsNotValidJSON(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	server := NewVisualizationServer(manager)
	visualization := &go_client.Visualization{
		Type:      go_client.Visualization_ROC_CURVE,
		InputPath: "gs://ml-pipeline/roc/data.csv",
		Arguments: "{",
	}
	request := &go_client.CreateVisualizationRequest{
		Visualization: visualization,
	}
	err := server.validateCreateVisualizationRequest(request)
	assert.Contains(t, err.Error(), "A visualization requires valid JSON to be provided as Arguments. Received {")
}

func TestGetArgumentsAsJSONFromRequest(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	server := NewVisualizationServer(manager)
	visualization := &go_client.Visualization{
		Type:      go_client.Visualization_ROC_CURVE,
		InputPath: "gs://ml-pipeline/roc/data.csv",
		Arguments: "{\"is_generated\": \"True\"}",
	}
	request := &go_client.CreateVisualizationRequest{
		Visualization: visualization,
	}
	arguments, err := server.getArgumentsAsJSONFromRequest(request)
	assert.Equal(t, []byte("{\"is_generated\":\"True\"}"), arguments)
	assert.Nil(t, err)
}

func TestGetArgumentsAsJSONFromRequest_ArgumentsNotValidJSON(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	server := NewVisualizationServer(manager)
	visualization := &go_client.Visualization{
		Type:      go_client.Visualization_ROC_CURVE,
		InputPath: "gs://ml-pipeline/roc/data.csv",
		Arguments: "{",
	}
	request := &go_client.CreateVisualizationRequest{
		Visualization: visualization,
	}
	arguments, err := server.getArgumentsAsJSONFromRequest(request)
	assert.Nil(t, arguments)
	assert.Contains(t, err.Error(), "Unable to parse provided JSON.")
}

func TestCreatePythonArgumentsFromRequest(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	server := NewVisualizationServer(manager)
	visualization := &go_client.Visualization{
		Type:      go_client.Visualization_ROC_CURVE,
		InputPath: "gs://ml-pipeline/roc/data.csv",
		Arguments: "{}",
	}
	request := &go_client.CreateVisualizationRequest{
		Visualization: visualization,
	}
	pythonArguments, err := server.createPythonArgumentsFromRequest(request)
	assert.Equal(t, "--type roc_curve --input_path 'gs://ml-pipeline/roc/data.csv' --arguments '{}'", pythonArguments)
	assert.Nil(t, err)
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
	visualization := &go_client.Visualization{
		Type:      go_client.Visualization_ROC_CURVE,
		InputPath: "gs://ml-pipeline/roc/data.csv",
		Arguments: "{}",
	}
	request := &go_client.CreateVisualizationRequest{
		Visualization: visualization,
	}
	body, err := server.generateVisualizationFromRequest(request)
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
	visualization := &go_client.Visualization{
		Type:      go_client.Visualization_ROC_CURVE,
		InputPath: "gs://ml-pipeline/roc/data.csv",
		Arguments: "{}",
	}
	request := &go_client.CreateVisualizationRequest{
		Visualization: visualization,
	}
	body, err := server.generateVisualizationFromRequest(request)
	assert.Nil(t, body)
	assert.Equal(t, "500 Internal Server Error", err.Error())
}
