import json
import os
import sys
import unittest
from unittest import mock

# Fixing module lookup in google3
sys.path.append(os.path.dirname(__file__))

import component


class ComponentTests(unittest.TestCase):

  def test_component(self):
    output_model_dir = self.create_tempdir().mkdir().full_path
    component.create_fully_connected_pytorch_network(
        input_size=10,
        model_path=output_model_dir + "/data",
    )


if __name__ == "__main__":
  unittest.main()
