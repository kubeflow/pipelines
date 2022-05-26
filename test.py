from google.cloud import aiplatform

gcp_project = ''
gcp_region = ''
gcs_base_path = ''

import os

root_dir = os.path.join(gcs_base_path, 'default_evaluation_pipeline')
x = os.path.expanduser(
    '~/workspace/pipelines/components/google-cloud/google_cloud_pipeline_components/aiplatform/batch_predict_job/component.yaml'
)
# On the model-evaluation-dev project:
# Classification Heart Study Model: 6842964547091824640
# Regression Heart Study Model: 682427284941963264

prediction_type = 'classification'
# prediction_type = 'regression'

if prediction_type == 'classification':
    # Classification Heart Study Model: 6842964547091824640
    prediction_type = 'classification'
    model_id = 6842964547091824640
    target = 'male'
elif prediction_type == 'regression':
    # Regression Heart Study Model: 682427284941963264
    prediction_type = 'regression'
    model_id = 682427284941963264
    target = 'age'

model_name = "projects/" + gcp_project + "/locations/" + gcp_region + "/models/" + str(
    model_id)
print(model_name)

parameter_values = {
    'project':
    gcp_project,
    'location':
    gcp_region,
    'root_dir':
    root_dir,
    'prediction_type':
    prediction_type,
    'model_name':
    model_name,
    'target_column_name':
    target,
    'batch_predict_gcs_source_uris':
    ['gs://xwx-us-central1-test/Heart_study_dataset_4k.csv'],
    'batch_predict_instances_format':
    'csv',
}

ar_job_id = "evaluation-default-pipeline-20220526-1"
ar_job = aiplatform.PipelineJob(display_name=ar_job_id,
                                template_path=x,
                                job_id=ar_job_id,
                                pipeline_root=root_dir,
                                parameter_values=parameter_values,
                                enable_caching=True)
ar_job.submit()
