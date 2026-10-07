// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy at http://www.apache.org/licenses/LICENSE-2.0

package history

import "encoding/json"

// These DTOs freeze release-2.18's archive wire format. Field order and anonymous
// embeds are intentional: its digest hashes encoding/json output of the structs.
// Never substitute current storage models here or add fields to this contract.
const legacyArchiveFormat = "kfp-namespace-transfer-mlmd-2.18/v1"
const legacyArchiveFormatV2 = "kfp-namespace-transfer-mlmd-2.18/v2"

type legacyArchive struct {
	Format            string                    `json:"format"`
	Source            string                    `json:"source"`
	Namespace         string                    `json:"namespace"`
	RuntimeNamespace  string                    `json:"runtime_namespace"`
	Schema            string                    `json:"schema"`
	Experiments       []legacyExperiment        `json:"experiments"`
	Pipelines         []legacyPipeline          `json:"pipelines"`
	CatalogDefaults   map[string]string         `json:"catalog_defaults,omitempty"`
	Versions          []legacyPipelineVersion   `json:"pipeline_versions"`
	Schedules         []legacyJob               `json:"schedules"`
	Runs              []legacyRunHistory        `json:"runs"`
	References        []legacyResourceReference `json:"references"`
	Metadata          legacyGraph               `json:"metadata"`
	Digest            string                    `json:"digest"`
	RuntimeParameters *legacyRuntimeParameters  `json:"runtime_parameters,omitempty"`
}

type legacyRuntimeParameters struct {
	Runs      map[string]string `json:"runs"`
	Schedules map[string]string `json:"schedules"`
}

type legacyRunHistory struct {
	Run     legacyRun         `json:"run"`
	Tasks   []legacyTask      `json:"tasks"`
	Metrics []legacyRunMetric `json:"metrics"`
}

type legacyGraph map[string][]map[string]any

type legacyExperiment struct {
	UUID                  string
	Name                  string
	Description           string
	CreatedAtInSec        int64
	LastRunCreatedAtInSec int64
	Namespace             string
	StorageState          string
}

type legacyPipeline struct {
	UUID           string
	CreatedAtInSec int64
	Name           string
	DisplayName    string
	Description    string
	Parameters     string
	Status         string
	// nolint:staticcheck // [ST1003] Released field spelling is part of the archive digest contract.
	DefaultVersionId string
	Namespace        string
	Tags             map[string]string
}

type legacyPipelineVersion struct {
	UUID           string
	CreatedAtInSec int64
	Name           string
	DisplayName    string
	Parameters     string
	// nolint:staticcheck // [ST1003] Released field spelling is part of the archive digest contract.
	PipelineId string
	Pipeline   legacyPipeline
	Status     string
	// nolint:staticcheck // [ST1003] Released field spelling is part of the archive digest contract.
	CodeSourceUrl   string
	Description     string
	PipelineSpec    string
	PipelineSpecURI string
	Tags            map[string]string
}

type legacyPipelineSpec struct {
	// nolint:staticcheck // [ST1003] Released field spelling is part of the archive digest contract.
	PipelineId string
	// nolint:staticcheck // [ST1003] Released field spelling is part of the archive digest contract.
	PipelineVersionId    string
	PipelineName         string
	PipelineSpecManifest string
	WorkflowSpecManifest string
	Parameters           string
	legacyRuntimeConfig
}

type legacyRuntimeConfig struct {
	Parameters   string
	PipelineRoot string
}

type legacyRun struct {
	UUID        string
	DisplayName string
	K8SName     string
	Description string
	Namespace   string
	// nolint:staticcheck // [ST1003] Released field spelling is part of the archive digest contract.
	ExperimentId string
	// nolint:staticcheck // [ST1003] Released field spelling is part of the archive digest contract.
	RecurringRunId     string
	ImportedFrom       *string
	ImportDigest       *string
	StorageState       string
	ServiceAccount     string
	Metrics            []*legacyRunMetric
	ResourceReferences []*legacyResourceReference
	legacyPipelineSpec
	legacyRunDetails
}

type legacyRunDetails struct {
	CreatedAtInSec          int64
	ScheduledAtInSec        int64
	FinishedAtInSec         int64
	Conditions              string
	State                   string
	StateHistoryString      string
	StateHistory            []*legacyRuntimeStatus
	PluginsInputString      *string
	PluginsOutputString     *string
	PipelineRuntimeManifest string
	WorkflowRuntimeManifest string
	// nolint:staticcheck // [ST1003] Released field spelling is part of the archive digest contract.
	PipelineContextId int64
	// nolint:staticcheck // [ST1003] Released field spelling is part of the archive digest contract.
	PipelineRunContextId int64
	RetryGeneration      int64
	RetryClaimedAtInSec  int64
	ArchivedAtInSec      int64
	TaskDetails          []*legacyTask
}

type legacyRunMetric struct {
	RunUUID     string
	NodeID      string
	Name        string
	NumberValue float64
	Format      string
	Payload     string
}

type legacyRuntimeStatus struct {
	UpdateTimeInSec int64           `json:"UpdateTimeInSec,omitempty"`
	State           string          `json:"State,omitempty"`
	Error           json.RawMessage `json:"Error,omitempty"`
}

type legacyTask struct {
	UUID              string
	Namespace         string
	PipelineName      string
	RunID             string
	Run               legacyRun
	PodName           string
	MLMDExecutionID   string
	CreatedTimestamp  int64
	StartedTimestamp  int64
	FinishedTimestamp int64
	Fingerprint       string
	Name              string
	// nolint:staticcheck // [ST1003] Released field spelling is part of the archive digest contract.
	ParentTaskId       string
	State              string
	StateHistoryString string
	MLMDInputs         string
	MLMDOutputs        string
	ChildrenPodsString string
	StateHistory       []*legacyRuntimeStatus
	ChildrenPods       []string
	Payload            string
}

type legacyJob struct {
	UUID           string
	DisplayName    string
	K8SName        string
	Namespace      string
	ServiceAccount string
	Description    string
	MaxConcurrency int64
	NoCatchup      bool
	CreatedAtInSec int64
	UpdatedAtInSec int64
	Enabled        bool
	// nolint:staticcheck // [ST1003] Released field spelling is part of the archive digest contract.
	ExperimentId       string
	ResourceReferences []*legacyResourceReference
	legacyTrigger
	legacyPipelineSpec
	Conditions         string
	PluginsInputString *string
}

type legacyTrigger struct {
	legacyCronSchedule
	legacyPeriodicSchedule
}

type legacyCronSchedule struct {
	CronScheduleStartTimeInSec *int64
	CronScheduleEndTimeInSec   *int64
	Cron                       *string
}

type legacyPeriodicSchedule struct {
	PeriodicScheduleStartTimeInSec *int64
	PeriodicScheduleEndTimeInSec   *int64
	IntervalSecond                 *int64
}

type legacyResourceReference struct {
	ResourceUUID  string
	ResourceType  string
	ReferenceUUID string
	ReferenceName string
	ReferenceType string
	Relationship  string
	Payload       string
}
