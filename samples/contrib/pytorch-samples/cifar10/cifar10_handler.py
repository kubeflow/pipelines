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
# pylint: disable=no-self-use,too-many-arguments,unused-argument,not-callable,no-member,attribute-defined-outside-init
""" Cifar10 Custom Handler."""

import base64
import io
import json
import logging
import os
from abc import ABC
from base64 import b64encode
from io import BytesIO

import numpy as np
import torch
from PIL import Image
from captum.attr import (
    IntegratedGradients, Occlusion, LayerGradCam, LayerAttribution
)
from captum.attr import visualization as viz
from classifier import CIFAR10CLASSIFIER
from matplotlib.colors import LinearSegmentedColormap
from torchvision import transforms
from ts.torch_handler.image_classifier import ImageClassifier

logger = logging.getLogger(__name__)


class CIFAR10Classification(ImageClassifier, ABC):
    """
    Base class for all vision handlers
    """

    def initialize(self, ctx): # pylint: disable=arguments-differ
        """In this initialize function, the CIFAR10 trained model is loaded and
        the Integrated Gradients,occlusion and layer_gradcam Algorithm for
        Captum Explanations is initialized here.
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
            "cuda:" + str(properties.get("gpu_id")) if torch.cuda.is_available(
            ) else "cpu"
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
            with open(mapping_file_path) as pointer:
                self.mapping = json.load(pointer)
        else:
            print("Mapping file missing")
            logger.warning("Missing the class_mapping.json file.")

        self.ig = IntegratedGradients(self.model)
        self.layer_gradcam = LayerGradCam(
            self.model, self.model.model_conv.layer4[2].conv3
        )
        self.occlusion = Occlusion(self.model)
        self.initialized = True
        self.image_processing = transforms.Compose([
            transforms.Resize(224),
            transforms.CenterCrop(224),
            transforms.ToTensor(),
            transforms.Normalize(
                mean=[0.485, 0.456, 0.406], std=[0.229, 0.224, 0.225]
            ),
        ])

    def _get_img(self, row):
        """Compat layer: normally the envelope should just return the data
        directly, but older version of KFServing envelope and
        Torchserve in general didn't have things set up right
        """

        if isinstance(row, dict):
            image = row.get("data") or row.get("body")
        else:
            image = row

        if isinstance(image, str):
            # if the image is a string of bytesarray.
            image = base64.b64decode(image)

        return image

    def preprocess(self, data):
        """The preprocess function of cifar10 program
        converts the input data to a float tensor
        Args:
            data (List): Input data from the request is in the form of a Tensor
        Returns:
            list : The preprocess function returns
            the input image as a list of float tensors.
        """
        images = []

        for row in data:
            image = self._get_img(row)

            # If the image is sent as bytesarray
            if isinstance(image, (bytearray, bytes)):
                image = Image.open(io.BytesIO(image))
                image = self.image_processing(image)
            else:
                # if the image is a list
                image = torch.FloatTensor(image)

            images.append(image)

        return torch.stack(images).to(self.device)

    def attribute_image_features(self, algorithm, data, **kwargs):
        """Calculate tensor attributions"""
        self.model.zero_grad()
        tensor_attributions = algorithm.attribute(data, target=0, **kwargs)
        return tensor_attributions

    def output_bytes(self, fig):
        """Convert image to bytes"""
        fout = BytesIO()
        fig.savefig(fout, format="png")
        fout.seek(0)
        return fout.getvalue()

    def get_insights(self, tensor_data, _, target=0):
        default_cmap = LinearSegmentedColormap.from_list(
            "custom blue",
            [(0, "#ffffff"), (0.25, "#0000ff"), (1, "#0000ff")],
            N=256,
        )

        attributions_ig, _ = self.attribute_image_features(
            self.ig,
            tensor_data,
            baselines=tensor_data * 0,
            return_convergence_delta=True,
            n_steps=15,
        )

        attributions_occ = self.attribute_image_features(
            self.occlusion,
            tensor_data,
            strides=(3, 8, 8),
            sliding_window_shapes=(3, 15, 15),
            baselines=tensor_data * 0,
        )

        attributions_lgc = self.attribute_image_features(
            self.layer_gradcam, tensor_data
        )

        upsamp_attr_lgc = LayerAttribution.interpolate(
            attributions_lgc, tensor_data.shape[2:]
        )

        matplot_viz_ig, _ = viz.visualize_image_attr_multiple(
            np.transpose(
                attributions_ig.squeeze().cpu().detach().numpy(), (1, 2, 0)
            ),
            np.transpose(
                tensor_data.squeeze().cpu().detach().numpy(), (1, 2, 0)
            ),
            use_pyplot=False,
            methods=["original_image", "heat_map"],
            cmap=default_cmap,
            show_colorbar=True,
            signs=["all", "positive"],
            titles=["Original", "Integrated Gradients"],
        )

        matplot_viz_occ, _ = viz.visualize_image_attr_multiple(
            np.transpose(
                attributions_occ.squeeze().cpu().detach().numpy(), (1, 2, 0)
            ),
            np.transpose(
                tensor_data.squeeze().cpu().detach().numpy(), (1, 2, 0)
            ),
            [
                "original_image",
                "heat_map",
                "heat_map",
            ],
            ["all", "positive", "negative"],
            show_colorbar=True,
            titles=[
                "Original",
                "Positive Attribution",
                "Negative Attribution",
            ],
            fig_size=(18, 6),
            use_pyplot=False,
        )

        matplot_viz_lgc, _ = viz.visualize_image_attr_multiple(
            upsamp_attr_lgc[0].cpu().permute(1, 2, 0).detach().numpy(),
            tensor_data.squeeze().permute(1, 2, 0).cpu().numpy(),
            use_pyplot=False,
            methods=["original_image", "blended_heat_map", "blended_heat_map"],
            signs=["all", "positive", "negative"],
            show_colorbar=True,
            titles=[
                "Original",
                "Positive Attribution",
                "Negative Attribution",
            ],
            fig_size=(18, 6)
        )

        occ_bytes = self.output_bytes(matplot_viz_occ)
        ig_bytes = self.output_bytes(matplot_viz_ig)
        lgc_bytes = self.output_bytes(matplot_viz_lgc)

        output = [{
            "b64": b64encode(row).decode("utf8")
        } if isinstance(row, (bytes, bytearray)) else row
                  for row in [ig_bytes, occ_bytes, lgc_bytes]]
        return output
