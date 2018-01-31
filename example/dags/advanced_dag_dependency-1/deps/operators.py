import logging

from airflow.models import BaseOperator
from airflow.plugins_manager import AirflowPlugin
from airflow.utils.decorators import apply_defaults
from datetime import datetime, timedelta
import sys

log = logging.getLogger(__name__)

class MyFirstOperator(BaseOperator):

    @apply_defaults
    def __init__(self, *args, **kwargs):
        super(MyFirstOperator, self).__init__(*args, **kwargs)

    def execute(self, context):
        with open("/root/airflow/dag_1_my_first_operator_task.txt", "a") as fh:
            fh.write("sys.path: " + str(sys.path)+" \n")
    	return 1

    @staticmethod
    def print_something():
    	return "Print from MyFirstOperator in Dag 1"

class MyFirstPlugin(AirflowPlugin):
    name = "my_first_plugin"
    operators = [MyFirstOperator]
