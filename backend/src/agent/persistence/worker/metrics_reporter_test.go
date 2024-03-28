// Copyright 2018 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package worker

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"google.golang.org/protobuf/testing/protocmp"

	workflowapi "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	api "github.com/kubeflow/pipelines/backend/api/v1beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/agent/persistence/client"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestReportMetrics_NoCompletedNode_NoOP(t *testing.T) {
	pipelineFake := client.NewPipelineClientFake()

	reporter := NewMetricsReporter(pipelineFake)

	workflow := util.NewWorkflow(&workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "MY_NAMESPACE",
			Name:      "MY_NAME",
			UID:       types.UID("run-1"),
		},
		Status: workflowapi.WorkflowStatus{
			Nodes: map[string]workflowapi.NodeStatus{
				"node-1": workflowapi.NodeStatus{
					ID:           "node-1",
					TemplateName: "template-1",
					Phase:        workflowapi.NodeRunning,
				},
			},
		},
	})
	err := reporter.ReportMetrics(workflow)
	assert.Nil(t, err)
	assert.Nil(t, pipelineFake.GetReportedMetricsRequest())
}

func TestReportMetrics_NoRunID_NoOP(t *testing.T) {
	pipelineFake := client.NewPipelineClientFake()

	reporter := NewMetricsReporter(pipelineFake)

	workflow := util.NewWorkflow(&workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "MY_NAMESPACE",
			Name:      "MY_NAME",
			UID:       types.UID("run-1"),
		},
		Status: workflowapi.WorkflowStatus{
			Nodes: map[string]workflowapi.NodeStatus{
				"node-1": workflowapi.NodeStatus{
					ID:    "node-1",
					Phase: workflowapi.NodeSucceeded,
				},
			},
		},
	})
	err := reporter.ReportMetrics(workflow)
	assert.Nil(t, err)
	assert.Nil(t, pipelineFake.GetReadArtifactRequest())
	assert.Nil(t, pipelineFake.GetReportedMetricsRequest())
}

func TestReportMetrics_NoArtifact_NoOP(t *testing.T) {
	pipelineFake := client.NewPipelineClientFake()

	reporter := NewMetricsReporter(pipelineFake)

	workflow := util.NewWorkflow(&workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "MY_NAMESPACE",
			Name:      "MY_NAME",
			UID:       types.UID("run-1"),
			Labels:    map[string]string{util.LabelKeyWorkflowRunId: "run-1"},
		},
		Status: workflowapi.WorkflowStatus{
			Nodes: map[string]workflowapi.NodeStatus{
				"node-1": workflowapi.NodeStatus{
					ID:           "node-1",
					TemplateName: "template-1",
					Phase:        workflowapi.NodeSucceeded,
				},
			},
		},
	})
	err := reporter.ReportMetrics(workflow)
	assert.Nil(t, err)
	assert.Nil(t, pipelineFake.GetReadArtifactRequest())
	assert.Nil(t, pipelineFake.GetReportedMetricsRequest())
}

func TestReportMetrics_NoMetricsArtifact_NoOP(t *testing.T) {
	pipelineFake := client.NewPipelineClientFake()

	reporter := NewMetricsReporter(pipelineFake)

	workflow := util.NewWorkflow(&workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "MY_NAMESPACE",
			Name:      "MY_NAME",
			UID:       types.UID("run-1"),
			Labels:    map[string]string{util.LabelKeyWorkflowRunId: "run-1"},
		},
		Status: workflowapi.WorkflowStatus{
			Nodes: map[string]workflowapi.NodeStatus{
				"node-1": workflowapi.NodeStatus{
					ID:           "node-1",
					TemplateName: "template-1",
					Phase:        workflowapi.NodeSucceeded,
					Outputs: &workflowapi.Outputs{
						Artifacts: []workflowapi.Artifact{{Name: "mlpipeline-ui-metadata"}},
					},
				},
			},
		},
	})
	err := reporter.ReportMetrics(workflow)
	assert.Nil(t, err)
	assert.Nil(t, pipelineFake.GetReadArtifactRequest())
	assert.Nil(t, pipelineFake.GetReportedMetricsRequest())
}

