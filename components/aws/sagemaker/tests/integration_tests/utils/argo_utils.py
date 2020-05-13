import utils


def print_workflow_logs(workflow_name):
    output = utils.run_command(
        f"argo logs {workflow_name} -n {utils.get_kfp_namespace()}"
    )
    print(f"workflow logs:\n", output.decode())
