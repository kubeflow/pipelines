import argparse
import custom_job

def _make_parent_dirs_and_return_path(file_path: str):
    import os
    os.makedirs(os.path.dirname(file_path), exist_ok=True)
    return file_path

_parser = argparse.ArgumentParser(prog='Vertex Pipelines service launcher', description='')
_parser.add_argument("--type", dest="type", type=str, required=True, default=argparse.SUPPRESS)
_parser.add_argument("--gcp_project", dest="gcp_project", type=str, required=True, default=argparse.SUPPRESS)
_parser.add_argument("--gcp_region", dest="gcp_region", type=str, required=True, default=argparse.SUPPRESS)
_parser.add_argument("--payload", dest="payload", type=str, required=True, default=argparse.SUPPRESS)
_parser.add_argument("--gcp_resources", dest="gcp_resources", type=_make_parent_dirs_and_return_path, required=True, default=argparse.SUPPRESS)
_parsed_args = vars(_parser.parse_args())

if _parsed_args['type']=='CustomJob':
    _outputs = custom_job.create_custom_job(**_parsed_args)
