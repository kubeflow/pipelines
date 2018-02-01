import logging

from airflow.models import DAG
from airflow.operators.python_operator import PythonOperator
from datetime import datetime, timedelta
import sys

def some_string():
    return "from dependency in dag 1"

def python_task_callback(**kwargs):
    with open("/root/airflow/from_subdag_1.txt", "a") as fh:
        fh.write("from subdag 1"+'\n')
    return 1

def subdag(parent_dag_name, child_dag_name, args):
    dag_subdag = DAG(
        dag_id='%s.%s' % (parent_dag_name, child_dag_name),
        default_args=args,
        schedule_interval="@once",
    )

    with open("/root/airflow/from_subdag_1_subgraph_construction.txt", "a") as fh:
        fh.write("from subdag 1 construction"+'\n')
        fh.write("parent dag"+ parent_dag_name + '\n')        
        fh.write(str(sys.path)+"\n")

    for i in range(10):
        PythonOperator(
            task_id='%s-task-%s' % (child_dag_name, i + 1),
            provide_context=True,
            python_callable=python_task_callback,
            dag=dag_subdag)
    return dag_subdag

