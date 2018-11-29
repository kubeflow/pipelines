import argparse
import subprocess
import re
from shutil import copyfile


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

    bin_path = "../tools/google-cloud-sdk/bin/"
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

    # Initialize gsutil creds
    command = bin_path + "gcloud auth activate-service-account " \
                         "--key-file=${GOOGLE_APPLICATION_CREDENTIALS}"
    print("auth command", command)
    return_code = subprocess.call(command, shell=True)
    print("return code", return_code)

    # Downloading input model
    command = bin_path + "gsutil cp -r " + args.input_path + " ."
    print("gsutil download command", command)
    return_code = subprocess.call(command, shell=True)
    print("return code", return_code)

    # Executing model optimization
    command = "python3 ../mo.py " + args.mo_options
    print("mo command", command)
    output = subprocess.run(command, shell=True, stdout=subprocess.PIPE,
                            universal_newlines=True)
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

    # copy generated model file to use them as workflow artifacts
    copyfile(BIN, "/tmp/model.bin")
    copyfile(XML, "/tmp/model.xml")

    command = bin_path + "gsutil cp " + XML + " " + args.output_path
    print("gsutil upload command", command)
    return_code = subprocess.call(command, shell=True)
    print("return code", return_code)
    command = bin_path + "gsutil cp " + BIN + " " + args.output_path
    print("gsutil upload command", command)
    return_code = subprocess.call(command, shell=True)
    print("return code", return_code)


if __name__ == "__main__":
    main()
