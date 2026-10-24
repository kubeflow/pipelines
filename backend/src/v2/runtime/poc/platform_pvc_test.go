package poc

import (
	"context"
	"testing"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubernetesfake "k8s.io/client-go/kubernetes/fake"
)

func TestPlatformPVCRecoveryUsesMaterializedTaskIdentity(t *testing.T) {
	client := kubernetesfake.NewSimpleClientset()
	coordinator := NewCoordinator(nil, nil, nil, nil, util.NewFakeTimeForEpoch(), &fakeKubernetesCoreWithClientset{clientSet: client}, nil, false)
	input := &pipelinespec.ExecutorInput{Inputs: &pipelinespec.ExecutorInput_Inputs{
		ParameterValues: map[string]*structpb.Value{
			"size": structpb.NewStringValue("1Gi"),
			"access_modes": structpb.NewListValue(&structpb.ListValue{
				Values: []*structpb.Value{structpb.NewStringValue("ReadWriteOnce")},
			}),
		},
	}}
	run, task := &model.Run{UUID: "run", Namespace: "ns"}, &model.Task{UUID: "task"}
	first, err := coordinator.createPVCPlatformTask(context.Background(), run, task, input, &pipelinespec.PipelineTaskSpec{})
	require.NoError(t, err)
	recovered, err := coordinator.createPVCPlatformTask(context.Background(), run, task, input, &pipelinespec.PipelineTaskSpec{})
	require.NoError(t, err)
	require.Equal(t, first.OutputParameters["name"].GetStringValue(), recovered.OutputParameters["name"].GetStringValue(), "takeover must recover the same PVC after a lost database checkpoint")
	pvcs, err := client.CoreV1().PersistentVolumeClaims("ns").List(context.Background(), metav1.ListOptions{})
	require.NoError(t, err)
	require.Len(t, pvcs.Items, 1)
	input.Inputs.ParameterValues["size"] = structpb.NewStringValue("2Gi")
	_, err = coordinator.createPVCPlatformTask(context.Background(), run, task, input, &pipelinespec.PipelineTaskSpec{})
	require.ErrorContains(t, err, "requirements", "recovery must reject a different request at the same stable identity")
}
