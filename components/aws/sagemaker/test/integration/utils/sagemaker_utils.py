def describe_training_job(client, training_job_name):
    response = client.describe_training_job(TrainingJobName=training_job_name)
    return response
