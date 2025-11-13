package api

type DriverPluginArgs struct {
	CachedDecisionPath   string `json:"cached_decision_path"`
	Component            string `json:"component,omitempty"`
	Container            string `json:"container,omitempty"`
	DagExecutionID       string `json:"dag_execution_id"`
	IterationIndex       string `json:"iteration_index"`
	HttpProxy            string `json:"http_proxy"`
	HttpsProxy           string `json:"https_proxy"`
	NoProxy              string `json:"no_proxy"`
	KubernetesConfig     string `json:"kubernetes_config,omitempty"`
	RuntimeConfig        string `json:"runtime_config,omitempty"`
	PipelineName         string `json:"pipeline_name"`
	PublishLogs          string `json:"publish_logs,omitempty"`
	RunID                string `json:"run_id"`
	RunName              string `json:"run_name"`
	RunDisplayName       string `json:"run_display_name"`
	TaskName             string `json:"task_name"`
	Task                 string `json:"task"`
	Type                 string `json:"type"`
	CacheDisabledFlag    bool   `json:"cache_disabled"`
	ExecutionIdPath      string `json:"execution_id_path"`
	IterationCountPath   string `json:"iteration_count_path"`
	ConditionPath        string `json:"condition_path"`
	PodSpecPathPath      string `json:"pod_spec_patch_path"`
	MlPipelineTLSEnabled bool   `json:"ml_pipeline_tls_enabled"`
	MetadataTLSEnabled   bool   `json:"metadata_tls_enabled"`
	CACertPath           string `json:"ca_cert_path"`
}

type DriverPlugin struct {
	DriverPlugin *DriverPluginContainer `json:"driver-plugin"`
}

type DriverPluginContainer struct {
	Args *DriverPluginArgs `json:"args"`
}

type DriverTemplate struct {
	Plugin *DriverPlugin `json:"plugin"`
}

type DriverRequest struct {
	Template *DriverTemplate `json:"template"`
}
