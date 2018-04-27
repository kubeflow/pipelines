package resource

import (
	"ml/src/model"
	"ml/src/storage"
	"ml/src/util"
	"testing"

	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
)

func createPkg(name string) *model.Package {
	return &model.Package{Name: name}
}

func TestCreatePipelineInternalNoSchedule(t *testing.T) {
	store := storage.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)
	store.PackageStore().CreatePackage(createPkg("pkg1"))
	store.PackageManager().CreatePackageFile([]byte("kind: Workflow"), "pkg1")

	pipeline := &model.Pipeline{
		Name:      "MY_PIPELINE",
		PackageId: 1}
	pipeline, err := manager.CreatePipeline(pipeline)

	assert.Nil(t, err, "There should not be an error: %v", err)

	expected := model.Pipeline{
		ID:             1,
		CreatedAtInSec: 2,
		UpdatedAtInSec: 2,
		Name:           "MY_PIPELINE",
		PackageId:      1,
		Schedule:       "",
		Enabled:        true,
		EnabledAtInSec: 2}

	assert.Equalf(t, expected, *pipeline, "Unexpected pipeline structure. Expect %v. Got %v.",
		expected, *pipeline)
	assert.Equal(t, 1, store.WorkflowClientFake().GetWorkflowCount(), "Unexpected number of workflows.")
}

func TestCreatePipelineFormatWorkflow(t *testing.T) {
	store := storage.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	// Prepare store
	workflow := &v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{Name: "workflow-name-"},
		Spec: v1alpha1.WorkflowSpec{
			Arguments: v1alpha1.Arguments{
				Parameters: []v1alpha1.Parameter{
					{Name: "param1", Value: util.StringPointer("value1-[[schedule]]")},
					{Name: "param2", Value: util.StringPointer("value2-[[now]]-suffix")},
				},
			}}}
	manager := NewResourceManager(store)
	store.PackageStore().CreatePackage(createPkg("pkg1"))
	store.PackageManager().CreatePackageFile(util.MarshalOrFail(workflow), "pkg1")

	// Create pipeline
	pipeline := &model.Pipeline{
		Name:      "MY_PIPELINE",
		PackageId: 1}
	pipeline, err := manager.CreatePipeline(pipeline)
	assert.Nil(t, err)

	// Check pipeline
	expected := model.Pipeline{
		ID:             1,
		CreatedAtInSec: 2,
		UpdatedAtInSec: 2,
		Name:           "MY_PIPELINE",
		PackageId:      1,
		Schedule:       "",
		Enabled:        true,
		EnabledAtInSec: 2}
	assert.Equal(t, expected, *pipeline)

	// Check workflow
	assert.Equal(t, 1, store.WorkflowClientFake().GetWorkflowCount(), "Unexpected number of workflows.")

	expectedWorkflow := &v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{Name: "workflow-name-123e4567-e89b-12d3-a456-426655440000"},
		Spec: v1alpha1.WorkflowSpec{
			Arguments: v1alpha1.Arguments{
				Parameters: []v1alpha1.Parameter{
					{Name: "param1", Value: util.StringPointer("value1-19700101000003")},
					{Name: "param2", Value: util.StringPointer("value2-19700101000004-suffix")},
				},
			}}}

	jobDetail, err := store.JobStore().GetJob(1, "workflow-name-123e4567-e89b-12d3-a456-426655440000")
	assert.Nil(t, err)
	assert.Equal(t, expectedWorkflow, jobDetail.Workflow)
}

func TestCreatePipelineInternalValidSchedule(t *testing.T) {
	store := storage.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)
	store.PackageStore().CreatePackage(createPkg("pkg1"))
	store.PackageManager().CreatePackageFile([]byte("kind: Workflow"), "pkg1")

	pipeline := &model.Pipeline{
		Name:      "MY_PIPELINE",
		PackageId: 1,
		Schedule:  "1 0 * * *"}
	pipeline, err := manager.CreatePipeline(pipeline)

	assert.Nil(t, err, "There should not be an error: %v", err)

	expected := model.Pipeline{
		ID:             1,
		CreatedAtInSec: 2,
		UpdatedAtInSec: 2,
		Name:           "MY_PIPELINE",
		PackageId:      1,
		Schedule:       "1 0 * * *",
		Enabled:        true,
		EnabledAtInSec: 2}

	assert.Equalf(t, expected, *pipeline, "Unexpected pipeline structure. Expect %v. Got %v.",
		expected, *pipeline)
	assert.Equal(t, 0, store.WorkflowClientFake().GetWorkflowCount(), "Unexpected number of workflows.")
}

