# Copyright 2020 Google Inc. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

from typing import Any, List, NamedTuple, Optional


def run_diagnose_me(
    bucket: str,
    execution_mode: str,
    project_id: str,
    target_apis: str,
    quota_check: list = None,
) -> NamedTuple('Outputs', [('bucket', str), ('project_id', str)]):
  """ Performs environment verification specific to this pipeline.

      args:
          bucket:
              string name of the bucket to be checked. Must be of the format
              gs://bucket_root/any/path/here/is/ignored where any path beyond root
              is ignored.
          execution_mode:
              If set to HALT_ON_ERROR will case any error to raise an exception.
              This is intended to stop the data processing of a pipeline. Can set
              to False to only report Errors/Warnings.
          project_id:
              GCP project ID which is assumed to be the project under which
              current pod is executing.
          target_apis:
              String consisting of a comma separated list of apis to be verified.
          quota_check:
              List of entries describing how much quota is required. Each entry
              has three fields: region, metric and quota_needed. All
              string-typed.
      Raises:
          RuntimeError: If configuration is not setup properly and
          HALT_ON_ERROR flag is set.
      """

  # Installing pip3 and kfp, since the base image 'google/cloud-sdk:279.0.0'
  # does not come with pip3 pre-installed.
  import subprocess
  subprocess.run([
      'curl', 'https://bootstrap.pypa.io/get-pip.py', '-o', 'get-pip.py'
  ],
                 capture_output=True)
  subprocess.run(['apt-get', 'install', 'python3-distutils', '--yes'],
                 capture_output=True)
  subprocess.run(['python3', 'get-pip.py'], capture_output=True)
  subprocess.run(['python3', '-m', 'pip', 'install', 'kfp>=0.1.31', '--quiet'],
                 capture_output=True)

  import sys
  from kfp.cli.diagnose_me import gcp

  config_error_observed = False

  quota_list = gcp.get_gcp_configuration(
      gcp.Commands.GET_QUOTAS, human_readable=False
  )

  if quota_list.has_error:
    print('Failed to retrieve project quota with error %s\n' % (quota_list.stderr))
    config_error_observed = True
  else:
    # Check quota.
    quota_dict = {}  # Mapping from region to dict[metric, available]
    for region_quota in quota_list.json_output:
      quota_dict[region_quota['name']] = {}
      for quota in region_quota['quotas']:
        quota_dict[region_quota['name']][quota['metric']
                                        ] = quota['limit'] - quota['usage']

    quota_check = [] or quota_check
    for single_check in quota_check:
      if single_check['region'] not in quota_dict:
        print(
            'Regional quota for %s does not exist in current project.\n' %
            (single_check['region'])
        )
        config_error_observed = True
      else:
        if quota_dict[single_check['region']][single_check['metric']
                                          ] < single_check['quota_needed']:
          print(
              'Insufficient quota observed for %s at %s: %s is needed but only %s is available.\n'
              % (
                  single_check['metric'], single_check['region'],
                  str(single_check['quota_needed']
                     ), str(quota_dict[single_check['region']][single_check['metric']])
              )
          )
          config_error_observed = True

  # Get the project ID
  # from project configuration
  project_config = gcp.get_gcp_configuration(
      gcp.Commands.GET_GCLOUD_DEFAULT, human_readable=False
  )
  if not project_config.has_error:
    auth_project_id = project_config.parsed_output['core']['project']
    print(
        'GCP credentials are configured with access to project: %s ...\n' %
        (project_id)
    )
    print('Following account(s) are active under this pipeline:\n')
    subprocess.run(['gcloud', 'auth', 'list', '--format', 'json'])
    print('\n')
  else:
    print(
        'Project configuration is not accessible with error  %s\n' %
        (project_config.stderr),
        file=sys.stderr
    )
    config_error_observed = True

  if auth_project_id != project_id:
    print(
        'User provided project ID %s does not match the configuration %s\n' %
        (project_id, auth_project_id),
        file=sys.stderr
    )
    config_error_observed = True

  # Get project buckets
  get_project_bucket_results = gcp.get_gcp_configuration(
      gcp.Commands.GET_STORAGE_BUCKETS, human_readable=False
  )

  if get_project_bucket_results.has_error:
    print(
        'could not retrieve project buckets with error: %s' %
        (get_project_bucket_results.stderr),
        file=sys.stderr
    )
    config_error_observed = True

  # Get the root of the user provided bucket i.e. gs://root.
  bucket_root = '/'.join(bucket.split('/')[0:3])

  print(
      'Checking to see if the provided GCS bucket\n  %s\nis accessible ...\n' %
      (bucket)
  )

  if bucket_root in get_project_bucket_results.json_output:
    print(
        'Provided bucket \n   %s\nis accessible within the project\n   %s\n' %
        (bucket, project_id)
    )

  else:
    print(
        'Could not find the bucket %s in project %s' % (bucket, project_id) +
        'Please verify that you have provided the correct GCS bucket name.\n' +
        'Only the following buckets are visible in this project:\n%s' %
        (get_project_bucket_results.parsed_output),
        file=sys.stderr
    )
    config_error_observed = True

  # Verify APIs that are required are enabled
  api_config_results = gcp.get_gcp_configuration(gcp.Commands.GET_APIS)

  api_status = {}

  if api_config_results.has_error:
    print(
        'could not retrieve API status with error: %s' %
        (api_config_results.stderr),
        file=sys.stderr
    )
    config_error_observed = True

  print('Checking APIs status ...')
  for item in api_config_results.parsed_output:
    api_status[item['config']['name']] = item['state']
    # printing the results in stdout for logging purposes
    print('%s %s' % (item['config']['name'], item['state']))

  # Check if target apis are enabled
  api_check_results = True
  for api in target_apis.replace(' ', '').split(','):
    if 'ENABLED' != api_status.get(api, 'DISABLED'):
      api_check_results = False
      print(
          'API \"%s\" is not accessible or not enabled. To enable this api go to '
          % (api) +
          'https://console.cloud.google.com/apis/library/%s?project=%s' %
          (api, project_id),
          file=sys.stderr
      )
      config_error_observed = True

  if 'HALT_ON_ERROR' in execution_mode and config_error_observed:
    raise RuntimeError(
        'There was an error in your environment configuration.\n' +
        'Note that resolving such issues generally require a deep knowledge of Kubernetes.\n'
        + '\n' +
        'We highly recommend that you recreate the cluster and check "Allow access ..." \n'
        +
        'checkbox during cluster creation to have the cluster configured automatically.\n'
        +
        'For more information on this and other troubleshooting instructions refer to\n'
        + 'our troubleshooting guide.\n' + '\n' +
        'If you have intentionally modified the cluster configuration, you may\n'
        +
        'bypass this error by removing the execution_mode HALT_ON_ERROR flag.\n'
    )

  return (project_id, bucket)


if __name__ == '__main__':
  import kfp.components as comp

  comp.func_to_container_op(
      run_diagnose_me,
      base_image='google/cloud-sdk:279.0.0',
      output_component_file='component.yaml',
  )
