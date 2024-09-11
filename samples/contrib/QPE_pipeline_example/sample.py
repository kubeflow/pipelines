import os
import unittest
from dataclasses import dataclass
from pprint import pprint
from typing import List

import kfp
from kfp.dsl.graph_component import GraphComponent
import QPE

_MINUTE = 60  # seconds
_DEFAULT_TIMEOUT = 5 * _MINUTE


@dataclass
class TestCase:
    pipeline_func: GraphComponent
    timeout: int = _DEFAULT_TIMEOUT


class SampleTest(unittest.TestCase):
    _kfp_host_and_port = os.getenv('KFP_API_HOST_AND_PORT', 'http://localhost:8888')
    _kfp_ui_and_port = os.getenv('KFP_UI_HOST_AND_PORT', 'http://localhost:8080')
    _client = kfp.Client(host=_kfp_host_and_port, ui_host=_kfp_ui_and_port)

    def test(self):
        # 只測試 hello_world 範例
        test_case = TestCase(pipeline_func=QPE.qpe_pipeline)

        # 直接執行測試案例
        self.run_test_case(test_case.pipeline_func, test_case.timeout)

    def run_test_case(self, pipeline_func: GraphComponent, timeout: int):
        # 執行指定的管道函數，並等待完成
        with self.subTest(pipeline=pipeline_func, msg=pipeline_func.name):
            run_result = self._client.create_run_from_pipeline_func(pipeline_func=pipeline_func)
            run_response = run_result.wait_for_run_completion(timeout)

            # 打印運行的詳細信息
            pprint(run_response.run_details)
            print("Run details page URL:")
            print(f"{self._kfp_ui_and_port}/#/runs/details/{run_response.run_id}")

            # 檢查運行狀態是否成功
            self.assertEqual(run_response.state, "SUCCEEDED")


if __name__ == '__main__':
    unittest.main()
