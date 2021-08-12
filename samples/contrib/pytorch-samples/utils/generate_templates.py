#!/usr/bin/env/python3
# Copyright (c) Facebook, Inc. and its affiliates.
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Generate component.yaml from templates"""
import json
import os
import shutil
import sys

import yaml

CURRENT_FOLDER = os.path.dirname(os.path.abspath(__file__))

PIPELINES_HOME = os.path.join(CURRENT_FOLDER.split("pipelines")[0], "pipelines")

TEMPLATE_PATH = os.path.join(
    PIPELINES_HOME, "components/PyTorch/pytorch-kfp-components/templates"
)

OUTPUT_YAML_FOLDER = "yaml"


def create_output_folder():
    """Removes the `yaml` folder and recreates it"""
    if os.path.exists(OUTPUT_YAML_FOLDER):
        shutil.rmtree(OUTPUT_YAML_FOLDER)

    os.mkdir(OUTPUT_YAML_FOLDER)


def get_templates_list():
    """Get the list of template files from `templates` directory"""
    assert os.path.exists(TEMPLATE_PATH)
    templates_list = os.listdir(TEMPLATE_PATH)
    return templates_list


def read_template(template_path: str):
    """Read the `componanent.yaml` template"""
    with open(template_path, "r") as stream:
        try:
            template_dict = yaml.safe_load(stream)
        except yaml.YAMLError as exc:
            print(exc)

    return template_dict


def replace_keys_in_template(template_dict: dict, mapping: dict):
    """Replace the keys, values in `component.yaml` based on `mapping` dict"""

    # Sample mapping will be as below
    # { "implementation.container.image" : "image_name" }
    for nested_key, value in mapping.items():

        # parse through each nested key

        keys = nested_key.split(".")
        accessable = template_dict
        for k in keys[:-1]:
            accessable = accessable[k]
        accessable[keys[-1]] = value

    return template_dict


def write_to_yaml_file(template_dict: dict, yaml_path: str):
    """Write yaml output into file"""
    with open(yaml_path, "w") as pointer:
        yaml.dump(template_dict, pointer)


def generate_component_yaml(mapping_template_path: str):
    """Method to generate component.yaml based on the template"""
    mapping: dict = {}
    if os.path.exists(mapping_template_path):
        with open(mapping_template_path) as pointer:
            mapping = json.load(pointer)
    create_output_folder()
    template_list = get_templates_list()

    for template_name in template_list:
        print("Processing {}".format(template_name))

        # if the template name is not present in the mapping dictionary
        # There is no change in the template, we can simply copy the template
        # into the output yaml folder.
        src = os.path.join(TEMPLATE_PATH, template_name)
        dest = os.path.join(OUTPUT_YAML_FOLDER, template_name)
        if not mapping or template_name not in mapping:
            shutil.copy(src, dest)
        else:
            # if the mapping is specified, replace the key value pairs
            # and then save the file
            template_dict = read_template(template_path=src)
            template_dict = replace_keys_in_template(
                template_dict=template_dict, mapping=mapping[template_name]
            )
            write_to_yaml_file(template_dict=template_dict, yaml_path=dest)


if __name__ == "__main__":
    if len(sys.argv) != 2:
        raise Exception(
            "\n\nUsage: "
            "python utils/generate_templates.py "
            "cifar10/template_mapping.json\n\n"
        )
    input_template_path = sys.argv[1]
    generate_component_yaml(mapping_template_path=input_template_path)
