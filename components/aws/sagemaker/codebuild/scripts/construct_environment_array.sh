#!/usr/bin/env bash
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# This script breaks up a string of environment variable names into a list of
# parameters that `docker run` accepts. This needs to be made into a script
# for CodeBuild because these commands do not run in dash - the default terminal
# on the CodeBuild standard images.

IFS=' ' read -a variable_array <<< $CONTAINER_VARIABLES
printf -v CONTAINER_VARIABLE_ARGUMENTS -- "--env %s " "${variable_array[@]}"
echo $CONTAINER_VARIABLE_ARGUMENTS