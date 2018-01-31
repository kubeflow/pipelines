"""
airflow trigger_dag dag_plugin
"""
from airflow import DAG
from airflow.operators.bash_operator import BashOperator
from airflow.operators.python_operator import PythonOperator
from datetime import datetime, timedelta
from deps import MyFirstOperator

default_args = {
    'owner': 'airflow',
    'depends_on_past': False,
    'start_date': datetime(2015, 6, 1),
}

dag = DAG(
    'dag_plugin_1', default_args=default_args, schedule_interval="@once") 


operator_task = MyFirstOperator(
    task_id='my_first_operator_task', 
    params={"python_task": {"output_path": "/root/airflow/python_output.txt"}},
    dag=dag)
