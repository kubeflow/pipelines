package driver

import (
	"context"
	"testing"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata"
	"github.com/kubeflow/pipelines/kubernetes_platform/go/kubernetesplatform"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/structpb"
	k8score "k8s.io/api/core/v1"
	k8sres "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
		})
	}
}

func Test_makePodSpecPatch_nodeSelector(t *testing.T) {
	viper.Set("KFP_POD_NAME", "MyWorkflowPod")
	viper.Set("KFP_POD_UID", "a1b2c3d4-a1b2-a1b2-a1b2-a1b2c3d4e5f6")
	tests := []struct {
		name        string
		k8sExecCfg  *kubernetesplatform.KubernetesExecutorConfig
		expected    *k8score.PodSpec
		inputParams map[string]*structpb.Value
	}{
		{
			"Valid - NVIDIA GPU on GKE",
			&kubernetesplatform.KubernetesExecutorConfig{
				NodeSelector: &kubernetesplatform.NodeSelector{
					Labels: map[string]string{
						"cloud.google.com/gke-accelerator": "nvidia-tesla-k80",
					},
				},
			},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
					},
				},
				NodeSelector: map[string]string{"cloud.google.com/gke-accelerator": "nvidia-tesla-k80"},
			},
			nil,
		},
		{
			"Valid - operating system and arch",
			&kubernetesplatform.KubernetesExecutorConfig{
				NodeSelector: &kubernetesplatform.NodeSelector{
					Labels: map[string]string{
						"beta.kubernetes.io/os":   "linux",
						"beta.kubernetes.io/arch": "amd64",
					},
				},
			},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
					},
				},
				NodeSelector: map[string]string{"beta.kubernetes.io/arch": "amd64", "beta.kubernetes.io/os": "linux"},
			},
			nil,
		},
		{
			"Valid - Json Parameter",
			&kubernetesplatform.KubernetesExecutorConfig{
				NodeSelector: &kubernetesplatform.NodeSelector{
					NodeSelectorJson: inputParamComponent("param_1"),
				},
			},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
					},
				},
				NodeSelector: map[string]string{"beta.kubernetes.io/arch": "amd64", "beta.kubernetes.io/os": "linux"},
			},
			map[string]*structpb.Value{
				"param_1": validValueStructOrPanic(map[string]interface{}{
					"beta.kubernetes.io/arch": "amd64",
					"beta.kubernetes.io/os":   "linux",
				}),
			},
		},
		{
			"Valid - empty",
			&kubernetesplatform.KubernetesExecutorConfig{},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
					},
				},
			},
			nil,
		},
		{
			"Valid - empty json",
			&kubernetesplatform.KubernetesExecutorConfig{
				NodeSelector: &kubernetesplatform.NodeSelector{
					NodeSelectorJson: inputParamComponent("param_1"),
				},
			},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
					},
				},
				// valid node selector, pod can be scheduled on any node
				NodeSelector: map[string]string{},
			},
			map[string]*structpb.Value{
				"param_1": validValueStructOrPanic(map[string]interface{}{}),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := &k8score.PodSpec{Containers: []k8score.Container{
				{
					Name: "main",
				},
			}}
			err := extendPodSpecPatch(
				context.Background(),
				got,
				Options{KubernetesExecutorConfig: tt.k8sExecCfg},
				nil,
				nil,
				nil,
				tt.inputParams,
			)
			assert.Nil(t, err)
			assert.NotNil(t, got)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func Test_extendPodSpecPatch_Secret(t *testing.T) {
	tests := []struct {
		name        string
		k8sExecCfg  *kubernetesplatform.KubernetesExecutorConfig
		podSpec     *k8score.PodSpec
		expected    *k8score.PodSpec
		inputParams map[string]*structpb.Value
	}{
		{
			"Valid - secret as volume (deprecated)",
			&kubernetesplatform.KubernetesExecutorConfig{
				SecretAsVolume: []*kubernetesplatform.SecretAsVolume{
					{
						SecretName: "secret1",
						MountPath:  "/data/path",
					},
				},
			},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
					},
				},
			},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
						VolumeMounts: []k8score.VolumeMount{
							{
								Name:      "secret1",
								MountPath: "/data/path",
							},
						},
					},
				},
				Volumes: []k8score.Volume{
					{
						Name: "secret1",
						VolumeSource: k8score.VolumeSource{
							Secret: &k8score.SecretVolumeSource{SecretName: "secret1", Optional: &[]bool{false}[0]},
						},
					},
				},
			},
			nil,
		},
		{
			"Valid - secret as volume",
			&kubernetesplatform.KubernetesExecutorConfig{
				SecretAsVolume: []*kubernetesplatform.SecretAsVolume{
					{
						SecretName:          "not-used",
						SecretNameParameter: inputParamConstant("secret1"),
						MountPath:           "/data/path",
					},
				},
			},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
					},
				},
			},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
						VolumeMounts: []k8score.VolumeMount{
							{
								Name:      "secret1",
								MountPath: "/data/path",
							},
						},
					},
				},
				Volumes: []k8score.Volume{
					{
						Name: "secret1",
						VolumeSource: k8score.VolumeSource{
							Secret: &k8score.SecretVolumeSource{SecretName: "secret1", Optional: &[]bool{false}[0]},
						},
					},
				},
			},
			nil,
		},
		{
			"Valid - secret as volume with optional false",
			&kubernetesplatform.KubernetesExecutorConfig{
				SecretAsVolume: []*kubernetesplatform.SecretAsVolume{
					{
						SecretName:          "not-used",
						SecretNameParameter: inputParamConstant("secret1"),
						MountPath:           "/data/path",
						Optional:            &[]bool{false}[0],
					},
				},
			},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
					},
				},
			},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
						VolumeMounts: []k8score.VolumeMount{
							{
								Name:      "secret1",
								MountPath: "/data/path",
							},
						},
					},
				},
				Volumes: []k8score.Volume{
					{
						Name: "secret1",
						VolumeSource: k8score.VolumeSource{
							Secret: &k8score.SecretVolumeSource{SecretName: "secret1", Optional: &[]bool{false}[0]},
						},
					},
				},
			},
			nil,
		},
		{
			"Valid - secret as volume with optional true",
			&kubernetesplatform.KubernetesExecutorConfig{
				SecretAsVolume: []*kubernetesplatform.SecretAsVolume{
					{
						SecretName:          "not-used",
						SecretNameParameter: inputParamConstant("secret1"),
						MountPath:           "/data/path",
						Optional:            &[]bool{true}[0],
					},
				},
			},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
					},
				},
			},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
						VolumeMounts: []k8score.VolumeMount{
							{
								Name:      "secret1",
								MountPath: "/data/path",
							},
						},
					},
				},
				Volumes: []k8score.Volume{
					{
						Name: "secret1",
						VolumeSource: k8score.VolumeSource{
							Secret: &k8score.SecretVolumeSource{SecretName: "secret1", Optional: &[]bool{true}[0]},
						},
					},
				},
			},
			nil,
		},
		{
			"Valid - secret not specified",
			&kubernetesplatform.KubernetesExecutorConfig{},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
					},
				},
			},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
					},
				},
			},
			nil,
		},
		{
			"Valid - secret as volume: component input parameter",
			&kubernetesplatform.KubernetesExecutorConfig{
				SecretAsVolume: []*kubernetesplatform.SecretAsVolume{
					{
						SecretName:          "not-used",
						SecretNameParameter: inputParamComponent("param_1"),
						MountPath:           "/data/path",
						Optional:            &[]bool{true}[0],
					},
				},
			},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
					},
				},
			},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
						VolumeMounts: []k8score.VolumeMount{
							{
								Name:      "secret-name",
								MountPath: "/data/path",
							},
						},
					},
				},
				Volumes: []k8score.Volume{
					{
						Name: "secret-name",
						VolumeSource: k8score.VolumeSource{
							Secret: &k8score.SecretVolumeSource{SecretName: "secret-name", Optional: &[]bool{true}[0]},
						},
					},
				},
			},
			map[string]*structpb.Value{
				"param_1": structpb.NewStringValue("secret-name"),
			},
		},
		{
			"Valid - secret as env (deprecated)",
			&kubernetesplatform.KubernetesExecutorConfig{
				SecretAsEnv: []*kubernetesplatform.SecretAsEnv{
					{
						SecretName: "my-secret",
						KeyToEnv: []*kubernetesplatform.SecretAsEnv_SecretKeyToEnvMap{
							{
								SecretKey: "password",
								EnvVar:    "SECRET_VAR",
							},
						},
					},
				},
			},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
					},
				},
			},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
						Env: []k8score.EnvVar{
							{
								Name: "SECRET_VAR",
								ValueFrom: &k8score.EnvVarSource{
									SecretKeyRef: &k8score.SecretKeySelector{
										LocalObjectReference: k8score.LocalObjectReference{Name: "my-secret"},
										Key:                  "password",
										Optional:             nil,
									},
								},
							},
						},
					},
				},
			},
			nil,
		},
		{
			"Valid - secret as env",
			&kubernetesplatform.KubernetesExecutorConfig{
				SecretAsEnv: []*kubernetesplatform.SecretAsEnv{
					{
						SecretName:          "not-used",
						SecretNameParameter: inputParamConstant("my-secret"),
						KeyToEnv: []*kubernetesplatform.SecretAsEnv_SecretKeyToEnvMap{
							{
								SecretKey: "password",
								EnvVar:    "SECRET_VAR",
							},
						},
					},
				},
			},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
					},
				},
			},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
						Env: []k8score.EnvVar{
							{
								Name: "SECRET_VAR",
								ValueFrom: &k8score.EnvVarSource{
									SecretKeyRef: &k8score.SecretKeySelector{
										LocalObjectReference: k8score.LocalObjectReference{Name: "my-secret"},
										Key:                  "password",
										Optional:             nil,
									},
								},
							},
						},
					},
				},
			},
			nil,
		},
		{
			"Valid - secret as env: component input parameter",
			&kubernetesplatform.KubernetesExecutorConfig{
				SecretAsEnv: []*kubernetesplatform.SecretAsEnv{
					{
						SecretNameParameter: inputParamComponent("param_1"),
						KeyToEnv: []*kubernetesplatform.SecretAsEnv_SecretKeyToEnvMap{
							{
								SecretKey: "password",
								EnvVar:    "SECRET_VAR",
							},
						},
					},
				},
			},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
					},
				},
			},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
						Env: []k8score.EnvVar{
							{
								Name: "SECRET_VAR",
								ValueFrom: &k8score.EnvVarSource{
									SecretKeyRef: &k8score.SecretKeySelector{
										LocalObjectReference: k8score.LocalObjectReference{Name: "secret-name"},
										Key:                  "password",
										Optional:             nil,
									},
								},
							},
						},
					},
				},
			},
			map[string]*structpb.Value{
				"param_1": structpb.NewStringValue("secret-name"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := extendPodSpecPatch(
				context.Background(),
				tt.podSpec,
				Options{KubernetesExecutorConfig: tt.k8sExecCfg},
				nil,
				nil,
				nil,
				tt.inputParams,
			)
			assert.Nil(t, err)
			assert.Equal(t, tt.expected, tt.podSpec)
		})
	}
}