func TestReportMetrics_Succeed(t *testing.T) {
	pipelineFake := client.NewPipelineClientFake()
	reporter := NewMetricsReporter(pipelineFake)
	workflow := util.NewWorkflow(&workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "MY_NAMESPACE",
			Name:      "MY_NAME",
			UID:       types.UID("run-1"),
			Labels:    map[string]string{util.LabelKeyWorkflowRunId: "run-1"},
		},
		Status: workflowapi.WorkflowStatus{
			Nodes: map[string]workflowapi.NodeStatus{
				"node-1": workflowapi.NodeStatus{
					ID:           "node-1",
					TemplateName: "template-1",
					Phase:        workflowapi.NodeSucceeded,
					Outputs: &workflowapi.Outputs{
						Artifacts: []workflowapi.Artifact{{Name: "mlpipeline-metrics"}},
					},
				},
			},
		},
	})
	metricsJSON := `{"metrics": [{"name": "accuracy", "numberValue": 0.77}, {"name": "logloss", "numberValue": 1.2}]}`
	artifactData, _ := util.ArchiveTgz(map[string]string{"file": metricsJSON})
	pipelineFake.StubArtifact(
		&api.ReadArtifactRequest{
			RunId:        "run-1",
			NodeId:       "MY_NAME-template-1-1",
			ArtifactName: "mlpipeline-metrics",
		},
		&api.ReadArtifactResponse{
			Data: []byte(artifactData),
		})
	pipelineFake.StubReportRunMetrics(&api.ReportRunMetricsResponse{
		Results: []*api.ReportRunMetricsResponse_ReportRunMetricResult{},
	}, nil)

	err1 := reporter.ReportMetrics(workflow)

	assert.Nil(t, err1)
	expectedMetricsRequest := &api.ReportRunMetricsRequest{
		RunId: "run-1",
		Metrics: []*api.RunMetric{
			{
				Name:   "accuracy",
				NodeId: "MY_NAME-template-1-1",
				Value:  &api.RunMetric_NumberValue{NumberValue: 0.77},
			},
			{
				Name:   "logloss",
				NodeId: "MY_NAME-template-1-1",
				Value:  &api.RunMetric_NumberValue{NumberValue: 1.2},
			},
		},
	}
	got := pipelineFake.GetReportedMetricsRequest()
	if diff := cmp.Diff(expectedMetricsRequest, got, cmpopts.EquateEmpty(), protocmp.Transform()); diff != "" {
		t.Errorf("parseRuntimeInfo() = %+v, want %+v\nDiff (-want, +got)\n%s", got, expectedMetricsRequest, diff)
		s, _ := json.MarshalIndent(expectedMetricsRequest, "", "  ")
		fmt.Printf("Want %s", s)
	}
}

func TestReportMetrics_EmptyArchive_Fail(t *testing.T) {
	pipelineFake := client.NewPipelineClientFake()
	reporter := NewMetricsReporter(pipelineFake)
	workflow := util.NewWorkflow(&workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "MY_NAMESPACE",
			Name:      "MY_NAME",
			UID:       types.UID("run-1"),
			Labels:    map[string]string{util.LabelKeyWorkflowRunId: "run-1"},
		},
		Status: workflowapi.WorkflowStatus{
			Nodes: map[string]workflowapi.NodeStatus{
				"node-1": workflowapi.NodeStatus{
					ID:           "node-1",
					TemplateName: "template-1",
					Phase:        workflowapi.NodeSucceeded,
					Outputs: &workflowapi.Outputs{
						Artifacts: []workflowapi.Artifact{{Name: "mlpipeline-metrics"}},
					},
				},
			},
		},
	})
	artifactData, _ := util.ArchiveTgz(map[string]string{})
	pipelineFake.StubArtifact(
		&api.ReadArtifactRequest{
			RunId:        "run-1",
			NodeId:       "MY_NAME-template-1-1",
			ArtifactName: "mlpipeline-metrics",
		},
		&api.ReadArtifactResponse{
			Data: []byte(artifactData),
		})

	err := reporter.ReportMetrics(workflow)

	assert.NotNil(t, err)
	assert.True(t, util.HasCustomCode(err, util.CUSTOM_CODE_PERMANENT))
	// Verify that ReportRunMetrics is not called.
	assert.Nil(t, pipelineFake.GetReportedMetricsRequest())
}

