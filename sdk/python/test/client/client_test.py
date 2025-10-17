# Copyright 2022 The Kubeflow Authors
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
import os
import time

import kfp
from kfp_server_api.models.v2beta1_experiment import V2beta1Experiment
from kfp_server_api.models.v2beta1_experiment_storage_state import \
    V2beta1ExperimentStorageState
from kfp_server_api.models.v2beta1_pipeline_version import \
    V2beta1PipelineVersion
import pytest

from backend.api.v2beta1.python_http_client.kfp_server_api.models.v2beta1_pipeline import \
    V2beta1Pipeline
from test_data.sdk_compiled_pipelines.valid.sequential_v2 import sequential

from ..test_utils.file_utils import FileUtils

KFP_URL = os.getenv("apiUrl", "http://localhost:8888")
NAMESPACE: str = os.getenv("namespace", "kubeflow")
VERIFY_SSL: bool = bool(os.getenv("verifySSL", "False"))
AUTH_TOKEN: str = os.getenv("authToken", None)
SSL_CA_CERT: str = os.getenv("sslCertPath", None)


@pytest.mark.client
class TestClient:

    hello_world_pipeline_file = f'{FileUtils.VALID_PIPELINE_FILES}/hello-world.yaml'
    kfp_client: kfp.Client

    @pytest.fixture(autouse=True)
    def setup_and_teardown(self):
        self.created_pipelines: list[str] = list()
        self.created_experiments: list[str] = list()
        self.created_runs: list[str] = list()
        self.created_recurring_runs: list[str] = list()
        self.kfp_client = kfp.Client(
            host=KFP_URL,
            namespace=NAMESPACE,
            verify_ssl=VERIFY_SSL,
            existing_token=AUTH_TOKEN,
            ssl_ca_cert=SSL_CA_CERT)
        yield
        for run_id in self.created_runs:
            print(f'Deleting run with id={run_id}')
            self.kfp_client.terminate_run(run_id)
            self.kfp_client.archive_run(run_id)
            self.kfp_client.delete_run(run_id)
        for recurring_run_id in self.created_recurring_runs:
            print(f'Deleting recurring run with id={recurring_run_id}')
            self.kfp_client.disable_recurring_run(recurring_run_id)
            self.kfp_client.delete_recurring_run(recurring_run_id)
        for experiment_id in self.created_experiments:
            print(f'Deleting experiment with id={experiment_id}')
            self.kfp_client.archive_experiment(experiment_id)
            self.kfp_client.delete_experiment(experiment_id)
        for pipeline_id in self.created_pipelines:
            print(f'Deleting pipeline with id={pipeline_id}')
            pipeline = self.kfp_client.get_pipeline(pipeline_id=pipeline_id)
            print(pipeline)
            if pipeline is not None:
                pipeline_versions: list[
                    V2beta1PipelineVersion] = self.kfp_client.list_pipeline_versions(
                        pipeline_id=pipeline_id,
                        page_size=50,
                    ).pipeline_versions
                for pipeline_version in pipeline_versions:
                    self.kfp_client.delete_pipeline_version(
                        pipeline_id=pipeline_id,
                        pipeline_version_id=pipeline_version.pipeline_version_id
                    )
                self.kfp_client.delete_pipeline(pipeline_id)

    def test_upload_pipeline(self):
        pipeline_name = f"hello-world-{self.get_current_time()}"
        pipeline_desc = "Test Hello World Pipeline"
        pipeline = self.kfp_client.upload_pipeline(
            pipeline_package_path=self.hello_world_pipeline_file,
            pipeline_name=pipeline_name,
            description=pipeline_desc)
        self.created_pipelines.append(pipeline.pipeline_id)
        assert pipeline.pipeline_id is not None, "Failed to upload pipeline"
        assert pipeline.name == pipeline_name, "Created Pipeline does not have the expected name"
        assert pipeline.description == pipeline_desc, "Description not same"

    def test_get_pipeline(self):
        pipeline_name = f"hello-world-{self.get_current_time()}"
        pipeline_desc = "Test Hello World Pipeline"
        created_pipeline = self.kfp_client.upload_pipeline(
            pipeline_package_path=self.hello_world_pipeline_file,
            pipeline_name=pipeline_name,
            description=pipeline_desc)
        self.created_pipelines.append(created_pipeline.pipeline_id)
        pipeline = self.kfp_client.get_pipeline(created_pipeline.pipeline_id)
        assert pipeline.pipeline_id == created_pipeline.pipeline_id, "Created pipeline not found in the DB"

    def test_list_pipelines(self):
        pipeline_name = f"hello-world-{self.get_current_time()}"
        pipeline_desc = "Test Hello World Pipeline"
        pipeline = self.kfp_client.upload_pipeline(
            pipeline_package_path=self.hello_world_pipeline_file,
            pipeline_name=pipeline_name,
            description=pipeline_desc)
        self.created_pipelines.append(pipeline.pipeline_id)
        pipelines: list[V2beta1Pipeline] = self.kfp_client.list_pipelines(
            page_size=50, sort_by='created_at desc').pipelines
        pipeline_exist = False
        for pipe in pipelines:
            if pipe.pipeline_id == pipeline.pipeline_id:
                pipeline_exist = True
                break
        assert pipeline_exist, "Created pipeline not found in the DB"

    def test_list_pipeline_versions(self):
        pipeline_name = f"hello-world-{self.get_current_time()}"
        pipeline = self.kfp_client.upload_pipeline(
            pipeline_package_path=self.hello_world_pipeline_file,
            pipeline_name=pipeline_name,
            description="Test Hello World Pipeline")
        self.created_pipelines.append(pipeline.pipeline_id)
        pipeline_versions = self.kfp_client.list_pipeline_versions(
            pipeline.pipeline_id,
            page_size=50,
            sort_by='created_at desc',
        ).pipeline_versions
        assert len(pipeline_versions
                  ) > 0, "No pipeline versions available after pipeline upload"

    def test_create_experiment(self):
        experiment_name = f"TestExperiment-{self.get_current_time()}"
        experiment_desc = "Python Client Tests Experiment"
        experiment = self.kfp_client.create_experiment(
            name=experiment_name, description=experiment_desc)
        self.created_experiments.append(experiment.experiment_id)
        assert experiment.experiment_id is not None, "Failed to create experiment"
        assert experiment.description == experiment_desc, "Description not same"
        assert experiment.display_name == experiment_name, "Name not same"

    def test_get_experiment(self):
        experiment_name = f"TestExperiment-{self.get_current_time()}"
        experiment_desc = "Python Client Tests Experiment"
        experiment = self.kfp_client.create_experiment(
            name=experiment_name, description=experiment_desc)
        self.created_experiments.append(experiment.experiment_id)
        experiment_list: list[
            V2beta1Experiment] = self.kfp_client.list_experiments(
                page_size=50).experiments
        experiment_found = False
        for exp in experiment_list:
            if exp.experiment_id == experiment.experiment_id:
                experiment_found = True
                break
        assert experiment_found, "Created experiment not found in the list of experiment"

    def test_archive_unarchive_experiment(self):
        experiment_name = f"TestExperiment-{self.get_current_time()}"
        experiment_desc = "Python Client Tests Experiment"
        experiment = self.kfp_client.create_experiment(
            name=experiment_name, description=experiment_desc)
        self.created_experiments.append(experiment.experiment_id)
        self.kfp_client.archive_experiment(experiment.experiment_id)
        archived_experiment = self.kfp_client.get_experiment(
            experiment_id=experiment.experiment_id)
        assert archived_experiment.storage_state == V2beta1ExperimentStorageState.ARCHIVED, "Experiment not in archived state"

        self.kfp_client.unarchive_experiment(experiment.experiment_id)
        archived_experiment = self.kfp_client.get_experiment(
            experiment_id=experiment.experiment_id)
        assert archived_experiment.storage_state == V2beta1ExperimentStorageState.AVAILABLE, "Experiment not Unarchived, its still in archived state"

    def test_create_run(self):
        # Upload Pipeline
        pipeline_name = f"hello-world-{self.get_current_time()}"
        pipeline_desc = "Test Hello World Pipeline"
        pipeline = self.kfp_client.upload_pipeline(
            pipeline_package_path=self.hello_world_pipeline_file,
            pipeline_name=pipeline_name,
            description=pipeline_desc)
        self.created_pipelines.append(pipeline.pipeline_id)
        pipeline_version = self.kfp_client.list_pipeline_versions(
            pipeline.pipeline_id).pipeline_versions[0]

        # Create Experiment
        experiment_name = f"TestExperiment-{self.get_current_time()}"
        experiment_desc = "Python Client Tests Experiment"
        experiment = self.kfp_client.create_experiment(
            name=experiment_name, description=experiment_desc)
        self.created_experiments.append(experiment.experiment_id)

        # Create Run
        run_name = f"TestRun-{self.get_current_time()}"
        run = self.kfp_client.run_pipeline(
            experiment_id=experiment.experiment_id,
            pipeline_id=pipeline.pipeline_id,
            version_id=pipeline_version.pipeline_version_id,
            job_name=run_name)
        self.created_runs.append(run.run_id)
        assert run.run_id is not None, "Run not created"
        assert run.display_name == run_name, "Run Name not same"

    def test_create_run_from_pipeline_package(self):
        # Create Experiment
        experiment_name = f"TestExperiment-{self.get_current_time()}"
        experiment_desc = "Python Client Tests Experiment"
        experiment = self.kfp_client.create_experiment(
            name=experiment_name, description=experiment_desc)
        self.created_experiments.append(experiment.experiment_id)

        # Create Run
        run_name = f"TestRun-{self.get_current_time()}"
        run = self.kfp_client.create_run_from_pipeline_package(
            pipeline_file=self.hello_world_pipeline_file,
            experiment_id=experiment.experiment_id,
            run_name=run_name)
        self.created_runs.append(run.run_id)
        assert run.run_id is not None, "Run not created"
        assert run.run_info.display_name == run_name, "Run Name not same"

    def test_get_run(self):
        # Create Experiment
        experiment_name = f"TestExperiment-{self.get_current_time()}"
        experiment_desc = "Python Client Tests Experiment"
        experiment = self.kfp_client.create_experiment(
            name=experiment_name, description=experiment_desc)
        self.created_experiments.append(experiment.experiment_id)

        # Create Run
        run_name = f"TestRun-{self.get_current_time()}"
        created_run = self.kfp_client.create_run_from_pipeline_package(
            pipeline_file=self.hello_world_pipeline_file,
            experiment_id=experiment.experiment_id,
            run_name=run_name)
        self.created_runs.append(created_run.run_id)

        # Get Run
        run = self.kfp_client.get_run(created_run.run_id)
        assert run.display_name == created_run.run_info.display_name, "Run name not same in the DB"

    def test_list_runs(self):
        # Create Experiment
        experiment_name = f"TestExperiment-"
        experiment_desc = "Python Client Tests Experiment"
        experiment = self.kfp_client.create_experiment(
            name=experiment_name, description=experiment_desc)
        self.created_experiments.append(experiment.experiment_id)

        # Create Run
        run_name = f"TestRun-{self.get_current_time()}"
        created_run = self.kfp_client.create_run_from_pipeline_package(
            pipeline_file=self.hello_world_pipeline_file,
            experiment_id=experiment.experiment_id,
            run_name=run_name)
        self.created_runs.append(created_run.run_id)

        # List Runs
        runs = self.kfp_client.list_runs(
            page_size=50, experiment_id=experiment.experiment_id).runs
        run_created = False
        for run in runs:
            if run.run_id == created_run.run_id:
                run_created = True
                break
        assert run_created, "Run not found in the DB"

    def test_create_run_from_pipeline_func(self):
        # Create Experiment
        experiment_name = f"TestExperiment-{self.get_current_time()}"
        experiment_desc = "Python Client Tests Experiment"
        experiment = self.kfp_client.create_experiment(
            name=experiment_name, description=experiment_desc)
        self.created_experiments.append(experiment.experiment_id)

        # Create Run
        run_name = f"TestRun-{self.get_current_time()}"
        run = self.kfp_client.create_run_from_pipeline_func(
            pipeline_func=sequential,
            arguments={'url': 'gs://sample-data/test.txt'},
            experiment_id=experiment.experiment_id,
            run_name=run_name)
        self.created_runs.append(run.run_id)
        assert run.run_id is not None, "Run not created"
        assert run.run_info.display_name == run_name, "Run Name not same"

    def test_create_scheduled_run(self):
        # Create Experiment
        experiment_name = f"TestExperiment-{self.get_current_time()}"
        experiment_desc = "Python Client Tests Experiment"
        experiment = self.kfp_client.create_experiment(
            name=experiment_name, description=experiment_desc)
        self.created_experiments.append(experiment.experiment_id)

        # Create Recurring Run
        run_name = f"TestRecurringRun-{self.get_current_time()}"
        recurring_run = self.kfp_client.create_recurring_run(
            experiment_id=experiment.experiment_id,
            pipeline_package_path=self.hello_world_pipeline_file,
            job_name=run_name,
            interval_second=300)
        self.created_recurring_runs.append(recurring_run.recurring_run_id)
        assert recurring_run.recurring_run_id is not None, "Run not created"
        assert recurring_run.display_name == run_name, "Run Name not same"

    def test_get_scheduled_run(self):
        # Create Experiment
        experiment_name = f"TestExperiment-{self.get_current_time()}"
        experiment_desc = "Python Client Tests Experiment"
        experiment = self.kfp_client.create_experiment(
            name=experiment_name, description=experiment_desc)
        self.created_experiments.append(experiment.experiment_id)

        # Create Recurring Run
        run_name = f"TestRecurringRun-{self.get_current_time()}"
        recurring_run = self.kfp_client.create_recurring_run(
            experiment_id=experiment.experiment_id,
            pipeline_package_path=self.hello_world_pipeline_file,
            job_name=run_name,
            interval_second=300)
        self.created_recurring_runs.append(recurring_run.recurring_run_id)

        # Get Recurring Run
        created_recurring_run = self.kfp_client.get_recurring_run(
            recurring_run.recurring_run_id)
        assert created_recurring_run.display_name == recurring_run.display_name, "Run name not same in the DB"

    def get_current_time(self) -> str:
        return str(int(time.time() * 10000))
