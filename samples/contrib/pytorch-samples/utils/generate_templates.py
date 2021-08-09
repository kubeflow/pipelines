import os
import yaml
import shutil
import sys
import json

CURRENT_FOLDER = os.path.dirname(os.path.abspath(__file__))

PIPELINES_HOME = os.path.join(CURRENT_FOLDER.split("pipelines")[0], "pipelines")

TEMPLATE_PATH = os.path.join(PIPELINES_HOME, "components/PyTorch/pytorch-kfp-components/templates")

OUTPUT_YAML_FOLDER = "yaml"


def create_output_folder():
    if os.path.exists(OUTPUT_YAML_FOLDER):
        shutil.rmtree(OUTPUT_YAML_FOLDER)

    os.mkdir(OUTPUT_YAML_FOLDER)


def get_templates_list():
    assert os.path.exists(TEMPLATE_PATH)
    templates_list = os.listdir(TEMPLATE_PATH)
    return templates_list


def read_template(template_path: str):
    with open(template_path, 'r') as stream:
        try:
            template_dict = yaml.safe_load(stream)
        except yaml.YAMLError as exc:
            print(exc)

    return template_dict


def replace_keys_in_template(template_dict: dict, mapping: dict):

    # Sample mapping will be as below
    # { "implementation.container.image" : "image_name" }
    for nested_key, value in mapping.items():

        # parse through each nested key

        keys = nested_key.split('.')
        accessable = template_dict
        for k in keys[:-1]:
            accessable = accessable[k]
        accessable[keys[-1]] = value

    return template_dict


def write_to_yaml_file(template_dict: dict, yaml_path: str):
    with open(yaml_path, 'w') as fp:
        yaml.dump(template_dict, fp)


def generate_component_yaml(mapping_template_path: str):

    mapping: dict = {}
    if os.path.exists(mapping_template_path):
        with open(mapping_template_path) as fp:
            mapping = json.load(fp)
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


if __name__ == '__main__':
    if len(sys.argv) != 2:
        raise Exception("\n\nUsage: "
                        "python utils/generate_templates.py cifar10/template_mapping.json\n\n")
    mapping_template_path = sys.argv[1]
    generate_component_yaml(mapping_template_path=mapping_template_path)