func TestCreatePipelineInternalInvalidSchedule(t *testing.T) {
	store := storage.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)
	store.PackageStore().CreatePackage(createPkg("pkg1"))
	store.PackageManager().CreatePackageFile([]byte("kind: Workflow"), "pkg1")

	pipeline := &model.Pipeline{
		Name:      "MY_PIPELINE",
		PackageId: 1,
		Schedule:  "abcdef"}
	pipeline, err := manager.CreatePipeline(pipeline)

	assert.Contains(t, err.Error(),
		"InvalidInputError: The pipeline schedule cannot be parsed: abcdef: Expected 5 to 6 fields")
	assert.Equal(t, 0, store.WorkflowClientFake().GetWorkflowCount(), "Unexpected number of workflows.")
}

func TestCreateJobFromPipelineID(t *testing.T) {
	// Use a real UUID in this test case to guarantee job with unique name
	store, err := storage.NewFakeClientManager(util.NewFakeTimeForEpoch(), util.NewUUIDGenerator())
	assert.Nil(t, err)
	defer store.Close()
	manager := NewResourceManager(store)

	// Create package.
	_, err = store.PackageStore().CreatePackage(createPkg("pkg1"))
	assert.Nil(t, err)

	err = store.PackageManager().CreatePackageFile([]byte("kind: Workflow"), "pkg1")
	assert.Nil(t, err)
	assert.Equal(t, 0, store.WorkflowClientFake().GetWorkflowCount())

	// Create pipeline with a schedule.
	pipeline := &model.Pipeline{
		Name:      "MY_PIPELINE",
		PackageId: 1,
		Schedule:  "* * * * * *"}
	pipeline, err = store.PipelineStore().CreatePipeline(pipeline)
	assert.Nil(t, err)
	assert.Equal(t, 0, store.WorkflowClientFake().GetWorkflowCount())

	// Create job.
	scheduledAtInSec := int64(5)
	jobDetail1, err := manager.CreateJobFromPipelineID(pipeline.ID, scheduledAtInSec)
	assert.Nil(t, err)
	expectedJob1 := &model.Job{
		Name:             jobDetail1.Workflow.Name,
		CreatedAtInSec:   3,
		UpdatedAtInSec:   4,
		Status:           model.JobExecutionPending,
		ScheduledAtInSec: 5,
		PipelineID:       1,
	}
	assert.Equal(t, map[string]bool{jobDetail1.Workflow.Name: true},
		store.WorkflowClientFake().GetWorkflowKeys())
	assert.Equal(t, expectedJob1, jobDetail1.Job)

	_, err = store.JobStore().GetJob(pipeline.ID, jobDetail1.Workflow.Name)
	assert.Nil(t, err)

	jobs, err := store.JobStore().ListJobs(pipeline.ID)
	assert.Nil(t, err)
	assert.Len(t, jobs, 1)

	// Create another job.
	scheduledAtInSec = int64(6)
	jobDetail2, err := manager.CreateJobFromPipelineID(pipeline.ID, scheduledAtInSec)
	assert.Nil(t, err)
	expectedJob2 := &model.Job{
		Name:             jobDetail2.Workflow.Name,
		CreatedAtInSec:   5,
		UpdatedAtInSec:   6,
		Status:           model.JobExecutionPending,
		ScheduledAtInSec: 6,
		PipelineID:       1,
	}
	assert.Equal(t, map[string]bool{jobDetail1.Workflow.Name: true, jobDetail2.Workflow.Name: true},
		store.WorkflowClientFake().GetWorkflowKeys())
	assert.Equal(t, expectedJob2, jobDetail2.Job)

	_, err = store.JobStore().GetJob(pipeline.ID, jobDetail2.Workflow.Name)
	assert.Nil(t, err)

	jobs, err = store.JobStore().ListJobs(pipeline.ID)
	assert.Nil(t, err)
	assert.Len(t, jobs, 2)
}