func Test_extendPodSpecPatch_ConfigMap(t *testing.T) {
	tests := []struct {
		name        string
		k8sExecCfg  *kubernetesplatform.KubernetesExecutorConfig
		podSpec     *k8score.PodSpec
		expected    *k8score.PodSpec
		inputParams map[string]*structpb.Value
	}{
		{
			"Valid - config map as volume (deprecated)",
			&kubernetesplatform.KubernetesExecutorConfig{
				ConfigMapAsVolume: []*kubernetesplatform.ConfigMapAsVolume{
					{
						ConfigMapName: "cm1",
						MountPath:     "/data/path",
					},
				},
			},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
					},
				},
			},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
						VolumeMounts: []k8score.VolumeMount{
							{
								Name:      "cm1",
								MountPath: "/data/path",
							},
						},
					},
				},
				Volumes: []k8score.Volume{
					{
						Name: "cm1",
						VolumeSource: k8score.VolumeSource{
							ConfigMap: &k8score.ConfigMapVolumeSource{
								LocalObjectReference: k8score.LocalObjectReference{Name: "cm1"},
								Optional:             &[]bool{false}[0],
							},
						},
					},
				},
			},
			nil,
		},
		{
			"Valid - config map as volume",
			&kubernetesplatform.KubernetesExecutorConfig{
				ConfigMapAsVolume: []*kubernetesplatform.ConfigMapAsVolume{
					{
						ConfigMapName:          "not-used",
						ConfigMapNameParameter: inputParamConstant("cm1"),
						MountPath:              "/data/path",
					},
				},
			},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
					},
				},
			},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
						VolumeMounts: []k8score.VolumeMount{
							{
								Name:      "cm1",
								MountPath: "/data/path",
							},
						},
					},
				},
				Volumes: []k8score.Volume{
					{
						Name: "cm1",
						VolumeSource: k8score.VolumeSource{
							ConfigMap: &k8score.ConfigMapVolumeSource{
								LocalObjectReference: k8score.LocalObjectReference{Name: "cm1"},
								Optional:             &[]bool{false}[0]},
						},
					},
				},
			},
			nil,
		},
		{
			"Valid - config map as volume with optional false",
			&kubernetesplatform.KubernetesExecutorConfig{
				ConfigMapAsVolume: []*kubernetesplatform.ConfigMapAsVolume{
					{
						ConfigMapName:          "not-used",
						ConfigMapNameParameter: inputParamConstant("cm1"),
						MountPath:              "/data/path",
						Optional:               &[]bool{false}[0],
					},
				},
			},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
					},
				},
			},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
						VolumeMounts: []k8score.VolumeMount{
							{
								Name:      "cm1",
								MountPath: "/data/path",
							},
						},
					},
				},
				Volumes: []k8score.Volume{
					{
						Name: "cm1",
						VolumeSource: k8score.VolumeSource{
							ConfigMap: &k8score.ConfigMapVolumeSource{
								LocalObjectReference: k8score.LocalObjectReference{Name: "cm1"},
								Optional:             &[]bool{false}[0],
							},
						},
					},
				},
			},
			nil,
		},
		{
			"Valid - config map as volume with optional true",
			&kubernetesplatform.KubernetesExecutorConfig{
				ConfigMapAsVolume: []*kubernetesplatform.ConfigMapAsVolume{
					{
						ConfigMapName:          "not-used",
						ConfigMapNameParameter: inputParamConstant("cm1"),
						MountPath:              "/data/path",
						Optional:               &[]bool{true}[0],
					},
				},
			},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
					},
				},
			},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
						VolumeMounts: []k8score.VolumeMount{
							{
								Name:      "cm1",
								MountPath: "/data/path",
							},
						},
					},
				},
				Volumes: []k8score.Volume{
					{
						Name: "cm1",
						VolumeSource: k8score.VolumeSource{
							ConfigMap: &k8score.ConfigMapVolumeSource{
								LocalObjectReference: k8score.LocalObjectReference{Name: "cm1"},
								Optional:             &[]bool{true}[0],
							},
						},
					},
				},
			},
			nil,
		},
		{
			"Valid - config map not specified",
			&kubernetesplatform.KubernetesExecutorConfig{},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
					},
				},
			},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
					},
				},
			},
			nil,
		},
		{
			"Valid - config map volume: component input parameter",
			&kubernetesplatform.KubernetesExecutorConfig{
				ConfigMapAsVolume: []*kubernetesplatform.ConfigMapAsVolume{
					{
						ConfigMapName:          "not-used",
						ConfigMapNameParameter: inputParamComponent("param_1"),
						MountPath:              "/data/path",
						Optional:               &[]bool{true}[0],
					},
				},
			},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
					},
				},
			},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
						VolumeMounts: []k8score.VolumeMount{
							{
								Name:      "cm-name",
								MountPath: "/data/path",
							},
						},
					},
				},
				Volumes: []k8score.Volume{
					{
						Name: "cm-name",
						VolumeSource: k8score.VolumeSource{
							ConfigMap: &k8score.ConfigMapVolumeSource{
								LocalObjectReference: k8score.LocalObjectReference{Name: "cm-name"},
								Optional:             &[]bool{true}[0]},
						},
					},
				},
			},
			map[string]*structpb.Value{
				"param_1": structpb.NewStringValue("cm-name"),
			},
		},
		{
			"Valid - config map as env (deprecated)",
			&kubernetesplatform.KubernetesExecutorConfig{
				ConfigMapAsEnv: []*kubernetesplatform.ConfigMapAsEnv{
					{
						ConfigMapName: "my-cm",
						KeyToEnv: []*kubernetesplatform.ConfigMapAsEnv_ConfigMapKeyToEnvMap{
							{
								ConfigMapKey: "foo",
								EnvVar:       "CONFIG_MAP_VAR",
							},
						},
					},
				},
			},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
					},
				},
			},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
						Env: []k8score.EnvVar{
							{
								Name: "CONFIG_MAP_VAR",
								ValueFrom: &k8score.EnvVarSource{
									ConfigMapKeyRef: &k8score.ConfigMapKeySelector{
										LocalObjectReference: k8score.LocalObjectReference{Name: "my-cm"},
										Key:                  "foo",
										Optional:             nil,
									},
								},
							},
						},
					},
				},
			},
			nil,
		},
		{
			"Valid - config map as env",
			&kubernetesplatform.KubernetesExecutorConfig{
				ConfigMapAsEnv: []*kubernetesplatform.ConfigMapAsEnv{
					{
						ConfigMapName:          "not-used",
						ConfigMapNameParameter: inputParamConstant("my-cm"),
						KeyToEnv: []*kubernetesplatform.ConfigMapAsEnv_ConfigMapKeyToEnvMap{
							{
								ConfigMapKey: "foo",
								EnvVar:       "CONFIG_MAP_VAR",
							},
						},
					},
				},
			},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
					},
				},
			},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
						Env: []k8score.EnvVar{
							{
								Name: "CONFIG_MAP_VAR",
								ValueFrom: &k8score.EnvVarSource{
									ConfigMapKeyRef: &k8score.ConfigMapKeySelector{
										LocalObjectReference: k8score.LocalObjectReference{Name: "my-cm"},
										Key:                  "foo",
										Optional:             nil,
									},
								},
							},
						},
					},
				},
			},
			nil,
		},
		{
			"Valid - config map as env: component input parameter",
			&kubernetesplatform.KubernetesExecutorConfig{
				ConfigMapAsEnv: []*kubernetesplatform.ConfigMapAsEnv{
					{
						ConfigMapName:          "not-used",
						ConfigMapNameParameter: inputParamComponent("param_1"),
						KeyToEnv: []*kubernetesplatform.ConfigMapAsEnv_ConfigMapKeyToEnvMap{
							{
								ConfigMapKey: "foo",
								EnvVar:       "CONFIG_MAP_VAR",
							},
						},
					},
				},
			},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
					},
				},
			},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
						Env: []k8score.EnvVar{
							{
								Name: "CONFIG_MAP_VAR",
								ValueFrom: &k8score.EnvVarSource{
									ConfigMapKeyRef: &k8score.ConfigMapKeySelector{
										LocalObjectReference: k8score.LocalObjectReference{Name: "cm-name"},
										Key:                  "foo",
										Optional:             nil,
									},
								},
							},
						},
					},
				},
			},
			map[string]*structpb.Value{
				"param_1": structpb.NewStringValue("cm-name"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := extendPodSpecPatch(
				context.Background(),
				tt.podSpec,
				Options{KubernetesExecutorConfig: tt.k8sExecCfg},
				nil,
				nil,
				nil,
				tt.inputParams,
			)
			assert.Nil(t, err)
			assert.Equal(t, tt.expected, tt.podSpec)
		})
	}
}

