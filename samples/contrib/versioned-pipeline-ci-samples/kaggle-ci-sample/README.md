# Kaggle Competition Pipeline Sample

## Pipeline Overview

This is a pipeline for [house price prediction](https://www.kaggle.com/c/house-prices-advanced-regression-techniques), an entry-level competition in kaggle. We demonstrate how to complete a kaggle competition by creating a pipeline of steps including downloading data, preprocessing and visualize data, train model and submitting results to kaggle website. 

* We refer to [this notebook](https://www.kaggle.com/rajgupta5/house-price-prediction) and [this notebook](https://www.kaggle.com/neviadomski/how-to-get-to-top-25-with-simple-model-sklearn) in terms of model implementation as well as data visualization.

* We use [kaggle python api](https://github.com/Kaggle/kaggle-api) to interact with kaggle site, such as downloading data and submiting result. More usage can be found in their documentation.

* We use [cloud build](https://cloud.google.com/cloud-build/) for CI process. That is, we automatically triggered a build and run as soon as we pushed our code to github repo. You need to setup a trigger on cloud build for your github repo branch to achieve the CI process.

## Usage

* Substitute the constants in cloudbuild.yaml
* Fill in your kaggle_username and kaggle_key in Dockerfiles to authenticate to kaggle. You can get them from an API token created from your kaggle account page.
* Set up cloud build triggers for Continuous Integration
* Change the images in pipeline.py to the ones you built in cloudbuild.yaml 