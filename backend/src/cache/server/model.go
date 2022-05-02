package server

type ExecutionTemplate struct {
	Name   string `json:"name"`
	Inputs struct {
		Artifacts []struct {
			Name string `json:"name"`
			Path string `json:"path"`
			Raw  struct {
				Data string `json:"data"`
			} `json:"raw"`
		} `json:"artifacts"`
	} `json:"inputs"`
	Outputs struct {
		Artifacts []struct {
			Name string `json:"name"`
			Path string `json:"path"`
		} `json:"artifacts"`
	} `json:"outputs"`
	Metadata struct {
		Annotations struct {
			PipelinesKubeflowOrgComponentRef  string `json:"pipelines.kubeflow.org/component_ref"`
			PipelinesKubeflowOrgComponentSpec string `json:"pipelines.kubeflow.org/component_spec"`
			SidecarIstioIoInject              string `json:"sidecar.istio.io/inject"`
		} `json:"annotations"`
		Labels struct {
			PipelinesKubeflowOrgCacheEnabled    string `json:"pipelines.kubeflow.org/cache_enabled"`
			PipelinesKubeflowOrgEnableCaching   string `json:"pipelines.kubeflow.org/enable_caching"`
			PipelinesKubeflowOrgKfpSdkVersion   string `json:"pipelines.kubeflow.org/kfp_sdk_version"`
			PipelinesKubeflowOrgPipelineSdkType string `json:"pipelines.kubeflow.org/pipeline-sdk-type"`
		} `json:"labels"`
	} `json:"metadata"`
	Container struct {
		Name      string   `json:"name"`
		Image     string   `json:"image"`
		Command   []string `json:"command"`
		Args      []string `json:"args"`
		Resources struct {
		} `json:"resources"`
	} `json:"container"`
	ArchiveLocation struct {
		ArchiveLogs bool `json:"archiveLogs"`
		S3          struct {
			Endpoint        string `json:"endpoint"`
			Bucket          string `json:"bucket"`
			Insecure        bool   `json:"insecure"`
			AccessKeySecret struct {
				Name string `json:"name"`
				Key  string `json:"key"`
			} `json:"accessKeySecret"`
			SecretKeySecret struct {
				Name string `json:"name"`
				Key  string `json:"key"`
			} `json:"secretKeySecret"`
			Key string `json:"key"`
		} `json:"s3"`
	} `json:"archiveLocation"`
}

type ExecutionOutput struct {
	PipelinesKubeflowOrgMetadataExecutionID string `json:"pipelines.kubeflow.org/metadata_execution_id"`
	WorkflowsArgoprojIoOutputs              string `json:"workflows.argoproj.io/outputs"`
}

type WorkflowsArgoprojIoOutputs struct {
	Artifacts []struct {
		Name string `json:"name"`
		S3   struct {
			Key string `json:"key"`
		} `json:"s3"`
	} `json:"artifacts"`
}
