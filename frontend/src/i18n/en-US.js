export default {
    PipelineList: {
        UploadPipeline: 'Upload pipeline',
        Pipelines: 'Pipelines',
        PipelineName: 'Pipeline name',
        Description: 'Description',
        UploadedOn: 'Uploaded on',
        FilterPipelines: 'Filter pipelines',
        EmptyMessage: 'No pipelines found. Click "Upload pipeline" to start.',
        FailedToRetrieveList: 'Error: failed to retrieve list of pipelines.',
        FailedToUpload: 'Failed to upload pipeline',
    },
    Page: {
        ClickDetails: ' Click Details for more information.',
        Dismiss: 'Dismiss'
    },
    Banner: {
        dialogTitle1: 'An error occurred',
        dialogTitle2: 'Warning',
        dialogTitle3: 'Info',
        TroubleshootingGuide: 'Troubleshooting guide',
        Details: 'Details',
        Refresh: 'Refresh',
        Dismiss: 'Dismiss'
    },
    NewPipelineVersion: {
        UploadPipelineWithTheSpecifiedPackage: 'Upload pipeline with the specified package',
        PipelineName: 'Pipeline name',
        Description: 'Description',
        UploadedOn: 'Uploaded on',
        PipelineVersions: 'Pipeline Versions',
        UploadPipeline: 'Upload Pipeline or Pipeline Version',
        CreateNewPipeline: 'Create a new pipeline',
        CreateNewUnderExistingPipeline: 'Create a new pipeline version under an existing pipeline',
        PipelineDescription: 'Pipeline Description',
        ChoosePipeline: 'Choose a pipeline',
        FilterPipelines: 'Filter pipelines',
        NoPipelinesFound: 'No pipelines found. Upload a pipeline and then try again.',
        Cancel: 'Cancel',
        UseThisPipeline: 'Use this pipeline',
        PipelineVersionName: 'Pipeline Version name',
        PipelineVersionDescription: 'Pipeline Version Description',
        ChoosePipelineFromComputer: 'Choose a pipeline package file from your computer, and give the pipeline a unique name.',
        YouCanDragDrop: 'You can also drag and drop the file here.',
        URLMustBePubliclyAccessible: 'URL must be publicly accessible.',
        UploadFile: 'Upload a file',
        File: 'File',
        ChooseFile: 'Choose file',
        ImportByUrl: 'Import by url',
        PackageUrl: 'Package Url',
        CodeSource: 'Code Source (optional)',
        Create: 'Create',
        SuccessfullyCreate: 'Successfully created new pipeline version:',
        PipelineVersionCreationFailed: 'Pipeline version creation failed',
        FileShouldBeSelected: 'File should be selected',
        MustSpecifyEitherPackageUrlOrFileIn: 'Must specify either package url  or file in .yaml, .zip, or .tar.gz',
        PipelineRequired: 'Pipeline is required',
        PipelineVersionNameRequired: 'Pipeline version name is required',
        PleaseSpecifyEitherPackageUrlOrFileIn: 'Please specify either package url or file in .yaml, .zip, or .tar.gz'
    },
    PipelineDetails: {
        UploadVersion: 'Upload version',
        PipelineDetails: 'Pipeline details',
        FailedToParse: 'Failed to parse pipeline spec from run with ID',
        AllRuns: 'All runs',
        CannotRetrieveRunDetails: 'Cannot retrieve run details.',
        CannotRetrievePipelineVersion: 'Cannot retrieve pipeline version.',
        CannotRetrievePipelineTemplate: 'Cannot retrieve pipeline template.',
        UnableToConvertStringResponse: 'Unable to convert string response from server to Argo workflow template',
        FailedToGeneratePipelineGraph: 'Error: failed to generate Pipeline graph.'
    },
    Buttons: {
        "selectAtLeastOneResourceToRetry": "Select at least one resource to retry",
        "archive": "Archive",
        "selectAtLeastOneResourceToArchive": "Select at least one resource to archive",
        "selectARunToClone": "Select a run to clone",
        "cloneRun": "Clone run",
        "createACopyFromThisRunsInitialState": "Create a copy from this runs initial state",
        "selectARecurringRunToClone": "Select a recurring run to clone",
        "cloneRecurringRun": "Clone recurring run",
        "retry": "Retry",
        "retryThisRun": "Retry this run",
        "collapseAll": "Collapse all",
        "collapseAllSections": "Collapse all sections",
        "selectMultipleRunsToCompare": "Select multiple runs to compare",
        "compareRuns": "Compare runs",
        "compareUpTo10SelectedRuns": "Compare up to 10 selected runs",
        "selectAtLeastOne": "Select at least one",
        "toDelete": "to delete",
        "delete": "Delete",
        "selectAtLeastOnePipelineAndOrOnePipelineVersionToDelete": "Select at least one pipeline and/or one pipeline version to delete",
        "runScheduleAlreadyDisabled": "Run schedule already disabled",
        "disable": "Disable",
        "disableTheRunSTrigger": "Disable the run's trigger",
        "runScheduleAlreadyEnabled": "Run schedule already enabled",
        "enable": "Enable",
        "enableTheRunSTrigger": "Enable the run's trigger",
        "expandAll": "Expand all",
        "expandAllSections": "Expand all sections",
        "createExperiment": "Create experiment",
        "createANewExperiment": "Create a new experiment",
        "createRun": "Create run",
        "createANewRun": "Create a new run",
        "createRecurringRun": "Create recurring run",
        "createANewRecurringRun": "Create a new recurring run",
        "uploadPipelineVersion": "Upload pipeline version",
        "refresh": "Refresh",
        "refreshTheList": "Refresh the list",
        "selectAtLeastOneResourceToRestore": "Select at least one resource to restore",
        "restore": "Restore",
        "restoreTheArchivedRunSToOriginalLocation": "Restore the archived run(s) to original location",
        "selectAtLeastOneRunToTerminate": "Select at least one run to terminate",
        "terminate": "Terminate",
        "terminateExecutionOfARun": "Terminate execution of a run",
        "uploadPipeline": "Upload pipeline",
        "willBeMovedToTheArchiveSectionWhereYouCanStillView": "will be moved to the Archive section, where you can still view",
        "its": "its",
        "their": "their",
        "detailsPleaseNoteThatTheRunWillNot": "details. Please note that the run will not",
        "beStoppedIfItSRunningWhenItSArchivedUseTheRestoreActionToRestoreThe": "be stopped if it's running when it's archived. Use the Restore action to restore the",
        "to": "to",
        "doYouWantToRestore": "Do you want to restore",
        "thisRunToIts": "this run to its",
        "theseRunsToTheir": "these runs to their",
        "originalLocation": "original location",
        "thisExperimentToIts": "this experiment to its",
        "theseExperimentsToTheir": "these experiments to their",
        "originalLocationAllRunsAndJobsIn": "original location? All runs and jobs in",
        "thisExperiment": "this experiment",
        "theseExperiments": "these experiments",
        "willStayAtTheirCurrentLocationsInSpiteThat": "will stay at their current locations in spite that",
        "willBeMovedTo": "will be moved to",
        "doYouWantToDelete": "Do you want to delete",
        "thisPipeline": "this Pipeline",
        "thesePipelines": "these Pipelines",
        "thisActionCannotBeUndone": "This action cannot be undone.",
        "thisPipelineVersion": "this Pipeline Version",
        "thesePipelineVersions": "these Pipeline Versions",
        "doYouWantToDeleteThisRecurringRunConfigThisActionCannotBeUndone": "Do you want to delete this recurring run config? This action cannot be undone.",
        "doYouWantToTerminateThisRunThisActionCannotBeUndoneThisWillTerminateAny": "Do you want to terminate this run? This action cannot be undone. This will terminate any",
        "runningPodsButTheyWillNotBeDeleted": " running pods, but they will not be deleted.",
        "doYouWantToDeleteTheSelectedRunsThisActionCannotBeUndone": "Do you want to delete the selected runs? This action cannot be undone.",
        "cancel": "Cancel",
        "failedTo": "Failed to",
        "withError": "with error",
        "succeededFor": "succeeded for",
        "dismiss": "Dismiss",
        "recurringRun": "recurring run",
        "failedToDeletePipeline": "Failed to delete pipeline",
        "failedToDeletePipelineVersion": "Failed to delete pipeline version",
        "deletionSucceededFor": "Deletion succeeded for",
        "failedToDeleteSomePipelinesAndOrSomePipelineVersions": "Failed to delete some pipelines and/or some pipeline versions",
        "detailsAllRunsInThisArchivedExperiment": "details. All runs in this archived experiment will be archived. All jobs in this archived experiment will be disabled. Use the Restore action on the experiment details page to restore the experiment"
    },
    ExecutionDetailsContent: {
        declaredInputs : "Declared Inputs",
        input: "Input",
        declaredOutputs: "Declared Outputs",
        outputs: "Outputs",
        failedToFetchArtifactTypes: "Failed to fetch artifact types",
        warning: "warning",
        invalidExecutionId: "Invalid execution id",
        error: "error",
        noExecutionIdentifiedById: "No execution identified by id",
        foundMultipleExecutionsWithId: "Found multiple executions with ID",
        cannotFindExecutionTypeWithId: "Cannot find execution type with id",
        moreThanOneExecutionTypeFoundWithId: "More than one execution type found with id",
    },
    ExecutionList: {
        name: "Name",
        state: "State",
        type: "Type",
        failedGettingExecutions: "Failed getting executions: ",
    }

}
