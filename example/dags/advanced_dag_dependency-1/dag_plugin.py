"""
airflow trigger_dag dag_plugin
"""
from airflow import DAG
from airflow.operators.bash_operator import BashOperator
from airflow.operators.python_operator import PythonOperator
from datetime import datetime, timedelta
from deps import MyFirstOperator
import sys 

default_args = {
    'owner': 'airflow',
    'depends_on_past': False,
    'start_date': datetime(2015, 6, 1),
}

with open("/root/airflow/dag_1_outside_dag.txt", "a") as fh:
    fh.write("sys.path :" + str(sys.path)+" \n")
    fh.write(MyFirstOperator.print_something()+" \n")

dag = DAG(
    'dag_plugin_1', default_args=default_args, schedule_interval="@once") 


operator_task = MyFirstOperator(
    task_id='my_first_operator_task', 
    dag=dag)

def python_task_callback(**kwargs):
    with open("/root/airflow/dag_1_python_task.txt", "a") as fh:
    	fh.write("sys.path :" + str(sys.path)+" \n")
    	fh.write(MyFirstOperator.print_something()+" \n")
    return 1

python_task = PythonOperator(
    task_id='python_task',
    provide_context=True,
    python_callable=python_task_callback,
    dag=dag)

