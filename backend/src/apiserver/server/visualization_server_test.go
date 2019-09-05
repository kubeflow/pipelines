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
	server := &VisualizationServer{
		resourceManager:    manager,
	}
	visualization := &go_client.Visualization{
		Type:      go_client.Visualization_ROC_CURVE,
		Source:    "gs://ml-pipeline/roc/data.csv",
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
	server := &VisualizationServer{
		resourceManager:    manager,
	}
	visualization := &go_client.Visualization{
		Type:      go_client.Visualization_ROC_CURVE,
		Source:    "gs://ml-pipeline/roc/data.csv",
		Arguments: "",
	}
	request := &go_client.CreateVisualizationRequest{
		Visualization: visualization,
	}
	err := server.validateCreateVisualizationRequest(request)
	assert.Nil(t, err)
}

func TestValidateCreateVisualizationRequest_SourceIsEmpty(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	server := &VisualizationServer{
		resourceManager:    manager,
	}
	visualization := &go_client.Visualization{
		Type:      go_client.Visualization_ROC_CURVE,
		Source:    "",
		Arguments: "{}",
	}
	request := &go_client.CreateVisualizationRequest{
		Visualization: visualization,
	}
	err := server.validateCreateVisualizationRequest(request)
	assert.Contains(t, err.Error(), "A visualization requires a Source to be provided. Received")
}

func TestValidateCreateVisualizationRequest_SourceIsEmptyAndTypeIsCustom(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	server := &VisualizationServer{
		resourceManager:    manager,
	}
	visualization := &go_client.Visualization{
		Type:      go_client.Visualization_CUSTOM,
		Arguments: "{}",
	}
	request := &go_client.CreateVisualizationRequest{
		Visualization: visualization,
	}
	err := server.validateCreateVisualizationRequest(request)
	assert.Nil(t, err)
}

func TestValidateCreateVisualizationRequest_ArgumentsNotValidJSON(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	server := &VisualizationServer{
		resourceManager:    manager,
	}
	visualization := &go_client.Visualization{
		Type:      go_client.Visualization_ROC_CURVE,
		Source:    "gs://ml-pipeline/roc/data.csv",
		Arguments: "{",
	}
	request := &go_client.CreateVisualizationRequest{
		Visualization: visualization,
	}
	err := server.validateCreateVisualizationRequest(request)
	assert.Contains(t, err.Error(), "A visualization requires valid JSON to be provided as Arguments. Received {")
}

func TestGenerateVisualization(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	httpServer := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		assert.Equal(t, "/", req.URL.String())
		rw.Write([]byte("roc_curve"))
	}))
	defer httpServer.Close()
	server := &VisualizationServer{
		resourceManager:    manager,
		serviceURL:         httpServer.URL,
	}
	visualization := &go_client.Visualization{
		Type:      go_client.Visualization_ROC_CURVE,
		Source:    "gs://ml-pipeline/roc/data.csv",
		Arguments: "{}",
	}
	request := &go_client.CreateVisualizationRequest{
		Visualization: visualization,
	}
	body, err := server.generateVisualizationFromRequest(request)
	assert.Nil(t, err)
	assert.Equal(t, []byte("roc_curve"), body)
}

func TestGenerateVisualization_ServiceNotAvailableError(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	httpServer := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		assert.Equal(t, "/", req.URL.String())
		if req.Method == http.MethodGet {
			rw.WriteHeader(500)
		} else {
			rw.WriteHeader(200)
		}
	}))
	server := &VisualizationServer{
		resourceManager:    manager,
		serviceURL:         httpServer.URL,
	}
	visualization := &go_client.Visualization{
		Type:      go_client.Visualization_ROC_CURVE,
		Source:    "gs://ml-pipeline/roc/data.csv",
		Arguments: "{}",
	}
	request := &go_client.CreateVisualizationRequest{
		Visualization: visualization,
	}
	body, err := server.generateVisualizationFromRequest(request)
	assert.Nil(t, body)
	assert.Equal(t, "InternalServerError: Service not available: service not available", err.Error())
}

func TestGenerateVisualization_ServerError(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	httpServer := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		assert.Equal(t, "/", req.URL.String())
		// The get requests 200s to indicate the service is alive, but the
		// visualization request fails with a 500.
		if req.Method == http.MethodGet {
			rw.WriteHeader(200)
		} else {
			rw.WriteHeader(500)
		}
	}))
	defer httpServer.Close()
	server := &VisualizationServer{
		resourceManager:    manager,
		serviceURL:         httpServer.URL,
	}
	visualization := &go_client.Visualization{
		Type:      go_client.Visualization_ROC_CURVE,
		Source:    "gs://ml-pipeline/roc/data.csv",
		Arguments: "{}",
	}
	request := &go_client.CreateVisualizationRequest{
		Visualization: visualization,
	}
	body, err := server.generateVisualizationFromRequest(request)
	assert.Nil(t, body)
	assert.Equal(t, "500 Internal Server Error", err.Error())
}
