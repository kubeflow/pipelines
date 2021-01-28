# Copyright 2021 Google LLC
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
# See the License for the specific language governing permissions and
# limitations under the License.
import absl
import argparse
import logging
from ml_metadata.proto import metadata_store_pb2
import os
import sys

MLMD_HOST_ENV = 'METADATA_GRPC_SERVICE_HOST'
MLMD_PORT_ENV = 'METADATA_GRPC_SERVICE_PORT'


def _get_metadata_connection_config(
) -> metadata_store_pb2.MetadataStoreClientConfig:
  """Constructs the metadata grpc connection config.

  Returns:
    A metadata_store_pb2.MetadataStoreClientConfig object.
  """
  connection_config = metadata_store_pb2.MetadataStoreClientConfig()
  connection_config.host = os.getenv(MLMD_HOST_ENV)
  connection_config.port = int(os.getenv(MLMD_PORT_ENV))

  return connection_config

def main():
  logging.basicConfig(stream=sys.stdout, level=logging.INFO)
  logging.getLogger().setLevel(logging.INFO)

  parser = argparse.ArgumentParser()
