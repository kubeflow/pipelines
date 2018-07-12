package worker

import (
	"fmt"
	"testing"

	"github.com/googleprivate/ml/backend/src/agent/persistence/client"
	"github.com/googleprivate/ml/backend/src/common/util"
	swfapi "github.com/googleprivate/ml/backend/src/crd/pkg/apis/scheduledworkflow/v1alpha1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

func TestScheduledWorkflow_Save_Success(t *testing.T) {
	swfFake := client.NewScheduledWorkflowClientFake()
	pipelineFake := client.NewPipelineClientFake()

	workflow := util.NewScheduledWorkflow(&swfapi.ScheduledWorkflow{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "MY_NAMESPACE",
			Name:      "MY_NAME",
		},
	})

	swfFake.Put("MY_NAMESPACE", "MY_NAME", workflow)

	saver := NewScheduledWorkflowSaver(
		swfFake,
		pipelineFake)

	err := saver.Save("MY_KEY", "MY_NAMESPACE", "MY_NAME", 20)

	assert.Equal(t, false, util.HasCustomCode(err, util.CUSTOM_CODE_TRANSIENT))
	assert.Equal(t, nil, err)
}

func TestScheduledWorkflow_Save_NotFoundDuringGet(t *testing.T) {
	swfFake := client.NewScheduledWorkflowClientFake()
	pipelineFake := client.NewPipelineClientFake()

	saver := NewScheduledWorkflowSaver(
		swfFake,
		pipelineFake)

	err := saver.Save("MY_KEY", "MY_NAMESPACE", "MY_NAME", 20)

	assert.Equal(t, false, util.HasCustomCode(err, util.CUSTOM_CODE_TRANSIENT))
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Workflow not found")
}

func TestScheduledWorkflow_Save_ErrorDuringGet(t *testing.T) {
	swfFake := client.NewScheduledWorkflowClientFake()
	pipelineFake := client.NewPipelineClientFake()

	swfFake.Put("MY_NAMESPACE", "MY_NAME", nil)

	saver := NewScheduledWorkflowSaver(
		swfFake,
		pipelineFake)

	err := saver.Save("MY_KEY", "MY_NAMESPACE", "MY_NAME", 20)

	assert.Equal(t, true, util.HasCustomCode(err, util.CUSTOM_CODE_TRANSIENT))
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "transient failure")
}

func TestScheduledWorkflow_Save_PermanentFailureWhileReporting(t *testing.T) {
	swfFake := client.NewScheduledWorkflowClientFake()
	pipelineFake := client.NewPipelineClientFake()

	pipelineFake.SetError(util.NewCustomError(fmt.Errorf("Error"), util.CUSTOM_CODE_PERMANENT,
		"My Permanent Error"))

	workflow := util.NewScheduledWorkflow(&swfapi.ScheduledWorkflow{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "MY_NAMESPACE",
			Name:      "MY_NAME",
		},
	})

	swfFake.Put("MY_NAMESPACE", "MY_NAME", workflow)

	saver := NewScheduledWorkflowSaver(
		swfFake,
		pipelineFake)

	err := saver.Save("MY_KEY", "MY_NAMESPACE", "MY_NAME", 20)

	assert.Equal(t, false, util.HasCustomCode(err, util.CUSTOM_CODE_TRANSIENT))
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "permanent failure")
}

func TestScheduledWorkflow_Save_TransientFailureWhileReporting(t *testing.T) {
	swfFake := client.NewScheduledWorkflowClientFake()
	pipelineFake := client.NewPipelineClientFake()

	pipelineFake.SetError(util.NewCustomError(fmt.Errorf("Error"), util.CUSTOM_CODE_TRANSIENT,
		"My Transient Error"))

	workflow := util.NewScheduledWorkflow(&swfapi.ScheduledWorkflow{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "MY_NAMESPACE",
			Name:      "MY_NAME",
		},
	})

	swfFake.Put("MY_NAMESPACE", "MY_NAME", workflow)

	saver := NewScheduledWorkflowSaver(
		swfFake,
		pipelineFake)

	err := saver.Save("MY_KEY", "MY_NAMESPACE", "MY_NAME", 20)

	assert.Equal(t, true, util.HasCustomCode(err, util.CUSTOM_CODE_TRANSIENT))
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "transient failure")
}
