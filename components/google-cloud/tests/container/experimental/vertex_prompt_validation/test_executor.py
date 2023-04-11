from google_cloud_pipeline_components.container.experimental.vertex_prompt_validation import executor

import unittest


class TestVertexPromptValidationExecutor(unittest.TestCase):

  def test_executor(self):
    with self.assertRaises(NotImplementedError):
      executor.main()
