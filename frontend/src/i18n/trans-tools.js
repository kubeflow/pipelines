// Replace with your value
const viStr = "Tên đường ống$$mô tả$$Đã tải lên$$Tên phiên bản$$Tên Experiment $$Được tạo lúc$$Bắt đầu một Run mới$$Chọn$$Phiên bản Pipeline$$Chọn một Pipeline$$Lọc Pipeline $$Không tìm thấy Pipeline . Tải lên Pipeline và sau đó thử lại.$$Huỷ bỏ$$Sử dụng Pipeline này$$Chọn phiên bản Pipeline $$Lọc phiên bản Pipeline $$Không tìm thấy phiên bản đường ống nào. Chọn hoặc tải lên một đường dẫn, sau đó thử lại.$$Sử dụng phiên bản đường ống này$$Lọc thử nghiệm$$Chọn một thử nghiệm$$Không có thử nghiệm nào được tìm thấy. Tạo một thử nghiệm và sau đó thử lại.$$Sử dụng thử nghiệm này$$Tên cấu hình chạy định kỳ$$Tên chạy$$Mô tả (không bắt buộc)$$Lần chạy này sẽ được kết hợp với thử nghiệm sau$$Lần chạy này sẽ sử dụng tài khoản dịch vụ Kubernetes sau.$$Lưu ý, tài khoản dịch vụ cần$$quyền tối thiểu được yêu cầu bởi quy trình làm việc argo$$và các quyền bổ sung mà tác vụ cụ thể yêu cầu.$$Tài khoản dịch vụ (Tùy chọn)$$Loại Run$$Định kỳ$$Một lần$$Kích hoạt Run$$Chọn một phương pháp mà các lần chạy mới sẽ được kích hoạt$$Bắt đầu$$Bỏ qua bước này$$Một số tham số bị thiếu giá trị$$Lỗi: không thể truy xuất run ban đầu:$$Lỗi: không thể truy xuất Run lặp lại ban đầu:$$Lỗi: không thể truy xuất phiên bản pipeline:$$Lỗi: không truy xuất được pipeline:$$Lỗi: không thể truy xuất experiment được liên kết:$$Sao chép$$run định kỳ$$run$$Bắt đầu run định kỳ$$Bắt đầu run mới$$Không tải lên được pipeline$$Lỗi: không thể truy xuất run đã chỉ định:$$Lỗi: bằng cách nào đó run được cung cấp trong tham số truy vấn:$$Không có embedded pipeline.$$Sử dụng pipeline từ trang trước.$$Lỗi: không thể phân tích cú pháp thông số kỹ thuật của pipeline được nhúng:$$Không thể phân tích cú pháp thông số kỹ thuật của pipeline được nhúng khi run:$$Không thể lấy chi tiết run được sao chép$$Lỗi: không tìm thấy phiên bản pipeline tương ứng với phiên bản của run ban đầu: $$Lỗi: không tìm thấy pipeline tương ứng với pipeline của run ban đầu:$$Lỗi: không đọc được định nghĩa pipeline của bản sao run.$$Sử dụng pipeline từ run nhân bản$$Không thể tìm thấy định nghĩa pipeline của run nhân bản.$$Lỗi: run$$không có bảng kê khai quy trình làm việc$$Chỉ định các thông số theo yêu cầu của pipeline$$pipeline không có tham số$$Các thông số sẽ xuất hiện sau khi bạn chọn một pipeline$$Runtạo không thành công /',/' Không thể bắt đầu run nếu không có phiên bản pipeline$$Không thể bắt đầu run nếu không có phiên bản pipeline$$Run tạo không thành công$$Lỗi khi tạo Run$$Đã bắt đầu thành công Run mới:$$Sao chép {{number}} trong số$$Run của$$Một phiên bản pipeline phải được chọn$$Tên run là bắt buộc$$Ngày / giờ kết thúc không được sớm hơn ngày / giờ bắt đầu$$Đối với các lần run được kích hoạt, số lần run đồng thời tối đa phải là một số dương";

const viArr = viStr.split("$$");

// Replace with your value
const enObj = {
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
    startANewRun: "Start a new run",
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
    runCreationFailedCannotStartRunWithoutPipelineVersion: "Run creation failed', 'Cannot start run without pipeline version",
    cannotStartRunWithoutPipelineVersion: "Cannot start run without pipeline version",
    runCreationFailed: "Run creation failed",
    errorCreatingRun: "Error creating Run:",
    successfullyStartedNewRun: "Successfully started new Run:",
    cloneOf: "Clone {{number}} of ",
    runOf: "Run of",
    aPipelineVersionMustBeSelected: "A pipeline version must be selected",
    runNameIsRequired: "Run name is required",
    endDateTimeCannotBeEarlierThanStartDateTime: "End date/time cannot be earlier than start date/time",
    forTriggeredRunsMaximumConcurrentRunsMustBeAPositiveNumber: "For triggered runs, maximum concurrent runs must be a positive number"
};

// Convert enObj to enArr
const enArr = Object.keys(enObj).map((key) => ({[key]: enObj[key]}));

if (viArr.length !== enArr.length) {
    alert(`================> Dịch thiếu @@ viArr.length: ${viArr.length} !== enArr.length: ${enArr.length} <===============`);
} else {
    const viObj = {};
    for (let i = 0; i < viArr.length; i++) {
        for (const key in enArr[i]) {
            viObj[key] = viArr[i];
        }
    }
    console.log('viObj@@', viObj);
}