func Test_extendPodSpecPatch_EmptyVolumeMount(t *testing.T) {
	medium := "Memory"
	sizeLimit := "1Gi"
	var sizeLimitResource *k8sres.Quantity
	r := k8sres.MustParse(sizeLimit)
	sizeLimitResource = &r

	tests := []struct {
		name       string
		k8sExecCfg *kubernetesplatform.KubernetesExecutorConfig
		podSpec    *k8score.PodSpec
		expected   *k8score.PodSpec
	}{
		{
			"Valid - emptydir mount with no medium or size limit",
			&kubernetesplatform.KubernetesExecutorConfig{
				EmptyDirMounts: []*kubernetesplatform.EmptyDirMount{
					{
						VolumeName: "emptydir1",
						MountPath:  "/data/path",
					},
				},
			},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
					},
				},
			},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
						VolumeMounts: []k8score.VolumeMount{
							{
								Name:      "emptydir1",
								MountPath: "/data/path",
							},
						},
					},
				},
				Volumes: []k8score.Volume{
					{
						Name: "emptydir1",
						VolumeSource: k8score.VolumeSource{
							EmptyDir: &k8score.EmptyDirVolumeSource{},
						},
					},
				},
			},
		},
		{
			"Valid - emptydir mount with medium and size limit",
			&kubernetesplatform.KubernetesExecutorConfig{
				EmptyDirMounts: []*kubernetesplatform.EmptyDirMount{
					{
						VolumeName: "emptydir1",
						MountPath:  "/data/path",
						Medium:     &medium,
						SizeLimit:  &sizeLimit,
					},
				},
			},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
					},
				},
			},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
						VolumeMounts: []k8score.VolumeMount{
							{
								Name:      "emptydir1",
								MountPath: "/data/path",
							},
						},
					},
				},
				Volumes: []k8score.Volume{
					{
						Name: "emptydir1",
						VolumeSource: k8score.VolumeSource{
							EmptyDir: &k8score.EmptyDirVolumeSource{
								Medium:    k8score.StorageMedium(medium),
								SizeLimit: sizeLimitResource,
							},
						},
					},
				},
			},
		},
		{
			"Valid - multiple emptydir mounts",
			&kubernetesplatform.KubernetesExecutorConfig{
				EmptyDirMounts: []*kubernetesplatform.EmptyDirMount{
					{
						VolumeName: "emptydir1",
						MountPath:  "/data/path",
					},
					{
						VolumeName: "emptydir2",
						MountPath:  "/data/path2",
					},
				},
			},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
					},
				},
			},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
						VolumeMounts: []k8score.VolumeMount{
							{
								Name:      "emptydir1",
								MountPath: "/data/path",
							},
							{
								Name:      "emptydir2",
								MountPath: "/data/path2",
							},
						},
					},
				},
				Volumes: []k8score.Volume{
					{
						Name: "emptydir1",
						VolumeSource: k8score.VolumeSource{
							EmptyDir: &k8score.EmptyDirVolumeSource{},
						},
					},
					{
						Name: "emptydir2",
						VolumeSource: k8score.VolumeSource{
							EmptyDir: &k8score.EmptyDirVolumeSource{},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := extendPodSpecPatch(
				context.Background(),
				tt.podSpec,
				Options{KubernetesExecutorConfig: tt.k8sExecCfg},
				nil,
				nil,
				nil,
				map[string]*structpb.Value{},
			)
			assert.Nil(t, err)
			assert.Equal(t, tt.expected, tt.podSpec)
		})
	}
}

