"""
airflow trigger_dag print_pythonpath
airflow test print_pythonpath python_task 2017-03-18T18:00:00.0
"""
from airflow import DAG
from airflow.operators.bash_operator import BashOperator
from datetime import datetime, timedelta
from airflow.operators.python_operator import PythonOperator
import requests
import sys
import os

default_args = {
    'owner': 'airflow',
    'depends_on_past': False,
    'start_date': datetime(2015, 6, 1),
    'email': ['airflow@example.com'],
    'email_on_failure': False,
    'email_on_retry': False,
    'retries': 1,
    'retry_delay': timedelta(minutes=5),
    # 'queue': 'bash_queue',
    # 'pool': 'backfill',
    # 'priority_weight': 10,
    # 'end_date': datetime(2016, 1, 1),
}

dag = DAG(
    'print_requests_version_from_zip', default_args=default_args, schedule_interval="@once") 


def python_task_callback(**kwargs):
    with open("/root/airflow/python_output.txt", "a") as fh:
        fh.write("requests version from zip: "+requests.__version__+" \n")
        fh.write("sys.path from zip:" + str(sys.path)+" \n")
        fh.write("requests in zip loaded from path:" + os.path.abspath(requests.__file__)+"\n")

python_task = PythonOperator(
    task_id='python_task',
    provide_context=True,
    python_callable=python_task_callback,
    dag=dag)

