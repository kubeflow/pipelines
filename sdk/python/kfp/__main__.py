import fire
from ._client import Client
import sys
import subprocess
import pprint
import time
import json

from tabulate import tabulate

class KFPWrapper(object):
    def __init__(self, host = None, client_id = None, namespace = 'kubeflow'):
        self._client = Client(host, client_id, namespace)
        self._namespace = namespace

    def run(self, experiment_name, run_name = None, pipeline_package_path = None, pipeline_id = None, watch = True, args = {}):
        if not run_name:
            run_name = experiment_name

        if not pipeline_package_path and not pipeline_id:
            print('You must provide one of [pipeline_package_path, pipeline_id].')
            sys.exit(1)

        print(args)
        experiment = self._client.create_experiment(experiment_name)
        run = self._client.run_pipeline(experiment.id, run_name, pipeline_package_path, args, pipeline_id)
        print('Run {} is submitted'.format(run.id))
        self.get_run(run.id, watch)

    def get_run(self, run_id, watch=True):
        run = self._client.get_run(run_id).run
        self._print_runs([run])
        if not watch:
            return
        argo_workflow_name = None
        while True:
            time.sleep(1)
            run_detail = self._client.get_run(run_id)
            run = run_detail.run
            if run_detail.pipeline_runtime and run_detail.pipeline_runtime.workflow_manifest:
                manifest = json.loads(run_detail.pipeline_runtime.workflow_manifest)
                if manifest['metadata'] and manifest['metadata']['name']:
                    argo_workflow_name = manifest['metadata']['name']
                    break
            if run_detail.run.status in ['Succeeded', 'Skipped', 'Failed', 'Error']:
                print('Run is finished with status {}.'.format(run_detail.run.status))
                return
        if argo_workflow_name:
            subprocess.run(['argo', 'watch', argo_workflow_name, '-n', self._namespace])
            self._print_runs([run])

    def list_runs(self, experiment_id = None, page_size=100):
        response = self._client.list_runs(experiment_id=experiment_id, page_size=page_size, sort_by='created_at desc')
        self._print_runs(response.runs)

    def _print_runs(self, runs):
        headers = ['run id', 'name', 'status', 'created at']
        data = [[run.id, run.name, run.status, run.created_at.isoformat()] for run in runs]
        print(tabulate(data, headers=headers, tablefmt='grid'))


def main():
    fire.Fire(KFPWrapper, name='kfp')

if __name__ == '__main__':
    main()