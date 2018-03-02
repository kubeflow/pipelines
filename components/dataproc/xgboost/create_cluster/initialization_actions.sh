#!/bin/bash
#
# Initialization actions to run in dataproc setup.
# The script will be run on each node in a dataproc cluster.
 
easy_install pip
pip install tensorflow==1.4.1
pip install pandas==0.18.1
