import argparse
import os
import json
from pathlib import Path
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
    parser.add_argument(
        '--output_url_path', help='Url of the queued pipeline')


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

    # Write Output
    print("Creating output directory")
    output_url_path = args.output_url_path
    Path(output_url_path).parent.mkdir(parents=True, exist_ok=True)

    with open(output_url_path, 'w') as f:
        json.dump(results.url, f)


if __name__ == "__main__":
    main()
