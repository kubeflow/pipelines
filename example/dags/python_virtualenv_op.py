"""
airflow trigger_dag print_pythonpath
airflow test python_virtualenv_op python_virtualenv_task 2017-03-18T18:00:00.0
"""
from airflow import DAG
from airflow.operators.bash_operator import BashOperator
from datetime import datetime, timedelta
from airflow.operators.python_operator import PythonOperator, PythonVirtualenvOperator
import numpy
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
}

dag = DAG(
    'python_virtualenv_op', default_args=default_args, schedule_interval="@once") 

def python_task_callback(**kwargs):
    with open("/root/airflow/non_virtualenv_version.txt", "a") as fh:
        fh.write("numpy version: "+numpy.__version__+" \n")
        fh.write("sys.path: " + str(sys.path)+" \n")
        fh.write("numpy from path: " + os.path.abspath(numpy.__file__)+"\n")

python_task = PythonOperator(
    task_id='python_task',
    provide_context=True,
    python_callable=python_task_callback,
    dag=dag)


def python_virtualenv_task_callback(**kwargs):
    import numpy
    import os
    import sys
    with open("/root/airflow/virtualenv_version.txt", "a") as fh:
        fh.write("numpy version: "+numpy.__version__+" \n")
        fh.write("sys.path: " + str(sys.path)+" \n")
        fh.write("numpy from path: " + os.path.abspath(numpy.__file__)+"\n")

python_virtualenv_task = PythonVirtualenvOperator(
    task_id='python_virtualenv_task',
    python_callable=python_virtualenv_task_callback,
    requirements=["numpy==1.13"],
    dag=dag)