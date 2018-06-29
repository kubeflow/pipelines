package worker

import (
	"fmt"
	"testing"

	workflowapi "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/googleprivate/ml/backend/src/controller/persistence/client"
	"github.com/googleprivate/ml/backend/src/util"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

func TestWorkflow_Save_Success(t *testing.T) {
	workflowFake := client.NewWorkflowClientFake()
	pipelineFake := client.NewPipelineClientFake()

	workflow := util.NewWorkflow(&workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "MY_NAMESPACE",
			Name:      "MY_NAME",
		},
	})

	workflowFake.Put("MY_NAMESPACE", "MY_NAME", workflow)

	saver := NewWorkflowSaver(
		workflowFake,
		pipelineFake)

	err := saver.Save("MY_KEY", "MY_NAMESPACE", "MY_NAME", 20)

	assert.Equal(t, false, util.HasCustomCode(err, util.CUSTOM_CODE_TRANSIENT))
	assert.Equal(t, nil, err)
}

func TestWorkflow_Save_NotFoundDuringGet(t *testing.T) {
	workflowFake := client.NewWorkflowClientFake()
	pipelineFake := client.NewPipelineClientFake()

	saver := NewWorkflowSaver(
		workflowFake,
		pipelineFake)

	err := saver.Save("MY_KEY", "MY_NAMESPACE", "MY_NAME", 20)

	assert.Equal(t, false, util.HasCustomCode(err, util.CUSTOM_CODE_TRANSIENT))
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Workflow not found")
}

func TestWorkflow_Save_ErrorDuringGet(t *testing.T) {
	workflowFake := client.NewWorkflowClientFake()
	pipelineFake := client.NewPipelineClientFake()

	workflowFake.Put("MY_NAMESPACE", "MY_NAME", nil)

	saver := NewWorkflowSaver(
		workflowFake,
		pipelineFake)

	err := saver.Save("MY_KEY", "MY_NAMESPACE", "MY_NAME", 20)

	assert.Equal(t, true, util.HasCustomCode(err, util.CUSTOM_CODE_TRANSIENT))
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "transient failure")
}

func TestWorkflow_Save_PermanentFailureWhileReporting(t *testing.T) {
	workflowFake := client.NewWorkflowClientFake()
	pipelineFake := client.NewPipelineClientFake()

	pipelineFake.SetError(util.NewCustomError(fmt.Errorf("Error"), util.CUSTOM_CODE_PERMANENT,
		"My Permanent Error"))

	workflow := util.NewWorkflow(&workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "MY_NAMESPACE",
			Name:      "MY_NAME",
		},
	})

	workflowFake.Put("MY_NAMESPACE", "MY_NAME", workflow)

	saver := NewWorkflowSaver(
		workflowFake,
		pipelineFake)

	err := saver.Save("MY_KEY", "MY_NAMESPACE", "MY_NAME", 20)

	assert.Equal(t, false, util.HasCustomCode(err, util.CUSTOM_CODE_TRANSIENT))
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "permanent failure")
}

func TestWorkflow_Save_TransientFailureWhileReporting(t *testing.T) {
	workflowFake := client.NewWorkflowClientFake()
	pipelineFake := client.NewPipelineClientFake()

	pipelineFake.SetError(util.NewCustomError(fmt.Errorf("Error"), util.CUSTOM_CODE_TRANSIENT,
		"My Transient Error"))

	workflow := util.NewWorkflow(&workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "MY_NAMESPACE",
			Name:      "MY_NAME",
		},
	})

	workflowFake.Put("MY_NAMESPACE", "MY_NAME", workflow)

	saver := NewWorkflowSaver(
		workflowFake,
		pipelineFake)

	err := saver.Save("MY_KEY", "MY_NAMESPACE", "MY_NAME", 20)

	assert.Equal(t, true, util.HasCustomCode(err, util.CUSTOM_CODE_TRANSIENT))
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "transient failure")
}
