# A simple NVIDIA-accelerated ResNet Kubeflow pipeline
### This example demonstrates a simple end-to-end training & deployment of a Keras Resnet model on the CIFAR10 dataset utilizing the following NVIDIA technologies:
* [NVIDIA-Docker2](https://github.com/NVIDIA/nvidia-docker) to make the Docker containers GPU aware.
* [NVIDIA device plugin](https://github.com/NVIDIA/k8s-device-plugin) to allow Kubernetes to access GPU nodes.
* [TensorFlow-19.02](https://ngc.nvidia.com/catalog/containers/nvidia:tensorflow) containers from NVIDIA GPU Cloud container registry.
* [TensorRT](https://docs.nvidia.com/deeplearning/dgx/integrate-tf-trt/index.html) for optimizing the Inference Graph in FP16 for leveraging the dedicated use of Tensor Cores for Inference.
* [TensorRT Inference Server](https://github.com/NVIDIA/tensorrt-inference-server) for serving the trained model.

## System Requirements
* Ubuntu 16.04 and above
* NVIDIA GPU

## Quickstart
* Install NVIDIA Docker, Kubernetes and Kubeflow on your local machine:
    * `sudo ./install_kubeflow_and_dependencies.sh`
* Mount persistent volume to Kubeflow:
    * `sudo ./mount_persistent_volume.sh`
* Build the Preprocessing, Training, Serving, and Pipeline containers using the following script:
    * First, modify `build.sh` in `preprocess`, `train`, and `serve` directories to point to a container registry that is accessible to you
    * Then, `sudo ./build_pipeline.sh`
    * Note the `pipeline.py.tar.gz` file that appears on your working directory
* Determine the ambassador port using this command:
    * `sudo kubectl get svc -n kubeflow ambassador`
* Open the Kubeflow Dashboard on:
    * https://local-machine-ip-address:port-determined-from-previous-step
    * E.g. https://10.110.210.99:31342
* Click on the tab Pipeline Dashboard, upload the `pipeline.py.tar.gz` file under you working directory and create a run
* Once the training has completed (should take about 20 minutes for 50 epochs) and the model is being served, port forward the port of the serving pod (8000) to the local system:
    * Determine the name of the serving pod by selecting it on the Kubeflow Dashboard
    * Modify accordingly the variable `SERVING_POD` within `portforward_serving_port.sh`
    * Then, `sudo ./portforward_serving_port.sh`
* Build the client container and start a local server for the demo web UI on the host machine (about 15 mins):
    * `sudo ./test_trtis_client.sh`
* Now you have successfully set up the client through which you can ping the server with an image URL and obtain predictions:
    * Open the demo client UI on a web browser with the following IP address:
https://local-machine-ip-address:8050 
    * The port is specified in `demo_client_ui.py` and can be changed as needed
    * Copy an image URL (for one of the 10 CIFAR classes) and paste it in the UI
