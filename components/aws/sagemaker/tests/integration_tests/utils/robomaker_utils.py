def describe_simulation_application(client, sim_app_arn):
    return client.describe_simulation_application(application=sim_app_arn)


def describe_simulation_job(client, sim_job_arn):
    return client.describe_simulation_job(job=sim_job_arn)


def describe_simulation_job_batch(client, batch_job_id):
    return client.describe_simulation_job_batch(batch=batch_job_id)


def delete_simulation_application(client, sim_app_arn):
    return client.delete_simulation_application(application=sim_app_arn)


def cancel_simulation_job(client, sim_job_arn):
    return client.cancel_simulation_job(job=sim_job_arn)


def cancel_simulation_job_batch(client, batch_job_id):
    return client.cancel_simulation_job_batch(batch=batch_job_id)


def list_simulation_applications(client, sim_app_name):
    return client.list_simulation_applications(
        filters=[{"name": "name", "values": [sim_app_name]}]
    )


def create_robot_application(client, app_name, sources, robot_software_suite):
    return client.create_robot_application(
        name=app_name, sources=sources, robotSoftwareSuite=robot_software_suite
    )
