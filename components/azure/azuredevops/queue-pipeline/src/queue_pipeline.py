
"""
Copyright (C) Microsoft Corporation. All rights reserved.​
 ​
Microsoft Corporation (“Microsoft”) grants you a nonexclusive, perpetual,
royalty-free right to use, copy, and modify the software code provided by us
("Software Code"). You may not sublicense the Software Code or any use of it
(except to your affiliates and to vendors to perform work on your behalf)
through distribution, network access, service agreement, lease, rental, or
otherwise. This license does not purport to express any claim of ownership over
data you may have shared with Microsoft in the creation of the Software Code.
Unless applicable law gives you more rights, Microsoft reserves all other
rights not expressly granted herein, whether by implication, estoppel or
otherwise. ​
 ​
THE SOFTWARE CODE IS PROVIDED “AS IS”, WITHOUT WARRANTY OF ANY KIND, EXPRESS
OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL
MICROSOFT OR ITS LICENSORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO,
PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR
BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER
IN CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE)
ARISING IN ANY WAY OUT OF THE USE OF THE SOFTWARE CODE, EVEN IF ADVISED OF THE
POSSIBILITY OF SUCH DAMAGE.
"""

import argparse
import os
from azure.devops.connection import Connection
from msrest.authentication import BasicAuthentication


def get_client(organization, personal_access_token):
    organization_url = 'https://dev.azure.com/' + organization

    # Create a connection to the org
    credentials = BasicAuthentication('', personal_access_token)
    connection = Connection(base_url=organization_url,
                            creds=credentials)

    # Get the build client
    build_client = connection.clients_v6_0.get_build_client()
    return build_client


def define_build(id, source_branch, source_version, parameters):
    build = {
        'definition': {
            'id': id
        }
    }

    # Add optional parameters
    if source_branch:
        build["source_branch"] = source_branch
    if source_version:
        build["source_version"] = source_version
    if parameters:
        build["parameters"] = parameters

    return build


def queue_build(client, build, project):
    # The failure responses from Azure Pipelines are pretty good,
    # don't do any special handling.
    queue_build_response = client.queue_build(build, project)
    return queue_build_response


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('-o', '--organization',
                        required=True,
                        help='Azure DevOps organization')
    parser.add_argument('-p', '--project',
                        required=True,
                        help='Azure DevOps project')
    parser.add_argument('-i', '--id',
                        required=True,
                        help='Id of the pipeline definition')
    parser.add_argument('-ppe', '--pat_path_env',
                        help='Name of environment variable containing the path to the Azure DevOps PAT')  # noqa: E501
    parser.add_argument('-pe', '--pat_env',
                        help='Name of environment variable containing the Azure DevOps PAT')  # noqa: E501
    parser.add_argument(
        '--source_branch', help='Source branch for the pipeline')
    parser.add_argument(
        '--source_version', help='Source version for the pipeline')
    parser.add_argument(
        '--parameters', help='Parameters for the pipeline')

    args = parser.parse_args()

    if args.pat_env:
        # Read PAT from env var
        pat = os.environ[args.pat_env]
    elif args.pat_path_env:
        # Read PAT from file
        with open(os.environ[args.pat_path_env], 'r') as f:
            pat = f.readline()
        f.close
    else:
        raise Exception('Please provide a PAT via pat_env or pat_path_env')

    client = get_client(args.organization, pat)
    build = define_build(args.id,
                         args.source_branch,
                         args.source_version,
                         args.parameters)
    results = queue_build(client, build, args.project)

    # Print the url of the queued build
    print(results.url)


if __name__ == "__main__":
    main()
