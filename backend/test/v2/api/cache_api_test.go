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
	"context"
	"os"
	"path/filepath"
	"time"

	recurringrunparams "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/recurring_run_client/recurring_run_service"
	"github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/recurring_run_model"
	runparams "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/run_client/run_service"
	"github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/run_model"
	"github.com/kubeflow/pipelines/backend/test/constants"
	"github.com/kubeflow/pipelines/backend/test/testutil"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Pipeline cache reuse >", SpecPriority(1), Label(constants.APIServerTests, constants.PipelineCacheEnabled, "LegacyMigration", "LegacyCache"), func() {
	It("reuses a completed run without creating an executor pod", Label("LegacyCacheSingleRun"), func() {
		experiment := createExperiment(experimentName)
		pipeline := uploadAPipeline(cachePipelineFixture("hello-world-with-returning-component.yaml"), &testContext.Pipeline.PipelineGeneratedName)
		version := testutil.GetLatestPipelineVersion(pipelineClient, &pipeline.PipelineID)
		// A unique input prevents another spec or a previous CI run from warming this cache.
		inputs := map[string]interface{}{"message": randomName}
		first := createPipelineRun(&pipeline.PipelineID, &version.PipelineVersionID, &experiment.ExperimentID, inputs)
		expectCacheTask(first.RunID, "comp", run_model.PipelineTaskTaskStateSUCCEEDED)
		second := createPipelineRun(&pipeline.PipelineID, &version.PipelineVersionID, &experiment.ExperimentID, inputs)
		expectCacheTask(second.RunID, "comp", run_model.PipelineTaskTaskStateCACHED)
	})

	It("reuses a completed recurring run without creating an executor pod", Label("LegacyCacheRecurringRun"), func() {
		experiment := createExperiment(experimentName)
		pipeline := uploadAPipeline(cachePipelineFixture("hello-world-with-returning-component.yaml"), &testContext.Pipeline.PipelineGeneratedName)
		version := testutil.GetLatestPipelineVersion(pipelineClient, &pipeline.PipelineID)
		schedule, err := recurringRunClient.Create(&recurringrunparams.RecurringRunServiceCreateRecurringRunParams{RecurringRun: &recurring_run_model.V2beta1RecurringRun{
			DisplayName:              "cache-" + randomName,
			ExperimentID:             experiment.ExperimentID,
			ServiceAccount:           testutil.GetDefaultPipelineRunnerServiceAccount(),
			PipelineVersionReference: &recurring_run_model.V2beta1PipelineVersionReference{PipelineID: pipeline.PipelineID, PipelineVersionID: version.PipelineVersionID},
			MaxConcurrency:           1,
			NoCatchup:                true,
			Mode:                     recurring_run_model.RecurringRunModeENABLE.Pointer(),
			Trigger:                  &recurring_run_model.V2beta1Trigger{PeriodicSchedule: &recurring_run_model.V2beta1PeriodicSchedule{IntervalSecond: 60}},
			RuntimeConfig:            &recurring_run_model.V2beta1RuntimeConfig{Parameters: map[string]interface{}{"message": randomName}},
		}})
		Expect(err).NotTo(HaveOccurred())
		// Stop scheduling before the global cleanup deletes this spec's runs and experiment.
		DeferCleanup(func() {
			Expect(recurringRunClient.Delete(&recurringrunparams.RecurringRunServiceDeleteRecurringRunParams{RecurringRunID: schedule.RecurringRunID})).To(Succeed())
			runs, err := runClient.ListAll(&runparams.RunServiceListRunsParams{ExperimentID: &experiment.ExperimentID}, 100)
			Expect(err).NotTo(HaveOccurred())
			for _, run := range runs {
				testContext.PipelineRun.CreatedRunIds = append(testContext.PipelineRun.CreatedRunIds, run.RunID)
			}
		})
		var runs []*run_model.V2beta1Run
		sortBy := "created_at asc"
		Eventually(func(g Gomega) {
			var err error
			runs, err = runClient.ListAll(&runparams.RunServiceListRunsParams{ExperimentID: &experiment.ExperimentID, SortBy: &sortBy}, 100)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(len(runs)).To(BeNumerically(">=", 2))
			for _, run := range runs[:2] {
				g.Expect(run.RecurringRunID).To(Equal(schedule.RecurringRunID))
				g.Expect(run.State).To(Equal(run_model.V2beta1RuntimeStateSUCCEEDED.Pointer()))
			}
		}, 6*time.Minute, 5*time.Second).Should(Succeed())
		expectCacheTask(runs[0].RunID, "comp", run_model.PipelineTaskTaskStateSUCCEEDED)
		expectCacheTask(runs[1].RunID, "comp", run_model.PipelineTaskTaskStateCACHED)
	})

	It("caches both PVC tasks for the same claim and executes both for a different claim", Label("LegacyCachePVC"), func() {
		experiment := createExperiment(experimentName)
		pipeline := uploadAPipeline(cachePipelineFixture("pvc-mount.yaml"), &testContext.Pipeline.PipelineGeneratedName)
		version := testutil.GetLatestPipelineVersion(pipelineClient, &pipeline.PipelineID)
		pvcName := "cache-" + randomName
		createCachePVC(pvcName)
		createCachePVC(pvcName + "-alt")
		for _, tc := range []struct {
			claim string
			state run_model.PipelineTaskTaskState
		}{
			{pvcName, run_model.PipelineTaskTaskStateSUCCEEDED},
			{pvcName, run_model.PipelineTaskTaskStateCACHED},
			{pvcName + "-alt", run_model.PipelineTaskTaskStateSUCCEEDED},
		} {
			run := createPipelineRun(&pipeline.PipelineID, &version.PipelineVersionID, &experiment.ExperimentID, map[string]interface{}{"pvc_name": tc.claim})
			expectCacheTask(run.RunID, "producer", tc.state)
			expectCacheTask(run.RunID, "consumer", tc.state)
		}
	})
})

