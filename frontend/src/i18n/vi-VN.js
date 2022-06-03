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
}