func Test_extendPodSpecPatch_ImagePullSecrets(t *testing.T) {
	tests := []struct {
		name        string
		k8sExecCfg  *kubernetesplatform.KubernetesExecutorConfig
		expected    *k8score.PodSpec
		inputParams map[string]*structpb.Value
	}{
		{
			"Valid - SecretA and SecretB (deprecated)",
			&kubernetesplatform.KubernetesExecutorConfig{
				ImagePullSecret: []*kubernetesplatform.ImagePullSecret{
					{SecretName: "SecretA"},
					{SecretName: "SecretB"},
				},
			},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
					},
				},
				ImagePullSecrets: []k8score.LocalObjectReference{
					{Name: "SecretA"},
					{Name: "SecretB"},
				},
			},
			nil,
		},
		{
			"Valid - SecretA and SecretB",
			&kubernetesplatform.KubernetesExecutorConfig{
				ImagePullSecret: []*kubernetesplatform.ImagePullSecret{
					{SecretName: "SecretA", SecretNameParameter: inputParamConstant("SecretA")},
					{SecretName: "SecretB", SecretNameParameter: inputParamConstant("SecretB")},
				},
			},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
					},
				},
				ImagePullSecrets: []k8score.LocalObjectReference{
					{Name: "SecretA"},
					{Name: "SecretB"},
				},
			},
			nil,
		},
		{
			"Valid - No ImagePullSecrets",
			&kubernetesplatform.KubernetesExecutorConfig{
				ImagePullSecret: []*kubernetesplatform.ImagePullSecret{},
			},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
					},
				},
			},
			nil,
		},
		{
			"Valid - empty",
			&kubernetesplatform.KubernetesExecutorConfig{},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
					},
				},
			},
			nil,
		},
		{
			"Valid - multiple input parameter secret names",
			&kubernetesplatform.KubernetesExecutorConfig{
				ImagePullSecret: []*kubernetesplatform.ImagePullSecret{
					{SecretName: "not-used1", SecretNameParameter: inputParamComponent("param_1")},
					{SecretName: "not-used2", SecretNameParameter: inputParamComponent("param_2")},
				},
			},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
					},
				},
				ImagePullSecrets: []k8score.LocalObjectReference{
					{Name: "secret-name-1"},
					{Name: "secret-name-2"},
				},
			},
			map[string]*structpb.Value{
				"param_1": structpb.NewStringValue("secret-name-1"),
				"param_2": structpb.NewStringValue("secret-name-2"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := &k8score.PodSpec{Containers: []k8score.Container{
				{
					Name: "main",
				},
			}}
			err := extendPodSpecPatch(
				context.Background(),
				got,
				Options{KubernetesExecutorConfig: tt.k8sExecCfg},
				nil,
				nil,
				nil,
				tt.inputParams,
			)
			assert.Nil(t, err)
			assert.NotNil(t, got)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func Test_extendPodSpecPatch_Tolerations(t *testing.T) {
	tests := []struct {
		name        string
		k8sExecCfg  *kubernetesplatform.KubernetesExecutorConfig
		expected    *k8score.PodSpec
		inputParams map[string]*structpb.Value
	}{
		{
			"Valid - toleration",
			&kubernetesplatform.KubernetesExecutorConfig{
				Tolerations: []*kubernetesplatform.Toleration{
					{
						Key:      "key1",
						Operator: "Equal",
						Value:    "value1",
						Effect:   "NoSchedule",
					},
				},
			},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
					},
				},
				Tolerations: []k8score.Toleration{
					{
						Key:               "key1",
						Operator:          "Equal",
						Value:             "value1",
						Effect:            "NoSchedule",
						TolerationSeconds: nil,
					},
				},
			},
			nil,
		},
		{
			"Valid - no tolerations",
			&kubernetesplatform.KubernetesExecutorConfig{},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
					},
				},
			},
			nil,
		},
		{
			"Valid - only pass operator",
			&kubernetesplatform.KubernetesExecutorConfig{
				Tolerations: []*kubernetesplatform.Toleration{
					{
						Operator: "Contains",
					},
				},
			},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
					},
				},
				Tolerations: []k8score.Toleration{
					{
						Operator: "Contains",
					},
				},
			},
			nil,
		},
		{
			"Valid - toleration json - constant",
			&kubernetesplatform.KubernetesExecutorConfig{
				Tolerations: []*kubernetesplatform.Toleration{
					{
						TolerationJson: structInputParamConstant(map[string]interface{}{
							"key":               "key1",
							"operator":          "Equal",
							"value":             "value1",
							"effect":            "NoSchedule",
							"tolerationSeconds": nil,
						}),
					},
				},
			},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
					},
				},
				Tolerations: []k8score.Toleration{
					{
						Key:               "key1",
						Operator:          "Equal",
						Value:             "value1",
						Effect:            "NoSchedule",
						TolerationSeconds: nil,
					},
				},
			},
			nil,
		},
		{
			"Valid - toleration json - component input",
			&kubernetesplatform.KubernetesExecutorConfig{
				Tolerations: []*kubernetesplatform.Toleration{
					{
						TolerationJson: inputParamComponent("param_1"),
					},
				},
			},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
					},
				},
				Tolerations: []k8score.Toleration{
					{
						Key:               "key1",
						Operator:          "Equal",
						Value:             "value1",
						Effect:            "NoSchedule",
						TolerationSeconds: int64Ptr(3600),
					},
				},
			},
			map[string]*structpb.Value{
				"param_1": validValueStructOrPanic(map[string]interface{}{
					"key":               "key1",
					"operator":          "Equal",
					"value":             "value1",
					"effect":            "NoSchedule",
					"tolerationSeconds": 3600,
				}),
			},
		},

		{
			"Valid - toleration json - empty component input",
			&kubernetesplatform.KubernetesExecutorConfig{
				Tolerations: []*kubernetesplatform.Toleration{
					{
						TolerationJson: inputParamComponent("param_1"),
					},
				},
			},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
					},
				},
				Tolerations: nil,
			},
			map[string]*structpb.Value{
				"param_1": validValueStructOrPanic(map[string]interface{}{}),
			},
		},

		{
			"Valid - toleration json - multiple input types",
			&kubernetesplatform.KubernetesExecutorConfig{
				Tolerations: []*kubernetesplatform.Toleration{
					{
						TolerationJson: inputParamComponent("param_1"),
					},
					{
						TolerationJson: structInputParamConstant(map[string]interface{}{
							"key":               "key2",
							"operator":          "Equal",
							"value":             "value2",
							"effect":            "NoSchedule",
							"tolerationSeconds": 3602,
						}),
						// Json takes precedence, these should not get used
						Key:   "key3",
						Value: "value3",
					},
					{
						Key:               "key4",
						Operator:          "Equal",
						Value:             "value4",
						Effect:            "NoSchedule",
						TolerationSeconds: int64Ptr(3604),
					},
				},
			},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
					},
				},
				Tolerations: []k8score.Toleration{
					{
						Key:               "key1",
						Operator:          "Equal",
						Value:             "value1",
						Effect:            "NoSchedule",
						TolerationSeconds: int64Ptr(3601),
					},
					{
						Key:               "key2",
						Operator:          "Equal",
						Value:             "value2",
						Effect:            "NoSchedule",
						TolerationSeconds: int64Ptr(3602),
					},
					{
						Key:               "key4",
						Operator:          "Equal",
						Value:             "value4",
						Effect:            "NoSchedule",
						TolerationSeconds: int64Ptr(3604),
					},
				},
			},
			map[string]*structpb.Value{
				"param_1": validValueStructOrPanic(map[string]interface{}{
					"key":               "key1",
					"operator":          "Equal",
					"value":             "value1",
					"effect":            "NoSchedule",
					"tolerationSeconds": 3601,
				}),
			},
		},
		{
			"Valid - toleration json - toleration list",
			&kubernetesplatform.KubernetesExecutorConfig{
				Tolerations: []*kubernetesplatform.Toleration{
					{
						TolerationJson: inputParamComponent("param_1"),
					},
				},
			},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
					},
				},
				Tolerations: []k8score.Toleration{
					{
						Key:               "key1",
						Operator:          "Equal",
						Value:             "value1",
						Effect:            "NoSchedule",
						TolerationSeconds: int64Ptr(3601),
					},
					{
						Key:               "key2",
						Operator:          "Equal",
						Value:             "value2",
						Effect:            "NoSchedule",
						TolerationSeconds: int64Ptr(3602),
					},
					{
						Key:               "key3",
						Operator:          "Equal",
						Value:             "value3",
						Effect:            "NoSchedule",
						TolerationSeconds: int64Ptr(3603),
					},
				},
			},
			map[string]*structpb.Value{
				"param_1": validListOfStructsOrPanic([]map[string]interface{}{
					{
						"key":               "key1",
						"operator":          "Equal",
						"value":             "value1",
						"effect":            "NoSchedule",
						"tolerationSeconds": 3601,
					},
					{
						"key":               "key2",
						"operator":          "Equal",
						"value":             "value2",
						"effect":            "NoSchedule",
						"tolerationSeconds": 3602,
					},
					{
						"key":               "key3",
						"operator":          "Equal",
						"value":             "value3",
						"effect":            "NoSchedule",
						"tolerationSeconds": 3603,
					},
				}),
			},
		},
		{
			"Valid - toleration json - list toleration & single toleration & constant toleration",
			&kubernetesplatform.KubernetesExecutorConfig{
				Tolerations: []*kubernetesplatform.Toleration{
					{
						TolerationJson: inputParamComponent("param_1"),
					},
					{
						TolerationJson: inputParamComponent("param_2"),
					},
					{
						Key:      "key5",
						Operator: "Equal",
						Value:    "value5",
						Effect:   "NoSchedule",
					},
				},
			},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
					},
				},
				Tolerations: []k8score.Toleration{
					{
						Key:               "key1",
						Operator:          "Equal",
						Value:             "value1",
						Effect:            "NoSchedule",
						TolerationSeconds: int64Ptr(3601),
					},
					{
						Key:               "key2",
						Operator:          "Equal",
						Value:             "value2",
						Effect:            "NoSchedule",
						TolerationSeconds: int64Ptr(3602),
					},
					{
						Key:               "key3",
						Operator:          "Equal",
						Value:             "value3",
						Effect:            "NoSchedule",
						TolerationSeconds: int64Ptr(3603),
					},
					{
						Key:               "key4",
						Operator:          "Equal",
						Value:             "value4",
						Effect:            "NoSchedule",
						TolerationSeconds: int64Ptr(3604),
					},
					{
						Key:      "key5",
						Operator: "Equal",
						Value:    "value5",
						Effect:   "NoSchedule",
					},
				},
			},
			map[string]*structpb.Value{
				"param_1": validListOfStructsOrPanic([]map[string]interface{}{
					{
						"key":               "key1",
						"operator":          "Equal",
						"value":             "value1",
						"effect":            "NoSchedule",
						"tolerationSeconds": 3601,
					},
					{
						"key":               "key2",
						"operator":          "Equal",
						"value":             "value2",
						"effect":            "NoSchedule",
						"tolerationSeconds": 3602,
					},
					{
						"key":               "key3",
						"operator":          "Equal",
						"value":             "value3",
						"effect":            "NoSchedule",
						"tolerationSeconds": 3603,
					},
				}),
				"param_2": validValueStructOrPanic(map[string]interface{}{
					"key":               "key4",
					"operator":          "Equal",
					"value":             "value4",
					"effect":            "NoSchedule",
					"tolerationSeconds": 3604,
				}),
			},
		},

		{
			"Valid - toleration json - empty toleration list",
			&kubernetesplatform.KubernetesExecutorConfig{
				Tolerations: []*kubernetesplatform.Toleration{
					{
						TolerationJson: inputParamComponent("param_1"),
					},
				},
			},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
					},
				},
				Tolerations: nil,
			},
			map[string]*structpb.Value{
				"param_1": validListOfStructsOrPanic([]map[string]interface{}{}),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := &k8score.PodSpec{Containers: []k8score.Container{
				{
					Name: "main",
				},
			}}
			err := extendPodSpecPatch(
				context.Background(),
				got,
				Options{KubernetesExecutorConfig: tt.k8sExecCfg},
				nil,
				nil,
				nil,
				tt.inputParams,
			)
			assert.Nil(t, err)
			assert.NotNil(t, got)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func Test_extendPodSpecPatch_FieldPathAsEnv(t *testing.T) {
	tests := []struct {
		name       string
		k8sExecCfg *kubernetesplatform.KubernetesExecutorConfig
		expected   *k8score.PodSpec
	}{
		{
			"Valid - FieldPathAsEnv",
			&kubernetesplatform.KubernetesExecutorConfig{
				FieldPathAsEnv: []*kubernetesplatform.FieldPathAsEnv{
					{Name: "KFP_RUN_NAME", FieldPath: "metadata.annotations['pipelines.kubeflow.org/run_name']"},
				},
			},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
						Env: []k8score.EnvVar{
							{
								Name: "KFP_RUN_NAME",
								ValueFrom: &k8score.EnvVarSource{
									FieldRef: &k8score.ObjectFieldSelector{
										FieldPath: "metadata.annotations['pipelines.kubeflow.org/run_name']",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			"Valid - Mix env values",
			&kubernetesplatform.KubernetesExecutorConfig{
				SecretAsEnv: []*kubernetesplatform.SecretAsEnv{
					{
						SecretName:          "my-secret",
						SecretNameParameter: inputParamConstant("my-secret"),
						KeyToEnv: []*kubernetesplatform.SecretAsEnv_SecretKeyToEnvMap{
							{
								SecretKey: "password",
								EnvVar:    "SECRET_VAR",
							},
						},
					},
				},
				FieldPathAsEnv: []*kubernetesplatform.FieldPathAsEnv{
					{Name: "KFP_RUN_NAME", FieldPath: "metadata.annotations['pipelines.kubeflow.org/run_name']"},
				},
			},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
						Env: []k8score.EnvVar{
							{
								Name: "SECRET_VAR",
								ValueFrom: &k8score.EnvVarSource{
									SecretKeyRef: &k8score.SecretKeySelector{
										LocalObjectReference: k8score.LocalObjectReference{Name: "my-secret"},
										Key:                  "password",
										Optional:             nil,
									},
								},
							},
							{
								Name: "KFP_RUN_NAME",
								ValueFrom: &k8score.EnvVarSource{
									FieldRef: &k8score.ObjectFieldSelector{
										FieldPath: "metadata.annotations['pipelines.kubeflow.org/run_name']",
									},
								},
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := &k8score.PodSpec{Containers: []k8score.Container{
				{
					Name: "main",
				},
			}}
			err := extendPodSpecPatch(
				context.Background(),
				got,
				Options{KubernetesExecutorConfig: tt.k8sExecCfg},
				nil,
				nil,
				nil,
				map[string]*structpb.Value{},
			)
			assert.Nil(t, err)
			assert.NotNil(t, got)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func Test_extendPodSpecPatch_ActiveDeadlineSeconds(t *testing.T) {
	var timeoutSeconds int64 = 20
	var NegativeTimeoutSeconds int64 = -20
	tests := []struct {
		name       string
		k8sExecCfg *kubernetesplatform.KubernetesExecutorConfig
		expected   *k8score.PodSpec
	}{
		{
			"Valid - With ActiveDeadlineSeconds",
			&kubernetesplatform.KubernetesExecutorConfig{
				ActiveDeadlineSeconds: timeoutSeconds,
			},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
					},
				},
				ActiveDeadlineSeconds: &timeoutSeconds,
			},
		},
		{
			"Valid - Negative input ignored",
			&kubernetesplatform.KubernetesExecutorConfig{
				ActiveDeadlineSeconds: NegativeTimeoutSeconds,
			},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
					},
				},
			},
		},
		{
			"Valid - No ActiveDeadlineSeconds",
			&kubernetesplatform.KubernetesExecutorConfig{},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := &k8score.PodSpec{Containers: []k8score.Container{
				{
					Name: "main",
				},
			}}
			err := extendPodSpecPatch(
				context.Background(),
				got,
				Options{KubernetesExecutorConfig: tt.k8sExecCfg},
				nil,
				nil,
				nil,
				map[string]*structpb.Value{},
			)
			assert.Nil(t, err)
			assert.NotNil(t, got)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func Test_extendPodSpecPatch_ImagePullPolicy(t *testing.T) {
	tests := []struct {
		name       string
		k8sExecCfg *kubernetesplatform.KubernetesExecutorConfig
		podSpec    *k8score.PodSpec
		expected   *k8score.PodSpec
	}{
		{
			"Valid - Always",
			&kubernetesplatform.KubernetesExecutorConfig{
				ImagePullPolicy: "Always",
			},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
					},
				},
			},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name:            "main",
						ImagePullPolicy: "Always",
					},
				},
			},
		},
		{
			"Valid - IfNotPresent",
			&kubernetesplatform.KubernetesExecutorConfig{
				ImagePullPolicy: "IfNotPresent",
			},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
					},
				},
			},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name:            "main",
						ImagePullPolicy: "IfNotPresent",
					},
				},
			},
		},
		{
			"Valid - Never",
			&kubernetesplatform.KubernetesExecutorConfig{
				ImagePullPolicy: "Never",
			},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
					},
				},
			},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name:            "main",
						ImagePullPolicy: "Never",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := extendPodSpecPatch(
				context.Background(),
				tt.podSpec,
				Options{KubernetesExecutorConfig: tt.k8sExecCfg},
				nil,
				nil,
				nil,
				map[string]*structpb.Value{},
			)
			assert.Nil(t, err)
			assert.Equal(t, tt.expected, tt.podSpec)
		})
	}
}

