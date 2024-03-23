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
    model_name = "projects/project-1/locations/us-cental1/models/123"
    model_dict = {"name": model_name}

    mock_model = mock.MagicMock()
    mock_model.resource_name = model_name
    mock_model.to_dict.return_value = {"name": model_name}

    with mock.patch(
        "google.cloud.aiplatform.aiplatform.Model.upload_xgboost_model_file",
        return_value=mock_model), mock.patch("shutil.copyfile"):
      return_values = component.upload_XGBoost_model_to_Google_Cloud_Vertex_AI(
          model_path="model_path",)
    actual_model_name = return_values[0]
    actual_model_dict = return_values[1]
    self.assertEqual(actual_model_name, model_name)
    self.assertEqual(json.loads(actual_model_dict), model_dict)


if __name__ == "__main__":
  unittest.main()
