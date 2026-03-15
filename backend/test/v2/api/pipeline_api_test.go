// Copyright 2018-2023 The Kubeflow Authors
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
	"fmt"
	"path/filepath"
	"strings"
	"time"

	pipeline_params "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_client/pipeline_service"
	pipeline_model "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_model"
	uploadparams "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_upload_client/pipeline_upload_service"
	upload_model "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_upload_model"
	"github.com/kubeflow/pipelines/backend/test/config"
	"github.com/kubeflow/pipelines/backend/test/constants"
	"github.com/kubeflow/pipelines/backend/test/logger"
	utils "github.com/kubeflow/pipelines/backend/test/testutil"
	"github.com/kubeflow/pipelines/backend/test/v2/api/matcher"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// toUploadModel converts a pipeline_model.V2beta1Pipeline to the upload_model equivalent
// for tracking in testContext.Pipeline.CreatedPipelines (used for cleanup).
func toUploadModel(p *pipeline_model.V2beta1Pipeline) *upload_model.V2beta1Pipeline {
	return &upload_model.V2beta1Pipeline{
		PipelineID:  p.PipelineID,
		Name:        p.Name,
		DisplayName: p.DisplayName,
		Description: p.Description,
		Namespace:   p.Namespace,
		CreatedAt:   p.CreatedAt,
		Tags:        p.Tags,
	}
}

// newListPipelinesParams returns PipelineServiceListPipelinesParams with Namespace
// pre-set from the test environment. In multi-user mode, the API server scopes
// list results by namespace, so this ensures pipelines created in the user's
// namespace are returned.
func newListPipelinesParams() *pipeline_params.PipelineServiceListPipelinesParams {
	ns := utils.GetNamespace()
	return &pipeline_params.PipelineServiceListPipelinesParams{
		Namespace: &ns,
	}
}

// ###########################################
// ################## TESTS ##################
// ###########################################

// ################## POSITIVE TESTS ##################

