# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

if __name__ == "__main__":
    import argparse
    parser = argparse.ArgumentParser()
    parser.add_argument('--token', type=str, required=True)
    parser.add_argument('--url', type=str, required=True)
    parser.add_argument('--name', type=str)
    parser.add_argument('--output-secret-name-file', type=str)
    args = parser.parse_args()

    access_token = args.token
    config_file_path = args.url

    # download config file
    # the default creds.ini is in the public accesible github repo
    import subprocess
    import os
    config_file = os.path.basename(config_file_path)
    config_local_path = os.path.join('/tmp', config_file)
    command = ['curl', '-H', 'Authorization: token %s' % access_token, '-L', '-o', config_local_path, config_file_path]
    subprocess.run(command, check=True)

    secret_name = args.name
    if (not secret_name):
        secret_name = 'ai-pipeline-' + os.path.splitext(config_file)[0]

    try:
        command = ['kubectl', 'delete', 'secret', secret_name]
        subprocess.run(command, check=True)
    except Exception as e:
        print('No previous secret: ' + secret_name + '. Secret deletion is not performed.')

    # gather all secrets
    command = ['kubectl', 'create', 'secret', 'generic', secret_name]

    import configparser
    config = configparser.ConfigParser()
    config.read(config_local_path)
    for section in config.sections():
        for key in config[section]:
            command.append('--from-literal=%s=\'%s\'' % (key, config[section][key]))

    # create the secret
    subprocess.run(command, check=True)

    # verify secret is created
    subprocess.run(['kubectl', 'describe', 'secret', secret_name], check=True)

    # indicate that secret is created and pass the secret name forward
    from pathlib import Path
    Path(args.output_secret_name_file).parent.mkdir(parents=True, exist_ok=True)
    Path(args.output_secret_name_file).write_text(secret_name)
