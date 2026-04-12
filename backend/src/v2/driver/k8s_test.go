package driver

import (
	"context"
	"testing"

	"github.com/kubeflow/pipelines/backend/src/v2/metadata"
	"github.com/kubeflow/pipelines/kubernetes_platform/go/kubernetesplatform"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/structpb"
)

func Test_makeVolumeMountPatch(t *testing.T) {
	type args struct {
		pvcMount []*kubernetesplatform.PvcMount
		dag      *metadata.DAG
		dagTasks map[string]*metadata.Execution
	}

	tests := []struct {
		name        string
		args        args
		wantPath    string
		wantName    string
		inputParams map[string]*structpb.Value
	}{
		{
			"pvc name: constant (deprecated)",
			args{
				[]*kubernetesplatform.PvcMount{
					{
						MountPath:    "/mnt/path",
						PvcReference: &kubernetesplatform.PvcMount_Constant{Constant: "pvc-name"},
					},
				},
				nil,
				nil,
			},
			"/mnt/path",
			"pvc-name",
			nil,
		},
		{
			"pvc name: constant parameter",
			args{
				[]*kubernetesplatform.PvcMount{
					{
						MountPath:        "/mnt/path",
						PvcReference:     &kubernetesplatform.PvcMount_Constant{Constant: "not-used"},
						PvcNameParameter: inputParamConstant("pvc-name"),
					},
				},
				nil,
				nil,
			},
			"/mnt/path",
			"pvc-name",
			nil,
		},
		{
			"pvc name: component input parameter",
			args{
				[]*kubernetesplatform.PvcMount{
					{
						MountPath:        "/mnt/path",
						PvcNameParameter: inputParamComponent("param_1"),
					},
				},
				nil,
				nil,
			},
			"/mnt/path",
			"pvc-name",
			map[string]*structpb.Value{
				"param_1": structpb.NewStringValue("pvc-name"),
			},
		},
		{
			"pvc mount with sub_path",
			args{
				[]*kubernetesplatform.PvcMount{
					{
						MountPath:        "/mnt/data",
						SubPath:          "models",
						PvcNameParameter: inputParamConstant("my-pvc"),
					},
				},
				nil,
				nil,
			},
			"/mnt/data",
			"my-pvc",
			nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			volumeMounts, volumes, err := makeVolumeMountPatch(
				context.Background(),
				Options{},
				tt.args.pvcMount,
				tt.args.dag,
				nil,
				nil,
				tt.inputParams,
			)
			assert.Nil(t, err)
			assert.Equal(t, 1, len(volumeMounts))
			assert.Equal(t, 1, len(volumes))
			assert.Equal(t, volumeMounts[0].MountPath, tt.wantPath)
			assert.Equal(t, volumeMounts[0].Name, tt.wantName)
			assert.Equal(t, volumes[0].Name, tt.wantName)
			assert.Equal(t, volumes[0].PersistentVolumeClaim.ClaimName, tt.wantName)
			// Check subPath if specified in the test case
			if len(tt.args.pvcMount) > 0 && tt.args.pvcMount[0].SubPath != "" {
				assert.Equal(t, tt.args.pvcMount[0].SubPath, volumeMounts[0].SubPath)
			}
		})
	}
}
