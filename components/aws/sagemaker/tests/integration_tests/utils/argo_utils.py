import utils


def print_workflow_logs(workflow_name):
    output = get_workflow_logs(workflow_name)
    print(f"workflow logs:\n", output.decode())


def find_in_logs(workflow_name, sub_str):
    logs = get_workflow_logs(workflow_name).decode()
    return logs.find(sub_str) >= 0


def get_workflow_logs(workflow_name):
    return utils.run_command(
        f"argo logs {workflow_name} -n {utils.get_kfp_namespace()}"
    )


def error_in_cw_logs(workflow_name):
    ERROR_MESSAGE = "Error in fetching CloudWatch logs for SageMaker job"
    return find_in_logs(workflow_name, ERROR_MESSAGE)