func TestReportMetrics_MultipleFilesInArchive_Fail(t *testing.T) {
	pipelineFake := client.NewPipelineClientFake()
	reporter := NewMetricsReporter(pipelineFake)
	workflow := util.NewWorkflow(&workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "MY_NAMESPACE",
			Name:      "MY_NAME",
			UID:       types.UID("run-1"),
			Labels:    map[string]string{util.LabelKeyWorkflowRunId: "run-1"},
		},
		Status: workflowapi.WorkflowStatus{
			Nodes: map[string]workflowapi.NodeStatus{
				"node-1": workflowapi.NodeStatus{
					ID:           "MY_NAME-template-1-1",
					TemplateName: "template-1",
					Phase:        workflowapi.NodeSucceeded,
					Outputs: &workflowapi.Outputs{
						Artifacts: []workflowapi.Artifact{{Name: "mlpipeline-metrics"}},
					},
				},
			},
		},
	})
	validMetricsJSON := `{"metrics": [{"name": "accuracy", "numberValue": 0.77}, {"name": "logloss", "numberValue": 1.2}]}`
	invalidMetricsJSON := `invalid JSON`
	artifactData, _ := util.ArchiveTgz(map[string]string{"file1": validMetricsJSON, "file2": invalidMetricsJSON})
	pipelineFake.StubArtifact(
		&api.ReadArtifactRequest{
			RunId:        "run-1",
			NodeId:       "MY_NAME-template-1-1",
			ArtifactName: "mlpipeline-metrics",
		},
		&api.ReadArtifactResponse{
			Data: []byte(artifactData),
		})

	err := reporter.ReportMetrics(workflow)

	assert.NotNil(t, err)
	assert.True(t, util.HasCustomCode(err, util.CUSTOM_CODE_PERMANENT))
	// Verify that ReportRunMetrics is not called.
	assert.Nil(t, pipelineFake.GetReportedMetricsRequest())
}

func TestReportMetrics_InvalidMetricsJSON_Fail(t *testing.T) {
	pipelineFake := client.NewPipelineClientFake()
	reporter := NewMetricsReporter(pipelineFake)
	workflow := util.NewWorkflow(&workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "MY_NAMESPACE",
			Name:      "MY_NAME",
			UID:       types.UID("run-1"),
			Labels:    map[string]string{util.LabelKeyWorkflowRunId: "run-1"},
		},
		Status: workflowapi.WorkflowStatus{
			Nodes: map[string]workflowapi.NodeStatus{
				"node-1": workflowapi.NodeStatus{
					ID:           "node-1",
					TemplateName: "template-1",
					Phase:        workflowapi.NodeSucceeded,
					Outputs: &workflowapi.Outputs{
						Artifacts: []workflowapi.Artifact{{Name: "mlpipeline-metrics"}},
					},
				},
			},
		},
	})
	metricsJSON := `invalid JSON`
	artifactData, _ := util.ArchiveTgz(map[string]string{"file": metricsJSON})
	pipelineFake.StubArtifact(
		&api.ReadArtifactRequest{
			RunId:        "run-1",
			NodeId:       "MY_NAME-template-1-1",
			ArtifactName: "mlpipeline-metrics",
		},
		&api.ReadArtifactResponse{
			Data: []byte(artifactData),
		})

	err := reporter.ReportMetrics(workflow)

	assert.NotNil(t, err)
	assert.True(t, util.HasCustomCode(err, util.CUSTOM_CODE_PERMANENT))
	// Verify that ReportRunMetrics is not called.
	assert.Nil(t, pipelineFake.GetReportedMetricsRequest())
}

