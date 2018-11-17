from inference_engine import IENetwork, IEPlugin
import argparse
import numpy as np
from urllib.parse import urlparse
from google.cloud import storage
import datetime
from shutil import copy
import os


def get_local_file(source_path):
    parsed_path = urlparse(source_path)
    if parsed_path.scheme == "gs":
        bucket_name = parsed_path.netloc
        file_path = parsed_path.path[1:]
        file_name = os.path.split(parsed_path.path)[1]
        try:
            gs_client = storage.Client()
            bucket = gs_client.get_bucket(bucket_name)
            blob = bucket.blob(file_path)
            blob.download_to_filename(file_name)
        except Exception as er:
            print(er)
            return ""
    elif parsed_path.scheme == "":
        # in case of local path just pass the input argument
        if os.path.isfile(source_path):
            file_name = source_path
        else:
            print("file " + source_path + "is not accessible")
            file_name = ""
    return file_name


def upload_file(source_file, target_folder):
    parsed_path = urlparse(target_folder)
    if parsed_path.scheme == "gs":
        bucket_name = parsed_path.netloc
        print("bucket_name", bucket_name)
        folder_path = parsed_path.path[1:]
        print("folder path", folder_path)
        try:
            gs_client = storage.Client()
            bucket = gs_client.get_bucket(bucket_name)
            print(folder_path + "/" + source_file)
            blob = bucket.blob(folder_path + "/" + source_file)
            blob.upload_from_filename(source_file)
        except Exception as er:
            print(er)
            return False
    elif parsed_path.scheme == "":
        if target_folder != ".":
            copy(source_file, target_folder)
    return True


def main():
    parser = argparse.ArgumentParser(
        description='Component executing inference operation')
    parser.add_argument('--model_bin', type=str,
                        help='GCS or local path to model weights file (.bin)')
    parser.add_argument('--model_xml', type=str,
                        help='GCS or local path to model graph (.xml)')
    parser.add_argument('--input_numpy_file', type=str,
                        help='GCS or local path to input dataset numpy file')
    parser.add_argument('--output_folder', type=str,
                        help='GCS or local path to results upload folder')
    args = parser.parse_args()

    device = "CPU"
    plugin_dir = None

    model_xml = get_local_file(args.model_xml)
    print("model xml", model_xml)
    if model_xml == "":
        exit(1)
    model_bin = get_local_file(args.model_bin)
    print("model bin", model_bin)
    if model_bin == "":
        exit(1)
    input_numpy_file = get_local_file(args.input_numpy_file)
    print("input_numpy_file", input_numpy_file)
    if input_numpy_file == "":
        exit(1)

    cpu_extension = "/usr/local/lib/libcpu_extension.so"

    plugin = IEPlugin(device=device, plugin_dirs=plugin_dir)
    if cpu_extension and 'CPU' in device:
        plugin.add_cpu_extension(cpu_extension)

    print("inference engine:", model_xml, model_bin, device)

    # Read IR
    print("Reading IR...")
    net = IENetwork.from_ir(model=model_xml, weights=model_bin)

    input_blob = next(iter(net.inputs))
    output_blob = next(iter(net.outputs))
    print(output_blob)

    print("Loading IR to the plugin...")
    exec_net = plugin.load(network=net, num_requests=1)
    n, c, h, w = net.inputs[input_blob]
    imgs = np.load(input_numpy_file, mmap_mode='r', allow_pickle=False)
    imgs = imgs.transpose((0, 3, 1, 2))
    print("loaded data", imgs.shape)

    # batch size is taken from the size of input array first dimension
    batch_size = net.inputs[input_blob][0]

    combined_results = {}  # dictionary storing results for all model outputs

    for x in range(0, imgs.shape[0] - batch_size, batch_size):
        img = imgs[x:(x + batch_size)]
        start_time = datetime.datetime.now()
        results = exec_net.infer(inputs={input_blob: img})
        end_time = datetime.datetime.now()
        duration = (end_time - start_time).total_seconds() * 1000
        print("inference duration:", duration, "ms")
        for output in results.keys():
            if output in combined_results:
                combined_results[output] = np.append(combined_results[output],
                                                     results[output], 0)
            else:
                combined_results[output] = results[output]

    for output in results.keys():
        filename = output.replace("/", "_") + ".npy"
        np.save(filename, combined_results[output])
        status = upload_file(filename, args.output_folder)
        print("upload status", status)


if __name__ == "__main__":
    main()
