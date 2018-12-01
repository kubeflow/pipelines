#!/usr/bin/python3

import argparse
import subprocess
import re
import os


def is_insecure_path(path):
    # check if the path do not include insecure characters
    if not re.match(r"^gs:\/\/[\.\w\/-]*$", path):
        is_insecure = True
    else:
        is_insecure = False
    return is_insecure


def are_insecure_mo_options(all_options):
    # check if all passed options do not include insecure characters
    is_insecure = False
    for option in all_options.split():
        if not re.match(r"^[\.\w:\/-]*$", option):
            is_insecure = True
    return is_insecure


def main():
    parser = argparse.ArgumentParser(
      description='Model converter to OpenVINO IR format')
    parser.add_argument(
      '--input_path', type=str, help='GCS path of input model file or folder')
    parser.add_argument(
      '--mo_options', type=str, help='OpenVINO Model Optimizer options')
    parser.add_argument(
      '--output_path', type=str, help='GCS path of output folder')
    args = parser.parse_args()

    # Validate parameters

    if is_insecure_path(args.input_path):
        print("Invalid input format")
        exit(1)

    if is_insecure_path(args.output_path):
        print("Invalid output format")
        exit(1)

    if are_insecure_mo_options(args.mo_options):
        print("Invalid model optimizer options")
        exit(1)

    # Initialize gsutil creds if needed
    if "GOOGLE_APPLICATION_CREDENTIALS" in os.environ:
        command = "gcloud auth activate-service-account " \
                         "--key-file=${GOOGLE_APPLICATION_CREDENTIALS}"
        print("auth command", command)
        return_code = subprocess.call(command, shell=True)
        print("return code", return_code)

    # Downloading input model or GCS folder with a model to current folder
    command = "gsutil cp -r " + args.input_path + " ."
    print("gsutil download command", command)
    return_code = subprocess.call(command, shell=True)
    print("return code", return_code)
    if return_code:
        exit(1)

    # Executing model optimization
    command = "mo.py " + args.mo_options
    print("Starting model optimization:", command)
    output = subprocess.run(command, shell=True, stdout=subprocess.PIPE,
                            universal_newlines=True)
    print("Model optimization output",output.stdout)
    XML = ""
    BIN = ""
    for line in output.stdout.splitlines():
        if "[ SUCCESS ] XML file" in line:
            XML = line.split(":")[1].strip()
        if "[ SUCCESS ] BIN file" in line:
            BIN = line.split(":")[1].strip()
    if XML == "" or BIN == "":
        print("Error, model optimization failed")
        exit(1)

    command = "gsutil cp " + XML + " " + os.path.join(args.output_path, os.path.split(XML)[1])
    print("gsutil upload command", command)
    return_code = subprocess.call(command, shell=True)
    print("return code", return_code)
    command = "gsutil cp " + BIN + " " + os.path.join(args.output_path, os.path.split(BIN)[1])
    print("gsutil upload command", command)
    return_code = subprocess.call(command, shell=True)
    print("return code", return_code)
    if return_code:
        exit(1)

    with open('/tmp/output_path.txt', 'w') as f:
        f.write(args.output_path)
    with open('/tmp/bin_path.txt', 'w') as f:
        f.write(os.path.join(args.output_path, os.path.split(BIN)[1]))
    with open('/tmp/xml_path.txt', 'w') as f:
        f.write(os.path.join(args.output_path, os.path.split(XML)[1]))

    print("Model successfully generated and uploaded to ", args.output_path)

if __name__ == "__main__":
    main()
