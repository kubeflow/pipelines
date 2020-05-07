def describe_training_job(client, training_job_name):
    return client.describe_training_job(TrainingJobName=training_job_name)


def describe_model(client, model_name):
    return client.describe_model(ModelName=model_name)
