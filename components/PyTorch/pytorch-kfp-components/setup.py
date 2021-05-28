from setuptools import setup, find_packages

with open("README.md", "r", encoding="utf-8") as fh:
    long_description = fh.read()

setup(
    name="pytorch-kfp-components",
    version="1.2.8",
    description="PyTorch Kubeflow Pipeline",
    long_description=long_description,
    long_description_content_type="text/markdown",
    packages=find_packages(),
    install_requires=[
        "torch",
        "kfp",
        "torchserve",
        "pytorch-lightning"
        ],
    #entry_points = {"kubeflow.pipeline":"component:pytorch_kfp_components"}
   	)
