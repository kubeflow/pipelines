#!/bin/bash

# Install Yarn
conda install -c conda-forge yarn -y

# Install Pip Packages
pip install captum torchvision matplotlib pillow pytorch-lightning ipywidgets minio 

pip install Werkzeug==2.0.0 flask flask-compress

# Install Jupyter Notebook Widgets
jupyter nbextension install --py --symlink --sys-prefix captum.insights.attr_vis.widget

# Enable Jupyter Notebook Extensions
jupyter nbextension enable --py widgetsnbextension
jupyter nbextension enable captum.insights.attr_vis.widget --py --sys-prefix