func TestCreateJobFromPipelineIDGetPipelineError(t *testing.T) {
	store := storage.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)

	// Create package.
	_, err := store.PackageStore().CreatePackage(createPkg("pkg1"))
	assert.Nil(t, err)

	err = store.PackageManager().CreatePackageFile([]byte("kind: Workflow"), "pkg1")
	assert.Nil(t, err)
	assert.Equal(t, 0, store.WorkflowClientFake().GetWorkflowCount())

	// We do not create the pipeline!

	// Create job.
	scheduledAtInSec := int64(5)
	_, err = manager.CreateJobFromPipelineID(55, scheduledAtInSec)
	assert.Contains(t, err.Error(), "Could not get pipeline from pipeline ID")
}

func TestCreateJobFromPipelineIDGetPackageError(t *testing.T) {
	store := storage.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)

	// We do not create the package!

	// Create pipeline.
	pipeline := &model.Pipeline{
		Name:      "MY_PIPELINE",
		PackageId: 1,
		Schedule:  "* * * * * *"}
	pipeline, err := store.PipelineStore().CreatePipeline(pipeline)
	assert.Nil(t, err)
	assert.Equal(t, 0, store.WorkflowClientFake().GetWorkflowCount())

	// Create job.
	scheduledAtInSec := int64(5)
	_, err = manager.CreateJobFromPipelineID(1, scheduledAtInSec)
	assert.Contains(t, err.Error(), "Could not get the package from the package ID")
}

func TestListJob(t *testing.T) {
	store := storage.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)
	store.PackageStore().CreatePackage(createPkg("pkg1"))
	store.PackageManager().CreatePackageFile([]byte("kind: Workflow"), "pkg1")

	pipeline := &model.Pipeline{
		Name:      "MY_PIPELINE",
		PackageId: 1}
	pipeline, err := manager.CreatePipeline(pipeline)
	assert.Nil(t, err, "There should not be an error: %v", err)

	jobs, err := manager.ListJobs(pipeline.ID)
	jobsExpected := []model.Job{{
		CreatedAtInSec:   4,
		UpdatedAtInSec:   5,
		Name:             "123e4567-e89b-12d3-a456-426655440000",
		Status:           model.JobExecutionPending,
		ScheduledAtInSec: 3,
		PipelineID:       1,
	}}

	assert.Equal(t, jobsExpected, jobs)
}

func TestListJob_PipelineNotFoundError(t *testing.T) {
	store := storage.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)

	_, err := manager.ListJobs(1)
	assert.Contains(t, err.Error(), "Pipeline 1 not found")
}

func TestGetJob(t *testing.T) {
	store := storage.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)
	store.PackageStore().CreatePackage(createPkg("pkg1"))
	store.PackageManager().CreatePackageFile([]byte(""), "pkg1")

	pipeline := &model.Pipeline{
		Name:      "MY_PIPELINE",
		PackageId: 1}
	pipeline, err := manager.CreatePipeline(pipeline)
	assert.Nil(t, err, "There should not be an error: %v", err)

	job, err := manager.GetJob(pipeline.ID, "123e4567-e89b-12d3-a456-426655440000")
	jobExpected := &model.Job{
		CreatedAtInSec:   4,
		UpdatedAtInSec:   5,
		Name:             "123e4567-e89b-12d3-a456-426655440000",
		Status:           model.JobExecutionPending,
		ScheduledAtInSec: 3,
		PipelineID:       1,
	}
	assert.Equal(t, jobExpected, job.Job)
	assert.Equal(t, "123e4567-e89b-12d3-a456-426655440000", job.Workflow.Name)
}

func TestGetJob_PipelineNotFoundError(t *testing.T) {
	store := storage.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)

	_, err := manager.GetJob(1, "foo")
	assert.Contains(t, err.Error(), "Pipeline 1 not found")
}
