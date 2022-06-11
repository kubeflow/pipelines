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
    ExperimentList:{
        failedToRetrieveListOfExperiments: "Error: failed to retrieve list of experiments.",
        failedToRetrieveRunStatusesForExperiment: "Error: failed to retrieve run statuses for experiment:",
        failedToLoadTheLast5RunsOfThisExperiment: "Failed to load the last 5 runs of this experiment",
        noExperimentsFound: "No experiments found. Click \"Create experiment\" to start.",
        filterExperiments: "Filter experiments",
        experimentName: "Experiment name",
        description: "Description",
        last5Runs: "Last 5 runs",
        failedToParseRequestFilter: "Error: failed to parse request filter: "
    },
    ExperimentDetails: {
        experimentDescription: "Experiment description",
        failedToRetrieveRecurringRunsForExperiment: "Error: failed to retrieve recurring runs for experiment:",
        fetchingRecurringRunsForExperiment: "Error: fetching recurring runs for experiment",
        errorFailedToRetrieveExperiment: "Error: failed to retrieve experiment",
        errorLoadingExperiment: "Error loading experiment:",
    },
    NewExperiment: {
        newExperiment: "New experiment",
        experimentName: "Experiment name",
        successfullyCreatedNewExperiment: "Successfully created new Experiment:",
        experimentCreationFailed: "Experiment creation failed",
        errorCreatingExperiment: "Error creating experiment:",
        experimentNameIsRequired: "Experiment name is required",
        description: "Description (optional)",
        next: "Next",
        cancel: "Cancel",
        experimentDetails: "Experiment details",
        thinkOfAnExperiment: "Think of an Experiment as a space that contains the history of all pipelines and their associated runs"
    },
    AllExperimentsAndArchive: {
        active: "Active",
        archived: "Archived",
    },
    AllRecurringRusList: {
        recurringRuns: "Recurring Runs"
    },
    AllRunsAndArchive: {
        active: "Active",
        archived: "Archived",
    },
    NewRun: {
        runDetails: "Run details",
        viewPipeline: "View pipeline",
        pipelineName: "Pipeline Name",
        description: "Description",
        uploadedOn: "Uploaded On",
        versionName: "Version Name",
        experimentName: "Experiment Name",
        createdAt: "Created at",
        startANewRun: "Start a new run",
        choose: "Choose",
        pipelineVersion: "Pipeline Version",
        chooseAPipeline: "Choose a pipeline",
        filterPipelines: "Filter pipelines",
        noPipelinesFound: "No pipelines found. Upload a pipeline and then try again.",
        cancel: "Cancel",
        useThisPipeline: "Use This Pipeline",
        chooseAPipelineVersion: "Choose a pipeline version",
        filterPipelineVersions: "Filter pipeline versions",
        noPipelineVersionsFound: "No pipeline versions found. Select or upload a pipeline then try again.",
        useThisPipelineVersion: "Use this pipeline version",
        filterExperiments: "Filter experiments",
        chooseAnExperiment: "Choose an experiment",
        noExperimentsFound: "No experiments found. Create an experiment and then try again.",
        useThisExperiment: "Use this experiment",
        recurringRunConfigName: "Recurring run config name",
        runName: "Run Name",
        descriptionOption: "Description (optional)",
        thisRunWillBeAssociatedWithTheFollowingExperiment: "This run will be associated with the following experiment",
        thisRunWillUseTheFollowingKubernetesServiceAccount: "This run will use the following Kubernetes service account.",
        noteTheServiceAccountNeeds: "Note, the service account needs",
        minimumPermissionsRequiredByArgoWorkflows: "minimum permissions required by argo workflows",
        andExtraPermissionsTheSpecificTaskRequires: "and extra permissions the specific task requires.",
        serviceAccountOptional: "Service Account (Optional)",
        runType: "Run Type",
        recurring: "Recurring",
        oneOff: "One-off",
        runTrigger: "Run trigger",
        chooseAMethodByWhichNewRunsWillBeTriggered: "Choose a method by which new runs will be triggered",
        start: "Start",
        skipThisStep: "Skip this step",
        someParametersAreMissingValues: "Some parameters are missing values",
        errorFailedToRetrieveOriginalRun: "Error: failed to retrieve original run:",
        errorFailedToRetrieveOriginalRecurringRun: "Error: failed to retrieve original recurring run:",
        errorFailedToRetrievePipelineVersion: "Error: failed to retrieve pipeline version:",
        errorFailedToRetrievePipeline: "Error: failed to retrieve pipeline:",
        errorFailedToRetrieveAssociatedExperiment: "Error: failed to retrieve associated experiment:",
        clone: "Clone",
        aRecurringRun: "a recurring run",
        aRun: "a run",
        startARecurringRun: "Start a recurring run",
        failedToUploadPipeline: "Failed to upload pipeline",
        errorFailedToRetrieveTheSpecifiedRun: "Error: failed to retrieve the specified run:",
        errorSomehowTheRunProvidedInTheQueryParams: "Error: somehow the run provided in the query params:",
        hadNoEmbeddedPipeline: "had no embedded pipeline.",
        usingPipelineFromPreviousPage: "Using pipeline from previous page.",
        errorFailedToParseTheEmbeddedPipelineSSpec: "Error: failed to parse the embedded pipeline's spec:",
        failedToParseTheEmbeddedPipelineSSpecFromRun: "Failed to parse the embedded pipeline's spec from run:",
        couldNotGetClonedRunDetails: "Could not get cloned run details",
        errorFailedToFindAPipelineVersionCorrespondingToThatOfTheOriginalRun: "Error: failed to find a pipeline version corresponding to that of the original run:",
        errorFailedToFindAPipelineCorrespondingToThatOfTheOriginalRun: "Error: failed to find a pipeline corresponding to that of the original run:",
        errorFailedToReadTheCloneRunSPipelineDefinition: "Error: failed to read the clone run's pipeline definition.",
        usingPipelineFromClonedRun: "Using pipeline from cloned run",
        couldNotFindTheClonedRunSPipelineDefinition: "Could not find the cloned run's pipeline definition.",
        errorRun: "Error: run",
        hadNoWorkflowManifest: "had no workflow manifest",
        specifyParametersRequiredByThePipeline: "Specify parameters required by the pipeline",
        thisPipelineHasNoParameters: "This pipeline has no parameters",
        parametersWillAppearAfterYouSelectAPipeline: "Parameters will appear after you select a pipeline",
        cannotStartRunWithoutPipelineVersion: "Cannot start run without pipeline version",
        runCreationFailed: "Run creation failed",
        errorCreatingRun: "Error creating Run:",
        successfullyStartedNewRun: "Successfully started new Run:",
        cloneOf: "Clone {{number}} of ",
        runOf: "Run of",
        aPipelineVersionMustBeSelected: "A pipeline version must be selected",
        runNameIsRequired: "Run name is required",
        endDateTimeCannotBeEarlierThanStartDateTime: "End date/time cannot be earlier than start date/time",
        forTriggeredRunsMaximumConcurrentRunsMustBeAPositiveNumber: "For triggered runs, maximum concurrent runs must be a positive number",
    },
    ExecutionDetailsContent: {
        declaredInputs : "Declared Inputs",
        input: "Input",
        declaredOutputs: "Declared Outputs",
        outputs: "Outputs",
        failedToFetchArtifactTypes: "Failed to fetch artifact types",
        invalidExecutionId: "Invalid execution id",
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
        noExecutionsFound: "No executions found."
    },
    RecurringRunList: {
        "recurringRunName": "Recurring Run Name",
        "status": "Status",
        "trigger": "Trigger",
        "experiment": "Experiment",
        "createdAt": "Created at",
        "filterRecurringRuns": "Filter recurring runs",
        "noAvailableRecurringRunsFound": "No available recurring runs found​",
        "forThisExperiment": "for this experiment",
        "forThisNamespace": "for this namespace",
        "failedToFetchRecurringRuns": "Error: failed to fetch recurring runs.",
    },
    RecurringRunDetails: {
        "recurringRunDetails": "Recurring run details",
        "runTrigger": "Run trigger",
        "runParameters": "Run parameters",
        "failedToRetrieveRecurringRun": "Error: failed to retrieve recurring run:",
        "failedToRetrieveThisRecurringRunSExperiment": "Error: failed to retrieve this recurring run's experiment.",



    },
    ArtifactList: {
        "name": "Name",
        "type": "Type",
        "createdAt": "Created at",
        "noArtifactsFound":"No artifacts found.",
        "filter": "Filter",
    },
    EnhancedArtifactDetails: {
        "noArtifactIdentifiedById": "No artifact identified by id:",
        "foundMultipleArtifactsWithId": "Found multiple artifacts with ID:",
        "unknownSelectedTab": "Unknown selected tab",
        
    },
    PipelineVersionList: {
        versionName: 'Version name',
        description: 'Description',
        uploadedOn: 'Uploaded on',
        noPipelineVersionsFound: 'No pipeline versions found',
        errorFailedToFetchRuns: 'Error: failed to fetch runs.'
    },
    NewRunSwitcher: {
        currentlyLoadingPipelineInformation: "Currently loading pipeline information"
    },
    InputOutputTab: {
        errorInRetrievingArtifacts: 'Error in retrieving Artifacts.',
        thereIsNoInputOutputParameterOrArtifact: 'There is no input/output parameter or artifact.'
    },
    MetricsTab: {
        taskIsInUnknownState: 'Task is in unknown state.',
        taskHasNotCompleted: 'Task has not completed.',
        metricsIsLoading: 'Metrics is loading.',
        errorInRetrievingMetricsInformation: 'Error in retrieving metrics information.',
        errorInRetrievingArtifactTypesInformation: 'Error in retrieving artifact types information.'
    },
    RuntimeNodeDetailsV2: {
        inputOutput: 'Input/Output',
        taskDetails: 'Task Details',
        taskName: 'Task name',
        status: 'Status',
        createdAt: 'Created At',
        finishedAt: 'Finished At',
        artifactInfo: 'Artifact Info',
        visualization: 'Visualization',
        upstreamTaskName: 'Upstream Task Name',
        artifactName: 'Artifact Name',
        artifactType: 'Artifact Type',
    },
    StaticNodeDetailsV2: {
        contained: 'contained',
        inputArtifacts: 'Input Artifacts',
        inputParameters: 'Input Parameters',
        outputArtifacts: 'Output Artifacts',
        outputParameters: 'Output Parameters',
        artifactName: 'Artifact Name',
        artifactType: 'Artifact Type',
    },
    FrontendFeatures: {
        contained: 'contained'
    },
    GettingStarted: {
        gettingStarted: 'Getting Started',
    },
    ResourceSelector: {
        errorRetrievingResources: 'Error retrieving resources'
    },
    Status: {
        unknownStatus: 'Unknown status',
        errorWhileRunningThisResource: 'Error while running this resource',
        resourceFailedToExecute: 'Resource failed to execute',
        pendingExecution: 'Pending execution',
        running: 'Running',
        runIsTerminating: 'Run is terminating',
        executionHasBeenSkippedForThisResource: 'Execution has been skipped for this resource',
        executedSuccessfully: 'Executed successfully',
        executionWasSkippedAndOutputsWereTakenFromCache: 'Execution was skipped and outputs were taken from cache',
        runWasManuallyTerminated: 'Run was manually terminated',
        runWasOmittedBecauseThePreviousStepFailed: 'Run was omitted because the previous step failed.',
    },
    NewRunV2: {
        startANewRun: "Start a new run",
        runNameCanNotBeEmpty: "Run name can not be empty.",
        successfullyStartedNewRun: "Successfully started new Run:",
        runCreationFailed:"Run creation failed",
        pipelineVersion:"Pipeline Version",
        description: "Description (optional)",
        thisRunWillBeAssociatedWithTheFollowingExperiment: "This run will be associated with the following experiment",
        start: "Start",
        cancel: "Cancel",
        experimentName: "Experiment name",
        description: "Description",
        createdAt: "Created at",
        experiment: "Kinh nghiệm",
        chooseAnExperiment: "Choose an experiment",
        filterExperiments: "Filter experiments",
        noExperimentsFoundCreateAnExperimentAndThenTryAgain: "No experiments found. Create an experiment and then try again.",
    },
    Toolbar: {
        back: "Back",
    },
    Trigger: {
        triggerType: "Trigger type",
        maximumConcurrentRuns: "Maximum concurrent runs",
        hasStartDate: "Has start date",
        hasEndDate: "Has end date",
        startDate: "Start date",
        endDate: "End date",
        startTime: "Start time",
        endTime: "End time",
        cronExpression: "cron expression"

    },
    UploadPipelineDialog: {
        uploadAFile: "Up Load File",
        importByURL: "Import by URL",
        file: "File",
        pipelineName: "Pipeline name",
        versionSource: "Version source",
        uploadOn: "Uploaded on",
        pipelineDescription: "Pipeline Description",
        showSummary: "showSummary",
    },
    PipelineVersionCard: {
        hide: "Hide",
        version: "Version",
        versionSource: "Version Source",
        uploadOn:"Up load",
        pipelineDescription: "Pipeline Description",
        showSummary: "Show summary",
        filterRecurringRuns: "Filter recurring runs",
        

    },
    RecurringRunsManager: {
        runName: "Tên Run",
        createdAt: "Ngày tạo",
        filterRecurringRuns: "Lọc RecurringRuns",
        enabled: "Kích hoạt",
        disabled: "Vô hiệu hóa",
        errorRetrievingRecurringRunConfigs: "Error retrieving recurring run configs",
        couldNotGetListOfRecurringRuns: "Could not get list of recurring runs",
        error: "Error",
        errorChangingEnabledStateOfRecurringRun: "Error changing enabled state of recurring run",
    }
}
