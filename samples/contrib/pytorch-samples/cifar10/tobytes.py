# !/usr/bin/env/python3
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
import base64
import json
import argparse
import uuid

parser = argparse.ArgumentParser()
parser.add_argument("filename", help="converts image to bytes array", type=str)
args = parser.parse_args()

image = open(args.filename, "rb")  # open binary file in read mode
image_read = image.read()
image_64_encode = base64.b64encode(image_read)
bytes_array = image_64_encode.decode("utf-8")
request = {
    "inputs": [{"name": str(uuid.uuid4()), "shape": -1, "datatype": "BYTES", "data": bytes_array}]
}

result_file = "{filename}.{ext}".format(filename=str(args.filename).split(".")[0], ext="json")
with open(result_file, "w") as outfile:
    json.dump(request, outfile, indent=4, sort_keys=True)