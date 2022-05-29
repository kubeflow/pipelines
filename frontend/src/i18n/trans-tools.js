// Replace with your value
const viStr = 'Chọn ít nhất một tài nguyên để lưu trữ$$lưu trữ$$Chọn một run để sao chép$$Sao chép run $$Chọn ít nhất một tài nguyên để thử lại$$Tạo một bản sao từ trạng thái ban đầu run này$$Chọn một run định kỳ để sao chép$$Sao chép run định kỳ$$Thử lại$$Thử lại run này$$Thu gọn tất cả$$Thu gọn tất cả các phần$$Chọn nhiều run để so sánh$$So sánh các run$$So sánh tối đa 10 runs đã chọn$$Chọn ít nhất một$$xóa$$Xóa$$Chọn ít nhất một pipeline và / hoặc một phiên bản pipeline để xóa$$Lịch trình run đã bị vô hiệu hóa$$Vô hiệu$$Vô hiệu hóa trình kích hoạt của lần chạy$$Lịch trình run đã được bật$$Cho phép$$Kích hoạt trình kích hoạt run$$Mở rộng tất cả$$Mở rộng tất cả các phần$$Tạo experiment$$Tạo một experiment mới$$Tạo run$$Tạo run mới$$Tạo run định kỳ$$Tạo một cuộc chạy định kỳ mới$$Tải lên phiên bản pipeline$$Làm mới$$Làm mới danh sách$$Chọn ít nhất một tài nguyên để khôi phục$$Khôi phục$$Khôi phục (các) run đã lưu trữ về vị trí ban đầu$$Chọn ít nhất một run để kết thúc$$Chấm dứt$$Chấm dứt thực hiện run$$Tải lên pipeline$$sẽ được chuyển đến phần Lưu trữ, nơi bạn vẫn có thể xem$$của nó$$của chúng$$thông tin chi tiết. Xin lưu ý rằng run sẽ không$$bị dừng nếu nó đang chạy khi nó được lưu trữ. Sử dụng hành động Khôi phục để khôi phục$$đến$$Bạn có muốn khôi phục lại không$$run này cho nó$$những run này cho chúng$$vị trí ban đầu$$experiment này cho nó$$những experiment này cho họ$$vị trí ban đầu? Tất cả các hoạt động và công việc trong$$experiment này$$những experiment này$$sẽ ở lại vị trí hiện tại của chúng mặc dù$$sẽ được chuyển đến$$Bạn có muốn xóa không$$Pipeline này$$những Pipeline này$$Hành động này không thể được hoàn tác.$$Phiên bản Pipeline này$$những Phiên bản Pipeline này$$Bạn có muốn xóa cấu hình run định kỳ này không? Hành động này không thể được hoàn tác.$$Bạn có muốn kết thúc run này không? Hành động này không thể được hoàn tác. Điều này sẽ chấm dứt bất kỳ$$đang chạy các pods, nhưng chúng sẽ không bị xóa.$$Bạn có muốn xóa các run đã chọn không? Hành động này không thể được hoàn tác$$Hủy bỏ$$Không thành công$$có lỗi$$đã thành công cho$$Bỏ qua$$run định kỳ$$Không xóa được pipeline$$Không xóa được phiên bản pipeline$$Đã xóa thành công cho$$Không xóa được một số pipeline và / hoặc một số phiên bản pipeline$$thông tin chi tiết. Tất cả các runs trong experiment được lưu trữ này sẽ được lưu trữ. Tất cả các công việc trong experiment được lưu trữ này sẽ bị vô hiệu hóa. Sử dụng hành động Khôi phục trên trang chi tiết experiment để khôi phụcexperiment ';

const viArr = viStr.split("$$");

// Replace with your value
const enObj = {
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