func TestReportMetrics_InvalidMetricsJSON_PartialFail(t *testing.T) {
	pipelineFake := client.NewPipelineClientFake()
	reporter := NewMetricsReporter(pipelineFake)
	workflow := util.NewWorkflow(&workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "MY_NAMESPACE",
			Name:      "MY_NAME",
			UID:       types.UID("run-1"),
			Labels:    map[string]string{util.LabelKeyWorkflowRunId: "run-1"},
		},
		Status: workflowapi.WorkflowStatus{
			Nodes: map[string]workflowapi.NodeStatus{
				"node-1": workflowapi.NodeStatus{
					ID:           "node-1",
					TemplateName: "template-1",
					Phase:        workflowapi.NodeSucceeded,
					Outputs: &workflowapi.Outputs{
						Artifacts: []workflowapi.Artifact{{Name: "mlpipeline-metrics"}},
					},
				},
				"node-2": workflowapi.NodeStatus{
					ID:           "node-2",
					TemplateName: "template-2",
					Phase:        workflowapi.NodeSucceeded,
					Outputs: &workflowapi.Outputs{
						Artifacts: []workflowapi.Artifact{{Name: "mlpipeline-metrics"}},
					},
				},
			},
		},
	})
	validMetricsJSON := `{"metrics": [{"name": "accuracy", "numberValue": 0.77}, {"name": "logloss", "numberValue": 1.2}]}`
	invalidMetricsJSON := `invalid JSON`
	validArtifactData, _ := util.ArchiveTgz(map[string]string{"file": validMetricsJSON})
	invalidArtifactData, _ := util.ArchiveTgz(map[string]string{"file": invalidMetricsJSON})
	// Stub two artifacts, node-1 is invalid, node-2 is valid.
	pipelineFake.StubArtifact(
		&api.ReadArtifactRequest{
			RunId:        "run-1",
			NodeId:       "MY_NAME-template-1-1",
			ArtifactName: "mlpipeline-metrics",
		},
		&api.ReadArtifactResponse{
			Data: []byte(invalidArtifactData),
		})
	pipelineFake.StubArtifact(
		&api.ReadArtifactRequest{
			RunId:        "run-1",
			NodeId:       "MY_NAME-template-2-2",
			ArtifactName: "mlpipeline-metrics",
		},
		&api.ReadArtifactResponse{
			Data: []byte(validArtifactData),
		})

	err := reporter.ReportMetrics(workflow)

	// Partial failure is reported while valid metrics are reported.
	assert.NotNil(t, err)
	assert.True(t, util.HasCustomCode(err, util.CUSTOM_CODE_PERMANENT))
	expectedMetricsRequest := &api.ReportRunMetricsRequest{
		RunId: "run-1",
		Metrics: []*api.RunMetric{
			&api.RunMetric{
				Name:   "accuracy",
				NodeId: "MY_NAME-template-2-2",
				Value:  &api.RunMetric_NumberValue{NumberValue: 0.77},
			},
			&api.RunMetric{
				Name:   "logloss",
				NodeId: "MY_NAME-template-2-2",
				Value:  &api.RunMetric_NumberValue{NumberValue: 1.2},
			},
		},
	}
	got := pipelineFake.GetReportedMetricsRequest()
	if diff := cmp.Diff(expectedMetricsRequest, got, cmpopts.EquateEmpty(), protocmp.Transform()); diff != "" {
		t.Errorf("parseRuntimeInfo() = %+v, want %+v\nDiff (-want, +got)\n%s", got, expectedMetricsRequest, diff)
		s, _ := json.MarshalIndent(expectedMetricsRequest, "", "  ")
		fmt.Printf("Want %s", s)
	}
}

func TestReportMetrics_CorruptedArchiveFile_Fail(t *testing.T) {
	pipelineFake := client.NewPipelineClientFake()
	reporter := NewMetricsReporter(pipelineFake)
	workflow := util.NewWorkflow(&workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "MY_NAMESPACE",
			Name:      "MY_NAME",
			UID:       types.UID("run-1"),
			Labels:    map[string]string{util.LabelKeyWorkflowRunId: "run-1"},
		},
		Status: workflowapi.WorkflowStatus{
			Nodes: map[string]workflowapi.NodeStatus{
				"node-1": workflowapi.NodeStatus{
					ID:           "node-1",
					TemplateName: "template-1",
					Phase:        workflowapi.NodeSucceeded,
					Outputs: &workflowapi.Outputs{
						Artifacts: []workflowapi.Artifact{{Name: "mlpipeline-metrics"}},
					},
				},
			},
		},
	})
	pipelineFake.StubArtifact(
		&api.ReadArtifactRequest{
			RunId:        "run-1",
			NodeId:       "MY_NAME-template-1-1",
			ArtifactName: "mlpipeline-metrics",
		},
		&api.ReadArtifactResponse{
			Data: []byte("invalid tgz content"),
		})

	err := reporter.ReportMetrics(workflow)

	assert.NotNil(t, err)
	assert.True(t, util.HasCustomCode(err, util.CUSTOM_CODE_PERMANENT))
	// Verify that ReportRunMetrics is not called.
	assert.Nil(t, pipelineFake.GetReportedMetricsRequest())
}