func Test_extendPodSpecPatch_GenericEphemeralVolume(t *testing.T) {
	storageClass := "storageClass"
	tests := []struct {
		name       string
		k8sExecCfg *kubernetesplatform.KubernetesExecutorConfig
		podSpec    *k8score.PodSpec
		expected   *k8score.PodSpec
	}{
		{
			"Valid - single volume added (default storage class)",
			&kubernetesplatform.KubernetesExecutorConfig{
				GenericEphemeralVolume: []*kubernetesplatform.GenericEphemeralVolume{
					{
						VolumeName:          "volume",
						MountPath:           "/data/path",
						AccessModes:         []string{"ReadWriteOnce"},
						Size:                "5Gi",
						DefaultStorageClass: true,
					},
				},
			},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
					},
				},
			},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
						VolumeMounts: []k8score.VolumeMount{
							{
								Name:      "volume",
								MountPath: "/data/path",
							},
						},
					},
				},
				Volumes: []k8score.Volume{
					{
						Name: "volume",
						VolumeSource: k8score.VolumeSource{
							Ephemeral: &k8score.EphemeralVolumeSource{
								VolumeClaimTemplate: &k8score.PersistentVolumeClaimTemplate{
									Spec: k8score.PersistentVolumeClaimSpec{
										AccessModes: []k8score.PersistentVolumeAccessMode{k8score.ReadWriteOnce},
										Resources: k8score.VolumeResourceRequirements{
											Requests: k8score.ResourceList{
												k8score.ResourceStorage: k8sres.MustParse("5Gi"),
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			"Valid - no generic volumes specified",
			&kubernetesplatform.KubernetesExecutorConfig{},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
					},
				},
			},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
					},
				},
			},
		},
		{
			"Valid - multiple volumes specified (one with labels, one with storage class)",
			&kubernetesplatform.KubernetesExecutorConfig{
				GenericEphemeralVolume: []*kubernetesplatform.GenericEphemeralVolume{
					{
						VolumeName:          "volume",
						MountPath:           "/data/path",
						AccessModes:         []string{"ReadWriteOnce"},
						Size:                "5Gi",
						DefaultStorageClass: true,
					},
					{
						VolumeName:       "volume2",
						MountPath:        "/data/path2",
						AccessModes:      []string{"ReadWriteOnce"},
						Size:             "10Gi",
						StorageClassName: storageClass,
						Metadata: &kubernetesplatform.PodMetadata{
							Annotations: map[string]string{
								"annotation1": "a1",
							},
							Labels: map[string]string{
								"label1": "l1",
							},
						},
					},
				},
			},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
					},
				},
			},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
						VolumeMounts: []k8score.VolumeMount{
							{
								Name:      "volume",
								MountPath: "/data/path",
							},
							{
								Name:      "volume2",
								MountPath: "/data/path2",
							},
						},
					},
				},
				Volumes: []k8score.Volume{
					{
						Name: "volume",
						VolumeSource: k8score.VolumeSource{
							Ephemeral: &k8score.EphemeralVolumeSource{
								VolumeClaimTemplate: &k8score.PersistentVolumeClaimTemplate{
									Spec: k8score.PersistentVolumeClaimSpec{
										AccessModes: []k8score.PersistentVolumeAccessMode{k8score.ReadWriteOnce},
										Resources: k8score.VolumeResourceRequirements{
											Requests: k8score.ResourceList{
												k8score.ResourceStorage: k8sres.MustParse("5Gi"),
											},
										},
									},
								},
							},
						},
					},
					{
						Name: "volume2",
						VolumeSource: k8score.VolumeSource{
							Ephemeral: &k8score.EphemeralVolumeSource{
								VolumeClaimTemplate: &k8score.PersistentVolumeClaimTemplate{
									ObjectMeta: metav1.ObjectMeta{
										Annotations: map[string]string{
											"annotation1": "a1",
										},
										Labels: map[string]string{
											"label1": "l1",
										},
									},
									Spec: k8score.PersistentVolumeClaimSpec{
										AccessModes: []k8score.PersistentVolumeAccessMode{k8score.ReadWriteOnce},
										Resources: k8score.VolumeResourceRequirements{
											Requests: k8score.ResourceList{
												k8score.ResourceStorage: k8sres.MustParse("10Gi"),
											},
										},
										StorageClassName: &storageClass,
									},
								},
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := extendPodSpecPatch(
				context.Background(),
				tt.podSpec,
				Options{KubernetesExecutorConfig: tt.k8sExecCfg},
				nil,
				nil,
				nil,
				map[string]*structpb.Value{},
			)
			assert.Nil(t, err)
			assert.Equal(t, tt.expected, tt.podSpec)
		})
	}
}

func validListOfStructsOrPanic(data []map[string]interface{}) *structpb.Value {
	var listValues []*structpb.Value
	for _, item := range data {
		s, err := structpb.NewStruct(item)
		if err != nil {
			panic(err)
		}
		listValues = append(listValues, structpb.NewStructValue(s))
	}
	return structpb.NewListValue(&structpb.ListValue{Values: listValues})
}

func validValueStructOrPanic(data map[string]interface{}) *structpb.Value {
	s, err := structpb.NewStruct(data)
	if err != nil {
		panic(err)
	}
	return structpb.NewStructValue(s)
}

func structInputParamConstant(value map[string]interface{}) *pipelinespec.TaskInputsSpec_InputParameterSpec {
	return &pipelinespec.TaskInputsSpec_InputParameterSpec{
		Kind: &pipelinespec.TaskInputsSpec_InputParameterSpec_RuntimeValue{
			RuntimeValue: &pipelinespec.ValueOrRuntimeParameter{
				Value: &pipelinespec.ValueOrRuntimeParameter_Constant{
					Constant: validValueStructOrPanic(value),
				},
			},
		},
	}
}

func int64Ptr(val int64) *int64 {
	return &val
}
