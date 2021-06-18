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
# pylint: disable=no-self-use,too-many-arguments,unused-argument,not-callable


""" Cifar10 Custom Handler."""
from abc import ABC
import os
import torch
import json
import numpy as np
from captum.attr import IntegratedGradients, Occlusion, LayerGradCam
from ts.torch_handler.image_classifier import ImageClassifier
from classifier import CIFAR10CLASSIFIER
import logging

logger = logging.getLogger(__name__)


class CIFAR10Classification(ImageClassifier, ABC):
    """
    Base class for all vision handlers
    """

    def initialize(self, ctx):
        """In this initialize function, the Titanic trained model is loaded and
        the Integrated Gradients Algorithm for Captum Explanations
        is initialized here.
        Args:
            ctx (context): It is a JSON Object containing information
            pertaining to the model artifacts parameters.
        """
        self.manifest = ctx.manifest
        properties = ctx.system_properties
        model_dir = properties.get("model_dir")
        print("Model dir is {}".format(model_dir))
        serialized_file = self.manifest["model"]["serializedFile"]
        model_pt_path = os.path.join(model_dir, serialized_file)
        self.device = torch.device(
            "cuda:" + str(properties.get("gpu_id"))
            if torch.cuda.is_available()
            else "cpu"
        )

        self.model = CIFAR10CLASSIFIER()
        self.model.load_state_dict(torch.load(model_pt_path))
        self.model.to(self.device)
        self.model.eval()
        self.model.zero_grad()
        logger.info("CIFAR10 model from path %s loaded successfully", model_dir)

        # Read the mapping file, index to object name
        mapping_file_path = os.path.join(model_dir, "class_mapping.json")
        if os.path.isfile(mapping_file_path):
            print("Mapping file present")
            with open(mapping_file_path) as f:
                self.mapping = json.load(f)
        else:
            print("Mapping file missing")
            logger.warning("Missing the class_mapping.json file.")

        self.ig = IntegratedGradients(self.model)
        self.layer_gradcam = LayerGradCam(
            self.model, self.model.model_conv.layer4[2].conv3
        )
        self.occlusion = Occlusion(self.model)
        self.initialized = True

    def attribute_image_features(self, algorithm, data, **kwargs):
        self.model.zero_grad()
        tensor_attributions = algorithm.attribute(data, target=0, **kwargs)
        return tensor_attributions

    def get_insights(self, tensor_data, _, target=0):
        explanation_ig = {}
        explanation_lgc = {}
        explanation_occ = {}
        attr_ig, _ = self.attribute_image_features(
            self.ig,
            tensor_data,
            baselines=tensor_data * 0,
            return_convergence_delta=True,
            n_steps=15,
        )
        attr_ig = np.transpose(
            attr_ig.squeeze().cpu().detach().numpy(), (1, 2, 0)
        )
        explanation_ig["attributions_ig"] = attr_ig.tolist()

        attributions_lgc = self.attribute_image_features(
            self.layer_gradcam, tensor_data
        )
        explanation_lgc["attributions_lgc"] = attributions_lgc.tolist()
        attributions_occ = self.attribute_image_features(
            self.occlusion,
            tensor_data,
            strides=(3, 8, 8),
            sliding_window_shapes=(3, 15, 15),
            baselines=tensor_data * 0,
        )
        attributions_occ = np.transpose(
            attributions_occ.squeeze().cpu().detach().numpy(), (1, 2, 0)
        )

        explanation_occ["attributions_occ"] = attributions_occ.tolist()

        return [explanation_ig, explanation_lgc, explanation_occ]
