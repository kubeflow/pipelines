// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package api

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	pipelineparams "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_client/pipeline_service"
	"github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_model"
	uploadparams "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_upload_client/pipeline_upload_service"
	runparams "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/run_client/run_service"
	"github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/run_model"
	apiserver "github.com/kubeflow/pipelines/backend/src/common/client/api_server/v2"
	"github.com/kubeflow/pipelines/backend/test/config"
	"github.com/kubeflow/pipelines/backend/test/constants"
	"github.com/kubeflow/pipelines/backend/test/testutil"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/yaml"
)

var _ = Describe("Legacy API coverage migration", Label(constants.APIServerTests, constants.FullRegression, constants.PipelineCacheEnabled, "LegacyMigration"), func() {
	It("reports the configured pipeline store in healthz", Label("LegacyAPIHealth"), func() {
		var tlsConfig *tls.Config
		var err error
		if *config.TLSEnabled {
			tlsConfig, err = testutil.GetTLSConfig(*config.CaCertPath)
			Expect(err).NotTo(HaveOccurred())
		}
		client, err := apiserver.NewHealthzClient(testutil.GetClientConfig(*config.Namespace), *config.DebugMode, tlsConfig)
		Expect(err).NotTo(HaveOccurred())
		health, err := client.GetHealthz()
		Expect(err).NotTo(HaveOccurred())
		Expect(health).NotTo(BeNil())
		expectedStore := "database"
		if *config.PipelineStoreKubernetes {
			expectedStore = "kubernetes"
		}
		Expect(health.PipelineStore).To(Equal(expectedStore))
	})

	It("terminates a long-running run and preserves its API metadata", Label("LegacyAPITermination"), func() {
		name := "terminate-" + randomName
		pipeline := uploadAPipeline(filepath.Join("..", "resources", "long-running.yaml"), &name)
		version := testutil.GetLatestPipelineVersion(pipelineClient, &pipeline.PipelineID)
		experiment := createExperiment("terminate-" + randomName)
		payload := createPipelineRunPayload(&pipeline.PipelineID, &version.PipelineVersionID, &experiment.ExperimentID, nil)
		// Single-user legacy requests rely on the server-defaulted service account.
		payload.RuntimeConfig = nil
		if !*config.MultiUserMode && !*config.KubeflowMode {
			payload.ServiceAccount = ""
		}
		run, err := runClient.Create(&runparams.RunServiceCreateRunParams{Run: payload})
		Expect(err).NotTo(HaveOccurred())
		testContext.PipelineRun.CreatedRunIds = append(testContext.PipelineRun.CreatedRunIds, run.RunID)
		Expect(runClient.Terminate(&runparams.RunServiceTerminateRunParams{RunID: run.RunID})).To(Succeed())
		var terminated *run_model.V2beta1Run
		Eventually(func(g Gomega) {
			var getErr error
			terminated, getErr = runClient.Get(&runparams.RunServiceGetRunParams{RunID: run.RunID})
			g.Expect(getErr).NotTo(HaveOccurred())
			g.Expect(terminated.State).NotTo(BeNil())
			g.Expect(*terminated.State).To(BeElementOf(run_model.V2beta1RuntimeStateCANCELING, run_model.V2beta1RuntimeStateCANCELED))
		}, time.Minute, time.Second).Should(Succeed())
		expected := &run_model.V2beta1Run{
			RunID: run.RunID, Namespace: terminated.Namespace,
			DisplayName: payload.DisplayName, Description: payload.Description,
			State: terminated.State, RunDetails: terminated.RunDetails, TaskCount: terminated.TaskCount,
			StateHistory: terminated.StateHistory, StorageState: terminated.StorageState,
			ServiceAccount: testutil.GetDefaultPipelineRunnerServiceAccount(),
			PipelineSpec:   terminated.PipelineSpec, PluginsInput: terminated.PluginsInput,
			PluginsOutput: terminated.PluginsOutput, ExperimentID: experiment.ExperimentID,
			PipelineVersionReference: payload.PipelineVersionReference,
			CreatedAt:                terminated.CreatedAt, ScheduledAt: terminated.ScheduledAt, FinishedAt: terminated.FinishedAt,
		}
		Expect(terminated).To(Equal(expected))
	})

	It("uploads ZIP pipelines and versions and rejects duplicate names", Label("LegacyAPIUploads"), func() {
		file := filepath.Join("..", "resources", "arguments_parameters.zip")
		name := "zip-" + randomName
		namespace := testutil.GetNamespace()
		uploadParams := &uploadparams.UploadPipelineParams{Name: &name}
		if !*config.UploadPipelinesWithKubernetes {
			uploadParams.Namespace = &namespace
		}
		// Upload the archive intact; UploadPipeline rewrites YAML specs.
		pipeline, err := pipelineUploadClient.UploadFile(file, uploadParams)
		Expect(err).NotTo(HaveOccurred())
		testContext.Pipeline.CreatedPipelines = append(testContext.Pipeline.CreatedPipelines, pipeline)
		Expect(pipeline.DisplayName).To(Equal(name))
		_, err = pipelineUploadClient.UploadFile(file, uploadParams)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Failed to upload pipeline"))
		versionName := "zip-version-" + randomName
		params := &uploadparams.UploadPipelineVersionParams{Name: &versionName, Pipelineid: &pipeline.PipelineID}
		version, err := pipelineUploadClient.UploadPipelineVersion(file, params)
		Expect(err).NotTo(HaveOccurred())
		Expect(version.DisplayName).To(Equal(versionName))
		Expect(version.PipelineID).To(Equal(pipeline.PipelineID))
		_, err = pipelineUploadClient.UploadPipelineVersion(file, params)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Failed to upload pipeline version"))
		defaultVersion, err := pipelineUploadClient.UploadPipelineVersion(filepath.Join("..", "resources", "arguments-parameters.yaml"), &uploadparams.UploadPipelineVersionParams{Pipelineid: &pipeline.PipelineID})
		Expect(err).NotTo(HaveOccurred())
		Expect(defaultVersion.DisplayName).To(Equal("arguments-parameters.yaml"))
		for _, fixture := range []string{"arguments-parameters.yaml", "hello-world.yaml"} {
			versionName := "yaml-" + strings.TrimSuffix(fixture, ".yaml") + "-" + randomName
			sourcePath := filepath.Join("..", "resources", fixture)
			uploaded, err := pipelineUploadClient.UploadPipelineVersion(sourcePath, &uploadparams.UploadPipelineVersionParams{Name: &versionName, Pipelineid: &pipeline.PipelineID})
			Expect(err).NotTo(HaveOccurred())
			Expect(uploaded.DisplayName).To(Equal(versionName))
			var fetched *pipeline_model.V2beta1PipelineVersion
			Eventually(func(g Gomega) {
				var getErr error
				fetched, getErr = pipelineClient.GetPipelineVersion(&pipelineparams.PipelineServiceGetPipelineVersionParams{PipelineID: pipeline.PipelineID, PipelineVersionID: uploaded.PipelineVersionID})
				g.Expect(getErr).NotTo(HaveOccurred())
				g.Expect(fetched).NotTo(BeNil())
				g.Expect(fetched.DisplayName).To(Equal(versionName))
			}, informerSyncTimeout, informerSyncInterval).Should(Succeed())
			Expect(time.Time(fetched.CreatedAt).IsZero()).To(BeFalse())
			source, err := os.ReadFile(sourcePath)
			Expect(err).NotTo(HaveOccurred())
			expectedJSON, err := yaml.YAMLToJSON(source)
			Expect(err).NotTo(HaveOccurred())
			for _, spec := range []interface{}{uploaded.PipelineSpec, fetched.PipelineSpec} {
				actualJSON, err := json.Marshal(spec)
				Expect(err).NotTo(HaveOccurred())
				// Match the legacy normalization of the store-assigned pipeline name.
				normalized := strings.Replace(string(actualJSON), "pipeline/"+name, "echo", 1)
				Expect(normalized).To(MatchJSON(expectedJSON))
			}
		}

	})

	It("rejects unsupported sort fields for pipelines, versions, and runs", Label("LegacyAPISortValidation"), func() {
		name := "sort-" + randomName
		pipeline := uploadAPipeline(filepath.Join("..", "resources", "hello-world.yaml"), &name)
		unknown := "unknownfield"
		namespace := testutil.GetNamespace()
		_, _, _, err := pipelineClient.List(&pipelineparams.PipelineServiceListPipelinesParams{SortBy: &unknown, Namespace: &namespace})
		Expect(err).To(HaveOccurred())
		_, _, _, err = pipelineClient.ListPipelineVersions(&pipelineparams.PipelineServiceListPipelineVersionsParams{PipelineID: pipeline.PipelineID, SortBy: &unknown})
		Expect(err).To(HaveOccurred())
		experiment := createExperiment("sort-" + randomName)
		_, _, _, err = runClient.List(&runparams.RunServiceListRunsParams{ExperimentID: &experiment.ExperimentID, SortBy: &unknown})
		Expect(err).To(HaveOccurred())
	})
	It("imports YAML and ZIP package URLs and preserves version metadata", Label("LegacyAPIURLImports"), func() {
		namespace := testutil.GetNamespace()
		name, display, description := "url-"+randomName, "URL pipeline "+randomName, "URL imported pipeline"
		pipelineURL, err := testutil.GetRepoBranchURLRAW(*config.REPO_NAME, *config.BranchName, "backend/test/v2/resources/sequential-v2.yaml")
		Expect(err).NotTo(HaveOccurred())
		pipeline, err := pipelineClient.CreatePipelineAndVersion(&pipelineparams.PipelineServiceCreatePipelineAndVersionParams{
			Body: &pipeline_model.V2beta1CreatePipelineAndVersionRequest{
				Pipeline:        &pipeline_model.V2beta1Pipeline{Name: name, DisplayName: display, Description: description, Namespace: namespace},
				PipelineVersion: &pipeline_model.V2beta1PipelineVersion{PackageURL: &pipeline_model.V2beta1URL{PipelineURL: pipelineURL}},
			},
		})
		Expect(err).NotTo(HaveOccurred())
		testContext.Pipeline.CreatedPipelines = append(testContext.Pipeline.CreatedPipelines, toUploadModel(pipeline))
		Expect(pipeline.Name).To(Equal(name))
		Expect(pipeline.DisplayName).To(Equal(display))
		Expect(pipeline.Description).To(Equal(description))
		Eventually(func(g Gomega) {
			versions, total, _, err := pipelineClient.ListPipelineVersions(&pipelineparams.PipelineServiceListPipelineVersionsParams{PipelineID: pipeline.PipelineID})
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(total).To(Equal(1))
			g.Expect(versions).To(HaveLen(1))
			g.Expect(versions[0].Name).To(Equal(name))
			g.Expect(versions[0].DisplayName).To(Equal(display))
			g.Expect(versions[0].Description).To(Equal(description))
			g.Expect(versions[0].PipelineID).To(Equal(pipeline.PipelineID))
			g.Expect(versions[0].PackageURL).NotTo(BeNil())
			g.Expect(versions[0].PackageURL.PipelineURL).To(Equal(pipelineURL))
		}, informerSyncTimeout, informerSyncInterval).Should(Succeed())

		zipURL, err := testutil.GetRepoBranchURLRAW(*config.REPO_NAME, *config.BranchName, "backend/test/v2/resources/arguments_parameters.zip")
		Expect(err).NotTo(HaveOccurred())
		version, err := pipelineClient.CreatePipelineVersion(&pipelineparams.PipelineServiceCreatePipelineVersionParams{
			PipelineID:      pipeline.PipelineID,
			PipelineVersion: &pipeline_model.V2beta1PipelineVersion{Name: "zip-" + randomName, DisplayName: "ZIP version", Description: "ZIP import", PipelineID: pipeline.PipelineID, PackageURL: &pipeline_model.V2beta1URL{PipelineURL: zipURL}},
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(version.DisplayName).To(Equal("ZIP version"))
		Expect(version.Description).To(Equal("ZIP import"))
		Expect(version.PipelineID).To(Equal(pipeline.PipelineID))
		Expect(version.PackageURL).NotTo(BeNil())
		Expect(version.PackageURL.PipelineURL).To(Equal(zipURL))
		yamlVersion, err := pipelineClient.CreatePipelineVersion(&pipelineparams.PipelineServiceCreatePipelineVersionParams{
			PipelineID:      pipeline.PipelineID,
			PipelineVersion: &pipeline_model.V2beta1PipelineVersion{Name: "yaml-" + randomName, DisplayName: "YAML version", PipelineID: pipeline.PipelineID, PackageURL: &pipeline_model.V2beta1URL{PipelineURL: pipelineURL}},
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(yamlVersion.Name).To(Equal("yaml-" + randomName))
		Expect(yamlVersion.DisplayName).To(Equal("YAML version"))

	})

	It("creates an embedded-spec run and preserves parameters and description", Label("LegacyAPIEmbeddedRun"), func() {
		source, err := os.ReadFile(filepath.Join("..", "resources", "arguments-parameters.yaml"))
		Expect(err).NotTo(HaveOccurred())
		specJSON, err := yaml.YAMLToJSON(source)
		Expect(err).NotTo(HaveOccurred())
		var spec map[string]interface{}
		Expect(json.Unmarshal(specJSON, &spec)).To(Succeed())
		experiment := createExperiment("embedded-" + randomName)
		payload := &run_model.V2beta1Run{
			DisplayName: "embedded-" + randomName, Description: "embedded manifest with parameter overrides",
			ExperimentID: experiment.ExperimentID, PipelineSpec: spec,
			ServiceAccount: testutil.GetDefaultPipelineRunnerServiceAccount(),
			RuntimeConfig:  &run_model.V2beta1RuntimeConfig{Parameters: map[string]interface{}{"param1": "goodbye", "param2": "world"}},
		}
		run, err := runClient.Create(&runparams.RunServiceCreateRunParams{Run: payload})
		Expect(err).NotTo(HaveOccurred())
		testContext.PipelineRun.CreatedRunIds = append(testContext.PipelineRun.CreatedRunIds, run.RunID)
		fetched, err := runClient.Get(&runparams.RunServiceGetRunParams{RunID: run.RunID})
		Expect(err).NotTo(HaveOccurred())
		for _, actual := range []*run_model.V2beta1Run{run, fetched} {
			Expect(actual.DisplayName).To(Equal(payload.DisplayName))
			Expect(actual.Description).To(Equal(payload.Description))
			Expect(actual.ExperimentID).To(Equal(experiment.ExperimentID))
			Expect(actual.RuntimeConfig).To(Equal(payload.RuntimeConfig))
			Expect(actual.ServiceAccount).To(Equal(payload.ServiceAccount))
			Expect(actual.PipelineVersionReference).To(BeNil())
			actualJSON, err := json.Marshal(actual.PipelineSpec)
			Expect(err).NotTo(HaveOccurred())
			Expect(actualJSON).To(MatchJSON(specJSON))
		}
	})

	It("preserves exact pipeline and version counts and ordering across pages", Label("LegacyAPIPagination"), func() {
		prefix := "pages-" + randomName
		namespace := testutil.GetNamespace()
		file := filepath.Join("..", "resources", "hello-world.yaml")
		var pipelineIDs []string
		var names []string
		for _, suffix := range []string{"b", "a", "c"} {
			name := prefix + "-" + suffix
			pipeline := uploadAPipeline(file, &name)
			pipelineIDs = append(pipelineIDs, pipeline.PipelineID)
			names = append(names, name)
			Eventually(func(g Gomega) {
				fetched, err := pipelineClient.Get(&pipelineparams.PipelineServiceGetPipelineParams{PipelineID: pipeline.PipelineID})
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(fetched).NotTo(BeNil())
				g.Expect(fetched.DisplayName).To(Equal(name))
				g.Expect(time.Time(fetched.CreatedAt).IsZero()).To(BeFalse())
			}, informerSyncTimeout, informerSyncInterval).Should(Succeed())
			// Both stores persist creation timestamps at second precision.
			time.Sleep(time.Second)
		}
		filter := fmt.Sprintf(`{"predicates":[{"key":"name","operation":"IS_SUBSTRING","string_value":"%s"}]}`, prefix)
		for _, order := range []struct {
			sort    string
			indices []int
		}{{"name", []int{1, 0, 2}}, {"created_at", []int{0, 1, 2}}, {"name desc", []int{2, 0, 1}}} {
			Eventually(func(g Gomega) {
				size := int32(2)
				first, total, token, err := pipelineClient.List(&pipelineparams.PipelineServiceListPipelinesParams{Namespace: &namespace, Filter: &filter, SortBy: &order.sort, PageSize: &size})
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(first).To(HaveLen(2))
				g.Expect(total).To(Equal(3))
				g.Expect(token).NotTo(BeEmpty())
				second, total, end, err := pipelineClient.List(&pipelineparams.PipelineServiceListPipelinesParams{Namespace: &namespace, Filter: &filter, SortBy: &order.sort, PageSize: &size, PageToken: &token})
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(second).To(HaveLen(1))
				g.Expect(total).To(Equal(3))
				g.Expect(end).To(BeEmpty())
				for i, actual := range append(first, second...) {
					g.Expect(actual.PipelineID).To(Equal(pipelineIDs[order.indices[i]]))
					g.Expect(actual.DisplayName).To(Equal(names[order.indices[i]]))
				}
			}, informerSyncTimeout, informerSyncInterval).Should(Succeed())
		}

		pipelineID := pipelineIDs[0]
		for _, suffix := range []string{"a", "c"} {
			name := prefix + "-" + suffix
			_, err := pipelineUploadClient.UploadPipelineVersion(file, &uploadparams.UploadPipelineVersionParams{Name: &name, Pipelineid: &pipelineID})
			Expect(err).NotTo(HaveOccurred())
			time.Sleep(time.Second)
		}
		for _, order := range []struct {
			sort    string
			indices []int
		}{{"name", []int{1, 0, 2}}, {"created_at", []int{0, 1, 2}}, {"name desc", []int{2, 0, 1}}} {
			Eventually(func(g Gomega) {
				size := int32(2)
				first, total, token, err := pipelineClient.ListPipelineVersions(&pipelineparams.PipelineServiceListPipelineVersionsParams{PipelineID: pipelineID, SortBy: &order.sort, PageSize: &size})
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(first).To(HaveLen(2))
				g.Expect(total).To(Equal(3))
				g.Expect(token).NotTo(BeEmpty())
				second, total, end, err := pipelineClient.ListPipelineVersions(&pipelineparams.PipelineServiceListPipelineVersionsParams{PipelineID: pipelineID, SortBy: &order.sort, PageSize: &size, PageToken: &token})
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(second).To(HaveLen(1))
				g.Expect(total).To(Equal(3))
				g.Expect(end).To(BeEmpty())
				for i, actual := range append(first, second...) {
					g.Expect(actual.DisplayName).To(Equal(names[order.indices[i]]))
					g.Expect(actual.Name).To(Equal(names[order.indices[i]]))
					g.Expect(time.Time(actual.CreatedAt).IsZero()).To(BeFalse())
				}
			}, informerSyncTimeout, informerSyncInterval).Should(Succeed())
		}
	})

	It("preserves exact run pagination, time filtering, and archived list results", Label("LegacyAPIRunLists"), func() {
		name := "run-list-" + randomName
		pipeline := uploadAPipeline(filepath.Join("..", "resources", "hello-world.yaml"), &name)
		version := testutil.GetLatestPipelineVersion(pipelineClient, &pipeline.PipelineID)
		experiment := createExperiment(name)
		var runs []*run_model.V2beta1Run
		for _, suffix := range []string{"b", "a"} {
			payload := createPipelineRunPayload(&pipeline.PipelineID, &version.PipelineVersionID, &experiment.ExperimentID, nil)
			payload.DisplayName = name + "-" + suffix
			run, err := runClient.Create(&runparams.RunServiceCreateRunParams{Run: payload})
			Expect(err).NotTo(HaveOccurred())
			testContext.PipelineRun.CreatedRunIds = append(testContext.PipelineRun.CreatedRunIds, run.RunID)
			runs = append(runs, run)
			time.Sleep(time.Second)
		}
		size, sort := int32(1), "name"
		first, total, token, err := runClient.List(&runparams.RunServiceListRunsParams{ExperimentID: &experiment.ExperimentID, PageSize: &size, SortBy: &sort})
		Expect(err).NotTo(HaveOccurred())
		Expect(first).To(HaveLen(1))
		Expect(total).To(Equal(2))
		Expect(token).NotTo(BeEmpty())
		Expect(first[0].RunID).To(Equal(runs[1].RunID))
		second, total, end, err := runClient.List(&runparams.RunServiceListRunsParams{ExperimentID: &experiment.ExperimentID, PageSize: &size, SortBy: &sort, PageToken: &token})
		Expect(err).NotTo(HaveOccurred())
		Expect(second).To(HaveLen(1))
		Expect(total).To(Equal(2))
		Expect(end).To(BeEmpty())
		Expect(second[0].RunID).To(Equal(runs[0].RunID))
		filter := fmt.Sprintf(`{"predicates":[{"key":"created_at","operation":"LESS_THAN","string_value":"%d"}]}`, time.Time(runs[1].CreatedAt).Unix())
		filtered, total, _, err := runClient.List(&runparams.RunServiceListRunsParams{ExperimentID: &experiment.ExperimentID, Filter: &filter})
		Expect(err).NotTo(HaveOccurred())
		Expect(filtered).To(HaveLen(1))
		Expect(total).To(Equal(1))
		Expect(filtered[0].RunID).To(Equal(runs[0].RunID))
		Expect(runClient.Archive(&runparams.RunServiceArchiveRunParams{RunID: runs[0].RunID})).To(Succeed())
		listed, total, _, err := runClient.List(&runparams.RunServiceListRunsParams{ExperimentID: &experiment.ExperimentID})
		Expect(err).NotTo(HaveOccurred())
		Expect(listed).To(HaveLen(2))
		Expect(total).To(Equal(2))
		var archived *run_model.V2beta1Run
		for _, run := range listed {
			if run.RunID == runs[0].RunID {
				archived = run
			}
		}
		Expect(archived).NotTo(BeNil())
		Expect(archived.StorageState).NotTo(BeNil())
		Expect(*archived.StorageState).To(Equal(run_model.V2beta1RunStorageStateARCHIVED))
		Expect(archived.DisplayName).To(Equal(runs[0].DisplayName))
	})

})
