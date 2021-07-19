#!/bin/bash -ex
# 
# Copyright (c) Facebook, Inc. and its affiliates.
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.


source_root=$(pwd)
cd "$source_root/components/PyTorch/pytorch-kfp-components"

# Verify package build correctly
python setup.py bdist_wheel clean

# Verify package can be installed and loaded correctly
WHEEL_FILE=$(find "$source_root/components/PyTorch/pytorch-kfp-components/dist/" -name "pytorch_kfp_components*.whl")
pip3 install --upgrade $WHEEL_FILE

python -c "import pytorch_kfp_components"

echo `pwd`

# Run lint and tests
./tests/run_tests.sh
