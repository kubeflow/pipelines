package resource

import (
	"ml/src/message"
	"ml/src/storage"
	"ml/src/util"
	"testing"

	"github.com/stretchr/testify/assert"
)

func createPkg(name string) *message.Package {
	return &message.Package{Name: name}
}

func TestCreatePipelineInternalNoSchedule(t *testing.T) {
	store := storage.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)
	store.PackageStore().CreatePackage(createPkg("pkg1"))
	store.PackageManager().CreatePackageFile([]byte("kind: Workflow"), "pkg1")

	pipeline := &message.Pipeline{
		Name:      "MY_PIPELINE",
		PackageId: 1}
	err := manager.CreatePipeline(pipeline)

	assert.Nil(t, err, "There should not be an error: %v", err)

	expected := message.Pipeline{
		Metadata:       &message.Metadata{ID: 1},
		Name:           "MY_PIPELINE",
		PackageId:      1,
		Schedule:       "",
		Enabled:        true,
		EnabledAtInSec: 1}

	assert.Equalf(t, expected, *pipeline, "Unexpected pipeline structure. Expect %v. Got %v.",
		expected, *pipeline)
	assert.Equal(t, 1, store.WorkflowClientFake().GetWorkflowCount(), "Unexpected number of workflows.")
}

func TestCreatePipelineInternalValidSchedule(t *testing.T) {
	store := storage.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)
	store.PackageStore().CreatePackage(createPkg("pkg1"))
	store.PackageManager().CreatePackageFile([]byte("kind: Workflow"), "pkg1")

	pipeline := &message.Pipeline{
		Name:      "MY_PIPELINE",
		PackageId: 1,
		Schedule:  "1 0 * * *"}
	err := manager.CreatePipeline(pipeline)

	assert.Nil(t, err, "There should not be an error: %v", err)

	expected := message.Pipeline{
		Metadata:       &message.Metadata{ID: 1},
		Name:           "MY_PIPELINE",
		PackageId:      1,
		Schedule:       "1 0 * * *",
		Enabled:        true,
		EnabledAtInSec: 1}

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

	pipeline := &message.Pipeline{
		Name:      "MY_PIPELINE",
		PackageId: 1,
		Schedule:  "abcdef"}
	err := manager.CreatePipeline(pipeline)

	assert.Contains(t, err.Error(),
		"InvalidInputError: The pipeline schedule cannot be parsed: abcdef: Expected 5 to 6 fields")
	assert.Equal(t, 0, store.WorkflowClientFake().GetWorkflowCount(), "Unexpected number of workflows.")
}

func TestCreateJobFromPipelineID(t *testing.T) {
	store := storage.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)

	// Create package.
	err := store.PackageStore().CreatePackage(createPkg("pkg1"))
	assert.Nil(t, err)

	err = store.PackageManager().CreatePackageFile([]byte("kind: Workflow"), "pkg1")
	assert.Nil(t, err)
	assert.Equal(t, 0, store.WorkflowClientFake().GetWorkflowCount())

	// Create pipeline with a schedule.
	pipeline := &message.Pipeline{
		Name:      "MY_PIPELINE",
		PackageId: 1,
		Schedule:  "* * * * * *"}
	err = store.PipelineStore().CreatePipeline(pipeline)
	assert.Nil(t, err)
	assert.Equal(t, 0, store.WorkflowClientFake().GetWorkflowCount())

	// Create job.
	scheduledAtInSec := int64(5)
	jobDetail1, err := manager.CreateJobFromPipelineID(pipeline.ID, scheduledAtInSec)
	assert.Nil(t, err)
	jobDetail1.Job.Metadata = nil // TODO: make the fields in Metadata testable.
	expectedJob1 := &message.Job{
		Name:             jobDetail1.Workflow.Name,
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
	jobDetail2.Job.Metadata = nil // TODO: make the fields in Metadata testable.
	expectedJob2 := &message.Job{
		Name:             jobDetail2.Workflow.Name,
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
	err := store.PackageStore().CreatePackage(createPkg("pkg1"))
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
	pipeline := &message.Pipeline{
		Name:      "MY_PIPELINE",
		PackageId: 1,
		Schedule:  "* * * * * *"}
	err := store.PipelineStore().CreatePipeline(pipeline)
	assert.Nil(t, err)
	assert.Equal(t, 0, store.WorkflowClientFake().GetWorkflowCount())

	// Create job.
	scheduledAtInSec := int64(5)
	_, err = manager.CreateJobFromPipelineID(1, scheduledAtInSec)
	assert.Contains(t, err.Error(), "Could not get the package from the package ID")
}
