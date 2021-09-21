package resource

import "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"

func AddRuntimeMetadata(wf *v1alpha1.Workflow) {
	template := wf.Spec.Templates[0]
	template.Metadata.Annotations = map[string]string{"sidecar.istio.io/inject": "false"}
	template.Metadata.Labels = map[string]string{"pipelines.kubeflow.org/cache_enabled": "true"}
	wf.Spec.Templates[0] = template
}
