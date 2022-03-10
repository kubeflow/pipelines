from google_cloud_pipeline_components.container.experimental.vertex_notification_email import executor

import unittest


class TestVertexNotificationEmailExecutor(unittest.TestCase):

  def test_executor(self):
    with self.assertRaises(NotImplementedError):
      executor.main()
