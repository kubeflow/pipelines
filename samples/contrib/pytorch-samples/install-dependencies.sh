#!/bin/bash

# Install Pip Packages
pip install captum torchvision matplotlib pillow pytorch-lightning flask flask-compress ipywidgets minio

# Install Yarn
npm install

# Install Jupyter Notebook Widgets
jupyter nbextension install --py --symlink --sys-prefix captum.insights.attr_vis.widget

# Enable Jupyter Notebook Extensions
jupyter nbextension enable --py widgetsnbextension
jupyter nbextension enable captum.insights.attr_vis.widget --py --sys-prefix
