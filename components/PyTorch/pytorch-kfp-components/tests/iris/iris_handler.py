#!/usr/bin/env/python3
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

import ast
import logging

import numpy as np
import torch
from ts.torch_handler.base_handler import BaseHandler

logger = logging.getLogger(__name__)


class IRISClassifierHandler(BaseHandler):
    """
    IRISClassifier handler class. This handler takes an input tensor and
    output the type of iris based on the input
    """

    def __init__(self):
        super(IRISClassifierHandler, self).__init__()

    def preprocess(self, data):
        """
        preprocessing step - Reads the input array and converts it to tensor

        :param data: Input to be passed through the layers for prediction

        :return: output - Preprocessed input
        """

        input_data_str = data[0].get("data")
        if input_data_str is None:
            input_data_str = data[0].get("body")

        input_data = input_data_str.decode("utf-8")
        input_tensor = torch.Tensor(ast.literal_eval(input_data))
        return input_tensor

    def postprocess(self, inference_output):
        """
        Does postprocess after inference to be returned to user

        :param inference_output: Output of inference

        :return: output - Output after post processing
        """

        predicted_idx = str(np.argmax(inference_output.cpu().detach().numpy()))

        if self.mapping:
            return [self.mapping[str(predicted_idx)]]
        return [predicted_idx]


_service = IRISClassifierHandler()