func TestReportMetrics_MultiplMetricErrors_TransientErrowWin(t *testing.T) {
	pipelineFake := client.NewPipelineClientFake()
	reporter := NewMetricsReporter(pipelineFake)
	workflow := util.NewWorkflow(&workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "MY_NAMESPACE",
			Name:      "MY_NAME",
			UID:       types.UID("run-1"),
			Labels:    map[string]string{util.LabelKeyWorkflowRunId: "run-1"},
		},
		Status: workflowapi.WorkflowStatus{
			Nodes: map[string]workflowapi.NodeStatus{
				"node-1": workflowapi.NodeStatus{
					ID:           "node-1",
					TemplateName: "template-1",
					Phase:        workflowapi.NodeSucceeded,
					Outputs: &workflowapi.Outputs{
						Artifacts: []workflowapi.Artifact{{Name: "mlpipeline-metrics"}},
					},
				},
			},
		},
	})
	metricsJSON :=
		`{"metrics": [{"name": "accuracy", "numberValue": 0.77}, {"name": "log loss", "numberValue": 1.2}, {"name": "accuracy", "numberValue": 1.2}]}`
	artifactData, _ := util.ArchiveTgz(map[string]string{"file": metricsJSON})
	pipelineFake.StubArtifact(
		&api.ReadArtifactRequest{
			RunId:        "run-1",
			NodeId:       "MY_NAME-template-1-1",
			ArtifactName: "mlpipeline-metrics",
		},
		&api.ReadArtifactResponse{
			Data: []byte(artifactData),
		})
	pipelineFake.StubReportRunMetrics(&api.ReportRunMetricsResponse{
		Results: []*api.ReportRunMetricsResponse_ReportRunMetricResult{
			&api.ReportRunMetricsResponse_ReportRunMetricResult{
				MetricNodeId: "node-1",
				MetricName:   "accuracy",
				Status:       api.ReportRunMetricsResponse_ReportRunMetricResult_OK,
			},
			// Invalid argument error triggers permanent error
			&api.ReportRunMetricsResponse_ReportRunMetricResult{
				MetricNodeId: "node-1",
				MetricName:   "log loss",
				Status:       api.ReportRunMetricsResponse_ReportRunMetricResult_INVALID_ARGUMENT,
			},
			// Internal error triggers transient error
			&api.ReportRunMetricsResponse_ReportRunMetricResult{
				MetricNodeId: "node-1",
				MetricName:   "accuracy",
				Status:       api.ReportRunMetricsResponse_ReportRunMetricResult_INTERNAL_ERROR,
			},
		},
	}, nil)

	err := reporter.ReportMetrics(workflow)

	assert.NotNil(t, err)
	assert.True(t, util.HasCustomCode(err, util.CUSTOM_CODE_TRANSIENT))
}

func TestReportMetrics_Unauthorized(t *testing.T) {
	pipelineFake := client.NewPipelineClientFake()
	reporter := NewMetricsReporter(pipelineFake)

	workflow := util.NewWorkflow(&workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "MY_NAMESPACE",
			Name:      "MY_NAME",
			UID:       types.UID("run-1"),
			Labels:    map[string]string{util.LabelKeyWorkflowRunId: "run-1"},
		},
		Status: workflowapi.WorkflowStatus{
			Nodes: map[string]workflowapi.NodeStatus{
				"node-1": workflowapi.NodeStatus{
					ID:           "node-1",
					TemplateName: "template-1",
					Phase:        workflowapi.NodeSucceeded,
					Outputs: &workflowapi.Outputs{
						Artifacts: []workflowapi.Artifact{{Name: "mlpipeline-metrics"}},
					},
				},
			},
		},
	})
	metricsJSON := `{"metrics": [{"name": "accuracy", "numberValue": 0.77}, {"name": "logloss", "numberValue": 1.2}]}`
	artifactData, _ := util.ArchiveTgz(map[string]string{"file": metricsJSON})
	pipelineFake.StubArtifact(
		&api.ReadArtifactRequest{
			RunId:        "run-1",
			NodeId:       "MY_NAME-template-1-1",
			ArtifactName: "mlpipeline-metrics",
		},
		&api.ReadArtifactResponse{
			Data: []byte(artifactData),
		})
	pipelineFake.StubReportRunMetrics(&api.ReportRunMetricsResponse{
		Results: []*api.ReportRunMetricsResponse_ReportRunMetricResult{},
	}, errors.New("failed to read artifacts"))

	err1 := reporter.ReportMetrics(workflow)

	assert.NotNil(t, err1)
	assert.Contains(t, err1.Error(), "failed to read artifacts")
}
