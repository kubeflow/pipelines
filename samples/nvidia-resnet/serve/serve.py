# Copyright (c) 2019, NVIDIA CORPORATION. All rights reserved.
#
# Redistribution and use in source and binary forms, with or without
# modification, are permitted provided that the following conditions
# are met:
#  * Redistributions of source code must retain the above copyright
#    notice, this list of conditions and the following disclaimer.
#  * Redistributions in binary form must reproduce the above copyright
#    notice, this list of conditions and the following disclaimer in the
#    documentation and/or other materials provided with the distribution.
#  * Neither the name of NVIDIA CORPORATION nor the names of its
#    contributors may be used to endorse or promote products derived
#    from this software without specific prior written permission.
#
# THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS ``AS IS'' AND ANY
# EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
# IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR
# PURPOSE ARE DISCLAIMED.  IN NO EVENT SHALL THE COPYRIGHT OWNER OR
# CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL,
# EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO,
# PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR
# PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY
# OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
# (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
# OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.


import os
import shutil
import argparse


parser = argparse.ArgumentParser(
    description='Provide input directory')
parser.add_argument('--input_dir',
                    help='provide input directory')
args = parser.parse_args()


TRTIS_RESOURCE_DIR = 'trtis_resource'

# Modify according to your training script and config.pbtxt
GRAPHDEF_FILE = 'trt_graphdef.pb'
MODEL_NAME = 'resnet_graphdef'

# Check for resource dir
resource_dir = os.path.join(args.input_dir, TRTIS_RESOURCE_DIR)
if not os.path.isdir(resource_dir):
    raise IOError('Resource dir for TRTIS not found')

# Create a directory structure that TRTIS expects
config_file_path = os.path.join(resource_dir, 'config.pbtxt')
label_file_path = os.path.join(resource_dir, 'labels.txt')
graphdef_path = os.path.join(args.input_dir, GRAPHDEF_FILE)
model_dir = '/models/%s/1' % MODEL_NAME

os.makedirs(model_dir)
shutil.copy(config_file_path, '/models/%s' % MODEL_NAME)
shutil.copy(label_file_path, '/models/%s' % MODEL_NAME)
shutil.copyfile(graphdef_path, os.path.join(model_dir, 'model.graphdef'))

os.system('trtserver --model-store=/models')
