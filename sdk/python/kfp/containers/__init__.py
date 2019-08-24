# Copyright 2019 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the speci

_default_image_builder = None

def get_default_image_builder():
    global _default_image_builder
    if _default_image_builder is None:
        from ..compiler._container_builder import ContainerBuilder
        _default_image_builder = ContainerBuilder()

def set_default_image_builder(builder):
    global _default_image_builder
    _default_image_builder = builder

from ._build_image_api import *