func cachePipelineFixture(name string) string {
	directory := os.Getenv("KFP_CACHE_FIXTURE_DIRECTORY")
	if directory == "" {
		directory = filepath.Join("..", "resources")
	}
	return filepath.Join(directory, name)
}

func expectCacheTask(runID, taskName string, state run_model.PipelineTaskTaskState) {
	full := string(run_model.V2beta1GetRunRequestViewModeFULL)
	Eventually(func(g Gomega) {
		run, err := runClient.Get(&runparams.RunServiceGetRunParams{RunID: runID, View: &full})
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(run).NotTo(BeNil())
		g.Expect(run.State).To(Equal(run_model.V2beta1RuntimeStateSUCCEEDED.Pointer()))
		var matched *run_model.V2beta1PipelineTask
		for _, task := range run.Tasks {
			if task != nil && task.Name == taskName {
				matched = task
				break
			}
		}
		g.Expect(matched).NotTo(BeNil(), "task %s in run %s", taskName, runID)
		g.Expect(matched.State).To(Equal(state.Pointer()))
		if state == run_model.PipelineTaskTaskStateCACHED {
			for _, pod := range matched.Pods {
				g.Expect(pod).NotTo(BeNil())
				g.Expect(pod.Type).NotTo(Equal(run_model.PipelineTaskTaskPodTypeEXECUTOR.Pointer()), "cached task %s has executor pod %s", taskName, pod.Name)
			}
		}
	}, 6*time.Minute, 5*time.Second).Should(Succeed())
}

func createCachePVC(name string) {
	storageClass := "standard"
	_, err := k8Client.CoreV1().PersistentVolumeClaims(testutil.GetNamespace()).Create(context.Background(), &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			Resources:        corev1.VolumeResourceRequirements{Requests: corev1.ResourceList{corev1.ResourceStorage: resource.MustParse("5Mi")}},
			StorageClassName: &storageClass,
		},
	}, metav1.CreateOptions{})
	Expect(err).NotTo(HaveOccurred())
	DeferCleanup(func() {
		Expect(k8Client.CoreV1().PersistentVolumeClaims(testutil.GetNamespace()).Delete(context.Background(), name, metav1.DeleteOptions{})).To(Succeed())
	})
}
