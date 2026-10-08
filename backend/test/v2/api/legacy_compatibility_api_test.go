// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0.

package api

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	api "github.com/kubeflow/pipelines/backend/api/v2/go_client"
	"github.com/kubeflow/pipelines/backend/test/config"
	"github.com/kubeflow/pipelines/backend/test/constants"
	"github.com/kubeflow/pipelines/backend/test/testutil"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

const legacyAPICompatibilityEnvironment = "LEGACY_API_COMPATIBILITY_TESTS"

// These clients deliberately use literal HTTP prefixes. The legacy Go import
// aliases call v2 and would not exercise the deployed compatibility router.
type compatibilityHTTPClient struct {
	client   *http.Client
	endpoint string
	token    string
}

func newCompatibilityHTTPClient(endpoint, token string, tlsConfig *tls.Config) *compatibilityHTTPClient {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = tlsConfig
	return &compatibilityHTTPClient{
		client: &http.Client{Transport: transport, Timeout: 30 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		}},
		endpoint: endpoint,
		token:    token,
	}
}

func (c *compatibilityHTTPClient) request(ctx context.Context, method, version, resource string, body []byte, contentType string, wantStatus int) ([]byte, error) {
	request, err := http.NewRequestWithContext(ctx, method, strings.TrimRight(c.endpoint, "/")+"/apis/"+version+resource, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", contentType)
	if c.token != "" {
		request.Header.Set("Authorization", "Bearer "+c.token)
	}
	fmt.Fprintf(GinkgoWriter, "Compatibility request: %s %s\n", method, request.URL.Path)
	response, err := c.client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	contents, err := io.ReadAll(io.LimitReader(response.Body, (4<<20)+1))
	if err != nil {
		return nil, err
	}
	if len(contents) > 4<<20 {
		return nil, fmt.Errorf("%s %s exceeds the smoke test response limit; inspect the server response", method, request.URL.Path)
	}
	if response.StatusCode != wantStatus {
		return nil, fmt.Errorf("%s %s returned %s, want %d; verify both API prefixes are deployed: %.2048s", method, request.URL.Path, response.Status, wantStatus, contents)
	}
	return contents, nil
}

func (c *compatibilityHTTPClient) message(ctx context.Context, method, version, resource string, input, output proto.Message) error {
	var body []byte
	var err error
	if input != nil {
		body, err = (protojson.MarshalOptions{UseProtoNames: true}).Marshal(input)
		if err != nil {
			return err
		}
	}
	contents, err := c.request(ctx, method, version, resource, body, "application/json", http.StatusOK)
	if err != nil || output == nil {
		return err
	}
	return protojson.Unmarshal(contents, output)
}

func (c *compatibilityHTTPClient) cleanup(version, resource string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_, err := c.request(ctx, http.MethodDelete, version, resource, nil, "application/json", http.StatusOK)
	Expect(err).NotTo(HaveOccurred(), "delete only the compatibility test's own resource: %s", resource)
}

var _ = Describe("Legacy API compatibility on the deployed server >", Serial, Label(constants.APIServerTests, constants.POSITIVE, "LegacyAPICompatibility"), func() {
	var client *compatibilityHTTPClient
	var diagnosticRunID string

	BeforeEach(func(ctx SpecContext) {
		if os.Getenv(legacyAPICompatibilityEnvironment) != "true" {
			Skip("Enable LEGACY_API_COMPATIBILITY_TESTS=true to exercise both prefixes on an existing deployment")
		}
		Expect(*config.UseLegacyAPIPrefix).To(BeFalse(), "do not use old-server preparation mode to test parallel route support")
		var tlsConfig *tls.Config
		if *config.TLSEnabled {
			var err error
			tlsConfig, err = testutil.GetTLSConfig(*config.CaCertPath)
			Expect(err).NotTo(HaveOccurred())
		}
		if *config.DisableTLSCheck {
			tlsConfig = &tls.Config{InsecureSkipVerify: true}
		}
		client = newCompatibilityHTTPClient(*config.ApiUrl, userToken, tlsConfig)
		DeferCleanup(client.client.CloseIdleConnections)
		diagnosticRunID = ""
		// Fail before creating any resource if this is an old server, the legacy
		// router is absent, or a proxy redirects instead of preserving the route.
		for _, version := range []string{"v2beta1", "v2"} {
			_, err := client.request(ctx, http.MethodGet, version, "/healthz", nil, "application/json", http.StatusOK)
			Expect(err).NotTo(HaveOccurred())
		}
	})

	AfterEach(func() {
		if CurrentSpecReport().Failed() && diagnosticRunID != "" {
			AddReportEntry("Legacy API pipeline diagnostics", collectArgoCompatibilityDiagnostics(diagnosticRunID))
		}
	})

	It("shares experiment creation, archive, restore and deletion in both directions", func(ctx SpecContext) {
		for _, versions := range [][2]string{{"v2beta1", "v2"}, {"v2", "v2beta1"}} {
			writer, reader := versions[0], versions[1]
			created := new(api.Experiment)
			Expect(client.message(ctx, http.MethodPost, writer, "/experiments", &api.Experiment{
				DisplayName: "legacy-api-" + writer + "-" + randomName,
				Namespace:   testutil.GetNamespace(),
			}, created)).To(Succeed())
			Expect(created.ExperimentId).NotTo(BeEmpty())
			resource := "/experiments/" + url.PathEscape(created.ExperimentId)
			deleted := false
			DeferCleanup(func() {
				if !deleted {
					client.cleanup(writer, resource)
				}
			})
			fetched := new(api.Experiment)
			Expect(client.message(ctx, http.MethodGet, reader, resource, nil, fetched)).To(Succeed())
			Expect(fetched.ExperimentId).To(Equal(created.ExperimentId))
			Expect(fetched.DisplayName).To(Equal(created.DisplayName))
			Expect(client.message(ctx, http.MethodPost, reader, resource+":archive", nil, nil)).To(Succeed())
			Expect(client.message(ctx, http.MethodGet, writer, resource, nil, fetched)).To(Succeed())
			Expect(fetched.StorageState).To(Equal(api.Experiment_ARCHIVED))
			Expect(client.message(ctx, http.MethodPost, writer, resource+":unarchive", nil, nil)).To(Succeed())
			Expect(client.message(ctx, http.MethodGet, reader, resource, nil, fetched)).To(Succeed())
			Expect(fetched.StorageState).To(Equal(api.Experiment_AVAILABLE))
			Expect(client.message(ctx, http.MethodDelete, reader, resource, nil, nil)).To(Succeed())
			deleted = true
			_, err := client.request(ctx, http.MethodGet, writer, resource, nil, "application/json", http.StatusNotFound)
			Expect(err).NotTo(HaveOccurred())
		}
	}, SpecTimeout(2*time.Minute))

	It("executes a legacy-submitted pipeline and shares runs, schedules, tasks, artifacts and logs with v2", func(ctx SpecContext) {
		experiment := new(api.Experiment)
		Expect(client.message(ctx, http.MethodPost, "v2beta1", "/experiments", &api.Experiment{
			DisplayName: "legacy-api-run-" + randomName,
			Namespace:   testutil.GetNamespace(),
		}, experiment)).To(Succeed())
		Expect(experiment.ExperimentId).NotTo(BeEmpty())
		DeferCleanup(func() { client.cleanup("v2", "/experiments/"+url.PathEscape(experiment.ExperimentId)) })

		// Reuse the tiny cached-image fixture: no SDK install or new image build.
		fixture, err := os.ReadFile(filepath.Join(pipelineFilesRootDir, "argo_compatibility", "fast_artifact.yaml"))
		Expect(err).NotTo(HaveOccurred())
		var upload bytes.Buffer
		form := multipart.NewWriter(&upload)
		part, err := form.CreateFormFile("uploadfile", "legacy-compatibility.yaml")
		Expect(err).NotTo(HaveOccurred())
		_, err = part.Write(fixture)
		Expect(err).NotTo(HaveOccurred())
		Expect(form.Close()).To(Succeed())
		query := url.Values{"name": {"legacy-api-" + randomName}, "namespace": {testutil.GetNamespace()}}
		body, err := client.request(ctx, http.MethodPost, "v2beta1", "/pipelines/upload?"+query.Encode(), upload.Bytes(), form.FormDataContentType(), http.StatusOK)
		Expect(err).NotTo(HaveOccurred())
		pipeline := new(api.Pipeline)
		Expect(protojson.Unmarshal(body, pipeline)).To(Succeed())
		Expect(pipeline.PipelineId).NotTo(BeEmpty())
		pipelineResource := "/pipelines/" + url.PathEscape(pipeline.PipelineId)
		DeferCleanup(func() { client.cleanup("v2beta1", pipelineResource+"?cascade=true") })
		versions := new(api.ListPipelineVersionsResponse)
		Expect(client.message(ctx, http.MethodGet, "v2", pipelineResource+"/versions", nil, versions)).To(Succeed())
		Expect(versions.PipelineVersions).To(HaveLen(1))
		reference := &api.PipelineVersionReference{PipelineId: pipeline.PipelineId, PipelineVersionId: versions.PipelineVersions[0].PipelineVersionId}

		schedule := new(api.RecurringRun)
		Expect(client.message(ctx, http.MethodPost, "v2beta1", "/recurringruns", &api.RecurringRun{
			DisplayName: "legacy-api-schedule-" + randomName, ExperimentId: experiment.ExperimentId,
			PipelineSource: &api.RecurringRun_PipelineVersionReference{PipelineVersionReference: reference},
			ServiceAccount: testutil.GetDefaultPipelineRunnerServiceAccount(), MaxConcurrency: 1,
			Mode: api.RecurringRun_DISABLE, NoCatchup: true,
			Trigger: &api.Trigger{Trigger: &api.Trigger_CronSchedule{CronSchedule: &api.CronSchedule{Cron: "0 0 1 1 *"}}},
		}, schedule)).To(Succeed())
		Expect(schedule.RecurringRunId).NotTo(BeEmpty())
		scheduleResource := "/recurringruns/" + url.PathEscape(schedule.RecurringRunId)
		DeferCleanup(func() { client.cleanup("v2beta1", scheduleResource) })
		storedSchedule := new(api.RecurringRun)
		Expect(client.message(ctx, http.MethodGet, "v2", scheduleResource, nil, storedSchedule)).To(Succeed())
		Expect(storedSchedule.RecurringRunId).To(Equal(schedule.RecurringRunId))
		Expect(storedSchedule.Mode).To(Equal(api.RecurringRun_DISABLE))

		run := new(api.Run)
		Expect(client.message(ctx, http.MethodPost, "v2beta1", "/runs", &api.Run{
			DisplayName: "legacy-api-run-" + randomName, ExperimentId: experiment.ExperimentId,
			PipelineSource: &api.Run_PipelineVersionReference{PipelineVersionReference: reference},
			ServiceAccount: testutil.GetDefaultPipelineRunnerServiceAccount(),
		}, run)).To(Succeed())
		Expect(run.RunId).NotTo(BeEmpty())
		diagnosticRunID = run.RunId
		runResource := "/runs/" + url.PathEscape(run.RunId)
		DeferCleanup(func() {
			cleanupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			// Termination may report that an already-terminal run cannot be stopped.
			_ = client.message(cleanupCtx, http.MethodPost, "v2beta1", runResource+":terminate", nil, nil)
			client.cleanup("v2beta1", runResource)
		})
		checkImagePulls := testutil.NewRunImagePullCheck(k8Client, testutil.GetNamespace(), run.RunId)
		Eventually(func() (api.RuntimeState, error) {
			if err := checkImagePulls(); err != nil {
				return api.RuntimeState_RUNTIME_STATE_UNSPECIFIED, StopTrying(err.Error())
			}
			state := new(api.Run)
			if err := client.message(ctx, http.MethodGet, "v2beta1", runResource, nil, state); err != nil {
				return state.State, err
			}
			if state.State == api.RuntimeState_FAILED || state.State == api.RuntimeState_CANCELED {
				return state.State, StopTrying(fmt.Sprintf("legacy-submitted run %s ended in %s: %v", run.RunId, state.State, state.Error))
			}
			return state.State, nil
		}, 6*time.Minute, 2*time.Second).WithContext(ctx).Should(Equal(api.RuntimeState_SUCCEEDED))
		canonicalRun := new(api.Run)
		Expect(client.message(ctx, http.MethodGet, "v2", runResource, nil, canonicalRun)).To(Succeed())
		Expect(canonicalRun.RunId).To(Equal(run.RunId))
		Expect(canonicalRun.State).To(Equal(api.RuntimeState_SUCCEEDED))

		var executorPod, artifactID string
		Eventually(func(g Gomega) {
			tasks := new(api.ListTasksResponse)
			g.Expect(client.message(ctx, http.MethodGet, "v2beta1", runResource+"/tasks", nil, tasks)).To(Succeed())
			executorPod, artifactID = "", ""
			for _, task := range tasks.Tasks {
				for _, pod := range task.Pods {
					if pod.Type == api.PipelineTask_EXECUTOR {
						executorPod = pod.Name
					}
				}
				for _, output := range task.GetOutputs().GetArtifacts() {
					if output.ArtifactKey == "output" && len(output.Artifacts) > 0 {
						artifactID = output.Artifacts[0].ArtifactId
					}
				}
			}
			g.Expect(executorPod).NotTo(BeEmpty(), "execution must have produced an executor pod")
			g.Expect(artifactID).NotTo(BeEmpty(), "the fixture must publish its output artifact")
		}, 90*time.Second, 2*time.Second).WithContext(ctx).Should(Succeed())
		for _, version := range []string{"v2beta1", "v2"} {
			artifact := new(api.Artifact)
			Expect(client.message(ctx, http.MethodGet, version, "/artifacts/"+url.PathEscape(artifactID), nil, artifact)).To(Succeed())
			Expect(artifact.ArtifactId).To(Equal(artifactID))
			Expect(artifact.Uri).NotTo(BeEmpty())
			logs, err := client.request(ctx, http.MethodGet, version, runResource+"/nodes/"+url.PathEscape(executorPod)+"/log?follow=false", nil, "application/json", http.StatusOK)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(logs)).To(ContainSubstring("input:  foo"))
		}
	}, SpecTimeout(8*time.Minute))
})
