"""
airflow trigger_dag print_pythonpath
airflow test print_pythonpath python_task 2017-03-18T18:00:00.0
"""
from airflow import DAG
from airflow.operators.bash_operator import BashOperator
from datetime import datetime, timedelta
from airflow.operators.python_operator import PythonOperator
import sys

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
    'print_pythonpath', default_args=default_args, schedule_interval="@once") 


def python_task_callback(**kwargs):
    with open("/root/airflow/python_output.txt", "a") as fh:
        print(str(sys.path)+'\n')
    return 1

python_task = PythonOperator(
    task_id='python_task',
    provide_context=True,
    python_callable=python_task_callback,
    dag=dag)

