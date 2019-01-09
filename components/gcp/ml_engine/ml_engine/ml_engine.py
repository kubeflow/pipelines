from fire import decorators

from ml_engine.create_job_op import CreateJobOp
from ml_engine.create_model_op import CreateModelOp
from ml_engine.create_version_op import CreateVersionOp
from ml_engine.delete_version_op import DeleteVersionOp

class MLEngine(object):

    @decorators.SetParseFns(runtime_version=str, python_version=str)
    def train(self,
              project_id,
              job_id,
              package_uris,
              python_module,
              region,
              scale_tier=None,
              args=None,
              runtime_version=None,
              python_version=None,
              job_dir=None,
              master_type=None,
              worker_type=None,
              parameter_server_type=None,
              worker_count=None,
              parameter_server_count=None,
              hyperparameters=None,
              labels=None):
        print(package_uris)
        print(type(package_uris))
        training_job = {
            'jobId': job_id,
            'trainingInput': {
                'packageUris': package_uris,
                'pythonModule': python_module,
                'region': region,
            }
        }
        if scale_tier:
            training_job['trainingInput']['scaleTier'] = scale_tier

        if args:
            training_job['trainingInput']['args'] = args

        if runtime_version:
            training_job['trainingInput']['runtimeVersion'] = runtime_version

        if python_version:
            training_job['trainingInput']['pythonVersion'] = python_version

        if job_dir:
            training_job['trainingInput']['jobDir'] = job_dir

        if labels:
            training_job['labels'] = labels

        CreateJobOp(project_id, training_job).execute()

    def create_model(self,
                     project_id,
                     name,
                     description=None,
                     regions=None,
                     onlinePredictionLogging=None,
                     labels=None,
                     etag=None):
        model = {
            'name': name
        }
        if description:
            model['description'] = description
        
        if regions:
            model['regions'] = regions

        if onlinePredictionLogging:
            model['onlinePredictionLogging'] = onlinePredictionLogging
        
        if labels:
            model['labels'] = labels

        if etag:
            model['etag'] = etag
        
        CreateModelOp(project_id, model).execute()

    def delete_version(self, 
                       project_id, 
                       model_name, 
                       version_name,
                       staging_dir=None,
                       wait_interval=30):
        DeleteVersionOp(project_id, model_name, version_name, staging_dir, wait_interval).execute()
    

    @decorators.SetParseFns(version_name=str, python_version=str, runtime_version=str)
    def create_version(self,
                       project_id,
                       model_name,
                       version_name,
                       deployment_uri,
                       description=None,
                       runtime_version=None,
                       machine_type=None,
                       labels=None,
                       framework=None,
                       python_version=None,
                       staging_dir=None,
                       wait_interval=30):
        version = {
            'name': version_name,
            'deploymentUri': deployment_uri
        }

        if description:
            version['description'] = description

        if runtime_version:
            version['runtimeVersion'] = runtime_version

        if machine_type:
            version['machineType'] = machine_type

        if labels:
            version['labels'] = labels

        if framework:
            version['framework'] = framework

        if python_version:
            version['pythonVersion'] = python_version

        CreateVersionOp(project_id, model_name, version, staging_dir, wait_interval).execute()
