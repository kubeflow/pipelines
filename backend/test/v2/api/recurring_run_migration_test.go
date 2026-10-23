// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package api

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/go-openapi/strfmt"
	"github.com/google/go-cmp/cmp"
	uploadparams "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_upload_client/pipeline_upload_service"
	recurringparams "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/recurring_run_client/recurring_run_service"
	recurringmodel "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/recurring_run_model"
	runparams "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/run_client/run_service"
	"github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/run_model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	swfapi "github.com/kubeflow/pipelines/backend/src/crd/pkg/apis/scheduledworkflow/v1beta1"
	swfclientset "github.com/kubeflow/pipelines/backend/src/crd/pkg/client/clientset/versioned"
	"github.com/kubeflow/pipelines/backend/test/config"
	"github.com/kubeflow/pipelines/backend/test/constants"
	"github.com/kubeflow/pipelines/backend/test/testutil"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	"google.golang.org/protobuf/types/known/structpb"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/yaml"
)

// Each spec owns an experiment. Schedules are removed before the global cleanup
// deletes their children, so parallel specs never delete each other's resources.
var _ = Describe("Migrated recurring run behavior", Label(constants.APIServerTests, "LegacyMigration", constants.PipelineCacheEnabled), func() {
	var experimentID, pipelineID, versionID string
	var jobs map[string]bool
	BeforeEach(func() {
		experimentID = createExperiment("recurring-migration-" + randomName).ExperimentID
		pipeline := uploadAPipeline("../resources/hello-world.yaml", &testContext.Pipeline.PipelineGeneratedName)
		pipelineID = pipeline.PipelineID
		versionID = testutil.GetLatestPipelineVersion(pipelineClient, &pipelineID).PipelineVersionID
		jobs = map[string]bool{}
		DeferCleanup(func() {
			for id, deleted := range jobs {
				if !deleted {
					Expect(recurringRunClient.Delete(&recurringparams.RecurringRunServiceDeleteRecurringRunParams{RecurringRunID: id})).To(Succeed())
				}
			}
			runs, err := runClient.ListAll(&runparams.RunServiceListRunsParams{ExperimentID: &experimentID}, 100)
			Expect(err).NotTo(HaveOccurred())
			for _, run := range runs {
				testContext.PipelineRun.CreatedRunIds = append(testContext.PipelineRun.CreatedRunIds, run.RunID)
			}
		})
	})
	request := func(name string) *recurringmodel.V2beta1RecurringRun {
		return &recurringmodel.V2beta1RecurringRun{DisplayName: name + "-" + randomName, Description: "migration coverage", ExperimentID: experimentID,
			ServiceAccount: testutil.GetDefaultPipelineRunnerServiceAccount(), MaxConcurrency: 10,
			Mode: recurringmodel.RecurringRunModeDISABLE.Pointer(), PipelineVersionReference: &recurringmodel.V2beta1PipelineVersionReference{PipelineID: pipelineID, PipelineVersionID: versionID}}
	}
	create := func(req *recurringmodel.V2beta1RecurringRun) *recurringmodel.V2beta1RecurringRun {
		job, err := recurringRunClient.Create(&recurringparams.RecurringRunServiceCreateRecurringRunParams{RecurringRun: req})
		Expect(err).NotTo(HaveOccurred())
		jobs[job.RecurringRunID] = false
		Expect(job.DisplayName).To(Equal(req.DisplayName))
		Expect(job.Description).To(Equal(req.Description))
		Expect(job.ExperimentID).To(Equal(experimentID))
		Expect(job.ServiceAccount).To(Equal(req.ServiceAccount))
		Expect(job.MaxConcurrency).To(Equal(req.MaxConcurrency))
		Expect(job.Mode).To(Equal(req.Mode))
		Expect(job.RuntimeConfig).To(Equal(req.RuntimeConfig))
		Expect(job.PipelineVersionReference).To(Equal(req.PipelineVersionReference))
		stored, err := recurringRunClient.Get(&recurringparams.RecurringRunServiceGetRecurringRunParams{RecurringRunID: job.RecurringRunID})
		Expect(err).NotTo(HaveOccurred())
		// Status may advance asynchronously between Create and Get.
		Expect(stored.DisplayName).To(Equal(job.DisplayName))
		Expect(stored.Description).To(Equal(job.Description))
		Expect(stored.ServiceAccount).To(Equal(job.ServiceAccount))
		Expect(stored.ExperimentID).To(Equal(job.ExperimentID))
		Expect(stored.PipelineVersionReference).To(Equal(job.PipelineVersionReference))
		Expect(stored.RuntimeConfig).To(Equal(job.RuntimeConfig))
		Expect(stored.MaxConcurrency).To(Equal(job.MaxConcurrency))
		Expect(stored.Mode).To(Equal(job.Mode))
		for _, trigger := range []*recurringmodel.V2beta1Trigger{job.Trigger, stored.Trigger} {
			Expect(trigger).To(matchRecurringTrigger(req.Trigger))
		}
		Expect(stored.NoCatchup).To(Equal(req.NoCatchup))
		return job
	}
	waitRuns := func(job *recurringmodel.V2beta1RecurringRun, count int) []*run_model.V2beta1Run {
		var result []*run_model.V2beta1Run
		Eventually(func(g Gomega) {
			runs, err := runClient.ListAll(&runparams.RunServiceListRunsParams{ExperimentID: &experimentID}, 100)
			g.Expect(err).NotTo(HaveOccurred())
			result = nil
			for _, run := range runs {
				if run.RecurringRunID == job.RecurringRunID {
					result = append(result, run)
				}
			}
			g.Expect(result).To(HaveLen(count))
			for _, run := range result {
				g.Expect(run.ExperimentID).To(Equal(experimentID))
				g.Expect(run.DisplayName).NotTo(BeEmpty())
			}
		}, 2*time.Minute, time.Second).Should(Succeed())
		return result
	}
	It("round trips, sorts, paginates and filters owned recurring runs", Label("LegacyRecurringList"), func() {
		first := create(request("b-first"))
		// API timestamps have second precision; cross the boundary before creating the next job.
		Eventually(func() bool { return time.Now().Unix() > time.Time(first.CreatedAt).Unix() }, 3*time.Second, 100*time.Millisecond).Should(BeTrue())
		second := create(request("a-second"))
		for _, sortBy := range []string{"created_at", "name"} {
			params := &recurringparams.RecurringRunServiceListRecurringRunsParams{ExperimentID: &experimentID, PageSize: util.Int32Pointer(1), SortBy: &sortBy}
			page, total, token, err := recurringRunClient.List(params)
			Expect(err).NotTo(HaveOccurred())
			Expect(total).To(Equal(2))
			Expect(page).To(HaveLen(1))
			Expect(token).NotTo(BeEmpty())
			expectedFirst, expectedSecond := first.RecurringRunID, second.RecurringRunID
			if sortBy == "name" {
				expectedFirst, expectedSecond = expectedSecond, expectedFirst
			}
			Expect(page[0].RecurringRunID).To(Equal(expectedFirst))
			params.PageToken = &token
			page, total, token, err = recurringRunClient.List(params)
			Expect(err).NotTo(HaveOccurred())
			Expect(total).To(Equal(2))
			Expect(page).To(HaveLen(1))
			Expect(token).To(BeEmpty())
			Expect(page[0].RecurringRunID).To(Equal(expectedSecond))
		}
		_, _, _, err := recurringRunClient.List(&recurringparams.RecurringRunServiceListRecurringRunsParams{ExperimentID: &experimentID, SortBy: util.StringPointer("unknown")})
		Expect(err).To(HaveOccurred())
		filter := fmt.Sprintf(`{"predicates":[{"key":"created_at","operation":"LESS_THAN","string_value":"%d"}]}`, time.Time(second.CreatedAt).Unix())
		page, total, _, err := recurringRunClient.List(&recurringparams.RecurringRunServiceListRecurringRunsParams{ExperimentID: &experimentID, Filter: &filter})
		Expect(err).NotTo(HaveOccurred())
		Expect(total).To(Equal(1))
		Expect(page).To(HaveLen(1))
		Expect(page[0].RecurringRunID).To(Equal(first.RecurringRunID))
	})
	It("schedules referenced and inline pipelines with runtime parameters", Label("LegacyRecurringExecution"), func() {
		req := request("reference")
		req.Mode = recurringmodel.RecurringRunModeENABLE.Pointer()
		job := create(req)
		runs := waitRuns(job, 1)
		Expect(runs[0].PipelineVersionReference.PipelineVersionID).To(Equal(versionID))
		Expect(runs[0].DisplayName).To(ContainSubstring("reference"))
		data, err := os.ReadFile("../resources/arguments-parameters.yaml")
		Expect(err).NotTo(HaveOccurred())
		spec := &structpb.Struct{}
		Expect(yaml.Unmarshal(data, spec)).To(Succeed())
		req = request("inline")
		req.Mode = recurringmodel.RecurringRunModeENABLE.Pointer()
		req.PipelineVersionReference = nil
		req.PipelineSpec = spec
		req.RuntimeConfig = &recurringmodel.V2beta1RuntimeConfig{Parameters: map[string]interface{}{"param1": "goodbye", "param2": "world"}}
		job = create(req)
		runs = waitRuns(job, 1)
		Expect(runs[0].DisplayName).To(ContainSubstring("inline"))
		Expect(runs[0].RuntimeConfig.Parameters).To(Equal(req.RuntimeConfig.Parameters))
	})
	It("resolves the latest version for a recurring run without a pinned version", Label("LegacyRecurringLatest"), func() {
		// Upload a second version after the API's second-resolution timestamp boundary.
		previous := testutil.GetLatestPipelineVersion(pipelineClient, &pipelineID)
		Eventually(func() bool { return time.Now().Unix() > time.Time(previous.CreatedAt).Unix() }, 3*time.Second, 100*time.Millisecond).Should(BeTrue())
		latest, err := uploadPipelineVersion("../resources/hello-world.yaml", &uploadparams.UploadPipelineVersionParams{Pipelineid: &pipelineID, Name: util.StringPointer("latest-" + randomName)})
		Expect(err).NotTo(HaveOccurred())
		versionID = latest.PipelineVersionID
		Eventually(func() string { return testutil.GetLatestPipelineVersion(pipelineClient, &pipelineID).PipelineVersionID }, 30*time.Second, time.Second).Should(Equal(versionID))
		req := request("latest")
		req.Mode = recurringmodel.RecurringRunModeENABLE.Pointer()
		req.PipelineVersionReference.PipelineVersionID = ""
		job := create(req)
		runs := waitRuns(job, 1)
		Expect(runs[0].PipelineVersionReference.PipelineVersionID).To(Equal(versionID))
		// Resolving metadata is insufficient: the root driver must execute the selected version.
		Eventually(func(g Gomega) {
			executed, err := runClient.Get(&runparams.RunServiceGetRunParams{RunID: runs[0].RunID})
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(executed.State).To(Equal(run_model.V2beta1RuntimeStateSUCCEEDED.Pointer()))
		}, 5*time.Minute, 5*time.Second).Should(Succeed())
	})
	for _, periodic := range []bool{true, false} {
		for _, noCatchup := range []bool{true, false} {
			It(fmt.Sprintf("honors noCatchup=%t for periodic=%t", noCatchup, periodic), Label(fmt.Sprintf("LegacyRecurringCatchupPeriodic%tNoCatchup%t", periodic, noCatchup)), func() {
				req := request("catchup")
				req.Mode = recurringmodel.RecurringRunModeENABLE.Pointer()
				req.NoCatchup = noCatchup
				start, end := strfmt.DateTime(time.Unix(36000, 0)), strfmt.DateTime(time.Unix(36120, 0))
				if periodic {
					req.Trigger = &recurringmodel.V2beta1Trigger{PeriodicSchedule: &recurringmodel.V2beta1PeriodicSchedule{StartTime: start, EndTime: end, IntervalSecond: 60}}
				} else {
					req.Trigger = &recurringmodel.V2beta1Trigger{CronSchedule: &recurringmodel.V2beta1CronSchedule{StartTime: start, EndTime: end, Cron: "0 * * * * ?"}}
				}
				job := create(req)
				count := 2
				if noCatchup {
					count = 1
				}
				waitRuns(job, count)
				Consistently(func() int {
					runs, err := runClient.ListAll(&runparams.RunServiceListRunsParams{ExperimentID: &experimentID}, 100)
					Expect(err).NotTo(HaveOccurred())
					return len(runs)
				}, 5*time.Second, time.Second).Should(Equal(count))
			})
		}
	}
	// These retain the legacy opt-in fixture requirement: the custom account must
	// be allowlisted and authorized for both the caller and schedule controller.
	for _, pinned := range []bool{false, true} {
		It(fmt.Sprintf("uses persisted schedule authorization despite CR tampering, pinned=%t", pinned), Label(fmt.Sprintf("LegacyRecurringCustomAccountPinned%t", pinned)), func() {
			account := os.Getenv("KFP_SCHEDULE_TEST_SERVICE_ACCOUNT")
			if (!*config.KubeflowMode && !*config.MultiUserMode) || account == "" {
				Skip("requires multi-user mode and KFP_SCHEDULE_TEST_SERVICE_ACCOUNT")
			}
			customPipelineName := "custom-account-" + randomName
			pipeline := uploadAPipeline("../resources/arguments-parameters.yaml", &customPipelineName)
			pipelineID = pipeline.PipelineID
			versionID = testutil.GetLatestPipelineVersion(pipelineClient, &pipelineID).PipelineVersionID
			req := request("authorized")
			req.ServiceAccount = account
			req.MaxConcurrency = 1
			req.RuntimeConfig = &recurringmodel.V2beta1RuntimeConfig{Parameters: map[string]interface{}{"param1": "authorized", "param2": "schedule"}}
			if !pinned {
				req.PipelineVersionReference.PipelineVersionID = ""
			}
			job := create(req)
			cfg, err := util.GetKubernetesConfig()
			Expect(err).NotTo(HaveOccurred())
			clients, err := swfclientset.NewForConfig(cfg)
			Expect(err).NotTo(HaveOccurred())
			client := clients.ScheduledworkflowV1beta1().ScheduledWorkflows(testutil.GetNamespace())
			var name string
			Eventually(func(g Gomega) {
				list, err := client.List(context.Background(), metav1.ListOptions{})
				g.Expect(err).NotTo(HaveOccurred())
				for _, workflow := range list.Items {
					if string(workflow.UID) == job.RecurringRunID {
						name = workflow.Name
					}
				}
				g.Expect(name).NotTo(BeEmpty())
			}, 30*time.Second, time.Second).Should(Succeed())
			Expect(retry.RetryOnConflict(retry.DefaultRetry, func() error {
				workflow, err := client.Get(context.Background(), name, metav1.GetOptions{})
				if err != nil {
					return err
				}
				workflow.Spec.ServiceAccount = "tampered-schedule-account"
				workflow.Spec.PipelineId = "tampered-pipeline-id"
				workflow.Spec.PipelineVersionId = "tampered-version-id"
				if workflow.Spec.Workflow == nil {
					workflow.Spec.Workflow = &swfapi.WorkflowResource{}
				}
				workflow.Spec.Workflow.Parameters = []swfapi.Parameter{{Name: "param1", Value: `"tampered"`}}
				_, err = client.Update(context.Background(), workflow)
				return err
			})).To(Succeed())
			Expect(recurringRunClient.Enable(&recurringparams.RecurringRunServiceEnableRecurringRunParams{RecurringRunID: job.RecurringRunID})).To(Succeed())
			patched, err := client.Get(context.Background(), name, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(patched.Spec.ServiceAccount).To(Equal("tampered-schedule-account"))
			Expect(patched.Spec.PipelineId).To(Equal("tampered-pipeline-id"))
			runs := waitRuns(job, 1)
			Eventually(func(g Gomega) {
				executed, err := runClient.Get(&runparams.RunServiceGetRunParams{RunID: runs[0].RunID})
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(executed.State).To(Equal(run_model.V2beta1RuntimeStateSUCCEEDED.Pointer()))
				g.Expect(executed.ServiceAccount).To(Equal(account))
				g.Expect(executed.RuntimeConfig.Parameters).To(Equal(req.RuntimeConfig.Parameters))
				g.Expect(executed.PipelineVersionReference.PipelineID).To(Equal(pipelineID))
				g.Expect(executed.PipelineVersionReference.PipelineVersionID).To(Equal(versionID))
			}, 10*time.Minute, 5*time.Second).Should(Succeed())
		})
	}
	It("creates a disabled ScheduledWorkflow, enables/disables it and deletes the job if its own workflow is missing", Label("LegacyRecurringLifecycle"), func() {
		req := request("disabled")
		req.NoCatchup = true
		req.Trigger = &recurringmodel.V2beta1Trigger{CronSchedule: &recurringmodel.V2beta1CronSchedule{Cron: "0 0 1 1 *"}}
		job := create(req)
		cfg, err := util.GetKubernetesConfig()
		Expect(err).NotTo(HaveOccurred())
		clients, err := swfclientset.NewForConfig(cfg)
		Expect(err).NotTo(HaveOccurred())
		client := clients.ScheduledworkflowV1beta1().ScheduledWorkflows(testutil.GetNamespace())
		var name string
		Eventually(func(g Gomega) {
			list, err := client.List(context.Background(), metav1.ListOptions{})
			g.Expect(err).NotTo(HaveOccurred())
			for _, swf := range list.Items {
				if string(swf.UID) == job.RecurringRunID {
					name = swf.Name
					g.Expect(swf.Spec.Enabled).To(BeFalse())
					g.Expect(swf.Spec.PipelineId).To(Equal(pipelineID))
					g.Expect(swf.Spec.PipelineVersionId).To(Equal(versionID))
				}
			}
			g.Expect(name).NotTo(BeEmpty())
		}, 30*time.Second, time.Second).Should(Succeed())
		Expect(recurringRunClient.Enable(&recurringparams.RecurringRunServiceEnableRecurringRunParams{RecurringRunID: job.RecurringRunID})).To(Succeed())
		stored, err := recurringRunClient.Get(&recurringparams.RecurringRunServiceGetRecurringRunParams{RecurringRunID: job.RecurringRunID})
		Expect(err).NotTo(HaveOccurred())
		Expect(stored.Mode).To(Equal(recurringmodel.RecurringRunModeENABLE.Pointer()))
		Expect(recurringRunClient.Disable(&recurringparams.RecurringRunServiceDisableRecurringRunParams{RecurringRunID: job.RecurringRunID})).To(Succeed())
		stored, err = recurringRunClient.Get(&recurringparams.RecurringRunServiceGetRecurringRunParams{RecurringRunID: job.RecurringRunID})
		Expect(err).NotTo(HaveOccurred())
		Expect(stored.Mode).To(Equal(recurringmodel.RecurringRunModeDISABLE.Pointer()))
		Expect(stored.Trigger.CronSchedule.Cron).To(Equal(req.Trigger.CronSchedule.Cron))
		Expect(client.Delete(context.Background(), name, &metav1.DeleteOptions{})).To(Succeed())
		Expect(recurringRunClient.Delete(&recurringparams.RecurringRunServiceDeleteRecurringRunParams{RecurringRunID: job.RecurringRunID})).To(Succeed())
		jobs[job.RecurringRunID] = true
		_, err = recurringRunClient.Get(&recurringparams.RecurringRunServiceGetRecurringRunParams{RecurringRunID: job.RecurringRunID})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("[404]"))
	})
})

// matchRecurringTrigger preserves all schedule fields while comparing timestamps
// by instant, independent of the timezone representation used by the API.
func matchRecurringTrigger(expected *recurringmodel.V2beta1Trigger) types.GomegaMatcher {
	value := func(trigger *recurringmodel.V2beta1Trigger) recurringmodel.V2beta1Trigger {
		// An absent trigger may round trip as an empty message.
		if trigger == nil {
			return recurringmodel.V2beta1Trigger{}
		}
		return *trigger
	}
	return WithTransform(value, BeComparableTo(value(expected), cmp.Comparer(func(a, b strfmt.DateTime) bool {
		return time.Time(a).Equal(time.Time(b))
	})))
}
