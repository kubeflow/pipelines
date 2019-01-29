# Copyright 2018 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

from fire import decorators

from gcp_common import validators
from ml_engine.create_job_op import CreateJobOp
from ml_engine.create_model_op import CreateModelOp
from ml_engine.create_version_op import CreateVersionOp
from ml_engine.delete_version_op import DeleteVersionOp

class MLEngine(object):
    """MLEngine commands entrypoint.

    Args:
        wait_interval: optional wait interval between calls
            to get job status. Defaults to 30.
    """
    @decorators.SetParseFns(wait_interval=int)
    def __init__(self, wait_interval=30):
        self._wait_interval = wait_interval

    @decorators.SetParseFns(project_id=str, job_id=str, python_module=str,
        region=str, scale_tier=str, runtime_version=str, python_version=str,
        job_dir=str, master_type=str, worker_type=str, 
        parameter_server_type=str, worker_count=str, 
        parameter_server_count=str)
    def train(self,
              project_id,
              job_id,
              package_uris,
              python_module,
              region,
              args=None,
              scale_tier=None,
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
        """Creates a MLEngine training job.

        Args:
            project_id: the ID of the parent project of the job.
            job_id: Required. The user-specified id of the job.
            package_uris: Required. The Google Cloud Storage location 
                of the packages with the training program and any 
                additional dependencies. The maximum number of package 
                URIs is 100.
            python_module: Required. The Python module name to run after 
                installing the packages.
            region: Required. The Google Compute Engine region to run 
                the training job in.
            args: Command line arguments to pass to the program.
            scale_tier: Specifies the machine types, the number of 
                replicas for workers and parameter servers.
            runtime_version: Optional. The Cloud ML Engine runtime 
                version to use for training. If not set, Cloud ML 
                Engine uses the default stable version, 1.0.
            python_version: Optional. The version of Python used in training. 
                If not set, the default version is '2.7'. Python '3.5' is 
                available when runtimeVersion is set to '1.4' and above. 
                Python '2.7' works with all supported runtime versions.
            job_dir: Optional. A Google Cloud Storage path in which to 
                store training outputs and other data needed for 
                training. This path is passed to your TensorFlow program as 
                the '--job-dir' command-line argument. The benefit of 
                specifying this field is that Cloud ML validates the path for 
                use in training.
            master_type: Optional. Specifies the type of virtual machine to 
                use for your training job's master worker.
            worker_type: Optional. Specifies the type of virtual machine to 
                use for your training job's worker nodes.
            parameter_server_type: Optional. Specifies the type of virtual 
                machine to use for your training job's parameter server.
            worker_count: Optional. The number of worker replicas to use for 
                the training job. Each replica in the cluster will be of the 
                type specified in workerType.
            parameter_server_count: Optional. The number of parameter server 
                replicas to use for the training job. Each replica in the 
                cluster will be of the type specified in parameterServerType.
            hyperparameters: Optional. The set of Hyperparameters to tune.
            labels: Optional. One or more labels that you can add, to 
                organize your jobs. Each label is a key-value pair, where 
                both the key and the value are arbitrary strings that you 
                supply.
        """
        validators.validate_required(project_id, 'project_id')
        validators.validate_project_id(project_id)
        validators.validate_required(job_id, 'job_id')
        validators.validate_required(package_uris, 'package_uris')
        validators.validate_type(package_uris, list, 'package_uris')
        validators.validate_required(python_module, 'python_module')
        validators.validate_required(region, 'region')
        training_job = {
            'jobId': job_id,
            'trainingInput': {
                'packageUris': package_uris,
                'pythonModule': python_module,
                'region': region,
            }
        }
        if labels:
            training_job['labels'] = validators.validate_dict(
                labels, 'labels')
        training_input = training_job['trainingInput']
        if scale_tier:
            training_input['scaleTier'] = scale_tier
        if args:
            training_input['args'] = validators.validate_list(
                args, 'args')
        if runtime_version:
            training_input['runtimeVersion'] = runtime_version
        if python_version:
            training_input['pythonVersion'] = python_version
        if job_dir:
            training_input['jobDir'] = validators.validate_gcs_path(
                job_dir, 'job_dir')    
        if hyperparameters:
            training_input['hyperparameters'] = validators.validate_dict(
                hyperparameters, 'hyperparameters')
        if master_type:
            training_input['masterType'] = master_type
        if worker_type:
            training_input['workerType'] = worker_type
        if parameter_server_type:
            training_input['parameterServerType'] = parameter_server_type
        if worker_count:
            training_input['workerCount'] = worker_count
        if parameter_server_count:
            training_input['parameterServerCount'] = parameter_server_count
        
        CreateJobOp(project_id, training_job, self._wait_interval).execute()

    @decorators.SetParseFns(project_id=str, job_id=str,
        input_data_format=str, output_path=str, region=str,
        model_name=str, model_version_name=str, model_uri=str,
        output_data_format=str, max_worker_count=str, 
        runtime_version=str, batch_size=str, signature_name=str)
    def batch_predict(self,
              project_id,
              job_id,
              input_paths,
              input_data_format,
              output_path,
              region,
              model_name=None,
              model_version_name=None,
              model_uri=None,
              output_data_format=None,
              max_worker_count=None,
              runtime_version=None,
              batch_size=None,
              signature_name=None,
              accelerator=None,
              labels=None):
        """Creates a MLEngine bactch predicion job.

        Args:
            project_id: the ID of the parent project of the job.
            job_id: Required. The user-specified id of the job.
            input_paths: Required. The Google Cloud Storage location of the 
                input data files. May contain wildcards.
            input_data_format: Required. The format of the input data files.
            output_path: Required. The output Google Cloud Storage location.
            region: Required. The Google Compute Engine region to run the 
                prediction job in. 
            model_name: Use this field if you want to use the default version 
                for the specified model. The string must use the following 
                format: "projects/YOUR_PROJECT/models/YOUR_MODEL".
            model_version_name: Use this field if you want to specify a 
                version of the model to use. The string is formatted the 
                same way as model_version, with the addition of the version 
                information: 
                "projects/YOUR_PROJECT/models/YOUR_MODEL/versions/YOUR_VERSION"
            model_uri: Use this field if you want to specify a Google Cloud 
                Storage path for the model to use.
            output_data_format: Optional. Format of the output data files, defaults 
                to JSON.
            max_worker_count: Optional. The maximum number of workers to be used for 
                parallel processing. Defaults to 10 if not specified.
            runtime_version: Optional. The Cloud ML Engine runtime version to use 
                for this batch prediction. If not set, Cloud ML Engine will pick 
                the runtime version used during the versions.create request for 
                this model version, or choose the latest stable version when model 
                version information is not available such as when the model is 
                specified by uri.
            batch_size: Optional. Number of records per batch, defaults to 64. 
                The service will buffer batchSize number of records in memory 
                before invoking one Tensorflow prediction call internally. So 
                take the record size and memory available into consideration 
                when setting this parameter.
            signature_name: Optional. The name of the signature defined in 
                the SavedModel to use for this job.
            accelerator: Optional. The type and number of accelerators to 
                be attached to each machine running the job.
            labels: Optional. One or more labels that you can add, to 
                organize your jobs. Each label is a key-value pair, where 
                both the key and the value are arbitrary strings that you 
                supply.
        """
        validators.validate_required(project_id, 'project_id')
        validators.validate_project_id(project_id)
        validators.validate_required(job_id, 'job_id')
        validators.validate_required(input_paths, 'input_paths')
        validators.validate_type(input_paths, list, 'input_paths')
        validators.validate_required(input_data_format, 'input_data_format')
        validators.validate_required(output_path, 'output_path')
        validators.validate_gcs_path(output_path, 'output_path')
        validators.validate_required(region, 'region')
        prediction_job = {
            'jobId': job_id,
            'predictionInput': {
                'inputPaths': input_paths,
                'dataFormat': input_data_format,
                'outputPath': output_path,
                'region': region,
            }
        }
        if labels:
            prediction_job['labels'] = validators.validate_dict(
                labels, 'labels')
        
        prediction_input = prediction_job['predictionInput']
        if not model_name and not model_version_name and not model_uri:
            raise ValueError('Must provide one of model_name, model_version_name or model_uri.')
        if model_name:
            prediction_input['modelName'] = model_name
        if model_version_name:
            prediction_input['versionName'] = model_version_name
        if model_uri:
            prediction_input['uri'] = model_uri
        if output_data_format:
            prediction_input['outputDataFormat'] = output_data_format
        if max_worker_count:
            prediction_input['maxWorkerCount'] = max_worker_count
        if runtime_version:
            prediction_input['runtimeVersion'] = runtime_version
        if batch_size:
            prediction_input['batchSize'] = batch_size
        if signature_name:
            prediction_input['signatureName'] = signature_name
        if accelerator:
            prediction_input['accelerator'] = validators.validate_dict(
                accelerator, 'accelerator')

        CreateJobOp(project_id, prediction_job, 
            self._wait_interval).execute()

    @decorators.SetParseFns(project_id=str, name=str, description=str,
        online_prediction_logging=bool)
    def create_model(self,
                     project_id,
                     name,
                     description=None,
                     regions=None,
                     online_prediction_logging=None,
                     labels=None):
        """Creates a MLEngine model.

        Args:
            project_id: the ID of the parent project of the job.
            name: Required. The name specified for the model when it was created.
            description: Optional. The description specified for the model 
                when it was created.
            regions: Optional. The list of regions where the model is going 
                to be deployed. Currently only one region per model is 
                supported. Defaults to 'us-central1' if nothing is set. See 
                the available regions for ML Engine services. Note: 
                * No matter where a model is deployed, it can always be 
                accessed by users from anywhere, both for online and batch 
                prediction. 
                * The region for a batch prediction job is set by the region 
                field when submitting the batch prediction job and does not 
                take its value from this field.
            online_prediction_logging: Optional. If true, enables StackDriver 
                Logging for online prediction. Default is false.
            labels: Optional. One or more labels that you can add, to 
                organize your jobs. Each label is a key-value pair, where 
                both the key and the value are arbitrary strings that you 
                supply.
        """
        validators.validate_required(project_id, 'project_id')
        validators.validate_project_id(project_id)
        validators.validate_required(name, 'name')
        model = {
            'name': name
        }
        if description:
            model['description'] = description        
        if regions:
            model['regions'] = validators.validate_list(
                regions, 'regions')
        if type(online_prediction_logging) is bool:
            model['onlinePredictionLogging'] = online_prediction_logging
        if labels:
            model['labels'] = validators.validate_dict(
                labels, 'labels')

        CreateModelOp(project_id, model).execute()

    @decorators.SetParseFns(project_id=str, model_name=str, 
        version_name=str)
    def delete_version(self, 
                       project_id, 
                       model_name, 
                       version_name):
        """Deletes a MLEngine version.

        Args:
            project_id: the ID of the parent project of the job.
            model_name: Required. The name of the parent model.
            version_name: Required. The name of the version to delete.
        """
        validators.validate_required(project_id, 'project_id')
        validators.validate_project_id(project_id)
        validators.validate_required(model_name, 'model_name')
        validators.validate_required(version_name, 'version_name')
        DeleteVersionOp(project_id, model_name, version_name, 
            self._wait_interval).execute()
    
    @decorators.SetParseFns(project_id=str, model_name=str, 
        version_name=str, deployment_uri=str, description=str,
        runtime_version=str, machine_type=str, framework=str,
        python_version=str, replace_existing=bool)
    def create_version(self,
                       project_id,
                       model_name,
                       version_name,
                       deployment_uri,
                       description=None,
                       runtime_version=None,
                       machine_type=None,
                       framework=None,
                       python_version=None,
                       auto_scaling=None,
                       manual_scaling=None,
                       replace_existing=False,
                       labels=None):
        """Creates a MLEngine version.

        Args:
            project_id: the ID of the parent project of the job.
            model_name: Required. The name of the parent model.
            version_name: Required.The name specified for the version 
                when it was created.
            deployment_uri: Required. The Google Cloud Storage location of 
                the trained model used to create the version. When passing 
                Version to projects.models.versions.create the model service 
                uses the specified location as the source of the model. 
                Once deployed, the model version is hosted by the prediction 
                service, so this location is useful only as a historical 
                record. The total number of model files can't exceed 1000.
            description: Optional. The description specified for the version 
                when it was created.
            runtime_version: Optional. The Cloud ML Engine runtime version 
                to use for this deployment. If not set, Cloud ML Engine uses 
                the default stable version, 1.0.
            machine_type: Optional. The type of machine on which to serve the 
                model. Currently only applies to online prediction service. 
                The following are currently supported and will be deprecated 
                in Beta release. mls1-highmem-1 1 core 2 Gb RAM mls1-highcpu-4 
                4 core 2 Gb RAM The following are available in Beta: mls1-c1-m2 
                1 core 2 Gb RAM Default mls1-c4-m2 4 core 2 Gb RAM
            framework: Optional. The machine learning framework Cloud ML Engine 
                uses to train this version of the model. Valid values are 
                TENSORFLOW, SCIKIT_LEARN, XGBOOST. If you do not specify a 
                framework, Cloud ML Engine will analyze files in the 
                deploymentUri to determine a framework. If you choose 
                SCIKIT_LEARN or XGBOOST, you must also set the runtime version 
                of the model to 1.4 or greater.
            python_version: Optional. The version of Python used in prediction. 
                If not set, the default version is '2.7'. Python '3.5' is 
                available when runtimeVersion is set to '1.4' and above. Python 
                '2.7' works with all supported runtime versions.
            auto_scaling: Optional. Sets the options for scaling. If not 
                specified, defaults to auto_scaling with min_nodes of 0. 
                Automatically scale the number of nodes used to serve 
                the model in response to increases and decreases in traffic. 
                Care should be taken to ramp up traffic according to the model's
                ability to scale or you will start seeing increases in latency 
                and 429 response codes.
            manual_scaling: Optional. Sets the options for scaling. If not 
                specified, defaults to auto_scaling with min_nodes of 0. Manually 
                select the number of nodes to use for serving the model. You 
                should generally use autoScaling with an appropriate minNodes 
                instead, but this option is available if you want more predictable 
                billing. Beware that latency and error rates will increase if the 
                traffic exceeds that capability of the system to serve it based 
                on the selected number of nodes.
            replace_existing: Optional. If true, replace existing version with new
                version in case of conflict.
            labels: Optional. One or more labels that you can add, to 
                organize your jobs. Each label is a key-value pair, where 
                both the key and the value are arbitrary strings that you 
                supply.
        """
        validators.validate_required(project_id, 'project_id')
        validators.validate_project_id(project_id)
        validators.validate_required(model_name, 'model_name')
        validators.validate_required(version_name, 'version_name')
        validators.validate_required(deployment_uri, 'deployment_uri')
        validators.validate_gcs_path(deployment_uri, 'deployment_uri')
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
            version['labels'] = validators.validate_dict(
                labels, 'labels')
        if framework:
            version['framework'] = framework
        if python_version:
            version['pythonVersion'] = python_version
        if auto_scaling:
            version['autoScaling'] = validators.validate_dict(
                auto_scaling, 'auto_scaling')
        if manual_scaling:
            version['manualScaling'] = validators.validate_dict(
                manual_scaling, 'manual_scaling')
        
        CreateVersionOp(project_id, model_name, version, 
            replace_existing=replace_existing, 
            wait_interval=self._wait_interval).execute()