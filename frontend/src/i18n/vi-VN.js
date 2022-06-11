export default {
    PipelineList: {
        UploadPipeline: 'Tải lên pipeline',
        Pipelines: 'Pipelines',
        PipelineName: 'Tên Pipeline',
        Description: 'Mô tả',
        UploadedOn: 'Đã tải lên',
        FilterPipelines: 'Lọc pipelines',
        EmptyMessage: 'Không tìm thấy pipeline. Nhấp vào "Tải lên pipeline" để bắt đầu.',
        FailedToRetrieveList: 'Lỗi: không thể truy xuất danh sách các pipelines.',
        FailedToUpload: 'Không tải lên được pipeline',
    },
    Page: {
        ClickDetails: ' Nhấp vào Chi tiết để biết thêm thông tin.',
        Dismiss: 'Bỏ qua'
    },
    Banner: {
        dialogTitle1: 'Đã xảy ra lỗi',
        dialogTitle2: 'Cảnh báo',
        dialogTitle3: 'Thông tin',
        TroubleshootingGuide: 'Hướng dẫn khắc phục sự cố',
        Details: 'Chi tiết',
        Refresh: 'Làm mới',
        Dismiss: 'Bỏ qua'
    },
    NewPipelineVersion: {
        UploadPipelineWithTheSpecifiedPackage: 'Tải lên pipeline với gói được chỉ định',
        PipelineName: 'Tên Pipeline',
        Description: 'Mô tả',
        UploadedOn: 'Đã tải lên',
        PipelineVersions: 'Phiên bản Pipeline',
        UploadPipeline: 'Tải lên Pipeline hoặc phiên bản Pipeline',
        CreateNewPipeline: 'Tạo pipeline mới',
        CreateNewUnderExistingPipeline: 'Tạo phiên bản pipeline mới theo pipeline hiện có',
        PipelineDescription: 'Mô tả pipeline',
        ChoosePipeline: 'Chọn một pipeline',
        FilterPipelines: 'Lọc pipelines',
        NoPipelinesFound: 'Không tìm thấy pipelines. Tải lên một pipeline và thử lại.',
        Cancel: 'Hủy bỏ',
        UseThisPipeline: 'Sử dụng pipeline này',
        PipelineVersionName: 'Tên phiên bản pipeline',
        PipelineVersionDescription: 'Mô tả phiên bản pipeline',
        ChoosePipelineFromComputer: 'Chọn tệp gói pipeline từ máy tính của bạn và đặt tên duy nhất cho pipeline.',
        YouCanDragDrop: 'Bạn cũng có thể kéo và thả tệp tại đây.',
        URLMustBePubliclyAccessible: 'URL phải có thể truy cập công khai.',
        UploadFile: 'Tải lên một tập tin',
        File: 'Tập tin',
        ChooseFile: 'Chọn tập tin',
        ImportByUrl: 'Nhập bằng url',
        PackageUrl: 'Url gói',
        CodeSource: 'Nguồn mã (tùy chọn)',
        Create: 'Tạo',
        SuccessfullyCreate: 'Đã tạo thành công phiên bản pipeline mới:',
        PipelineVersionCreationFailed: 'Không tạo được phiên bản pipeline',
        FileShouldBeSelected: 'Tệp nên được chọn',
        MustSpecifyEitherPackageUrlOrFileIn: 'Phải chỉ định url gói hoặc tệp ở dạng .yaml, .zip hoặc .tar.gz',
        PipelineRequired: 'Pipeline là bắt buộc',
        PipelineVersionNameRequired: 'Tên phiên bản pipeline là bắt buộc',
        PleaseSpecifyEitherPackageUrlOrFileIn: 'Vui lòng chỉ định url gói hoặc tệp ở dạng .yaml, .zip hoặc .tar.gz'
    },
    PipelineDetails: {
        UploadVersion: 'Tải lên phiên bản',
        PipelineDetails: 'Chi tiết pipeline',
        FailedToParse: 'Không thể phân tích cú pháp thông số pipeline khi chạy với ID',
        AllRuns: 'Tất cả các lần chạy',
        CannotRetrieveRunDetails: 'Không thể truy xuất chi tiết chạy.',
        CannotRetrievePipelineVersion: 'Không thể truy xuất phiên bản pipeline.',
        CannotRetrievePipelineTemplate: 'Không thể truy xuất mẫu pipeline.',
        UnableToConvertStringResponse: 'Không thể chuyển đổi phản hồi chuỗi từ máy chủ sang mẫu quy trình làm việc Argo',
        FailedToGeneratePipelineGraph: 'Lỗi: không tạo được biểu đồ pipeline.',
    },
    Buttons: {
        "selectAtLeastOneResourceToRetry": "Chọn ít nhất một tài nguyên để lưu trữ",
        "archive": "lưu trữ",
        "selectAtLeastOneResourceToArchive": "Chọn một run để sao chép",
        "selectARunToClone": "Sao chép run ",
        "cloneRun": "Chọn ít nhất một tài nguyên để thử lại",
        "createACopyFromThisRunsInitialState": "Tạo một bản sao từ trạng thái ban đầu run này",
        "selectARecurringRunToClone": "Chọn một run định kỳ để sao chép",
        "cloneRecurringRun": "Sao chép run định kỳ",
        "retry": "Thử lại",
        "retryThisRun": "Thử lại run này",
        "collapseAll": "Thu gọn tất cả",
        "collapseAllSections": "Thu gọn tất cả các phần",
        "selectMultipleRunsToCompare": "Chọn nhiều run để so sánh",
        "compareRuns": "So sánh các run",
        "compareUpTo10SelectedRuns": "So sánh tối đa 10 runs đã chọn",
        "selectAtLeastOne": "Chọn ít nhất một",
        "toDelete": "xóa",
        "delete": "Xóa",
        "selectAtLeastOnePipelineAndOrOnePipelineVersionToDelete": "Chọn ít nhất một pipeline và / hoặc một phiên bản pipeline để xóa",
        "runScheduleAlreadyDisabled": "Lịch trình run đã bị vô hiệu hóa",
        "disable": "Vô hiệu",
        "disableTheRunSTrigger": "Vô hiệu hóa trình kích hoạt của lần chạy",
        "runScheduleAlreadyEnabled": "Lịch trình run đã được bật",
        "enable": "Cho phép",
        "enableTheRunSTrigger": "Kích hoạt trình kích hoạt run",
        "expandAll": "Mở rộng tất cả",
        "expandAllSections": "Mở rộng tất cả các phần",
        "createExperiment": "Tạo experiment",
        "createANewExperiment": "Tạo một experiment mới",
        "createRun": "Tạo run",
        "createANewRun": "Tạo run mới",
        "createRecurringRun": "Tạo run định kỳ",
        "createANewRecurringRun": "Tạo một cuộc chạy định kỳ mới",
        "uploadPipelineVersion": "Tải lên phiên bản pipeline",
        "refresh": "Làm mới",
        "refreshTheList": "Làm mới danh sách",
        "selectAtLeastOneResourceToRestore": "Chọn ít nhất một tài nguyên để khôi phục",
        "restore": "Khôi phục",
        "restoreTheArchivedRunSToOriginalLocation": "Khôi phục (các) run đã lưu trữ về vị trí ban đầu",
        "selectAtLeastOneRunToTerminate": "Chọn ít nhất một run để kết thúc",
        "terminate": "Chấm dứt",
        "terminateExecutionOfARun": "Chấm dứt thực hiện run",
        "uploadPipeline": "Tải lên pipeline",
        "willBeMovedToTheArchiveSectionWhereYouCanStillView": "sẽ được chuyển đến phần Lưu trữ, nơi bạn vẫn có thể xem",
        "its": "của nó",
        "their": "của chúng",
        "detailsPleaseNoteThatTheRunWillNot": "thông tin chi tiết. Xin lưu ý rằng run sẽ không",
        "beStoppedIfItSRunningWhenItSArchivedUseTheRestoreActionToRestoreThe": "bị dừng nếu nó đang chạy khi nó được lưu trữ. Sử dụng hành động Khôi phục để khôi phục",
        "to": "đến",
        "doYouWantToRestore": "Bạn có muốn khôi phục lại không",
        "thisRunToIts": "run này cho nó",
        "theseRunsToTheir": "những run này cho chúng",
        "originalLocation": "vị trí ban đầu",
        "thisExperimentToIts": "experiment này cho nó",
        "theseExperimentsToTheir": "những experiment này cho họ",
        "originalLocationAllRunsAndJobsIn": "vị trí ban đầu? Tất cả các hoạt động và công việc trong",
        "thisExperiment": "experiment này",
        "theseExperiments": "những experiment này",
        "willStayAtTheirCurrentLocationsInSpiteThat": "sẽ ở lại vị trí hiện tại của chúng mặc dù",
        "willBeMovedTo": "sẽ được chuyển đến",
        "doYouWantToDelete": "Bạn có muốn xóa không",
        "thisPipeline": "Pipeline này",
        "thesePipelines": "những Pipeline này",
        "thisActionCannotBeUndone": "Hành động này không thể được hoàn tác.",
        "thisPipelineVersion": "Phiên bản Pipeline này",
        "thesePipelineVersions": "những Phiên bản Pipeline này",
        "doYouWantToDeleteThisRecurringRunConfigThisActionCannotBeUndone": "Bạn có muốn xóa cấu hình run định kỳ này không? Hành động này không thể được hoàn tác.",
        "doYouWantToTerminateThisRunThisActionCannotBeUndoneThisWillTerminateAny": "Bạn có muốn kết thúc run này không? Hành động này không thể được hoàn tác. Điều này sẽ chấm dứt bất kỳ",
        "runningPodsButTheyWillNotBeDeleted": "đang chạy các pods, nhưng chúng sẽ không bị xóa.",
        "doYouWantToDeleteTheSelectedRunsThisActionCannotBeUndone": "Bạn có muốn xóa các run đã chọn không? Hành động này không thể được hoàn tác",
        "cancel": "Hủy bỏ",
        "failedTo": "Không thành công",
        "withError": "có lỗi",
        "succeededFor": "đã thành công cho",
        "dismiss": "Bỏ qua",
        "recurringRun": "run định kỳ",
        "failedToDeletePipeline": "Không xóa được pipeline",
        "failedToDeletePipelineVersion": "Không xóa được phiên bản pipeline",
        "deletionSucceededFor": "Đã xóa thành công cho",
        "failedToDeleteSomePipelinesAndOrSomePipelineVersions": "Không xóa được một số pipeline và / hoặc một số phiên bản pipeline",
        "detailsAllRunsInThisArchivedExperiment": "thông tin chi tiết. Tất cả các runs trong experiment được lưu trữ này sẽ được lưu trữ. Tất cả các công việc trong experiment được lưu trữ này sẽ bị vô hiệu hóa. Sử dụng hành động Khôi phục trên trang chi tiết experiment để khôi phụcexperiment "
    },
    ExperimentList:{
        failedToRetrieveListOfExperiments: "Lỗi: không thể truy xuất danh sách các Experiment.",
        failedToRetrieveRunStatusesForExperiment: "Lỗi: không thể truy xuất các trạng thái chạy cho Experiments:",
        failedToLoadTheLast5RunsOfThisExperiment: "Không tải được 5 lần runs cuối cùng của Experiment này",
        noExperimentsFound: "Không có Experiment nào được tìm thấy. Nhấp vào \"Tạo Experiment \" để bắt đầu",
        filterExperiments: "Lọc Experiment",
        experimentName: "Tên Experiment",
        description: "Mô tả",
        last5Runs: "5 runs cuối cùng",
        failedToParseRequestFilter: "Lỗi: không thể phân tích cú pháp bộ lọc yêu cầu: "
    },
    ExperimentDetails: {
        experimentDescription: "Mô tả Experiment",
        failedToRetrieveRecurringRunsForExperiment: "Lỗi: không thể truy xuất các lần chạy định kỳ cho Experiment:",
        fetchingRecurringRunsForExperiment: "Lỗi: tìm nạp các lần chạy định kỳ cho Experiment",
        errorFailedToRetrieveExperiment: "Lỗi: không thể truy xuất Experiment",
        errorLoadingExperiment: "Lỗi khi tải Experiment",
    },
    NewExperiment: {
        newExperiment: "Experiment mới",
        experimentName: "Tên Experiment",
        successfullyCreatedNewExperiment: "Đã tạo thành công Experiment mới:",
        experimentCreationFailed: "Tạo Experiment không thành công",
        errorCreatingExperiment: "Lỗi khi tạo Experiment:",
        experimentNameIsRequired: "Tên Experiment là bắt buộc",
        description: "Mô tả (tuỳ chọn)",
        next: "Tiếp theo",
        cancel: "Huỷ bỏ",
        experimentDetails: "Chi tiết Experiment",
        thinkOfAnExperiment: "Hãy coi Experiment như một không gian chứa lịch sử của tất cả các pipeline và các lần chạy liên quan của chúng"
    },
    AllExperimentsAndArchive: {
        active: "Đang hoạt động",
        archived: "Được lưu trữ",
    },
    AllRecurringRusList: {
        recurringRuns: "Runs Định kỳ"
    },
    AllRunsAndArchive: {
        active: "Đang hoạt động",
        archived: "Được lưu trữ",
    },
    NewRun: {
        runDetails: "Chi tiết Run",
        viewPipeline: "Xem pipeline",
        pipelineName: "Tên Pipeline",
        description: "Mô tả",
        uploadedOn: "Đã tải lên",
        versionName: "Tên phiên bản",
        experimentName: "Tên Experiment",
        createdAt: "Được tạo lúc",
        startANewRun: "Bắt đầu một Run mới",
        choose: "Chọn",
        pipelineVersion: "Phiên bản Pipeline",
        chooseAPipeline: "Chọn một Pipeline",
        filterPipelines: "Lọc Pipeline",
        noPipelinesFound: "Không tìm thấy Pipeline . Tải lên Pipeline và sau đó thử lại.",
        cancel: "Huỷ bỏ",
        useThisPipeline: "Sử dụng Pipeline này",
        chooseAPipelineVersion: "Chọn phiên bản Pipeline",
        filterPipelineVersions: "Lọc phiên bản Pipeline",
        noPipelineVersionsFound: "Không tìm thấy phiên bản pipeline nào. Chọn hoặc tải lên một pipeline, sau đó thử lại.",
        useThisPipelineVersion: "Sử dụng phiên bản đường ống này",
        filterExperiments: "Lọc Experiments",
        chooseAnExperiment: "Chọn một experiment",
        noExperimentsFound: "Không có experiment nào được tìm thấy. Tạo một experiment và sau đó thử lại.",
        useThisExperiment: "Sử dụng experiment này",
        recurringRunConfigName: "Tên cấu hình run định kỳ",
        runName: "Tên Run",
        descriptionOption: "Mô tả (không bắt buộc)",
        thisRunWillBeAssociatedWithTheFollowingExperiment: "Lần run này sẽ được kết hợp với experiment sau",
        thisRunWillUseTheFollowingKubernetesServiceAccount: "Lần run này sẽ sử dụng tài khoản dịch vụ Kubernetes sau.",
        noteTheServiceAccountNeeds: "Lưu ý, tài khoản dịch vụ cần",
        minimumPermissionsRequiredByArgoWorkflows: "quyền tối thiểu được yêu cầu bởi quy trình làm việc argo",
        andExtraPermissionsTheSpecificTaskRequires: "và các quyền bổ sung mà tác vụ cụ thể yêu cầu.",
        serviceAccountOptional: "Tài khoản dịch vụ (Tùy chọn)",
        runType: "Loại Run",
        recurring: "Định kỳ",
        oneOff: "Một lần",
        runTrigger: "Kích hoạt Run",
        chooseAMethodByWhichNewRunsWillBeTriggered: "Chọn một phương pháp mà các lần chạy mới sẽ được kích hoạt",
        start: "Bắt đầu",
        skipThisStep: "Bỏ qua bước này",
        someParametersAreMissingValues: "Một số tham số bị thiếu giá trị",
        errorFailedToRetrieveOriginalRun: "Lỗi: không thể truy xuất run ban đầu:",
        errorFailedToRetrieveOriginalRecurringRun: "Lỗi: không thể truy xuất Run lặp lại ban đầu:",
        errorFailedToRetrievePipelineVersion: "Lỗi: không thể truy xuất phiên bản pipeline:",
        errorFailedToRetrievePipeline: "Lỗi: không truy xuất được pipeline:",
        errorFailedToRetrieveAssociatedExperiment: "Lỗi: không thể truy xuất experiment được liên kết:",
        clone: "Sao chép",
        aRecurringRun: "Run định kỳ",
        aRun: "run",
        startARecurringRun: "Bắt đầu run định kỳ",
        failedToUploadPipeline: "Không tải lên được pipeline",
        errorFailedToRetrieveTheSpecifiedRun: "Lỗi: không thể truy xuất run đã chỉ định:",
        errorSomehowTheRunProvidedInTheQueryParams: "Lỗi: bằng cách nào đó run được cung cấp trong tham số truy vấn:",
        hadNoEmbeddedPipeline: "Không có embedded pipeline.",
        usingPipelineFromPreviousPage: "Sử dụng pipeline từ trang trước.",
        errorFailedToParseTheEmbeddedPipelineSSpec: "Lỗi: không thể phân tích cú pháp thông số kỹ thuật của pipeline được nhúng:",
        failedToParseTheEmbeddedPipelineSSpecFromRun: "Không thể phân tích cú pháp thông số kỹ thuật của pipeline được nhúng khi run:",
        couldNotGetClonedRunDetails: "Không thể lấy chi tiết run được sao chép",
        errorFailedToFindAPipelineVersionCorrespondingToThatOfTheOriginalRun: "Lỗi: không tìm thấy phiên bản pipeline tương ứng với phiên bản của run ban đầu: ",
        errorFailedToFindAPipelineCorrespondingToThatOfTheOriginalRun: "Lỗi: không tìm thấy pipeline tương ứng với pipeline của run ban đầu:",
        errorFailedToReadTheCloneRunSPipelineDefinition: "Lỗi: không đọc được định nghĩa pipeline của bản sao run.",
        usingPipelineFromClonedRun: "Sử dụng pipeline từ run nhân bản",
        couldNotFindTheClonedRunSPipelineDefinition: "Không thể tìm thấy định nghĩa pipeline của run nhân bản.",
        errorRun: "Lỗi: run",
        hadNoWorkflowManifest: "Không có bảng kê khai quy trình làm việc",
        specifyParametersRequiredByThePipeline: "Chỉ định các thông số theo yêu cầu của pipeline",
        thisPipelineHasNoParameters: "Pipeline không có tham số",
        parametersWillAppearAfterYouSelectAPipeline: "Các thông số sẽ xuất hiện sau khi bạn chọn một pipeline",
        runCreationFailed:"Run tạo không thành công",
        cannotStartRunWithoutPipelineVersion: "Không thể bắt đầu run nếu không có phiên bản pipeline",
        errorCreatingRun: "Lỗi khi tạo Run",
        successfullyStartedNewRun: "Đã bắt đầu thành công Run mới:",
        cloneOf: "Sao chép {{number}} trong số",
        runOf: "Run của",
        aPipelineVersionMustBeSelected: "Một phiên bản pipeline phải được chọn",
        runNameIsRequired: "Tên run là bắt buộc",
        endDateTimeCannotBeEarlierThanStartDateTime: "Ngày / giờ kết thúc không được sớm hơn ngày / giờ bắt đầu",
        forTriggeredRunsMaximumConcurrentRunsMustBeAPositiveNumber: "Đối với các lần run được kích hoạt, số lần run đồng thời tối đa phải là một số dương"
    },
    ExecutionDetailsContent: {
        declaredInputs : "Đầu vào đã khai báo",
        input: "Đầu vào",
        declaredOutputs: "Đầu ra đã khai báo",
        outputs: "Đầu ra",
        failedToFetchArtifactTypes: "Lỗi không tìm thấy kiểu artifact",
        invalidExecutionId: "ID thực thi không hợp lệ",
        noExecutionIdentifiedById: "Không có sự thực thi nào được xác định bởi id",
        foundMultipleExecutionsWithId: "Tìm thấy nhiều lần thực thi với ID",
        cannotFindExecutionTypeWithId: "Không thể tìm thấy loại thực thi với id",
        moreThanOneExecutionTypeFoundWithId: "Nhiều loại thực thi được tìm thấy với id",
    },
    ExecutionList: {
        name: "Tên",
        state: "Trạng Thái",
        type: "Kiểu",
        failedGettingExecutions: "Không thực hiện được: ",
        noExecutionsFound: "Không tìm thấy executions."
    },
    RecurringRunList: {
        "recurringRunName": "Tên Recurring Run",
        "status": "Trạng thái",
        "trigger": "Kích hoạt",
        "experiment": "Kinh nghiệm",
        "createdAt": "Ngày tạo",
        "filterRecurringRuns": "Lọc Recurring Runs",
        "noAvailableRecurringRunsFound": "Không tìm thấy recurring runs",
        "forThisExperiment": "Cho thử nghiệm này",
        "forThisNamespace": "Cho không gian tên này",
        "failedToFetchRecurringRuns": "Lỗi: Không tìm được recurring runs.",
    },
    RecurringRunDetails: {
        "recurringRunDetails": "Chi tiết Recurring run",
        "runTrigger": "Kích hoạt Run",
        "runParameters": "Thông số Run",
        "failedToRetrieveRecurringRun": "Lỗi: không thể truy xuất lần chạy định kỳ:",
        "failedToRetrieveThisRecurringRunSExperiment": "Lỗi: không thể truy xuất thử nghiệm của lần chạy định kỳ này.",

    },

    ArtifactList: {
        "name": "Tên",
        "type": "Loại",
        "createdAt": "Ngày tạo",
        "noArtifactsFound": "Không tìm thấy artifacts",
        "filter": "Lọc"
    },
    EnhancedArtifactDetails: {
        "noArtifactIdentifiedById": "Không có cấu phần phần mềm nào được xác định bằng id:",
        "foundMultipleArtifactsWithId": "Đã tìm thấy nhiều Artifact có ID:",
        "unknownSelectedTab": "Tab đã chọn không xác định",
        
    },
    PipelineVersionList: {
        versionName: 'Tên phiên bản',
        description: 'Mô tả',
        uploadedOn: 'Đã tải lên',
        noPipelineVersionsFound: 'Không tìm thấy phiên bản pipeline.',
        errorFailedToFetchRuns: 'Lỗi: không tìm được các lần chạy.'
    },
    NewRunSwitcher: {
        currentlyLoadingPipelineInformation: "Hiện đang tải thông tin pipeline"
    },
    InputOutputTab: {
        errorInRetrievingArtifacts: 'Lỗi khi truy xuất Artifacts.',
        thereIsNoInputOutputParameterOrArtifact: 'Không có tham số đầu vào/đầu ra hoặc artifact.'
    },
    MetricsTab: {
        taskIsInUnknownState: 'Task ở trạng thái không xác định.',
        taskHasNotCompleted: 'Task chưa hoàn thành.',
        metricsIsLoading: 'Đang tải số liệu.',
        errorInRetrievingMetricsInformation: 'Lỗi khi truy xuất thông tin số liệu.',
        errorInRetrievingArtifactTypesInformation: 'Lỗi khi truy xuất thông tin các loại artifact.'
    },

    RuntimeNodeDetailsV2: {
        inputOutput: 'Đầu vào/Đầu ra',
        taskDetails: 'Chi tiết task',
        taskName: 'Tên task',
        status: 'Trạng thái',
        createdAt: 'Được tạo lúc',
        finishedAt: 'Hoàn thành lúc',
        artifactInfo: 'Thông tin Artifact',
        visualization: 'Hiển thị',
        upstreamTaskName: 'Tên Upstream Task',
        artifactName: 'Tên Artifact',
        artifactType: 'Kiểu Artifact',
    },
    StaticNodeDetailsV2: {
        contained: 'bao gồm',
        inputArtifacts: 'Artifacts đầu vào',
        inputParameters: 'Tham số đầu vào',
        outputArtifacts: 'Artifacts đầu ra',
        outputParameters: 'Tham số đầu ra',
        artifactName: 'Tên Artifact',
        artifactType: 'Kiểu Artifact',
    },
    FrontendFeatures: {
        contained: 'bao gồm'
    },
    GettingStarted: {
        gettingStarted: 'Bắt đầu',
    },
    ResourceSelector: {
        errorRetrievingResources: 'Lỗi khi truy xuất tài nguyên'
    },
    Status: {
        unknownStatus: 'Trạng thái không xác định',
        errorWhileRunningThisResource: 'Lỗi khi chạy tài nguyên này',
        resourceFailedToExecute: 'Tài nguyên không thực thi được',
        pendingExecution: 'Đang chờ thực hiện',
        running: 'Đang chạy',
        runIsTerminating: 'Đang kết thúc chạy',
        executionHasBeenSkippedForThisResource: 'Quá trình thực thi đã bị bỏ qua đối với tài nguyên này',
        executedSuccessfully: 'Đã thực thi thành công',
        executionWasSkippedAndOutputsWereTakenFromCache: 'Quá trình thực thi đã bị bỏ qua và kết quả đầu ra được lấy từ bộ nhớ cache',
        runWasManuallyTerminated: 'Đã kết thúc chạy theo cách thủ công',
        runWasOmittedBecauseThePreviousStepFailed: 'Chạy đã bị bỏ qua vì bước trước đó không thành công.',
    },
    NewRunV2: {
        startANewRun: "Bắt đầu Run mới",
        runNameCanNotBeEmpty: "Tên Run không được để trống.",
        successfullyStartedNewRun: "Bắt đầu thành công RUn mới:",
        runCreationFailed: "Run không thành công",
        pipelineVersion: "Phiên bản Pipeline",
        description: "Mô tả(không bắt buộc)",
        thisRunWillBeAssociatedWithTheFollowingExperiment: "Lần chạy này sẽ được kết hợp với thử nghiệm sau",
        start: "Bắt đầu",
        cancel: "Huỷ bỏ",
        experimentName:"tên thí nghiệm",
        description: "Mô tả",
        createdAt: "Ngày tạo",
        experiment: "Experiment",
        chooseAnExperiment: "Chọn kinh nghiệm",
        filterExperiments: "Lọc experiments",
        noExperimentsFoundCreateAnExperimentAndThenTryAgain: "Không có thử nghiệm nào được tìm thấy. Tạo một thử nghiệm và sau đó thử lại.",
    },
    Toolbar: {
        back: "Quay lại",
    },
    Trigger: {
        triggerType: "Loại Trigger",
        maximumConcurrentRuns: "Số lần chạy đồng thời tối đa",
        hasStartDate: "Có ngày bắt đầu",
        hasStartDate: "Có ngày kết thúc",
        startDate: "Ngày bắt đầu",
        startDate: "Ngày kết thúc",
        startTime: "Thời gian bắt đầu",
        endTime: "Thời gian kết thúc",
        cronExpression: "biểu thức cron",


    },
    UploadPipelineDialog: {
        uploadAFile: "Tải File",
        importByURL: "Nhập URL",
        file: "Tệp",
        pipelineName: "Tên Pipeline"
    },
    PipelineVersionCard: {
        hide: "Ẩn",
        version: "Phiên bản",
        versionSource: "Nguồn phiên bản",
        uploadOn:"Tải lên",
        pipelineDescription: "Mô tả Pipeline",
        showSummary: "Hiển thị tóm tắt"
    },
    RecurringRunsManager: {
        runName: "Tên Run",
        createdAt: "Ngày tạo",
        filterRecurringRuns: "Lọc RecurringRuns",
        enabled: "Kích hoạt",
        disabled: "Vô hiệu hóa",
        errorRetrievingRecurringRunConfigs: "Lỗi khi truy xuất cấu hình chạy định kỳ",
        couldNotGetListOfRecurringRuns: "Không thể nhận danh sách các lần chạy định kỳ",
        error: "Lỗi",
        errorChangingEnabledStateOfRecurringRun: "Lỗi khi thay đổi trạng thái chạy định kỳ đã bật"

    }

}
