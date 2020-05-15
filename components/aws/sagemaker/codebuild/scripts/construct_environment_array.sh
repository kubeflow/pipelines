#!/usr/bin/env bash

# This script breaks up a string of environment variable names into a list of
# parameters that `docker run` accepts. This needs to be made into a script
# for CodeBuild because these commands do not run in dash - the default terminal
# on the CodeBuild standard images.

IFS=' ' read -a variable_array <<< $CONTAINER_VARIABLES
printf -v CONTAINER_VARIABLE_ARGUMENTS -- "--env %s " "${variable_array[@]}"
echo $CONTAINER_VARIABLE_ARGUMENTS