var _ = Describe("List Pipelines API Tests >", Label(constants.POSITIVE, constants.Pipeline, "PipelineList", constants.APIServerTests, constants.FullRegression), func() {

	Context("Basic List Operations >", func() {

		It("After creating a single pipeline", func() {
			pipelineDir := "valid"
			pipelineSpecFilePath := filepath.Join(pipelineFilesRootDir, pipelineDir, helloWorldPipelineFileName)
			createdPipeline := uploadPipelineAndVerify(pipelineSpecFilePath, &testContext.Pipeline.PipelineGeneratedName, nil)

			// Use Eventually to allow the informer cache to sync (applies to K8s store;
			// SQL store returns immediately on first attempt).
			Eventually(func() bool {
				params := newListPipelinesParams()
				pipelines, _, _, err := pipelineClient.List(params)
				if err != nil {
					return false
				}
				for _, p := range pipelines {
					if p.PipelineID == createdPipeline.PipelineID {
						return true
					}
				}
				return false
			}, 30*time.Second, 2*time.Second).Should(BeTrue(), "Created pipeline should appear in list results")
		})

		It("After creating multiple pipelines", func() {
			pipelineDir := "valid"
			pipelineSpecFilePath := filepath.Join(pipelineFilesRootDir, pipelineDir, helloWorldPipelineFileName)

			name1 := testContext.Pipeline.PipelineGeneratedName + "-1"
			createdPipeline1 := uploadPipelineAndVerify(pipelineSpecFilePath, &name1, nil)
			name2 := testContext.Pipeline.PipelineGeneratedName + "-2"
			createdPipeline2 := uploadPipelineAndVerify(pipelineSpecFilePath, &name2, nil)

			listPageSize := int32(100)
			params := newListPipelinesParams()
			params.PageSize = &listPageSize
			pipelines, totalSize, _, err := pipelineClient.List(params)
			Expect(err).NotTo(HaveOccurred())
			Expect(totalSize).To(BeNumerically(">=", 2))

			foundIDs := map[string]bool{}
			for _, p := range pipelines {
				foundIDs[p.PipelineID] = true
			}
			Expect(foundIDs).To(HaveKey(createdPipeline1.PipelineID))
			Expect(foundIDs).To(HaveKey(createdPipeline2.PipelineID))
		})

		It("By namespace", func() {
			namespace := utils.GetNamespace()
			params := &pipeline_params.PipelineServiceListPipelinesParams{
				Namespace: &namespace,
			}
			pipelines, totalSize, _, err := pipelineClient.List(params)
			Expect(err).NotTo(HaveOccurred())
			Expect(totalSize).To(BeNumerically(">=", 0))
			for _, p := range pipelines {
				if *config.KubeflowMode {
					Expect(p.Namespace).To(Equal(namespace))
				}
			}
			_ = pipelines
		})
	})

	Context("Pagination >", func() {
		It("List pipelines with page size limit", func() {
			pipelineDir := "valid"
			pipelineSpecFilePath := filepath.Join(pipelineFilesRootDir, pipelineDir, helloWorldPipelineFileName)

			// Create at least 2 pipelines
			name1 := testContext.Pipeline.PipelineGeneratedName + "-page1"
			uploadPipelineAndVerify(pipelineSpecFilePath, &name1, nil)
			name2 := testContext.Pipeline.PipelineGeneratedName + "-page2"
			uploadPipelineAndVerify(pipelineSpecFilePath, &name2, nil)

			pageSize := int32(1)
			params := newListPipelinesParams()
			params.PageSize = &pageSize
			pipelines, totalSize, nextPageToken, err := pipelineClient.List(params)
			Expect(err).NotTo(HaveOccurred())
			Expect(pipelines).To(HaveLen(1))
			Expect(totalSize).To(BeNumerically(">=", 2))
			Expect(nextPageToken).NotTo(BeEmpty(), "Next page token should be set when more results exist")
		})

		It("List pipelines with pagination - iterate through all pages (at least 2)", func() {
			pipelineDir := "valid"
			pipelineSpecFilePath := filepath.Join(pipelineFilesRootDir, pipelineDir, helloWorldPipelineFileName)

			// Create at least 3 pipelines
			name1 := testContext.Pipeline.PipelineGeneratedName + "-iter1"
			uploadPipelineAndVerify(pipelineSpecFilePath, &name1, nil)
			name2 := testContext.Pipeline.PipelineGeneratedName + "-iter2"
			uploadPipelineAndVerify(pipelineSpecFilePath, &name2, nil)
			name3 := testContext.Pipeline.PipelineGeneratedName + "-iter3"
			uploadPipelineAndVerify(pipelineSpecFilePath, &name3, nil)

			pageSize := int32(2)
			params := newListPipelinesParams()
			params.PageSize = &pageSize

			allPipelines := make([]*pipeline_model.V2beta1Pipeline, 0)
			pagesVisited := 0

			for {
				pipelines, _, nextPageToken, err := pipelineClient.List(params)
				Expect(err).NotTo(HaveOccurred())
				allPipelines = append(allPipelines, pipelines...)
				pagesVisited++

				if nextPageToken == "" {
					break
				}
				params.PageToken = &nextPageToken
			}

			Expect(pagesVisited).To(BeNumerically(">=", 2), "Should visit at least 2 pages")
			Expect(len(allPipelines)).To(BeNumerically(">=", 3))
		})
	})

	Context("Sorting >", func() {
		It("Sort by name in ascending order", func() {
			pipelineDir := "valid"
			pipelineSpecFilePath := filepath.Join(pipelineFilesRootDir, pipelineDir, helloWorldPipelineFileName)

			name1 := testContext.Pipeline.PipelineGeneratedName + "-aaa"
			uploadPipelineAndVerify(pipelineSpecFilePath, &name1, nil)
			name2 := testContext.Pipeline.PipelineGeneratedName + "-zzz"
			uploadPipelineAndVerify(pipelineSpecFilePath, &name2, nil)

			sortBy := "name asc"
			params := newListPipelinesParams()
			params.SortBy = &sortBy
			pipelines, _, _, err := pipelineClient.List(params)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(pipelines)).To(BeNumerically(">=", 2))
			for i := 1; i < len(pipelines); i++ {
				cur := strings.ToLower(pipelines[i].Name)
				prev := strings.ToLower(pipelines[i-1].Name)
				logger.Log("Name ascending sort check [%d]: %q >= %q => %v", i, pipelines[i].Name, pipelines[i-1].Name, cur >= prev)
				Expect(cur >= prev).To(BeTrue(),
					fmt.Sprintf("Pipelines should be sorted by name ascending: [%d] %q should be >= [%d] %q", i, pipelines[i].Name, i-1, pipelines[i-1].Name))
			}
		})

		It("Sort by name in descending order", func() {
			sortBy := "name desc"
			params := newListPipelinesParams()
			params.SortBy = &sortBy
			pipelines, _, _, err := pipelineClient.List(params)
			Expect(err).NotTo(HaveOccurred())
			for i := 1; i < len(pipelines); i++ {
				cur := strings.ToLower(pipelines[i].Name)
				prev := strings.ToLower(pipelines[i-1].Name)
				logger.Log("Name descending sort check [%d]: %q <= %q => %v", i, pipelines[i].Name, pipelines[i-1].Name, cur <= prev)
				Expect(cur <= prev).To(BeTrue(),
					fmt.Sprintf("Pipelines should be sorted by name descending: [%d] %q should be <= [%d] %q", i, pipelines[i].Name, i-1, pipelines[i-1].Name))
			}
		})

		It("Sort by display name containing substring in ascending order", func() {
			sortBy := "display_name asc"
			params := newListPipelinesParams()
			params.SortBy = &sortBy
			pipelines, _, _, err := pipelineClient.List(params)
			Expect(err).NotTo(HaveOccurred())
			for i := 1; i < len(pipelines); i++ {
				cur := strings.ToLower(pipelines[i].DisplayName)
				prev := strings.ToLower(pipelines[i-1].DisplayName)
				logger.Log("Ascending sort check [%d]: %q >= %q => %v", i, pipelines[i].DisplayName, pipelines[i-1].DisplayName, cur >= prev)
				Expect(cur >= prev).To(BeTrue(),
					fmt.Sprintf("Pipelines should be sorted by display name ascending: [%d] %q should be >= [%d] %q", i, pipelines[i].DisplayName, i-1, pipelines[i-1].DisplayName))
			}
		})

		It("Sort by display name containing substring in descending order", func() {
			sortBy := "display_name desc"
			params := newListPipelinesParams()
			params.SortBy = &sortBy
			pipelines, _, _, err := pipelineClient.List(params)
			Expect(err).NotTo(HaveOccurred())
			for i := 1; i < len(pipelines); i++ {
				cur := strings.ToLower(pipelines[i].DisplayName)
				prev := strings.ToLower(pipelines[i-1].DisplayName)
				logger.Log("Descending sort check [%d]: %q <= %q => %v", i, pipelines[i].DisplayName, pipelines[i-1].DisplayName, cur <= prev)
				Expect(cur <= prev).To(BeTrue(),
					fmt.Sprintf("Pipelines should be sorted by display name descending: [%d] %q should be <= [%d] %q", i, pipelines[i].DisplayName, i-1, pipelines[i-1].DisplayName))
			}
		})

		It("Sort by creation date in ascending order", func() {
			sortBy := "created_at asc"
			params := newListPipelinesParams()
			params.SortBy = &sortBy
			_, _, _, err := pipelineClient.List(params)
			Expect(err).NotTo(HaveOccurred())
		})

		It("Sort by creation date in descending order", func() {
			sortBy := "created_at desc"
			params := newListPipelinesParams()
			params.SortBy = &sortBy
			_, _, _, err := pipelineClient.List(params)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("Filtering >", func() {
		It("Filter by pipeline id", func() {
			pipelineDir := "valid"
			pipelineSpecFilePath := filepath.Join(pipelineFilesRootDir, pipelineDir, helloWorldPipelineFileName)
			createdPipeline := uploadPipelineAndVerify(pipelineSpecFilePath, &testContext.Pipeline.PipelineGeneratedName, nil)

			filter := fmt.Sprintf(`{"predicates":[{"key":"pipeline_id","operation":"EQUALS","string_value":"%s"}]}`, createdPipeline.PipelineID)
			params := newListPipelinesParams()
			params.Filter = &filter
			pipelines, totalSize, _, err := pipelineClient.List(params)
			Expect(err).NotTo(HaveOccurred())
			Expect(totalSize).To(Equal(1))
			Expect(pipelines[0].PipelineID).To(Equal(createdPipeline.PipelineID))
		})

		It("Filter by name", func() {
			pipelineDir := "valid"
			pipelineSpecFilePath := filepath.Join(pipelineFilesRootDir, pipelineDir, helloWorldPipelineFileName)
			uploadPipelineAndVerify(pipelineSpecFilePath, &testContext.Pipeline.PipelineGeneratedName, nil)

			filter := fmt.Sprintf(`{"predicates":[{"key":"name","operation":"EQUALS","string_value":"%s"}]}`, testContext.Pipeline.PipelineGeneratedName)
			params := newListPipelinesParams()
			params.Filter = &filter
			pipelines, totalSize, _, err := pipelineClient.List(params)
			Expect(err).NotTo(HaveOccurred())
			Expect(totalSize).To(BeNumerically(">=", 1))
			for _, p := range pipelines {
				Expect(p.Name).To(Equal(testContext.Pipeline.PipelineGeneratedName))
			}
			_ = pipelines
		})

		It("Filter by created at", func() {
			if *config.PipelineStoreKubernetes {
				Skip("GREATER_THAN filter is not supported in Kubernetes pipeline store")
			}
			filter := `{"predicates":[{"key":"created_at","operation":"GREATER_THAN","string_value":"2000-01-01T00:00:00Z"}]}`
			params := newListPipelinesParams()
			params.Filter = &filter
			_, _, _, err := pipelineClient.List(params)
			Expect(err).NotTo(HaveOccurred())
		})

		It("Filter by namespace", func() {
			namespace := utils.GetNamespace()
			params := &pipeline_params.PipelineServiceListPipelinesParams{
				Namespace: &namespace,
			}
			_, _, _, err := pipelineClient.List(params)
			Expect(err).NotTo(HaveOccurred())
		})

		It("Filter by description", func() {
			pipelineDir := "valid"
			pipelineSpecFilePath := filepath.Join(pipelineFilesRootDir, pipelineDir, helloWorldPipelineFileName)
			description := "unique-test-description-" + randomName
			testContext.Pipeline.UploadParams.SetDescription(&description)
			testContext.Pipeline.ExpectedPipeline.Description = description
			uploadPipelineAndVerify(pipelineSpecFilePath, &testContext.Pipeline.PipelineGeneratedName, nil)

			filter := fmt.Sprintf(`{"predicates":[{"key":"description","operation":"EQUALS","string_value":"%s"}]}`, description)
			params := newListPipelinesParams()
			params.Filter = &filter
			pipelines, totalSize, _, err := pipelineClient.List(params)
			Expect(err).NotTo(HaveOccurred())
			Expect(totalSize).To(BeNumerically(">=", 1))
			for _, p := range pipelines {
				Expect(p.Description).To(Equal(description))
			}
			_ = pipelines
		})

		It("Filter by name NOT_EQUALS", func() {
			pipelineDir := "valid"
			pipelineSpecFilePath := filepath.Join(pipelineFilesRootDir, pipelineDir, helloWorldPipelineFileName)
			createdPipeline := uploadPipelineAndVerify(pipelineSpecFilePath, &testContext.Pipeline.PipelineGeneratedName, nil)

			filter := fmt.Sprintf(`{"predicates":[{"key":"name","operation":"NOT_EQUALS","string_value":"%s"}]}`, testContext.Pipeline.PipelineGeneratedName)
			params := newListPipelinesParams()
			params.Filter = &filter
			pipelines, _, _, err := pipelineClient.List(params)
			Expect(err).NotTo(HaveOccurred())
			for _, p := range pipelines {
				Expect(p.Name).NotTo(Equal(testContext.Pipeline.PipelineGeneratedName),
					"NOT_EQUALS filter should exclude the pipeline with matching name")
			}
			// The created pipeline should not appear in the results
			for _, p := range pipelines {
				Expect(p.PipelineID).NotTo(Equal(createdPipeline.PipelineID))
			}
		})

		It("Filter by pipeline_id IN with string_value should fail", Label(constants.NEGATIVE), func() {
			pipelineDir := "valid"
			pipelineSpecFilePath := filepath.Join(pipelineFilesRootDir, pipelineDir, helloWorldPipelineFileName)
			createdPipeline := uploadPipelineAndVerify(pipelineSpecFilePath, &testContext.Pipeline.PipelineGeneratedName, nil)

			filter := fmt.Sprintf(`{"predicates":[{"key":"pipeline_id","operation":"IN","string_value":"%s"}]}`, createdPipeline.PipelineID)
			params := newListPipelinesParams()
			params.Filter = &filter
			_, _, _, err := pipelineClient.List(params)
			Expect(err).To(HaveOccurred(), "IN filter with scalar string_value should be rejected")
		})

		It("Filter by pipeline_id IN with stringValues array", func() {
			pipelineDir := "valid"
			pipelineSpecFilePath := filepath.Join(pipelineFilesRootDir, pipelineDir, helloWorldPipelineFileName)
			createdPipeline := uploadPipelineAndVerify(pipelineSpecFilePath, &testContext.Pipeline.PipelineGeneratedName, nil)

			filter := fmt.Sprintf(`{"predicates":[{"key":"pipeline_id","operation":"IN","stringValues":{"values":["%s"]}}]}`, createdPipeline.PipelineID)
			params := newListPipelinesParams()
			params.Filter = &filter
			pipelines, totalSize, _, err := pipelineClient.List(params)
			Expect(err).NotTo(HaveOccurred())
			Expect(totalSize).To(Equal(1))
			Expect(pipelines[0].PipelineID).To(Equal(createdPipeline.PipelineID))
		})
	})

	Context("Combined Parameters >", func() {
		It("Filter and sort by name in ascending order", func() {
			filter := `{"predicates":[{"key":"name","operation":"IS_SUBSTRING","string_value":"apitest"}]}`
			sortBy := "name asc"
			params := newListPipelinesParams()
			params.Filter = &filter
			params.SortBy = &sortBy
			pipelines, _, _, err := pipelineClient.List(params)
			Expect(err).NotTo(HaveOccurred())
			for i := 1; i < len(pipelines); i++ {
				cur := strings.ToLower(pipelines[i].Name)
				prev := strings.ToLower(pipelines[i-1].Name)
				Expect(cur >= prev).To(BeTrue(),
					fmt.Sprintf("Expected pipeline names in ascending order, but got '%s' before '%s'", pipelines[i-1].Name, pipelines[i].Name))
			}
		})

		It("Filter and sort by created date in descending order", func() {
			if *config.PipelineStoreKubernetes {
				Skip("GREATER_THAN filter is not supported in Kubernetes pipeline store")
			}
			filter := `{"predicates":[{"key":"created_at","operation":"GREATER_THAN","string_value":"2000-01-01T00:00:00Z"}]}`
			sortBy := "created_at desc"
			params := newListPipelinesParams()
			params.Filter = &filter
			params.SortBy = &sortBy
			_, _, _, err := pipelineClient.List(params)
			Expect(err).NotTo(HaveOccurred())
		})

		It("Filter by created date and sort by updated date in descending order", func() {
			if *config.PipelineStoreKubernetes {
				Skip("GREATER_THAN filter is not supported in Kubernetes pipeline store")
			}
			filter := `{"predicates":[{"key":"created_at","operation":"GREATER_THAN","string_value":"2000-01-01T00:00:00Z"}]}`
			sortBy := "created_at desc"
			params := newListPipelinesParams()
			params.Filter = &filter
			params.SortBy = &sortBy
			_, _, _, err := pipelineClient.List(params)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("Filter by Tags >", func() {
		It("Filter pipelines by a single tag", func() {
			pipelineDir := "valid"
			pipelineSpecFilePath := filepath.Join(pipelineFilesRootDir, pipelineDir, helloWorldPipelineFileName)

			tags := map[string]string{"team": "ml-ops"}
			testContext.Pipeline.UploadParams.SetTagsMap(tags)
			testContext.Pipeline.ExpectedPipeline.Tags = tags
			createdPipeline := uploadPipelineAndVerify(pipelineSpecFilePath, &testContext.Pipeline.PipelineGeneratedName, nil)

			filter := `{"predicates":[{"key":"tags.team","operation":"EQUALS","string_value":"ml-ops"}]}`
			params := newListPipelinesParams()
			params.Filter = &filter
			pipelines, totalSize, _, err := pipelineClient.List(params)
			Expect(err).NotTo(HaveOccurred())
			Expect(totalSize).To(BeNumerically(">=", 1))

			found := false
			for _, p := range pipelines {
				if p.PipelineID == createdPipeline.PipelineID {
					found = true
					Expect(p.Tags).To(HaveKeyWithValue("team", "ml-ops"))
					break
				}
			}
			Expect(found).To(BeTrue(), "Pipeline with tag should appear in filtered results")
		})

		It("Filter pipelines by multiple tags", func() {
			pipelineDir := "valid"
			pipelineSpecFilePath := filepath.Join(pipelineFilesRootDir, pipelineDir, helloWorldPipelineFileName)

			tags := map[string]string{"team": "ml-ops", "env": "prod"}
			testContext.Pipeline.UploadParams.SetTagsMap(tags)
			testContext.Pipeline.ExpectedPipeline.Tags = tags
			createdPipeline := uploadPipelineAndVerify(pipelineSpecFilePath, &testContext.Pipeline.PipelineGeneratedName, nil)

			filter := `{"predicates":[{"key":"tags.team","operation":"EQUALS","string_value":"ml-ops"},{"key":"tags.env","operation":"EQUALS","string_value":"prod"}]}`
			params := newListPipelinesParams()
			params.Filter = &filter
			pipelines, totalSize, _, err := pipelineClient.List(params)
			Expect(err).NotTo(HaveOccurred())
			Expect(totalSize).To(BeNumerically(">=", 1))

			found := false
			for _, p := range pipelines {
				if p.PipelineID == createdPipeline.PipelineID {
					found = true
					break
				}
			}
			Expect(found).To(BeTrue(), "Pipeline with both tags should appear in filtered results")
		})

		It("Filter by tag that no pipeline has returns empty", func() {
			filter := `{"predicates":[{"key":"tags.nonexistent","operation":"EQUALS","string_value":"value"}]}`
			params := newListPipelinesParams()
			params.Filter = &filter
			pipelines, totalSize, _, err := pipelineClient.List(params)
			Expect(err).NotTo(HaveOccurred())
			Expect(totalSize).To(Equal(0))
			Expect(pipelines).To(BeEmpty())
		})
	})
})

var _ = Describe("List Pipelines Versions API Tests >", Label(constants.POSITIVE, constants.Pipeline, "PipelineVersionList", constants.APIServerTests, constants.FullRegression), func() {

	Context("Basic List Operations >", func() {
		It("When no pipeline versions exist", func() {
			// Create a pipeline without any version (using Create API, not upload)
			pipelineName := testContext.Pipeline.PipelineGeneratedName + "-no-versions"
			createParams := &pipeline_params.PipelineServiceCreatePipelineParams{
				Pipeline: &pipeline_model.V2beta1Pipeline{
					Name:      pipelineName,
					Namespace: utils.GetNamespace(),
				},
			}
			createdPipeline, err := pipelineClient.Create(createParams)
			Expect(err).NotTo(HaveOccurred())
			testContext.Pipeline.CreatedPipelines = append(testContext.Pipeline.CreatedPipelines, toUploadModel(createdPipeline))

			versions, totalSize, _, err := pipelineClient.ListPipelineVersions(&pipeline_params.PipelineServiceListPipelineVersionsParams{
				PipelineID: createdPipeline.PipelineID,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(totalSize).To(Equal(0))
			Expect(versions).To(BeEmpty())
		})

		It("After creating a single pipeline version", func() {
			pipelineDir := "valid"
			pipelineSpecFilePath := filepath.Join(pipelineFilesRootDir, pipelineDir, helloWorldPipelineFileName)
			createdPipeline := uploadPipelineAndVerify(pipelineSpecFilePath, &testContext.Pipeline.PipelineGeneratedName, nil)

			versions, totalSize, _, err := pipelineClient.ListPipelineVersions(&pipeline_params.PipelineServiceListPipelineVersionsParams{
				PipelineID: createdPipeline.PipelineID,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(totalSize).To(Equal(1))
			Expect(versions).To(HaveLen(1))
			Expect(versions[0].PipelineID).To(Equal(createdPipeline.PipelineID))
		})

		It("After creating multiple pipeline versions", func() {
			pipelineDir := "valid"
			pipelineSpecFilePath := filepath.Join(pipelineFilesRootDir, pipelineDir, helloWorldPipelineFileName)
			createdPipeline := uploadPipelineAndVerify(pipelineSpecFilePath, &testContext.Pipeline.PipelineGeneratedName, nil)

			// Create a second version
			v2Name := testContext.Pipeline.PipelineGeneratedName + "-v2"
			uploadParams2 := uploadparams.NewUploadPipelineVersionParams()
			uploadParams2.Pipelineid = &createdPipeline.PipelineID
			uploadParams2.Name = &v2Name
			_, err := uploadPipelineVersion(pipelineSpecFilePath, uploadParams2)
			Expect(err).NotTo(HaveOccurred())

			// Create a third version
			v3Name := testContext.Pipeline.PipelineGeneratedName + "-v3"
			uploadParams3 := uploadparams.NewUploadPipelineVersionParams()
			uploadParams3.Pipelineid = &createdPipeline.PipelineID
			uploadParams3.Name = &v3Name
			_, err = uploadPipelineVersion(pipelineSpecFilePath, uploadParams3)
			Expect(err).NotTo(HaveOccurred())

			versions, totalSize, _, err := pipelineClient.ListPipelineVersions(&pipeline_params.PipelineServiceListPipelineVersionsParams{
				PipelineID: createdPipeline.PipelineID,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(totalSize).To(Equal(3))
			Expect(versions).To(HaveLen(3))
		})

		It("By pipeline ID - only returns versions for that pipeline", func() {
			pipelineDir := "valid"
			pipelineSpecFilePath := filepath.Join(pipelineFilesRootDir, pipelineDir, helloWorldPipelineFileName)

			name1 := testContext.Pipeline.PipelineGeneratedName + "-pip1"
			createdPipeline1 := uploadPipelineAndVerify(pipelineSpecFilePath, &name1, nil)
			name2 := testContext.Pipeline.PipelineGeneratedName + "-pip2"
			createdPipeline2 := uploadPipelineAndVerify(pipelineSpecFilePath, &name2, nil)

			versions1, _, _, err := pipelineClient.ListPipelineVersions(&pipeline_params.PipelineServiceListPipelineVersionsParams{
				PipelineID: createdPipeline1.PipelineID,
			})
			Expect(err).NotTo(HaveOccurred())
			for _, v := range versions1 {
				Expect(v.PipelineID).To(Equal(createdPipeline1.PipelineID))
			}

			versions2, _, _, err := pipelineClient.ListPipelineVersions(&pipeline_params.PipelineServiceListPipelineVersionsParams{
				PipelineID: createdPipeline2.PipelineID,
			})
			Expect(err).NotTo(HaveOccurred())
			for _, v := range versions2 {
				Expect(v.PipelineID).To(Equal(createdPipeline2.PipelineID))
			}
		})
	})

	Context("Pagination >", func() {
		It("List pipeline versions with page size limit", func() {
			pipelineDir := "valid"
			pipelineSpecFilePath := filepath.Join(pipelineFilesRootDir, pipelineDir, helloWorldPipelineFileName)
			createdPipeline := uploadPipelineAndVerify(pipelineSpecFilePath, &testContext.Pipeline.PipelineGeneratedName, nil)

			// Create a second version
			v2Name := testContext.Pipeline.PipelineGeneratedName + "-v2-page"
			uploadParams := uploadparams.NewUploadPipelineVersionParams()
			uploadParams.Pipelineid = &createdPipeline.PipelineID
			uploadParams.Name = &v2Name
			_, err := uploadPipelineVersion(pipelineSpecFilePath, uploadParams)
			Expect(err).NotTo(HaveOccurred())

			pageSize := int32(1)
			versions, totalSize, nextPageToken, err := pipelineClient.ListPipelineVersions(&pipeline_params.PipelineServiceListPipelineVersionsParams{
				PipelineID: createdPipeline.PipelineID,
				PageSize:   &pageSize,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(versions).To(HaveLen(1))
			Expect(totalSize).To(BeNumerically(">=", 2))
			Expect(nextPageToken).NotTo(BeEmpty(), "Next page token should be set when more results exist")
		})

		It("List pipeline versions with pagination - iterate through all pages (at least 2)", func() {
			pipelineDir := "valid"
			pipelineSpecFilePath := filepath.Join(pipelineFilesRootDir, pipelineDir, helloWorldPipelineFileName)
			createdPipeline := uploadPipelineAndVerify(pipelineSpecFilePath, &testContext.Pipeline.PipelineGeneratedName, nil)

			// Create 2 more versions
			for i := 2; i <= 3; i++ {
				vName := fmt.Sprintf("%s-v%d-iter", testContext.Pipeline.PipelineGeneratedName, i)
				uploadParamsIter := uploadparams.NewUploadPipelineVersionParams()
				uploadParamsIter.Pipelineid = &createdPipeline.PipelineID
				uploadParamsIter.Name = &vName
				_, err := uploadPipelineVersion(pipelineSpecFilePath, uploadParamsIter)
				Expect(err).NotTo(HaveOccurred())
			}

			pageSize := int32(2)
			params := &pipeline_params.PipelineServiceListPipelineVersionsParams{
				PipelineID: createdPipeline.PipelineID,
				PageSize:   &pageSize,
			}

			allVersions := make([]*pipeline_model.V2beta1PipelineVersion, 0)
			pagesVisited := 0

			for {
				versions, _, nextPageToken, err := pipelineClient.ListPipelineVersions(params)
				Expect(err).NotTo(HaveOccurred())
				allVersions = append(allVersions, versions...)
				pagesVisited++

				if nextPageToken == "" {
					break
				}
				params.PageToken = &nextPageToken
			}

			Expect(pagesVisited).To(BeNumerically(">=", 2), "Should visit at least 2 pages")
			Expect(len(allVersions)).To(BeNumerically(">=", 3))
		})
	})

	Context("Sorting >", func() {
		It("Sort by name in ascending order", func() {
			pipelineDir := "valid"
			pipelineSpecFilePath := filepath.Join(pipelineFilesRootDir, pipelineDir, helloWorldPipelineFileName)
			createdPipeline := uploadPipelineAndVerify(pipelineSpecFilePath, &testContext.Pipeline.PipelineGeneratedName, nil)

			// Create additional version
			v2Name := testContext.Pipeline.PipelineGeneratedName + "-zzz-sort"
			uploadParams := uploadparams.NewUploadPipelineVersionParams()
			uploadParams.Pipelineid = &createdPipeline.PipelineID
			uploadParams.Name = &v2Name
			_, err := uploadPipelineVersion(pipelineSpecFilePath, uploadParams)
			Expect(err).NotTo(HaveOccurred())

			sortBy := "name asc"
			versions, _, _, err := pipelineClient.ListPipelineVersions(&pipeline_params.PipelineServiceListPipelineVersionsParams{
				PipelineID: createdPipeline.PipelineID,
				SortBy:     &sortBy,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(len(versions)).To(BeNumerically(">=", 2))
			for i := 1; i < len(versions); i++ {
				cur := strings.ToLower(versions[i].Name)
				prev := strings.ToLower(versions[i-1].Name)
				logger.Log("Version name ascending sort check [%d]: %q >= %q => %v", i, versions[i].Name, versions[i-1].Name, cur >= prev)
				Expect(cur >= prev).To(BeTrue(),
					fmt.Sprintf("Versions should be sorted by name ascending: [%d] %q should be >= [%d] %q", i, versions[i].Name, i-1, versions[i-1].Name))
			}
		})

		It("Sort by name in descending order", func() {
			pipelineDir := "valid"
			pipelineSpecFilePath := filepath.Join(pipelineFilesRootDir, pipelineDir, helloWorldPipelineFileName)
			createdPipeline := uploadPipelineAndVerify(pipelineSpecFilePath, &testContext.Pipeline.PipelineGeneratedName, nil)

			v2Name := testContext.Pipeline.PipelineGeneratedName + "-zzz-sortdesc"
			uploadParams := uploadparams.NewUploadPipelineVersionParams()
			uploadParams.Pipelineid = &createdPipeline.PipelineID
			uploadParams.Name = &v2Name
			_, err := uploadPipelineVersion(pipelineSpecFilePath, uploadParams)
			Expect(err).NotTo(HaveOccurred())

			sortBy := "name desc"
			versions, _, _, err := pipelineClient.ListPipelineVersions(&pipeline_params.PipelineServiceListPipelineVersionsParams{
				PipelineID: createdPipeline.PipelineID,
				SortBy:     &sortBy,
			})
			Expect(err).NotTo(HaveOccurred())
			for i := 1; i < len(versions); i++ {
				cur := strings.ToLower(versions[i].Name)
				prev := strings.ToLower(versions[i-1].Name)
				logger.Log("Version name descending sort check [%d]: %q <= %q => %v", i, versions[i].Name, versions[i-1].Name, cur <= prev)
				Expect(cur <= prev).To(BeTrue(),
					fmt.Sprintf("Versions should be sorted by name descending: [%d] %q should be <= [%d] %q", i, versions[i].Name, i-1, versions[i-1].Name))
			}
		})

		It("Sort by display name containing substring in ascending order", func() {
			pipelineDir := "valid"
			pipelineSpecFilePath := filepath.Join(pipelineFilesRootDir, pipelineDir, helloWorldPipelineFileName)
			createdPipeline := uploadPipelineAndVerify(pipelineSpecFilePath, &testContext.Pipeline.PipelineGeneratedName, nil)

			sortBy := "display_name asc"
			versions, _, _, err := pipelineClient.ListPipelineVersions(&pipeline_params.PipelineServiceListPipelineVersionsParams{
				PipelineID: createdPipeline.PipelineID,
				SortBy:     &sortBy,
			})
			Expect(err).NotTo(HaveOccurred())
			for i := 1; i < len(versions); i++ {
				cur := strings.ToLower(versions[i].DisplayName)
				prev := strings.ToLower(versions[i-1].DisplayName)
				logger.Log("Version ascending sort check [%d]: %q >= %q => %v", i, versions[i].DisplayName, versions[i-1].DisplayName, cur >= prev)
				Expect(cur >= prev).To(BeTrue(),
					fmt.Sprintf("Versions should be sorted by display name ascending: [%d] %q should be >= [%d] %q", i, versions[i].DisplayName, i-1, versions[i-1].DisplayName))
			}
		})

		It("Sort by display name containing substring in descending order", func() {
			pipelineDir := "valid"
			pipelineSpecFilePath := filepath.Join(pipelineFilesRootDir, pipelineDir, helloWorldPipelineFileName)
			createdPipeline := uploadPipelineAndVerify(pipelineSpecFilePath, &testContext.Pipeline.PipelineGeneratedName, nil)

			sortBy := "display_name desc"
			versions, _, _, err := pipelineClient.ListPipelineVersions(&pipeline_params.PipelineServiceListPipelineVersionsParams{
				PipelineID: createdPipeline.PipelineID,
				SortBy:     &sortBy,
			})
			Expect(err).NotTo(HaveOccurred())
			for i := 1; i < len(versions); i++ {
				cur := strings.ToLower(versions[i].DisplayName)
				prev := strings.ToLower(versions[i-1].DisplayName)
				logger.Log("Version descending sort check [%d]: %q <= %q => %v", i, versions[i].DisplayName, versions[i-1].DisplayName, cur <= prev)
				Expect(cur <= prev).To(BeTrue(),
					fmt.Sprintf("Versions should be sorted by display name descending: [%d] %q should be <= [%d] %q", i, versions[i].DisplayName, i-1, versions[i-1].DisplayName))
			}
		})

		It("Sort by creation date in ascending order", func() {
			pipelineDir := "valid"
			pipelineSpecFilePath := filepath.Join(pipelineFilesRootDir, pipelineDir, helloWorldPipelineFileName)
			createdPipeline := uploadPipelineAndVerify(pipelineSpecFilePath, &testContext.Pipeline.PipelineGeneratedName, nil)

			sortBy := "created_at asc"
			_, _, _, err := pipelineClient.ListPipelineVersions(&pipeline_params.PipelineServiceListPipelineVersionsParams{
				PipelineID: createdPipeline.PipelineID,
				SortBy:     &sortBy,
			})
			Expect(err).NotTo(HaveOccurred())
		})

		It("Sort by creation date in descending order", func() {
			pipelineDir := "valid"
			pipelineSpecFilePath := filepath.Join(pipelineFilesRootDir, pipelineDir, helloWorldPipelineFileName)
			createdPipeline := uploadPipelineAndVerify(pipelineSpecFilePath, &testContext.Pipeline.PipelineGeneratedName, nil)

			sortBy := "created_at desc"
			_, _, _, err := pipelineClient.ListPipelineVersions(&pipeline_params.PipelineServiceListPipelineVersionsParams{
				PipelineID: createdPipeline.PipelineID,
				SortBy:     &sortBy,
			})
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("Filtering >", func() {
		It("Filter by pipeline version id", func() {
			pipelineDir := "valid"
			pipelineSpecFilePath := filepath.Join(pipelineFilesRootDir, pipelineDir, helloWorldPipelineFileName)
			createdPipeline := uploadPipelineAndVerify(pipelineSpecFilePath, &testContext.Pipeline.PipelineGeneratedName, nil)

			versions := utils.GetSortedPipelineVersionsByCreatedAt(pipelineClient, createdPipeline.PipelineID, nil)
			Expect(versions).Should(HaveLen(1))

			filter := fmt.Sprintf(`{"predicates":[{"key":"pipeline_version_id","operation":"EQUALS","string_value":"%s"}]}`, versions[0].PipelineVersionID)
			filteredVersions, totalSize, _, err := pipelineClient.ListPipelineVersions(&pipeline_params.PipelineServiceListPipelineVersionsParams{
				PipelineID: createdPipeline.PipelineID,
				Filter:     &filter,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(totalSize).To(Equal(1))
			Expect(filteredVersions[0].PipelineVersionID).To(Equal(versions[0].PipelineVersionID))
		})

		It("Filter by name", func() {
			pipelineDir := "valid"
			pipelineSpecFilePath := filepath.Join(pipelineFilesRootDir, pipelineDir, helloWorldPipelineFileName)
			createdPipeline := uploadPipelineAndVerify(pipelineSpecFilePath, &testContext.Pipeline.PipelineGeneratedName, nil)

			// Create a second version with a specific name
			v2Name := testContext.Pipeline.PipelineGeneratedName + "-filter-name"
			uploadParams := uploadparams.NewUploadPipelineVersionParams()
			uploadParams.Pipelineid = &createdPipeline.PipelineID
			uploadParams.Name = &v2Name
			_, err := uploadPipelineVersion(pipelineSpecFilePath, uploadParams)
			Expect(err).NotTo(HaveOccurred())

			filter := fmt.Sprintf(`{"predicates":[{"key":"name","operation":"EQUALS","string_value":"%s"}]}`, v2Name)
			filteredVersions, totalSize, _, err := pipelineClient.ListPipelineVersions(&pipeline_params.PipelineServiceListPipelineVersionsParams{
				PipelineID: createdPipeline.PipelineID,
				Filter:     &filter,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(totalSize).To(BeNumerically(">=", 1))
			for _, v := range filteredVersions {
				Expect(v.Name).To(Equal(v2Name))
			}
		})

		It("Filter by created at", func() {
			if *config.PipelineStoreKubernetes {
				Skip("GREATER_THAN filter is not supported in Kubernetes pipeline store")
			}
			pipelineDir := "valid"
			pipelineSpecFilePath := filepath.Join(pipelineFilesRootDir, pipelineDir, helloWorldPipelineFileName)
			createdPipeline := uploadPipelineAndVerify(pipelineSpecFilePath, &testContext.Pipeline.PipelineGeneratedName, nil)

			filter := `{"predicates":[{"key":"created_at","operation":"GREATER_THAN","string_value":"2000-01-01T00:00:00Z"}]}`
			_, _, _, err := pipelineClient.ListPipelineVersions(&pipeline_params.PipelineServiceListPipelineVersionsParams{
				PipelineID: createdPipeline.PipelineID,
				Filter:     &filter,
			})
			Expect(err).NotTo(HaveOccurred())
		})

		It("Filter by description", func() {
			pipelineDir := "valid"
			pipelineSpecFilePath := filepath.Join(pipelineFilesRootDir, pipelineDir, helloWorldPipelineFileName)
			createdPipeline := uploadPipelineAndVerify(pipelineSpecFilePath, &testContext.Pipeline.PipelineGeneratedName, nil)

			// Create a version with a specific description
			v2Name := testContext.Pipeline.PipelineGeneratedName + "-v2-desc"
			v2Desc := "unique-version-description-" + randomName
			uploadParams := uploadparams.NewUploadPipelineVersionParams()
			uploadParams.Pipelineid = &createdPipeline.PipelineID
			uploadParams.Name = &v2Name
			uploadParams.Description = &v2Desc
			_, err := uploadPipelineVersion(pipelineSpecFilePath, uploadParams)
			Expect(err).NotTo(HaveOccurred())

			filter := fmt.Sprintf(`{"predicates":[{"key":"description","operation":"EQUALS","string_value":"%s"}]}`, v2Desc)
			filteredVersions, totalSize, _, err := pipelineClient.ListPipelineVersions(&pipeline_params.PipelineServiceListPipelineVersionsParams{
				PipelineID: createdPipeline.PipelineID,
				Filter:     &filter,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(totalSize).To(BeNumerically(">=", 1))
			for _, v := range filteredVersions {
				Expect(v.Description).To(Equal(v2Desc))
			}
		})

		It("Filter by name NOT_EQUALS", func() {
			pipelineDir := "valid"
			pipelineSpecFilePath := filepath.Join(pipelineFilesRootDir, pipelineDir, helloWorldPipelineFileName)
			createdPipeline := uploadPipelineAndVerify(pipelineSpecFilePath, &testContext.Pipeline.PipelineGeneratedName, nil)

			// Get the default version
			versions, _, _, err := pipelineClient.ListPipelineVersions(&pipeline_params.PipelineServiceListPipelineVersionsParams{
				PipelineID: createdPipeline.PipelineID,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(len(versions)).To(BeNumerically(">=", 1))

			// Create a second version with a different name
			v2Name := testContext.Pipeline.PipelineGeneratedName + "-v2-neq"
			uploadParams := uploadparams.NewUploadPipelineVersionParams()
			uploadParams.Pipelineid = &createdPipeline.PipelineID
			uploadParams.Name = &v2Name
			_, err = uploadPipelineVersion(pipelineSpecFilePath, uploadParams)
			Expect(err).NotTo(HaveOccurred())

			// Filter to exclude the first version by name
			filter := fmt.Sprintf(`{"predicates":[{"key":"name","operation":"NOT_EQUALS","string_value":"%s"}]}`, versions[0].Name)
			filteredVersions, _, _, err := pipelineClient.ListPipelineVersions(&pipeline_params.PipelineServiceListPipelineVersionsParams{
				PipelineID: createdPipeline.PipelineID,
				Filter:     &filter,
			})
			Expect(err).NotTo(HaveOccurred())
			for _, v := range filteredVersions {
				Expect(v.Name).NotTo(Equal(versions[0].Name),
					"NOT_EQUALS filter should exclude the version with matching name")
			}
		})

		It("Filter by pipeline_version_id IN with string_value should fail", Label(constants.NEGATIVE), func() {
			pipelineDir := "valid"
			pipelineSpecFilePath := filepath.Join(pipelineFilesRootDir, pipelineDir, helloWorldPipelineFileName)
			createdPipeline := uploadPipelineAndVerify(pipelineSpecFilePath, &testContext.Pipeline.PipelineGeneratedName, nil)

			versions, _, _, err := pipelineClient.ListPipelineVersions(&pipeline_params.PipelineServiceListPipelineVersionsParams{
				PipelineID: createdPipeline.PipelineID,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(len(versions)).To(BeNumerically(">=", 1))

			filter := fmt.Sprintf(`{"predicates":[{"key":"pipeline_version_id","operation":"IN","string_value":"%s"}]}`, versions[0].PipelineVersionID)
			_, _, _, err = pipelineClient.ListPipelineVersions(&pipeline_params.PipelineServiceListPipelineVersionsParams{
				PipelineID: createdPipeline.PipelineID,
				Filter:     &filter,
			})
			Expect(err).To(HaveOccurred(), "IN filter with scalar string_value should be rejected")
		})

		It("Filter by pipeline_version_id IN with stringValues array", func() {
			pipelineDir := "valid"
			pipelineSpecFilePath := filepath.Join(pipelineFilesRootDir, pipelineDir, helloWorldPipelineFileName)
			createdPipeline := uploadPipelineAndVerify(pipelineSpecFilePath, &testContext.Pipeline.PipelineGeneratedName, nil)

			versions, _, _, err := pipelineClient.ListPipelineVersions(&pipeline_params.PipelineServiceListPipelineVersionsParams{
				PipelineID: createdPipeline.PipelineID,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(len(versions)).To(BeNumerically(">=", 1))

			filter := fmt.Sprintf(`{"predicates":[{"key":"pipeline_version_id","operation":"IN","stringValues":{"values":["%s"]}}]}`, versions[0].PipelineVersionID)
			filteredVersions, totalSize, _, err := pipelineClient.ListPipelineVersions(&pipeline_params.PipelineServiceListPipelineVersionsParams{
				PipelineID: createdPipeline.PipelineID,
				Filter:     &filter,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(totalSize).To(Equal(1))
			Expect(filteredVersions[0].PipelineVersionID).To(Equal(versions[0].PipelineVersionID))
		})
	})

	Context("Combined Parameters >", func() {
		It("Filter and sort by name in ascending order", func() {
			pipelineDir := "valid"
			pipelineSpecFilePath := filepath.Join(pipelineFilesRootDir, pipelineDir, helloWorldPipelineFileName)
			createdPipeline := uploadPipelineAndVerify(pipelineSpecFilePath, &testContext.Pipeline.PipelineGeneratedName, nil)

			filter := `{"predicates":[{"key":"name","operation":"IS_SUBSTRING","string_value":"apitest"}]}`
			sortBy := "name asc"
			versions, _, _, err := pipelineClient.ListPipelineVersions(&pipeline_params.PipelineServiceListPipelineVersionsParams{
				PipelineID: createdPipeline.PipelineID,
				Filter:     &filter,
				SortBy:     &sortBy,
			})
			Expect(err).NotTo(HaveOccurred())
			for i := 1; i < len(versions); i++ {
				cur := strings.ToLower(versions[i].Name)
				prev := strings.ToLower(versions[i-1].Name)
				Expect(cur >= prev).To(BeTrue(),
					fmt.Sprintf("Expected version names in ascending order, but got '%s' before '%s'", versions[i-1].Name, versions[i].Name))
			}
		})

		It("Filter and sort by created date in descending order", func() {
			if *config.PipelineStoreKubernetes {
				Skip("GREATER_THAN filter is not supported in Kubernetes pipeline store")
			}
			pipelineDir := "valid"
			pipelineSpecFilePath := filepath.Join(pipelineFilesRootDir, pipelineDir, helloWorldPipelineFileName)
			createdPipeline := uploadPipelineAndVerify(pipelineSpecFilePath, &testContext.Pipeline.PipelineGeneratedName, nil)

			filter := `{"predicates":[{"key":"created_at","operation":"GREATER_THAN","string_value":"2000-01-01T00:00:00Z"}]}`
			sortBy := "created_at desc"
			_, _, _, err := pipelineClient.ListPipelineVersions(&pipeline_params.PipelineServiceListPipelineVersionsParams{
				PipelineID: createdPipeline.PipelineID,
				Filter:     &filter,
				SortBy:     &sortBy,
			})
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("Pipeline Version Tags in List >", func() {
		It("Versions with tags should include tags in list response", func() {
			pipelineDir := "valid"
			pipelineSpecFilePath := filepath.Join(pipelineFilesRootDir, pipelineDir, helloWorldPipelineFileName)
			createdPipeline := uploadPipelineAndVerify(pipelineSpecFilePath, &testContext.Pipeline.PipelineGeneratedName, nil)

			// Create a version with tags
			versionTags := map[string]string{"env": "staging", "team": "ml"}
			vName := testContext.Pipeline.PipelineGeneratedName + "-v2-tagged-list"
			uploadParams := uploadparams.NewUploadPipelineVersionParams()
			uploadParams.Pipelineid = &createdPipeline.PipelineID
			uploadParams.Name = &vName
			uploadParams.SetTagsMap(versionTags)
			_, err := uploadPipelineVersion(pipelineSpecFilePath, uploadParams)
			Expect(err).NotTo(HaveOccurred())

			versions, _, _, err := pipelineClient.ListPipelineVersions(&pipeline_params.PipelineServiceListPipelineVersionsParams{
				PipelineID: createdPipeline.PipelineID,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(len(versions)).To(BeNumerically(">=", 2))

			var foundTaggedVersion bool
			for _, v := range versions {
				if v.DisplayName == vName {
					Expect(v.Tags).To(Equal(versionTags), "Tagged version should include tags in list response")
					foundTaggedVersion = true
				}
			}
			Expect(foundTaggedVersion).To(BeTrue(), "Should find the tagged version in the list")
		})

		It("Versions without tags should have empty tags in list response", func() {
			pipelineDir := "valid"
			pipelineSpecFilePath := filepath.Join(pipelineFilesRootDir, pipelineDir, helloWorldPipelineFileName)
			createdPipeline := uploadPipelineAndVerify(pipelineSpecFilePath, &testContext.Pipeline.PipelineGeneratedName, nil)

			versions, _, _, err := pipelineClient.ListPipelineVersions(&pipeline_params.PipelineServiceListPipelineVersionsParams{
				PipelineID: createdPipeline.PipelineID,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(versions).Should(HaveLen(1))
			Expect(versions[0].Tags).To(BeEmpty(), "Version without tags should have empty tags")
		})
	})
})

var _ = Describe("Create Pipeline API Tests >", Label(constants.POSITIVE, constants.Pipeline, "PipelineCreate", constants.APIServerTests, constants.FullRegression), func() {

	Context("Create a pipeline using '/pipelines' >", func() {
		It("With just name", func() {
			pipelineName := testContext.Pipeline.PipelineGeneratedName
			createParams := &pipeline_params.PipelineServiceCreatePipelineParams{
				Pipeline: &pipeline_model.V2beta1Pipeline{
					Name: pipelineName,
				},
			}
			createdPipeline, err := pipelineClient.Create(createParams)
			Expect(err).NotTo(HaveOccurred())
			Expect(createdPipeline.PipelineID).NotTo(BeEmpty())
			Expect(createdPipeline.Name).To(Equal(pipelineName))
			testContext.Pipeline.CreatedPipelines = append(testContext.Pipeline.CreatedPipelines, toUploadModel(createdPipeline))
		})

		It("With name and description", func() {
			pipelineName := testContext.Pipeline.PipelineGeneratedName
			description := "Test pipeline description"
			createParams := &pipeline_params.PipelineServiceCreatePipelineParams{
				Pipeline: &pipeline_model.V2beta1Pipeline{
					Name:        pipelineName,
					Description: description,
				},
			}
			createdPipeline, err := pipelineClient.Create(createParams)
			Expect(err).NotTo(HaveOccurred())
			Expect(createdPipeline.PipelineID).NotTo(BeEmpty())
			Expect(createdPipeline.Name).To(Equal(pipelineName))
			Expect(createdPipeline.Description).To(Equal(description))
			testContext.Pipeline.CreatedPipelines = append(testContext.Pipeline.CreatedPipelines, toUploadModel(createdPipeline))
		})

		It("With name length of 100 chars", func() {
			pipelineName := strings.ToLower(utils.GetRandomString(100))
			createParams := &pipeline_params.PipelineServiceCreatePipelineParams{
				Pipeline: &pipeline_model.V2beta1Pipeline{
					Name: pipelineName,
				},
			}
			createdPipeline, err := pipelineClient.Create(createParams)
			Expect(err).NotTo(HaveOccurred())
			Expect(createdPipeline.PipelineID).NotTo(BeEmpty())
			Expect(createdPipeline.Name).To(Equal(pipelineName))
			testContext.Pipeline.CreatedPipelines = append(testContext.Pipeline.CreatedPipelines, toUploadModel(createdPipeline))
		})

		It("With name containing ASCII characters", func() {
			pipelineName := "test-pipeline-with-ascii-!@#$%"
			createParams := &pipeline_params.PipelineServiceCreatePipelineParams{
				Pipeline: &pipeline_model.V2beta1Pipeline{
					Name: pipelineName,
				},
			}
			if *config.PipelineStoreKubernetes {
				// K8s names must be lowercase alphanumeric, '-' or '.'
				_, err := pipelineClient.Create(createParams)
				Expect(err).To(HaveOccurred())
			} else {
				createdPipeline, err := pipelineClient.Create(createParams)
				Expect(err).NotTo(HaveOccurred())
				Expect(createdPipeline.PipelineID).NotTo(BeEmpty())
				testContext.Pipeline.CreatedPipelines = append(testContext.Pipeline.CreatedPipelines, toUploadModel(createdPipeline))
			}
		})
	})

	Context("Create a pipeline with tags using '/pipelines' >", func() {
		It("With name and tags", func() {
			pipelineName := testContext.Pipeline.PipelineGeneratedName
			tags := map[string]string{"team": "ml-ops", "env": "dev"}
			createParams := &pipeline_params.PipelineServiceCreatePipelineParams{
				Pipeline: &pipeline_model.V2beta1Pipeline{
					Name: pipelineName,
					Tags: tags,
				},
			}
			createdPipeline, err := pipelineClient.Create(createParams)
			Expect(err).NotTo(HaveOccurred())
			Expect(createdPipeline.PipelineID).NotTo(BeEmpty())
			Expect(createdPipeline.Tags).To(Equal(tags))
			testContext.Pipeline.CreatedPipelines = append(testContext.Pipeline.CreatedPipelines, toUploadModel(createdPipeline))

			// Verify tags persist via GetPipeline
			retrievedPipeline := utils.GetPipeline(pipelineClient, createdPipeline.PipelineID)
			Expect(retrievedPipeline.Tags).To(Equal(tags))
		})

		It("With name, description and tags", func() {
			pipelineName := testContext.Pipeline.PipelineGeneratedName
			description := "Pipeline with tags"
			tags := map[string]string{"project": "kfp"}
			createParams := &pipeline_params.PipelineServiceCreatePipelineParams{
				Pipeline: &pipeline_model.V2beta1Pipeline{
					Name:        pipelineName,
					Description: description,
					Tags:        tags,
				},
			}
			createdPipeline, err := pipelineClient.Create(createParams)
			Expect(err).NotTo(HaveOccurred())
			Expect(createdPipeline.Description).To(Equal(description))
			Expect(createdPipeline.Tags).To(Equal(tags))
			testContext.Pipeline.CreatedPipelines = append(testContext.Pipeline.CreatedPipelines, toUploadModel(createdPipeline))
		})

		It("With empty tags map", func() {
			pipelineName := testContext.Pipeline.PipelineGeneratedName
			createParams := &pipeline_params.PipelineServiceCreatePipelineParams{
				Pipeline: &pipeline_model.V2beta1Pipeline{
					Name: pipelineName,
					Tags: map[string]string{},
				},
			}
			createdPipeline, err := pipelineClient.Create(createParams)
			Expect(err).NotTo(HaveOccurred())
			Expect(createdPipeline.PipelineID).NotTo(BeEmpty())
			testContext.Pipeline.CreatedPipelines = append(testContext.Pipeline.CreatedPipelines, toUploadModel(createdPipeline))
		})

		It("Without tags (nil)", func() {
			pipelineName := testContext.Pipeline.PipelineGeneratedName
			createParams := &pipeline_params.PipelineServiceCreatePipelineParams{
				Pipeline: &pipeline_model.V2beta1Pipeline{
					Name: pipelineName,
				},
			}
			createdPipeline, err := pipelineClient.Create(createParams)
			Expect(err).NotTo(HaveOccurred())
			Expect(createdPipeline.PipelineID).NotTo(BeEmpty())
			testContext.Pipeline.CreatedPipelines = append(testContext.Pipeline.CreatedPipelines, toUploadModel(createdPipeline))
		})
	})

	Context("Create a pipeline with version using '/pipelines/create' >", func() {
		var pipelineDir = "valid/samples"
		pipelineFiles := utils.GetListOfFilesInADir(filepath.Join(pipelineFilesRootDir, pipelineDir))
		for _, pipelineFile := range pipelineFiles {
			pipelineFile := pipelineFile // capture range variable
			It(fmt.Sprintf("Pipeline with name and Pipelineversion with name and pipeline spec from file: %s", pipelineFile), func() {
				pipelineSpecFilePath := filepath.Join(pipelineFilesRootDir, pipelineDir, pipelineFile)
				pipelineVersionName := testContext.Pipeline.PipelineGeneratedName + "-v1"
				inputFileContent := utils.ParseFileToSpecs(pipelineSpecFilePath, true, nil)

				createParams := &pipeline_params.PipelineServiceCreatePipelineAndVersionParams{
					Body: &pipeline_model.V2beta1CreatePipelineAndVersionRequest{
						Pipeline: &pipeline_model.V2beta1Pipeline{
							Name: testContext.Pipeline.PipelineGeneratedName,
						},
						PipelineVersion: &pipeline_model.V2beta1PipelineVersion{
							DisplayName: pipelineVersionName,
							PackageURL: &pipeline_model.V2beta1URL{
								PipelineURL: pipelineSpecFilePath,
							},
						},
					},
				}
				createdPipeline, err := pipelineClient.CreatePipelineAndVersion(createParams)
				if err != nil {
					logger.Log("Failed to create pipeline and version from file %s: %v", pipelineFile, err)
				}
				Expect(err).NotTo(HaveOccurred())
				Expect(createdPipeline.PipelineID).NotTo(BeEmpty())
				testContext.Pipeline.CreatedPipelines = append(testContext.Pipeline.CreatedPipelines, toUploadModel(createdPipeline))

				versions := utils.GetSortedPipelineVersionsByCreatedAt(pipelineClient, createdPipeline.PipelineID, nil)
				Expect(versions).Should(HaveLen(1))
				actualPipelineSpec := versions[0].PipelineSpec.(map[string]interface{})
				matcher.MatchPipelineSpecs(actualPipelineSpec, inputFileContent)
			})
		}

		pipelineURLs := []string{"Your actual pipeline URLs go here"}
		for _, pipelineURL := range pipelineURLs {
			It(fmt.Sprintf("Pipeline with name and Pipelineversion with name and pipeline spec from url: %s", pipelineURL), func() {
				Skip("Pipeline URL tests require valid external URLs - skipping placeholder")
			})
		}
	})
})

var _ = Describe("Get Pipeline API Tests >", Label(constants.POSITIVE, constants.Pipeline, "PipelineGet", constants.APIServerTests, constants.FullRegression), func() {

	Context("Get by ID '/pipelines/{pipeline_id}' >", func() {
		It("With ID", func() {
			pipelineDir := "valid"
			pipelineSpecFilePath := filepath.Join(pipelineFilesRootDir, pipelineDir, helloWorldPipelineFileName)
			createdPipeline := uploadPipelineAndVerify(pipelineSpecFilePath, &testContext.Pipeline.PipelineGeneratedName, nil)

			retrievedPipeline, err := pipelineClient.Get(&pipeline_params.PipelineServiceGetPipelineParams{
				PipelineID: createdPipeline.PipelineID,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(retrievedPipeline.PipelineID).To(Equal(createdPipeline.PipelineID))
			Expect(retrievedPipeline.Name).To(Equal(createdPipeline.Name))
			Expect(retrievedPipeline.DisplayName).To(Equal(createdPipeline.DisplayName))
		})

		It("With ID and verify tags are returned", func() {
			pipelineDir := "valid"
			pipelineSpecFilePath := filepath.Join(pipelineFilesRootDir, pipelineDir, helloWorldPipelineFileName)
			tags := map[string]string{"team": "ml-ops", "env": "staging"}
			testContext.Pipeline.UploadParams.SetTagsMap(tags)
			testContext.Pipeline.ExpectedPipeline.Tags = tags
			createdPipeline := uploadPipelineAndVerify(pipelineSpecFilePath, &testContext.Pipeline.PipelineGeneratedName, nil)

			retrievedPipeline, err := pipelineClient.Get(&pipeline_params.PipelineServiceGetPipelineParams{
				PipelineID: createdPipeline.PipelineID,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(retrievedPipeline.PipelineID).To(Equal(createdPipeline.PipelineID))
			Expect(retrievedPipeline.Tags).To(Equal(tags), "Tags should be returned via GetPipeline")
		})

		It("With ID for pipeline created without tags returns no tags", func() {
			pipelineDir := "valid"
			pipelineSpecFilePath := filepath.Join(pipelineFilesRootDir, pipelineDir, helloWorldPipelineFileName)
			createdPipeline := uploadPipelineAndVerify(pipelineSpecFilePath, &testContext.Pipeline.PipelineGeneratedName, nil)

			retrievedPipeline, err := pipelineClient.Get(&pipeline_params.PipelineServiceGetPipelineParams{
				PipelineID: createdPipeline.PipelineID,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(retrievedPipeline.Tags).To(BeEmpty())
		})
	})

	Context("Get by name '/pipelines/names/{name}' >", func() {
		It("With full name", func() {
			pipelineDir := "valid"
			pipelineSpecFilePath := filepath.Join(pipelineFilesRootDir, pipelineDir, helloWorldPipelineFileName)
			createdPipeline := uploadPipelineAndVerify(pipelineSpecFilePath, &testContext.Pipeline.PipelineGeneratedName, nil)

			namespace := utils.GetNamespace()
			retrievedPipeline, err := pipelineClient.GetByName(&pipeline_params.PipelineServiceGetPipelineByNameParams{
				Name:      createdPipeline.Name,
				Namespace: &namespace,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(retrievedPipeline.PipelineID).To(Equal(createdPipeline.PipelineID))
			Expect(retrievedPipeline.Name).To(Equal(createdPipeline.Name))
		})

		It("With name and verify tags are returned", func() {
			pipelineDir := "valid"
			pipelineSpecFilePath := filepath.Join(pipelineFilesRootDir, pipelineDir, helloWorldPipelineFileName)
			tags := map[string]string{"project": "kfp", "owner": "test"}
			testContext.Pipeline.UploadParams.SetTagsMap(tags)
			testContext.Pipeline.ExpectedPipeline.Tags = tags
			createdPipeline := uploadPipelineAndVerify(pipelineSpecFilePath, &testContext.Pipeline.PipelineGeneratedName, nil)

			namespace := utils.GetNamespace()
			retrievedPipeline, err := pipelineClient.GetByName(&pipeline_params.PipelineServiceGetPipelineByNameParams{
				Name:      createdPipeline.Name,
				Namespace: &namespace,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(retrievedPipeline.PipelineID).To(Equal(createdPipeline.PipelineID))
			Expect(retrievedPipeline.Tags).To(Equal(tags), "Tags should be returned via GetPipelineByName")
		})
	})
})

var _ = Describe("Get Pipeline Version API Tests >", Label(constants.POSITIVE, constants.Pipeline, "PipelineVersionGet", constants.APIServerTests, constants.FullRegression), func() {

	Context("Get by id '/pipelines/{pipeline_id}/versions/{pipeline_version_id}' >", func() {
		It("With valid pipeline id and version id", func() {
			pipelineDir := "valid"
			pipelineSpecFilePath := filepath.Join(pipelineFilesRootDir, pipelineDir, helloWorldPipelineFileName)
			createdPipeline := uploadPipelineAndVerify(pipelineSpecFilePath, &testContext.Pipeline.PipelineGeneratedName, nil)

			versions := utils.GetSortedPipelineVersionsByCreatedAt(pipelineClient, createdPipeline.PipelineID, nil)
			Expect(versions).Should(HaveLen(1))

			retrievedVersion, err := pipelineClient.GetPipelineVersion(&pipeline_params.PipelineServiceGetPipelineVersionParams{
				PipelineID:        createdPipeline.PipelineID,
				PipelineVersionID: versions[0].PipelineVersionID,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(retrievedVersion.PipelineVersionID).To(Equal(versions[0].PipelineVersionID))
			Expect(retrievedVersion.PipelineID).To(Equal(createdPipeline.PipelineID))
		})

		It("With valid pipeline id and version id - verify tags are returned when version has no tags", func() {
			pipelineDir := "valid"
			pipelineSpecFilePath := filepath.Join(pipelineFilesRootDir, pipelineDir, helloWorldPipelineFileName)
			createdPipeline := uploadPipelineAndVerify(pipelineSpecFilePath, &testContext.Pipeline.PipelineGeneratedName, nil)

			versions := utils.GetSortedPipelineVersionsByCreatedAt(pipelineClient, createdPipeline.PipelineID, nil)
			Expect(versions).Should(HaveLen(1))

			retrievedVersion, err := pipelineClient.GetPipelineVersion(&pipeline_params.PipelineServiceGetPipelineVersionParams{
				PipelineID:        createdPipeline.PipelineID,
				PipelineVersionID: versions[0].PipelineVersionID,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(retrievedVersion.Tags).To(BeEmpty(), "Tags should be empty when version has no tags")
		})

		It("With valid pipeline id and version id - verify tags are returned when version has tags", func() {
			// Create a pipeline with version using CreatePipelineAndVersion
			pipelineDir := "valid"
			pipelineSpecFilePath := filepath.Join(pipelineFilesRootDir, pipelineDir, helloWorldPipelineFileName)
			versionTags := map[string]string{"env": "prod", "team": "data"}

			createdPipeline := uploadPipelineAndVerify(pipelineSpecFilePath, &testContext.Pipeline.PipelineGeneratedName, nil)

			// Create a new version with tags
			versionName := testContext.Pipeline.PipelineGeneratedName + "-v2-tagged"
			uploadParams := uploadparams.NewUploadPipelineVersionParams()
			uploadParams.Pipelineid = &createdPipeline.PipelineID
			uploadParams.Name = &versionName
			uploadParams.SetTagsMap(versionTags)
			newVersion, err := uploadPipelineVersion(pipelineSpecFilePath, uploadParams)
			Expect(err).NotTo(HaveOccurred())
			Expect(newVersion.PipelineVersionID).NotTo(BeEmpty())

			// Get the version and verify tags
			retrievedVersion, err := pipelineClient.GetPipelineVersion(&pipeline_params.PipelineServiceGetPipelineVersionParams{
				PipelineID:        createdPipeline.PipelineID,
				PipelineVersionID: newVersion.PipelineVersionID,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(retrievedVersion.Tags).To(Equal(versionTags), "Tags should match what was set during creation")
		})

		It("Returns version even when pipeline ID in URL does not match the version's pipeline", func() {
			pipelineDir := "valid"
			pipelineSpecFilePath := filepath.Join(pipelineFilesRootDir, pipelineDir, helloWorldPipelineFileName)

			// Create two pipelines
			name1 := testContext.Pipeline.PipelineGeneratedName + "-1"
			pipeline1 := uploadPipelineAndVerify(pipelineSpecFilePath, &name1, nil)
			name2 := testContext.Pipeline.PipelineGeneratedName + "-2"
			pipeline2 := uploadPipelineAndVerify(pipelineSpecFilePath, &name2, nil)

			// Get versions of pipeline1 (use explicit error checking)
			versionsList, _, _, err := pipelineClient.ListPipelineVersions(&pipeline_params.PipelineServiceListPipelineVersionsParams{
				PipelineID: pipeline1.PipelineID,
			})
			Expect(err).NotTo(HaveOccurred(), "Failed to list pipeline versions for pipeline1")
			Expect(versionsList).Should(HaveLen(1))

			// NOTE: The server does not validate that the version belongs to the specified pipeline.
			retrievedVersion, err := pipelineClient.GetPipelineVersion(&pipeline_params.PipelineServiceGetPipelineVersionParams{
				PipelineID:        pipeline2.PipelineID,
				PipelineVersionID: versionsList[0].PipelineVersionID,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(retrievedVersion.PipelineID).To(Equal(pipeline1.PipelineID))
			_ = pipeline2
		})
	})

	Context("List pipeline versions with tags '/pipelines/{pipeline_id}/versions' >", func() {
		It("List versions and verify tags are included in response", func() {
			pipelineDir := "valid"
			pipelineSpecFilePath := filepath.Join(pipelineFilesRootDir, pipelineDir, helloWorldPipelineFileName)

			createdPipeline := uploadPipelineAndVerify(pipelineSpecFilePath, &testContext.Pipeline.PipelineGeneratedName, nil)

			// Create a second version with tags
			versionTags := map[string]string{"stage": "beta", "owner": "alice"}
			versionName := testContext.Pipeline.PipelineGeneratedName + "-v2-list"
			uploadParams := uploadparams.NewUploadPipelineVersionParams()
			uploadParams.Pipelineid = &createdPipeline.PipelineID
			uploadParams.Name = &versionName
			uploadParams.SetTagsMap(versionTags)
			_, err := uploadPipelineVersion(pipelineSpecFilePath, uploadParams)
			Expect(err).NotTo(HaveOccurred())

			// List versions
			versions := utils.GetSortedPipelineVersionsByCreatedAt(pipelineClient, createdPipeline.PipelineID, nil)
			Expect(len(versions)).To(BeNumerically(">=", 2))

			// Find the tagged version and verify tags
			var foundTaggedVersion bool
			for _, v := range versions {
				if v.DisplayName == versionName {
					Expect(v.Tags).To(Equal(versionTags), "Listed version should include tags")
					foundTaggedVersion = true
					break
				}
			}
			Expect(foundTaggedVersion).To(BeTrue(), "Should find the tagged version in the list")
		})

		It("Version created without tags should have empty tags in list", func() {
			pipelineDir := "valid"
			pipelineSpecFilePath := filepath.Join(pipelineFilesRootDir, pipelineDir, helloWorldPipelineFileName)

			createdPipeline := uploadPipelineAndVerify(pipelineSpecFilePath, &testContext.Pipeline.PipelineGeneratedName, nil)

			versions := utils.GetSortedPipelineVersionsByCreatedAt(pipelineClient, createdPipeline.PipelineID, nil)
			Expect(versions).Should(HaveLen(1))
			Expect(versions[0].Tags).To(BeEmpty(), "Version created without tags should have empty tags")
		})
	})
})

var _ = Describe("Create Pipeline Version API Tests >", Label(constants.POSITIVE, constants.Pipeline, "PipelineVersionCreate", constants.APIServerTests, constants.FullRegression), func() {
	Context("Create version via CreatePipelineVersion RPC '/pipelines/{pipeline_id}/versions' >", func() {
		It("Create version with valid pipeline ID, pipeline version body including PipelineID", func() {
			pipelineDir := "valid"
			pipelineSpecFilePath := filepath.Join(pipelineFilesRootDir, pipelineDir, helloWorldPipelineFileName)
			createdPipeline := uploadPipelineAndVerify(pipelineSpecFilePath, &testContext.Pipeline.PipelineGeneratedName, nil)

			// CreatePipelineVersion requires an HTTP-accessible URL for the pipeline spec.
			// Use the upload API to create the version, then verify via GetPipelineVersion.
			// Direct CreatePipelineVersion with local file paths only works when the API server
			// can access the local filesystem (e.g., running outside a container).
			vName := testContext.Pipeline.PipelineGeneratedName + "-v-create-rpc"
			uploadParams := uploadparams.NewUploadPipelineVersionParams()
			uploadParams.Pipelineid = &createdPipeline.PipelineID
			uploadParams.Name = &vName
			newVersion, err := uploadPipelineVersion(pipelineSpecFilePath, uploadParams)
			Expect(err).NotTo(HaveOccurred())
			Expect(newVersion.PipelineVersionID).NotTo(BeEmpty())
			Expect(newVersion.PipelineID).To(Equal(createdPipeline.PipelineID))

			// Verify the version is retrievable
			retrievedVersion, err := pipelineClient.GetPipelineVersion(&pipeline_params.PipelineServiceGetPipelineVersionParams{
				PipelineID:        createdPipeline.PipelineID,
				PipelineVersionID: newVersion.PipelineVersionID,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(retrievedVersion.PipelineVersionID).To(Equal(newVersion.PipelineVersionID))
			Expect(retrievedVersion.PipelineID).To(Equal(createdPipeline.PipelineID))
		})

		It("Create version with tags", func() {
			pipelineDir := "valid"
			pipelineSpecFilePath := filepath.Join(pipelineFilesRootDir, pipelineDir, helloWorldPipelineFileName)
			createdPipeline := uploadPipelineAndVerify(pipelineSpecFilePath, &testContext.Pipeline.PipelineGeneratedName, nil)

			vName := testContext.Pipeline.PipelineGeneratedName + "-v-create-tags"
			tags := map[string]string{"env": "test", "team": "ml"}
			uploadParams := uploadparams.NewUploadPipelineVersionParams()
			uploadParams.Pipelineid = &createdPipeline.PipelineID
			uploadParams.Name = &vName
			uploadParams.SetTagsMap(tags)
			newVersion, err := uploadPipelineVersion(pipelineSpecFilePath, uploadParams)
			Expect(err).NotTo(HaveOccurred())

			// Verify tags are returned on GetPipelineVersion
			retrievedVersion, err := pipelineClient.GetPipelineVersion(&pipeline_params.PipelineServiceGetPipelineVersionParams{
				PipelineID:        createdPipeline.PipelineID,
				PipelineVersionID: newVersion.PipelineVersionID,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(retrievedVersion.Tags).To(Equal(tags), "Tags should match what was set during version creation")
		})

		It("Create version with description", func() {
			pipelineDir := "valid"
			pipelineSpecFilePath := filepath.Join(pipelineFilesRootDir, pipelineDir, helloWorldPipelineFileName)
			createdPipeline := uploadPipelineAndVerify(pipelineSpecFilePath, &testContext.Pipeline.PipelineGeneratedName, nil)

			vName := testContext.Pipeline.PipelineGeneratedName + "-v-create-desc"
			vDesc := "A versioned pipeline for testing"
			uploadParams := uploadparams.NewUploadPipelineVersionParams()
			uploadParams.Pipelineid = &createdPipeline.PipelineID
			uploadParams.Name = &vName
			uploadParams.Description = &vDesc
			newVersion, err := uploadPipelineVersion(pipelineSpecFilePath, uploadParams)
			Expect(err).NotTo(HaveOccurred())
			Expect(newVersion.PipelineVersionID).NotTo(BeEmpty())
		})

		It("Create multiple versions for the same pipeline", func() {
			pipelineDir := "valid"
			pipelineSpecFilePath := filepath.Join(pipelineFilesRootDir, pipelineDir, helloWorldPipelineFileName)
			createdPipeline := uploadPipelineAndVerify(pipelineSpecFilePath, &testContext.Pipeline.PipelineGeneratedName, nil)

			for i := 1; i <= 3; i++ {
				vName := fmt.Sprintf("%s-v%d", testContext.Pipeline.PipelineGeneratedName, i+1)
				uploadParams := uploadparams.NewUploadPipelineVersionParams()
				uploadParams.Pipelineid = &createdPipeline.PipelineID
				uploadParams.Name = &vName
				_, err := uploadPipelineVersion(pipelineSpecFilePath, uploadParams)
				Expect(err).NotTo(HaveOccurred())
			}

			// Verify all 4 versions exist (1 from upload + 3 created)
			versions := utils.GetSortedPipelineVersionsByCreatedAt(pipelineClient, createdPipeline.PipelineID, nil)
			Expect(len(versions)).To(Equal(4), "Should have 4 versions total")
		})
	})
})

var _ = Describe("Delete Pipeline API Tests >", Label(constants.POSITIVE, constants.Pipeline, "PipelineDelete", constants.APIServerTests, constants.FullRegression), func() {

	Context("Delete pipeline by ID '/pipelines/{pipeline_id}' >", func() {
		It("Delete pipeline by ID that does not have any versions", func() {
			pipelineName := testContext.Pipeline.PipelineGeneratedName
			createParams := &pipeline_params.PipelineServiceCreatePipelineParams{
				Pipeline: &pipeline_model.V2beta1Pipeline{
					Name: pipelineName,
				},
			}
			createdPipeline, err := pipelineClient.Create(createParams)
			Expect(err).NotTo(HaveOccurred())
			Expect(createdPipeline.PipelineID).NotTo(BeEmpty())

			// Delete the pipeline (no versions, so no need for cascade)
			err = pipelineClient.Delete(&pipeline_params.PipelineServiceDeletePipelineParams{
				PipelineID: createdPipeline.PipelineID,
			})
			Expect(err).NotTo(HaveOccurred())

			// Verify it's gone
			_, err = pipelineClient.Get(&pipeline_params.PipelineServiceGetPipelineParams{
				PipelineID: createdPipeline.PipelineID,
			})
			Expect(err).To(HaveOccurred())
		})

		It("Delete pipeline with tags and verify tags are cleaned up", func() {
			pipelineDir := "valid"
			pipelineSpecFilePath := filepath.Join(pipelineFilesRootDir, pipelineDir, helloWorldPipelineFileName)
			tags := map[string]string{"team": "ml-ops", "env": "staging"}
			testContext.Pipeline.UploadParams.SetTagsMap(tags)
			testContext.Pipeline.ExpectedPipeline.Tags = tags
			createdPipeline := uploadPipelineAndVerify(pipelineSpecFilePath, &testContext.Pipeline.PipelineGeneratedName, nil)

			// Delete all versions first, then delete the pipeline
			utils.DeleteAllPipelineVersions(pipelineClient, createdPipeline.PipelineID)
			err := pipelineClient.Delete(&pipeline_params.PipelineServiceDeletePipelineParams{
				PipelineID: createdPipeline.PipelineID,
			})
			Expect(err).NotTo(HaveOccurred())

			// Remove from cleanup list since we already deleted it
			remaining := make([]*upload_model.V2beta1Pipeline, 0)
			for _, p := range testContext.Pipeline.CreatedPipelines {
				if p.PipelineID != createdPipeline.PipelineID {
					remaining = append(remaining, p)
				}
			}
			testContext.Pipeline.CreatedPipelines = remaining

			// Verify pipeline is gone
			_, err = pipelineClient.Get(&pipeline_params.PipelineServiceGetPipelineParams{
				PipelineID: createdPipeline.PipelineID,
			})
			Expect(err).To(HaveOccurred())
		})
	})

	Context("Delete pipeline version by ID '/pipelines/{pipeline_id}/versions/{pipeline_version_id}' >", func() {
		It("Delete pipeline version by ID", func() {
			pipelineDir := "valid"
			pipelineSpecFilePath := filepath.Join(pipelineFilesRootDir, pipelineDir, helloWorldPipelineFileName)
			createdPipeline := uploadPipelineAndVerify(pipelineSpecFilePath, &testContext.Pipeline.PipelineGeneratedName, nil)

			// Get the pipeline version
			versions := utils.GetSortedPipelineVersionsByCreatedAt(pipelineClient, createdPipeline.PipelineID, nil)
			Expect(versions).Should(HaveLen(1))

			// Delete the version
			err := pipelineClient.DeletePipelineVersion(&pipeline_params.PipelineServiceDeletePipelineVersionParams{
				PipelineID:        createdPipeline.PipelineID,
				PipelineVersionID: versions[0].PipelineVersionID,
			})
			Expect(err).NotTo(HaveOccurred())

			// Verify version is gone
			versionsAfterDelete := utils.GetSortedPipelineVersionsByCreatedAt(pipelineClient, createdPipeline.PipelineID, nil)
			Expect(versionsAfterDelete).Should(HaveLen(0))
		})

		It("Deletes version even when pipeline ID in URL does not match the version's pipeline", func() {
			pipelineDir := "valid"
			pipelineSpecFilePath := filepath.Join(pipelineFilesRootDir, pipelineDir, helloWorldPipelineFileName)

			// Create two pipelines
			name1 := testContext.Pipeline.PipelineGeneratedName + "-1"
			pipeline1 := uploadPipelineAndVerify(pipelineSpecFilePath, &name1, nil)
			name2 := testContext.Pipeline.PipelineGeneratedName + "-2"
			pipeline2 := uploadPipelineAndVerify(pipelineSpecFilePath, &name2, nil)

			// Get versions of pipeline1 (use explicit error checking)
			versionsList, _, _, err := pipelineClient.ListPipelineVersions(&pipeline_params.PipelineServiceListPipelineVersionsParams{
				PipelineID: pipeline1.PipelineID,
			})
			Expect(err).NotTo(HaveOccurred(), "Failed to list pipeline versions for pipeline1")
			Expect(versionsList).Should(HaveLen(1))

			// NOTE: The server does not validate that the version belongs to the specified pipeline during deletion.
			err = pipelineClient.DeletePipelineVersion(&pipeline_params.PipelineServiceDeletePipelineVersionParams{
				PipelineID:        pipeline2.PipelineID,
				PipelineVersionID: versionsList[0].PipelineVersionID,
			})
			Expect(err).NotTo(HaveOccurred())

			// Verify version is actually deleted by trying to get it
			_, err = pipelineClient.GetPipelineVersion(&pipeline_params.PipelineServiceGetPipelineVersionParams{
				PipelineID:        pipeline1.PipelineID,
				PipelineVersionID: versionsList[0].PipelineVersionID,
			})
			Expect(err).To(HaveOccurred())
		})
	})
})

// ################## NEGATIVE TESTS ##################

var _ = Describe("Verify Pipeline Negative Tests >", Label("Negative", constants.Pipeline, constants.APIServerTests, constants.FullRegression), func() {
	Context("Create a pipeline with version using '/pipelines/create' >", func() {
		It("With a valid pipeline and pipeline version name but invalid pipeline spec file", func() {
			pipelineVersionName := testContext.Pipeline.PipelineGeneratedName + "-v1"
			createParams := &pipeline_params.PipelineServiceCreatePipelineAndVersionParams{
				Body: &pipeline_model.V2beta1CreatePipelineAndVersionRequest{
					Pipeline: &pipeline_model.V2beta1Pipeline{
						Name: testContext.Pipeline.PipelineGeneratedName,
					},
					PipelineVersion: &pipeline_model.V2beta1PipelineVersion{
						DisplayName: pipelineVersionName,
						PackageURL: &pipeline_model.V2beta1URL{
							PipelineURL: "/nonexistent/path/to/pipeline.yaml",
						},
					},
				},
			}
			_, err := pipelineClient.CreatePipelineAndVersion(createParams)
			Expect(err).To(HaveOccurred())
		})

		It("With a valid pipeline and pipeline version name but invalid pipeline spec url", func() {
			pipelineVersionName := testContext.Pipeline.PipelineGeneratedName + "-v1"
			createParams := &pipeline_params.PipelineServiceCreatePipelineAndVersionParams{
				Body: &pipeline_model.V2beta1CreatePipelineAndVersionRequest{
					Pipeline: &pipeline_model.V2beta1Pipeline{
						Name: testContext.Pipeline.PipelineGeneratedName,
					},
					PipelineVersion: &pipeline_model.V2beta1PipelineVersion{
						DisplayName: pipelineVersionName,
						PackageURL: &pipeline_model.V2beta1URL{
							PipelineURL: "https://invalid-url-that-does-not-exist.example.com/pipeline.yaml",
						},
					},
				},
			}
			_, err := pipelineClient.CreatePipelineAndVersion(createParams)
			Expect(err).To(HaveOccurred())
		})

		if *config.KubeflowMode {
			It("In a namespace you don't have access to", func() {
				Skip("Requires multi-user environment with restricted namespace")
			})
		}
	})

	Context("Create a pipeline using '/pipelines' >", func() {
		It("With 500 char name", func() {
			longName := utils.GetRandomString(500)
			createParams := &pipeline_params.PipelineServiceCreatePipelineParams{
				Pipeline: &pipeline_model.V2beta1Pipeline{
					Name: longName,
				},
			}
			_, err := pipelineClient.Create(createParams)
			Expect(err).To(HaveOccurred())
		})

		It("With CJK characters in the name", func() {
			cjkName := "\u4f60\u597d\u4e16\u754c-pipeline"
			createParams := &pipeline_params.PipelineServiceCreatePipelineParams{
				Pipeline: &pipeline_model.V2beta1Pipeline{
					Name: cjkName,
				},
			}
			createdPipeline, err := pipelineClient.Create(createParams)
			if err == nil {
				// If it succeeds, track for cleanup
				testContext.Pipeline.CreatedPipelines = append(testContext.Pipeline.CreatedPipelines, toUploadModel(createdPipeline))
			}
			// CJK characters may or may not be accepted depending on server validation
		})

		if *config.KubeflowMode {
			It("In a namespace you don't have access to", func() {
				Skip("Requires multi-user environment with restricted namespace")
			})
		}
	})

	Context("Get pipeline by ID >", func() {
		It("By non existing ID", func() {
			_, err := pipelineClient.Get(&pipeline_params.PipelineServiceGetPipelineParams{
				PipelineID: "00000000-0000-0000-0000-000000000000",
			})
			Expect(err).To(HaveOccurred())
		})

		It("By ID containing ASCII characters", func() {
			_, err := pipelineClient.Get(&pipeline_params.PipelineServiceGetPipelineParams{
				PipelineID: "invalid-id-!@#$%",
			})
			Expect(err).To(HaveOccurred())
		})

		if *config.KubeflowMode {
			It("In a namespace you don't have access to", func() {
				Skip("Requires multi-user environment with restricted namespace")
			})
		}
	})

	Context("Get pipeline version by ID >", func() {
		It("By non existing ID", func() {
			// First create a pipeline to have a valid pipeline ID
			pipelineDir := "valid"
			pipelineSpecFilePath := filepath.Join(pipelineFilesRootDir, pipelineDir, helloWorldPipelineFileName)
			createdPipeline := uploadPipelineAndVerify(pipelineSpecFilePath, &testContext.Pipeline.PipelineGeneratedName, nil)

			_, err := pipelineClient.GetPipelineVersion(&pipeline_params.PipelineServiceGetPipelineVersionParams{
				PipelineID:        createdPipeline.PipelineID,
				PipelineVersionID: "00000000-0000-0000-0000-000000000000",
			})
			Expect(err).To(HaveOccurred())
		})

		It("By ID containing ASCII characters", func() {
			_, err := pipelineClient.GetPipelineVersion(&pipeline_params.PipelineServiceGetPipelineVersionParams{
				PipelineID:        "invalid-id",
				PipelineVersionID: "invalid-version-id-!@#$%",
			})
			Expect(err).To(HaveOccurred())
		})

		It("By valid version ID but with the pipeline ID that does not contain this version should ideally fail", func() {
			Skip("Server does not currently validate pipeline ownership for GetPipelineVersion")
		})

		if *config.KubeflowMode {
			It("In a namespace you don't have access to", func() {
				Skip("Requires multi-user environment with restricted namespace")
			})
		}
	})

	Context("Delete by ID '/pipelines/{pipeline_id}' >", func() {
		It("Delete by ID that does have pipeline version(s)", func() {
			pipelineDir := "valid"
			pipelineSpecFilePath := filepath.Join(pipelineFilesRootDir, pipelineDir, helloWorldPipelineFileName)
			createdPipeline := uploadPipelineAndVerify(pipelineSpecFilePath, &testContext.Pipeline.PipelineGeneratedName, nil)

			// Try to delete pipeline that has versions (should fail without cascade)
			err := pipelineClient.Delete(&pipeline_params.PipelineServiceDeletePipelineParams{
				PipelineID: createdPipeline.PipelineID,
			})
			Expect(err).To(HaveOccurred())
		})

		It("Delete by non existing ID", func() {
			err := pipelineClient.Delete(&pipeline_params.PipelineServiceDeletePipelineParams{
				PipelineID: "00000000-0000-0000-0000-000000000000",
			})
			Expect(err).To(HaveOccurred())
		})

		It("Delete by ID containing ASCII characters", func() {
			err := pipelineClient.Delete(&pipeline_params.PipelineServiceDeletePipelineParams{
				PipelineID: "invalid-id-!@#$%",
			})
			Expect(err).To(HaveOccurred())
		})

		if *config.KubeflowMode {
			It("In a namespace you don't have access to", func() {
				Skip("Requires multi-user environment with restricted namespace")
			})
		}
	})

	Context("Delete pipeline version by ID '/pipelines/{pipeline_id}/versions/{pipeline_version_id}' >", func() {
		It("Delete pipeline version with an invalid ID", func() {
			pipelineDir := "valid"
			pipelineSpecFilePath := filepath.Join(pipelineFilesRootDir, pipelineDir, helloWorldPipelineFileName)
			createdPipeline := uploadPipelineAndVerify(pipelineSpecFilePath, &testContext.Pipeline.PipelineGeneratedName, nil)

			err := pipelineClient.DeletePipelineVersion(&pipeline_params.PipelineServiceDeletePipelineVersionParams{
				PipelineID:        createdPipeline.PipelineID,
				PipelineVersionID: "00000000-0000-0000-0000-000000000000",
			})
			Expect(err).To(HaveOccurred())
		})

		It("Delete pipeline version by ID with wrong pipeline ID should ideally fail", func() {
			Skip("Server does not currently validate pipeline ownership for DeletePipelineVersion")
		})

		It("Delete by ID containing ASCII characters", func() {
			err := pipelineClient.DeletePipelineVersion(&pipeline_params.PipelineServiceDeletePipelineVersionParams{
				PipelineID:        "invalid-id",
				PipelineVersionID: "invalid-version-!@#$%",
			})
			Expect(err).To(HaveOccurred())
		})

		if *config.KubeflowMode {
			It("In a namespace you don't have access to", func() {
				Skip("Requires multi-user environment with restricted namespace")
			})
		}
	})

	Context("List pipelines >", func() {
		It("By partial name", func() {
			filter := `{"predicates":[{"key":"name","operation":"EQUALS","string_value":"nonexistent-pipeline-name-xyz"}]}`
			params := newListPipelinesParams()
			params.Filter = &filter
			pipelines, totalSize, _, err := pipelineClient.List(params)
			Expect(err).NotTo(HaveOccurred())
			Expect(totalSize).To(Equal(0))
			Expect(pipelines).To(BeEmpty())
		})

		It("By invalid name", func() {
			filter := `{"predicates":[{"key":"name","operation":"EQUALS","string_value":""}]}`
			params := newListPipelinesParams()
			params.Filter = &filter
			_, _, _, err := pipelineClient.List(params)
			Expect(err).NotTo(HaveOccurred())
		})

		It("By invalid ID", func() {
			filter := `{"predicates":[{"key":"pipeline_id","operation":"EQUALS","string_value":"00000000-0000-0000-0000-000000000000"}]}`
			params := newListPipelinesParams()
			params.Filter = &filter
			pipelines, totalSize, _, err := pipelineClient.List(params)
			Expect(err).NotTo(HaveOccurred())
			Expect(totalSize).To(Equal(0))
			Expect(pipelines).To(BeEmpty())
		})

		It("By invalid ID containing ASCII characters", func() {
			filter := `{"predicates":[{"key":"pipeline_id","operation":"EQUALS","string_value":"invalid-!@#$^&*"}]}`
			params := newListPipelinesParams()
			params.Filter = &filter
			pipelines, totalSize, _, err := pipelineClient.List(params)
			Expect(err).NotTo(HaveOccurred())
			Expect(totalSize).To(Equal(0))
			Expect(pipelines).To(BeEmpty())
		})

		if *config.KubeflowMode {
			It("In a namespace you don't have access to", func() {
				Skip("Requires multi-user environment with restricted namespace")
			})
		}
	})

	Context("Negative tag tests >", func() {
		It("Filter by tag with non-EQUALS operation should fail", func() {
			filter := `{"predicates":[{"key":"tags.team","operation":"NOT_EQUALS","string_value":"ml-ops"}]}`
			params := newListPipelinesParams()
			params.Filter = &filter
			_, _, _, err := pipelineClient.List(params)
			Expect(err).To(HaveOccurred(), "Non-EQUALS operation on tag filter should fail")
		})

		It("Filter by tag with wrong value returns empty results", func() {
			pipelineDir := "valid"
			pipelineSpecFilePath := filepath.Join(pipelineFilesRootDir, pipelineDir, helloWorldPipelineFileName)
			tags := map[string]string{"team": "ml-ops"}
			testContext.Pipeline.UploadParams.SetTagsMap(tags)
			testContext.Pipeline.ExpectedPipeline.Tags = tags
			uploadPipelineAndVerify(pipelineSpecFilePath, &testContext.Pipeline.PipelineGeneratedName, nil)

			filter := `{"predicates":[{"key":"tags.team","operation":"EQUALS","string_value":"wrong-value"}]}`
			params := newListPipelinesParams()
			params.Filter = &filter
			pipelines, totalSize, _, err := pipelineClient.List(params)
			Expect(err).NotTo(HaveOccurred())
			Expect(totalSize).To(Equal(0))
			Expect(pipelines).To(BeEmpty())
		})

		It("Get tags of deleted pipeline should fail", func() {
			pipelineName := testContext.Pipeline.PipelineGeneratedName
			tags := map[string]string{"team": "ml-ops"}
			createParams := &pipeline_params.PipelineServiceCreatePipelineParams{
				Pipeline: &pipeline_model.V2beta1Pipeline{
					Name: pipelineName,
					Tags: tags,
				},
			}
			createdPipeline, err := pipelineClient.Create(createParams)
			Expect(err).NotTo(HaveOccurred())

			// Delete the pipeline
			err = pipelineClient.Delete(&pipeline_params.PipelineServiceDeletePipelineParams{
				PipelineID: createdPipeline.PipelineID,
			})
			Expect(err).NotTo(HaveOccurred())

			// Try to get the deleted pipeline
			_, err = pipelineClient.Get(&pipeline_params.PipelineServiceGetPipelineParams{
				PipelineID: createdPipeline.PipelineID,
			})
			Expect(err).To(HaveOccurred(), "Getting a deleted pipeline should fail")
		})
	})
})

// ################## PIPELINE UPDATE (WITH TAGS) TESTS ##################

var _ = Describe("Update Pipeline - Positive Tests >", Label(constants.POSITIVE, constants.Pipeline, "PipelineTags", constants.APIServerTests, constants.FullRegression), func() {

	Context("Update pipeline via PUT '/pipelines/{pipeline_id}' >", func() {
		It("Add tags to a pipeline that has no tags", func() {
			pipelineName := testContext.Pipeline.PipelineGeneratedName
			createParams := &pipeline_params.PipelineServiceCreatePipelineParams{
				Pipeline: &pipeline_model.V2beta1Pipeline{
					Name: pipelineName,
				},
			}
			createdPipeline, err := pipelineClient.Create(createParams)
			Expect(err).NotTo(HaveOccurred())
			testContext.Pipeline.CreatedPipelines = append(testContext.Pipeline.CreatedPipelines, toUploadModel(createdPipeline))

			// Add tags via UpdatePipeline
			newTags := map[string]string{"team": "ml-ops", "env": "prod"}
			updatedPipeline, err := pipelineClient.UpdatePipeline(&pipeline_params.PipelineServiceUpdatePipelineParams{
				PipelinePipelineID: createdPipeline.PipelineID,
				Pipeline: pipeline_params.PipelineServiceUpdatePipelineBody{
					Tags: newTags,
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedPipeline.Tags).To(Equal(newTags))

			// Verify via GetPipeline
			retrievedPipeline, err := pipelineClient.Get(&pipeline_params.PipelineServiceGetPipelineParams{
				PipelineID: createdPipeline.PipelineID,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(retrievedPipeline.Tags).To(Equal(newTags))
		})

		It("Replace existing tags with new tags", func() {
			pipelineName := testContext.Pipeline.PipelineGeneratedName
			initialTags := map[string]string{"team": "ml-ops", "env": "dev"}
			createParams := &pipeline_params.PipelineServiceCreatePipelineParams{
				Pipeline: &pipeline_model.V2beta1Pipeline{
					Name: pipelineName,
					Tags: initialTags,
				},
			}
			createdPipeline, err := pipelineClient.Create(createParams)
			Expect(err).NotTo(HaveOccurred())
			testContext.Pipeline.CreatedPipelines = append(testContext.Pipeline.CreatedPipelines, toUploadModel(createdPipeline))

			// Replace tags via UpdatePipeline
			newTags := map[string]string{"project": "kfp", "owner": "test"}
			updatedPipeline, err := pipelineClient.UpdatePipeline(&pipeline_params.PipelineServiceUpdatePipelineParams{
				PipelinePipelineID: createdPipeline.PipelineID,
				Pipeline: pipeline_params.PipelineServiceUpdatePipelineBody{
					Tags: newTags,
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedPipeline.Tags).To(Equal(newTags))
			Expect(updatedPipeline.Tags).NotTo(HaveKey("team"), "Old tags should be replaced")
			Expect(updatedPipeline.Tags).NotTo(HaveKey("env"), "Old tags should be replaced")
		})

		It("Remove all tags by passing empty map", func() {
			pipelineName := testContext.Pipeline.PipelineGeneratedName
			initialTags := map[string]string{"team": "ml-ops"}
			createParams := &pipeline_params.PipelineServiceCreatePipelineParams{
				Pipeline: &pipeline_model.V2beta1Pipeline{
					Name: pipelineName,
					Tags: initialTags,
				},
			}
			createdPipeline, err := pipelineClient.Create(createParams)
			Expect(err).NotTo(HaveOccurred())
			testContext.Pipeline.CreatedPipelines = append(testContext.Pipeline.CreatedPipelines, toUploadModel(createdPipeline))

			// Remove all tags via UpdatePipeline
			updatedPipeline, err := pipelineClient.UpdatePipeline(&pipeline_params.PipelineServiceUpdatePipelineParams{
				PipelinePipelineID: createdPipeline.PipelineID,
				Pipeline: pipeline_params.PipelineServiceUpdatePipelineBody{
					Tags: map[string]string{},
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedPipeline.Tags).To(BeEmpty(), "Tags should be empty after removal")

			// Verify via GetPipeline
			retrievedPipeline, err := pipelineClient.Get(&pipeline_params.PipelineServiceGetPipelineParams{
				PipelineID: createdPipeline.PipelineID,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(retrievedPipeline.Tags).To(BeEmpty())
		})

		It("Update tags and verify via list filter", func() {
			pipelineName := testContext.Pipeline.PipelineGeneratedName
			createParams := &pipeline_params.PipelineServiceCreatePipelineParams{
				Pipeline: &pipeline_model.V2beta1Pipeline{
					Name:      pipelineName,
					Namespace: utils.GetNamespace(),
				},
			}
			createdPipeline, err := pipelineClient.Create(createParams)
			Expect(err).NotTo(HaveOccurred())
			testContext.Pipeline.CreatedPipelines = append(testContext.Pipeline.CreatedPipelines, toUploadModel(createdPipeline))

			// Add tags via UpdatePipeline
			newTags := map[string]string{"team": "ut-" + randomName[:17]}
			_, err = pipelineClient.UpdatePipeline(&pipeline_params.PipelineServiceUpdatePipelineParams{
				PipelinePipelineID: createdPipeline.PipelineID,
				Pipeline: pipeline_params.PipelineServiceUpdatePipelineBody{
					Tags: newTags,
				},
			})
			Expect(err).NotTo(HaveOccurred())

			// Verify via list filter
			filter := fmt.Sprintf(`{"predicates":[{"key":"tags.team","operation":"EQUALS","string_value":"%s"}]}`, newTags["team"])
			listParams := newListPipelinesParams()
			listParams.Filter = &filter
			pipelines, totalSize, _, err := pipelineClient.List(listParams)
			Expect(err).NotTo(HaveOccurred())
			Expect(totalSize).To(BeNumerically(">=", 1))
			found := false
			for _, p := range pipelines {
				if p.PipelineID == createdPipeline.PipelineID {
					found = true
					break
				}
			}
			Expect(found).To(BeTrue(), "Pipeline should be found when filtering by updated tags")
		})

		It("Update tags with maximum allowed key and value lengths (20 chars)", func() {
			pipelineName := testContext.Pipeline.PipelineGeneratedName
			createParams := &pipeline_params.PipelineServiceCreatePipelineParams{
				Pipeline: &pipeline_model.V2beta1Pipeline{
					Name: pipelineName,
				},
			}
			createdPipeline, err := pipelineClient.Create(createParams)
			Expect(err).NotTo(HaveOccurred())
			testContext.Pipeline.CreatedPipelines = append(testContext.Pipeline.CreatedPipelines, toUploadModel(createdPipeline))

			// Add tags with exactly 20 char key and value
			maxKey := utils.GetRandomString(20)
			maxValue := utils.GetRandomString(20)
			tags := map[string]string{maxKey: maxValue}
			updatedPipeline, err := pipelineClient.UpdatePipeline(&pipeline_params.PipelineServiceUpdatePipelineParams{
				PipelinePipelineID: createdPipeline.PipelineID,
				Pipeline: pipeline_params.PipelineServiceUpdatePipelineBody{
					Tags: tags,
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedPipeline.Tags).To(Equal(tags))
		})

		It("Update pipeline display_name along with tags", func() {
			pipelineName := testContext.Pipeline.PipelineGeneratedName
			createParams := &pipeline_params.PipelineServiceCreatePipelineParams{
				Pipeline: &pipeline_model.V2beta1Pipeline{
					Name: pipelineName,
				},
			}
			createdPipeline, err := pipelineClient.Create(createParams)
			Expect(err).NotTo(HaveOccurred())
			testContext.Pipeline.CreatedPipelines = append(testContext.Pipeline.CreatedPipelines, toUploadModel(createdPipeline))

			// Update display_name and tags together
			newDisplayName := "Updated Display Name"
			newTags := map[string]string{"team": "ml-ops"}
			updatedPipeline, err := pipelineClient.UpdatePipeline(&pipeline_params.PipelineServiceUpdatePipelineParams{
				PipelinePipelineID: createdPipeline.PipelineID,
				Pipeline: pipeline_params.PipelineServiceUpdatePipelineBody{
					DisplayName: newDisplayName,
					Tags:        newTags,
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedPipeline.DisplayName).To(Equal(newDisplayName))
			Expect(updatedPipeline.Tags).To(Equal(newTags))
		})
	})
})

var _ = Describe("Update Pipeline - Negative Tests >", Label(constants.NEGATIVE, constants.Pipeline, "PipelineTags", constants.APIServerTests, constants.FullRegression), func() {

	Context("Update pipeline with invalid parameters >", func() {
		It("Update tags for non-existing pipeline ID", func() {
			_, err := pipelineClient.UpdatePipeline(&pipeline_params.PipelineServiceUpdatePipelineParams{
				PipelinePipelineID: "00000000-0000-0000-0000-000000000000",
				Pipeline: pipeline_params.PipelineServiceUpdatePipelineBody{
					Tags: map[string]string{"team": "ml-ops"},
				},
			})
			Expect(err).To(HaveOccurred())
		})

		It("Update tags with key exceeding 20 characters", func() {
			pipelineName := testContext.Pipeline.PipelineGeneratedName
			createParams := &pipeline_params.PipelineServiceCreatePipelineParams{
				Pipeline: &pipeline_model.V2beta1Pipeline{
					Name: pipelineName,
				},
			}
			createdPipeline, err := pipelineClient.Create(createParams)
			Expect(err).NotTo(HaveOccurred())
			testContext.Pipeline.CreatedPipelines = append(testContext.Pipeline.CreatedPipelines, toUploadModel(createdPipeline))

			longKey := utils.GetRandomString(21)
			_, err = pipelineClient.UpdatePipeline(&pipeline_params.PipelineServiceUpdatePipelineParams{
				PipelinePipelineID: createdPipeline.PipelineID,
				Pipeline: pipeline_params.PipelineServiceUpdatePipelineBody{
					Tags: map[string]string{longKey: "value"},
				},
			})
			Expect(err).To(HaveOccurred(), "Tag key exceeding 20 characters should be rejected")
		})

		It("Update tags with value exceeding 20 characters", func() {
			pipelineName := testContext.Pipeline.PipelineGeneratedName
			createParams := &pipeline_params.PipelineServiceCreatePipelineParams{
				Pipeline: &pipeline_model.V2beta1Pipeline{
					Name: pipelineName,
				},
			}
			createdPipeline, err := pipelineClient.Create(createParams)
			Expect(err).NotTo(HaveOccurred())
			testContext.Pipeline.CreatedPipelines = append(testContext.Pipeline.CreatedPipelines, toUploadModel(createdPipeline))

			longValue := utils.GetRandomString(21)
			_, err = pipelineClient.UpdatePipeline(&pipeline_params.PipelineServiceUpdatePipelineParams{
				PipelinePipelineID: createdPipeline.PipelineID,
				Pipeline: pipeline_params.PipelineServiceUpdatePipelineBody{
					Tags: map[string]string{"team": longValue},
				},
			})
			Expect(err).To(HaveOccurred(), "Tag value exceeding 20 characters should be rejected")
		})

		It("Update tags with empty key should fail", func() {
			pipelineName := testContext.Pipeline.PipelineGeneratedName
			createParams := &pipeline_params.PipelineServiceCreatePipelineParams{
				Pipeline: &pipeline_model.V2beta1Pipeline{
					Name: pipelineName,
				},
			}
			createdPipeline, err := pipelineClient.Create(createParams)
			Expect(err).NotTo(HaveOccurred())
			testContext.Pipeline.CreatedPipelines = append(testContext.Pipeline.CreatedPipelines, toUploadModel(createdPipeline))

			_, err = pipelineClient.UpdatePipeline(&pipeline_params.PipelineServiceUpdatePipelineParams{
				PipelinePipelineID: createdPipeline.PipelineID,
				Pipeline: pipeline_params.PipelineServiceUpdatePipelineBody{
					Tags: map[string]string{"": "value"},
				},
			})
			Expect(err).To(HaveOccurred(), "Empty tag key should be rejected")
		})

		It("Update tags with key containing dot should fail", func() {
			pipelineName := testContext.Pipeline.PipelineGeneratedName
			createParams := &pipeline_params.PipelineServiceCreatePipelineParams{
				Pipeline: &pipeline_model.V2beta1Pipeline{
					Name: pipelineName,
				},
			}
			createdPipeline, err := pipelineClient.Create(createParams)
			Expect(err).NotTo(HaveOccurred())
			testContext.Pipeline.CreatedPipelines = append(testContext.Pipeline.CreatedPipelines, toUploadModel(createdPipeline))

			_, err = pipelineClient.UpdatePipeline(&pipeline_params.PipelineServiceUpdatePipelineParams{
				PipelinePipelineID: createdPipeline.PipelineID,
				Pipeline: pipeline_params.PipelineServiceUpdatePipelineBody{
					Tags: map[string]string{"team.name": "ml-ops"},
				},
			})
			Expect(err).To(HaveOccurred(), "Tag key containing dot should be rejected")
		})

		It("Update tags exceeding maximum count of 10 should fail", func() {
			pipelineName := testContext.Pipeline.PipelineGeneratedName
			createParams := &pipeline_params.PipelineServiceCreatePipelineParams{
				Pipeline: &pipeline_model.V2beta1Pipeline{
					Name: pipelineName,
				},
			}
			createdPipeline, err := pipelineClient.Create(createParams)
			Expect(err).NotTo(HaveOccurred())
			testContext.Pipeline.CreatedPipelines = append(testContext.Pipeline.CreatedPipelines, toUploadModel(createdPipeline))

			tooManyTags := make(map[string]string)
			for i := 0; i < 11; i++ {
				tooManyTags[fmt.Sprintf("key%d", i)] = fmt.Sprintf("val%d", i)
			}
			_, err = pipelineClient.UpdatePipeline(&pipeline_params.PipelineServiceUpdatePipelineParams{
				PipelinePipelineID: createdPipeline.PipelineID,
				Pipeline: pipeline_params.PipelineServiceUpdatePipelineBody{
					Tags: tooManyTags,
				},
			})
			Expect(err).To(HaveOccurred(), "More than 10 tags should be rejected")
		})

		It("Update pipeline on a deleted pipeline should fail", func() {
			pipelineName := testContext.Pipeline.PipelineGeneratedName
			createParams := &pipeline_params.PipelineServiceCreatePipelineParams{
				Pipeline: &pipeline_model.V2beta1Pipeline{
					Name: pipelineName,
				},
			}
			createdPipeline, err := pipelineClient.Create(createParams)
			Expect(err).NotTo(HaveOccurred())

			// Delete the pipeline
			err = pipelineClient.Delete(&pipeline_params.PipelineServiceDeletePipelineParams{
				PipelineID: createdPipeline.PipelineID,
			})
			Expect(err).NotTo(HaveOccurred())

			// Try to update pipeline after deletion
			_, err = pipelineClient.UpdatePipeline(&pipeline_params.PipelineServiceUpdatePipelineParams{
				PipelinePipelineID: createdPipeline.PipelineID,
				Pipeline: pipeline_params.PipelineServiceUpdatePipelineBody{
					Tags: map[string]string{"team": "ml-ops"},
				},
			})
			Expect(err).To(HaveOccurred(), "Updating a deleted pipeline should fail")
		})

		It("Get pipeline by ID after deletion should not return tags", func() {
			pipelineName := testContext.Pipeline.PipelineGeneratedName
			tags := map[string]string{"team": "ml-ops", "env": "prod"}
			createParams := &pipeline_params.PipelineServiceCreatePipelineParams{
				Pipeline: &pipeline_model.V2beta1Pipeline{
					Name: pipelineName,
					Tags: tags,
				},
			}
			createdPipeline, err := pipelineClient.Create(createParams)
			Expect(err).NotTo(HaveOccurred())

			// Verify tags exist before deletion
			retrievedPipeline, err := pipelineClient.Get(&pipeline_params.PipelineServiceGetPipelineParams{
				PipelineID: createdPipeline.PipelineID,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(retrievedPipeline.Tags).To(Equal(tags))

			// Delete the pipeline
			err = pipelineClient.Delete(&pipeline_params.PipelineServiceDeletePipelineParams{
				PipelineID: createdPipeline.PipelineID,
			})
			Expect(err).NotTo(HaveOccurred())

			// Verify get fails after deletion
			_, err = pipelineClient.Get(&pipeline_params.PipelineServiceGetPipelineParams{
				PipelineID: createdPipeline.PipelineID,
			})
			Expect(err).To(HaveOccurred(), "Getting a deleted pipeline should fail")
		})

		It("Deleted pipeline tags should not appear in list filter results", func() {
			pipelineDir := "valid"
			pipelineSpecFilePath := filepath.Join(pipelineFilesRootDir, pipelineDir, helloWorldPipelineFileName)
			uniqueTagValue := "dt-" + randomName[:17]
			tags := map[string]string{"team": uniqueTagValue}
			testContext.Pipeline.UploadParams.SetTagsMap(tags)
			testContext.Pipeline.ExpectedPipeline.Tags = tags
			createdPipeline := uploadPipelineAndVerify(pipelineSpecFilePath, &testContext.Pipeline.PipelineGeneratedName, nil)

			// Verify pipeline appears in filter before deletion
			filter := fmt.Sprintf(`{"predicates":[{"key":"tags.team","operation":"EQUALS","string_value":"%s"}]}`, uniqueTagValue)
			listParams := newListPipelinesParams()
			listParams.Filter = &filter
			pipelines, totalSize, _, err := pipelineClient.List(listParams)
			Expect(err).NotTo(HaveOccurred())
			Expect(totalSize).To(BeNumerically(">=", 1))
			found := false
			for _, p := range pipelines {
				if p.PipelineID == createdPipeline.PipelineID {
					found = true
					break
				}
			}
			Expect(found).To(BeTrue(), "Pipeline should be found before deletion")

			// Delete the pipeline (clean up versions first)
			utils.DeleteAllPipelineVersions(pipelineClient, createdPipeline.PipelineID)
			err = pipelineClient.Delete(&pipeline_params.PipelineServiceDeletePipelineParams{
				PipelineID: createdPipeline.PipelineID,
			})
			Expect(err).NotTo(HaveOccurred())

			// Remove from cleanup list
			remaining := make([]*upload_model.V2beta1Pipeline, 0)
			for _, p := range testContext.Pipeline.CreatedPipelines {
				if p.PipelineID != createdPipeline.PipelineID {
					remaining = append(remaining, p)
				}
			}
			testContext.Pipeline.CreatedPipelines = remaining

			// Verify pipeline no longer appears in filter after deletion
			pipelines, _, _, err = pipelineClient.List(listParams)
			Expect(err).NotTo(HaveOccurred())
			for _, p := range pipelines {
				Expect(p.PipelineID).NotTo(Equal(createdPipeline.PipelineID), "Deleted pipeline should not appear in filtered results")
			}
		})
	})

	Context("Create pipeline with invalid tags >", func() {
		It("With tag key exceeding 20 characters", func() {
			pipelineName := testContext.Pipeline.PipelineGeneratedName
			longKey := utils.GetRandomString(21)
			createParams := &pipeline_params.PipelineServiceCreatePipelineParams{
				Pipeline: &pipeline_model.V2beta1Pipeline{
					Name: pipelineName,
					Tags: map[string]string{longKey: "value"},
				},
			}
			_, err := pipelineClient.Create(createParams)
			Expect(err).To(HaveOccurred(), "Tag key exceeding 20 characters should be rejected")
		})

		It("With tag value exceeding 20 characters", func() {
			pipelineName := testContext.Pipeline.PipelineGeneratedName
			longValue := utils.GetRandomString(21)
			createParams := &pipeline_params.PipelineServiceCreatePipelineParams{
				Pipeline: &pipeline_model.V2beta1Pipeline{
					Name: pipelineName,
					Tags: map[string]string{"team": longValue},
				},
			}
			_, err := pipelineClient.Create(createParams)
			Expect(err).To(HaveOccurred(), "Tag value exceeding 20 characters should be rejected")
		})

		It("With empty tag key", func() {
			pipelineName := testContext.Pipeline.PipelineGeneratedName
			createParams := &pipeline_params.PipelineServiceCreatePipelineParams{
				Pipeline: &pipeline_model.V2beta1Pipeline{
					Name: pipelineName,
					Tags: map[string]string{"": "value"},
				},
			}
			_, err := pipelineClient.Create(createParams)
			Expect(err).To(HaveOccurred(), "Empty tag key should be rejected")
		})

		It("With tag key containing dot", func() {
			pipelineName := testContext.Pipeline.PipelineGeneratedName
			createParams := &pipeline_params.PipelineServiceCreatePipelineParams{
				Pipeline: &pipeline_model.V2beta1Pipeline{
					Name: pipelineName,
					Tags: map[string]string{"team.name": "ml-ops"},
				},
			}
			_, err := pipelineClient.Create(createParams)
			Expect(err).To(HaveOccurred(), "Tag key containing dot should be rejected")
		})

		It("With more than 10 tags", func() {
			pipelineName := testContext.Pipeline.PipelineGeneratedName
			tooManyTags := make(map[string]string)
			for i := 0; i < 11; i++ {
				tooManyTags[fmt.Sprintf("key%d", i)] = fmt.Sprintf("val%d", i)
			}
			createParams := &pipeline_params.PipelineServiceCreatePipelineParams{
				Pipeline: &pipeline_model.V2beta1Pipeline{
					Name: pipelineName,
					Tags: tooManyTags,
				},
			}
			_, err := pipelineClient.Create(createParams)
			Expect(err).To(HaveOccurred(), "More than 10 tags should be rejected")
		})
	})

	Context("Create pipeline version with invalid tags >", func() {
		It("With tag key exceeding 20 characters on pipeline version", func() {
			pipelineDir := "valid"
			pipelineSpecFilePath := filepath.Join(pipelineFilesRootDir, pipelineDir, helloWorldPipelineFileName)
			createdPipeline := uploadPipelineAndVerify(pipelineSpecFilePath, &testContext.Pipeline.PipelineGeneratedName, nil)

			longKey := utils.GetRandomString(21)
			vName := testContext.Pipeline.PipelineGeneratedName + "-v-bad-key"
			uploadParams := uploadparams.NewUploadPipelineVersionParams()
			uploadParams.Pipelineid = &createdPipeline.PipelineID
			uploadParams.Name = &vName
			uploadParams.SetTagsMap(map[string]string{longKey: "value"})
			_, err := uploadPipelineVersion(pipelineSpecFilePath, uploadParams)
			Expect(err).To(HaveOccurred(), "Tag key exceeding 20 characters should be rejected")
		})

		It("With tag value exceeding 20 characters on pipeline version", func() {
			pipelineDir := "valid"
			pipelineSpecFilePath := filepath.Join(pipelineFilesRootDir, pipelineDir, helloWorldPipelineFileName)
			createdPipeline := uploadPipelineAndVerify(pipelineSpecFilePath, &testContext.Pipeline.PipelineGeneratedName, nil)

			longValue := utils.GetRandomString(21)
			vName := testContext.Pipeline.PipelineGeneratedName + "-v-bad-val"
			uploadParams := uploadparams.NewUploadPipelineVersionParams()
			uploadParams.Pipelineid = &createdPipeline.PipelineID
			uploadParams.Name = &vName
			uploadParams.SetTagsMap(map[string]string{"team": longValue})
			_, err := uploadPipelineVersion(pipelineSpecFilePath, uploadParams)
			Expect(err).To(HaveOccurred(), "Tag value exceeding 20 characters should be rejected")
		})

		It("With empty tag key on pipeline version", func() {
			pipelineDir := "valid"
			pipelineSpecFilePath := filepath.Join(pipelineFilesRootDir, pipelineDir, helloWorldPipelineFileName)
			createdPipeline := uploadPipelineAndVerify(pipelineSpecFilePath, &testContext.Pipeline.PipelineGeneratedName, nil)

			vName := testContext.Pipeline.PipelineGeneratedName + "-v-empty-key"
			uploadParams := uploadparams.NewUploadPipelineVersionParams()
			uploadParams.Pipelineid = &createdPipeline.PipelineID
			uploadParams.Name = &vName
			uploadParams.SetTagsMap(map[string]string{"": "value"})
			_, err := uploadPipelineVersion(pipelineSpecFilePath, uploadParams)
			Expect(err).To(HaveOccurred(), "Empty tag key should be rejected on pipeline version")
		})

		It("With tag key containing dot on pipeline version", func() {
			pipelineDir := "valid"
			pipelineSpecFilePath := filepath.Join(pipelineFilesRootDir, pipelineDir, helloWorldPipelineFileName)
			createdPipeline := uploadPipelineAndVerify(pipelineSpecFilePath, &testContext.Pipeline.PipelineGeneratedName, nil)

			vName := testContext.Pipeline.PipelineGeneratedName + "-v-dot-key"
			uploadParams := uploadparams.NewUploadPipelineVersionParams()
			uploadParams.Pipelineid = &createdPipeline.PipelineID
			uploadParams.Name = &vName
			uploadParams.SetTagsMap(map[string]string{"team.name": "ml-ops"})
			_, err := uploadPipelineVersion(pipelineSpecFilePath, uploadParams)
			Expect(err).To(HaveOccurred(), "Tag key containing dot should be rejected on pipeline version")
		})

		It("With more than 10 tags on pipeline version", func() {
			pipelineDir := "valid"
			pipelineSpecFilePath := filepath.Join(pipelineFilesRootDir, pipelineDir, helloWorldPipelineFileName)
			createdPipeline := uploadPipelineAndVerify(pipelineSpecFilePath, &testContext.Pipeline.PipelineGeneratedName, nil)

			tooManyTags := make(map[string]string)
			for i := 0; i < 11; i++ {
				tooManyTags[fmt.Sprintf("key%d", i)] = fmt.Sprintf("val%d", i)
			}
			vName := testContext.Pipeline.PipelineGeneratedName + "-v-many-tags"
			uploadParams := uploadparams.NewUploadPipelineVersionParams()
			uploadParams.Pipelineid = &createdPipeline.PipelineID
			uploadParams.Name = &vName
			uploadParams.SetTagsMap(tooManyTags)
			_, err := uploadPipelineVersion(pipelineSpecFilePath, uploadParams)
			Expect(err).To(HaveOccurred(), "More than 10 tags should be rejected on pipeline version")
		})
	})
})

var _ = Describe("Update Pipeline Version >", Label(constants.POSITIVE, constants.Pipeline, "PipelineVersionTags", constants.APIServerTests, constants.FullRegression), func() {

	Context("Update pipeline version tags via UpdatePipelineVersion >", func() {
		It("Add tags to an existing pipeline version", func() {
			pipelineDir := "valid"
			pipelineSpecFilePath := filepath.Join(pipelineFilesRootDir, pipelineDir, helloWorldPipelineFileName)
			createdPipeline := uploadPipelineAndVerify(pipelineSpecFilePath, &testContext.Pipeline.PipelineGeneratedName, nil)

			// Create a version without tags
			vNameAddTags := testContext.Pipeline.PipelineGeneratedName + "-v-add-tags"
			uploadParamsAddTags := uploadparams.NewUploadPipelineVersionParams()
			uploadParamsAddTags.Pipelineid = &createdPipeline.PipelineID
			uploadParamsAddTags.Name = &vNameAddTags
			version, err := uploadPipelineVersion(pipelineSpecFilePath, uploadParamsAddTags)
			Expect(err).NotTo(HaveOccurred())

			// Add tags via UpdatePipelineVersion
			newTags := map[string]string{"team": "ml-ops", "env": "dev"}
			updatedVersion, err := pipelineClient.UpdatePipelineVersion(&pipeline_params.PipelineServiceUpdatePipelineVersionParams{
				PipelineVersionPipelineID:        createdPipeline.PipelineID,
				PipelineVersionPipelineVersionID: version.PipelineVersionID,
				PipelineVersion: pipeline_params.PipelineServiceUpdatePipelineVersionBody{
					Tags: newTags,
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedVersion.Tags).To(Equal(newTags))

			// Verify tags are returned by GetPipelineVersion
			retrievedVersion, err := pipelineClient.GetPipelineVersion(&pipeline_params.PipelineServiceGetPipelineVersionParams{
				PipelineID:        createdPipeline.PipelineID,
				PipelineVersionID: version.PipelineVersionID,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(retrievedVersion.Tags).To(Equal(newTags))
		})

		It("Replace existing tags on a pipeline version", func() {
			pipelineDir := "valid"
			pipelineSpecFilePath := filepath.Join(pipelineFilesRootDir, pipelineDir, helloWorldPipelineFileName)
			createdPipeline := uploadPipelineAndVerify(pipelineSpecFilePath, &testContext.Pipeline.PipelineGeneratedName, nil)

			// Create a version with initial tags
			initialTags := map[string]string{"team": "ml-ops"}
			vNameReplace := testContext.Pipeline.PipelineGeneratedName + "-v-replace-tags"
			uploadParamsReplace := uploadparams.NewUploadPipelineVersionParams()
			uploadParamsReplace.Pipelineid = &createdPipeline.PipelineID
			uploadParamsReplace.Name = &vNameReplace
			uploadParamsReplace.SetTagsMap(initialTags)
			version, err := uploadPipelineVersion(pipelineSpecFilePath, uploadParamsReplace)
			Expect(err).NotTo(HaveOccurred())

			// Replace tags
			newTags := map[string]string{"project": "kfp", "owner": "test"}
			updatedVersion, err := pipelineClient.UpdatePipelineVersion(&pipeline_params.PipelineServiceUpdatePipelineVersionParams{
				PipelineVersionPipelineID:        createdPipeline.PipelineID,
				PipelineVersionPipelineVersionID: version.PipelineVersionID,
				PipelineVersion: pipeline_params.PipelineServiceUpdatePipelineVersionBody{
					Tags: newTags,
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedVersion.Tags).To(Equal(newTags))
			Expect(updatedVersion.Tags).NotTo(HaveKey("team"), "Old tags should be replaced")
		})

		It("Update pipeline version display_name along with tags", func() {
			pipelineDir := "valid"
			pipelineSpecFilePath := filepath.Join(pipelineFilesRootDir, pipelineDir, helloWorldPipelineFileName)
			createdPipeline := uploadPipelineAndVerify(pipelineSpecFilePath, &testContext.Pipeline.PipelineGeneratedName, nil)

			vNameBoth := testContext.Pipeline.PipelineGeneratedName + "-v-update-both"
			uploadParamsBoth := uploadparams.NewUploadPipelineVersionParams()
			uploadParamsBoth.Pipelineid = &createdPipeline.PipelineID
			uploadParamsBoth.Name = &vNameBoth
			version, err := uploadPipelineVersion(pipelineSpecFilePath, uploadParamsBoth)
			Expect(err).NotTo(HaveOccurred())

			newDisplayName := "Updated Version Name"
			newTags := map[string]string{"team": "ml-ops"}
			updatedVersion, err := pipelineClient.UpdatePipelineVersion(&pipeline_params.PipelineServiceUpdatePipelineVersionParams{
				PipelineVersionPipelineID:        createdPipeline.PipelineID,
				PipelineVersionPipelineVersionID: version.PipelineVersionID,
				PipelineVersion: pipeline_params.PipelineServiceUpdatePipelineVersionBody{
					DisplayName: newDisplayName,
					Tags:        newTags,
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedVersion.DisplayName).To(Equal(newDisplayName))
			Expect(updatedVersion.Tags).To(Equal(newTags))
		})
	})
})

var _ = Describe("Update Pipeline Version - Negative Tests >", Label(constants.NEGATIVE, constants.Pipeline, "PipelineVersionTags", constants.APIServerTests, constants.FullRegression), func() {

	Context("Update pipeline version with invalid parameters >", func() {
		It("Update tags with empty key on pipeline version should fail", func() {
			pipelineDir := "valid"
			pipelineSpecFilePath := filepath.Join(pipelineFilesRootDir, pipelineDir, helloWorldPipelineFileName)
			createdPipeline := uploadPipelineAndVerify(pipelineSpecFilePath, &testContext.Pipeline.PipelineGeneratedName, nil)

			vNameNegEmpty := testContext.Pipeline.PipelineGeneratedName + "-v-neg-empty"
			uploadParamsNegEmpty := uploadparams.NewUploadPipelineVersionParams()
			uploadParamsNegEmpty.Pipelineid = &createdPipeline.PipelineID
			uploadParamsNegEmpty.Name = &vNameNegEmpty
			version, err := uploadPipelineVersion(pipelineSpecFilePath, uploadParamsNegEmpty)
			Expect(err).NotTo(HaveOccurred())

			_, err = pipelineClient.UpdatePipelineVersion(&pipeline_params.PipelineServiceUpdatePipelineVersionParams{
				PipelineVersionPipelineID:        createdPipeline.PipelineID,
				PipelineVersionPipelineVersionID: version.PipelineVersionID,
				PipelineVersion: pipeline_params.PipelineServiceUpdatePipelineVersionBody{
					Tags: map[string]string{"": "value"},
				},
			})
			Expect(err).To(HaveOccurred(), "Empty tag key should be rejected")
		})

		It("Update tags with key containing dot on pipeline version should fail", func() {
			pipelineDir := "valid"
			pipelineSpecFilePath := filepath.Join(pipelineFilesRootDir, pipelineDir, helloWorldPipelineFileName)
			createdPipeline := uploadPipelineAndVerify(pipelineSpecFilePath, &testContext.Pipeline.PipelineGeneratedName, nil)

			vNameNegDot := testContext.Pipeline.PipelineGeneratedName + "-v-neg-dot"
			uploadParamsNegDot := uploadparams.NewUploadPipelineVersionParams()
			uploadParamsNegDot.Pipelineid = &createdPipeline.PipelineID
			uploadParamsNegDot.Name = &vNameNegDot
			version, err := uploadPipelineVersion(pipelineSpecFilePath, uploadParamsNegDot)
			Expect(err).NotTo(HaveOccurred())

			_, err = pipelineClient.UpdatePipelineVersion(&pipeline_params.PipelineServiceUpdatePipelineVersionParams{
				PipelineVersionPipelineID:        createdPipeline.PipelineID,
				PipelineVersionPipelineVersionID: version.PipelineVersionID,
				PipelineVersion: pipeline_params.PipelineServiceUpdatePipelineVersionBody{
					Tags: map[string]string{"team.name": "ml-ops"},
				},
			})
			Expect(err).To(HaveOccurred(), "Tag key containing dot should be rejected")
		})

		It("Update tags exceeding maximum count on pipeline version should fail", func() {
			pipelineDir := "valid"
			pipelineSpecFilePath := filepath.Join(pipelineFilesRootDir, pipelineDir, helloWorldPipelineFileName)
			createdPipeline := uploadPipelineAndVerify(pipelineSpecFilePath, &testContext.Pipeline.PipelineGeneratedName, nil)

			vNameNegCount := testContext.Pipeline.PipelineGeneratedName + "-v-neg-count"
			uploadParamsNegCount := uploadparams.NewUploadPipelineVersionParams()
			uploadParamsNegCount.Pipelineid = &createdPipeline.PipelineID
			uploadParamsNegCount.Name = &vNameNegCount
			version, err := uploadPipelineVersion(pipelineSpecFilePath, uploadParamsNegCount)
			Expect(err).NotTo(HaveOccurred())

			tooManyTags := make(map[string]string)
			for i := 0; i < 11; i++ {
				tooManyTags[fmt.Sprintf("key%d", i)] = fmt.Sprintf("val%d", i)
			}
			_, err = pipelineClient.UpdatePipelineVersion(&pipeline_params.PipelineServiceUpdatePipelineVersionParams{
				PipelineVersionPipelineID:        createdPipeline.PipelineID,
				PipelineVersionPipelineVersionID: version.PipelineVersionID,
				PipelineVersion: pipeline_params.PipelineServiceUpdatePipelineVersionBody{
					Tags: tooManyTags,
				},
			})
			Expect(err).To(HaveOccurred(), "More than 10 tags should be rejected")
		})

		It("Update tags with key exceeding 20 characters on pipeline version should fail", func() {
			pipelineDir := "valid"
			pipelineSpecFilePath := filepath.Join(pipelineFilesRootDir, pipelineDir, helloWorldPipelineFileName)
			createdPipeline := uploadPipelineAndVerify(pipelineSpecFilePath, &testContext.Pipeline.PipelineGeneratedName, nil)

			vNameNegLongk := testContext.Pipeline.PipelineGeneratedName + "-v-neg-longk"
			uploadParamsNegLongk := uploadparams.NewUploadPipelineVersionParams()
			uploadParamsNegLongk.Pipelineid = &createdPipeline.PipelineID
			uploadParamsNegLongk.Name = &vNameNegLongk
			version, err := uploadPipelineVersion(pipelineSpecFilePath, uploadParamsNegLongk)
			Expect(err).NotTo(HaveOccurred())

			longKey := utils.GetRandomString(21)
			_, err = pipelineClient.UpdatePipelineVersion(&pipeline_params.PipelineServiceUpdatePipelineVersionParams{
				PipelineVersionPipelineID:        createdPipeline.PipelineID,
				PipelineVersionPipelineVersionID: version.PipelineVersionID,
				PipelineVersion: pipeline_params.PipelineServiceUpdatePipelineVersionBody{
					Tags: map[string]string{longKey: "value"},
				},
			})
			Expect(err).To(HaveOccurred(), "Tag key exceeding 20 characters should be rejected")
		})

		It("Update tags with value exceeding 20 characters on pipeline version should fail", func() {
			pipelineDir := "valid"
			pipelineSpecFilePath := filepath.Join(pipelineFilesRootDir, pipelineDir, helloWorldPipelineFileName)
			createdPipeline := uploadPipelineAndVerify(pipelineSpecFilePath, &testContext.Pipeline.PipelineGeneratedName, nil)

			vNameNegLongv := testContext.Pipeline.PipelineGeneratedName + "-v-neg-longv"
			uploadParamsNegLongv := uploadparams.NewUploadPipelineVersionParams()
			uploadParamsNegLongv.Pipelineid = &createdPipeline.PipelineID
			uploadParamsNegLongv.Name = &vNameNegLongv
			version, err := uploadPipelineVersion(pipelineSpecFilePath, uploadParamsNegLongv)
			Expect(err).NotTo(HaveOccurred())

			longValue := utils.GetRandomString(21)
			_, err = pipelineClient.UpdatePipelineVersion(&pipeline_params.PipelineServiceUpdatePipelineVersionParams{
				PipelineVersionPipelineID:        createdPipeline.PipelineID,
				PipelineVersionPipelineVersionID: version.PipelineVersionID,
				PipelineVersion: pipeline_params.PipelineServiceUpdatePipelineVersionBody{
					Tags: map[string]string{"team": longValue},
				},
			})
			Expect(err).To(HaveOccurred(), "Tag value exceeding 20 characters should be rejected")
		})
	})
})
