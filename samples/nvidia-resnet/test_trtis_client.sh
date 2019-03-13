#!/bin/bash
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


CLIENT_DIR=$(pwd)/trtis_client
REPO_DIR=/tmp/tensorrt-inference-server
SCRIPT_DIR=$REPO_DIR/src/clients/python/
IMAGE=tensorrtserver_clients
UPDATED_CLIENT_FILE=updated_image_client.py
DEMO_CLIENT_UI=demo_client_ui.py

CMD="pip install dash && python3 /workspace/src/clients/python/$DEMO_CLIENT_UI"
# If you don't want to test the UI, just use the command below
# CMD="python3 /workspace/src/clients/python/$UPDATED_CLIENT_FILE -m resnet_graphdef -s RESNET images/mug.jpg"

git clone https://github.com/NVIDIA/tensorrt-inference-server.git $REPO_DIR 
cp $CLIENT_DIR/$UPDATED_CLIENT_FILE $SCRIPT_DIR
cp $CLIENT_DIR/$DEMO_CLIENT_UI $SCRIPT_DIR
docker build -t $IMAGE -f $REPO_DIR/Dockerfile.client $REPO_DIR
docker run -it --rm --net=host $IMAGE /bin/bash -c "$CMD"
