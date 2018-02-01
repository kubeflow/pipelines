"""
This is a dag that contrary to the Airflow's philosophy. 
foo_flag is a boolean flag that gets evaluated everytime scheduler reads this DAG. 
The DAG topology is determind based on this flag and thus dynamic. 
The tree/graph view change when I refresh the page. 
"""
import airflow
from airflow.operators.python_operator import BranchPythonOperator
from airflow.operators.dummy_operator import DummyOperator
from airflow.models import DAG
import random


args = {
    'owner': 'airflow',
    'start_date': airflow.utils.dates.days_ago(2)
}

import datetime, time
foo_flag = (time.mktime(datetime.datetime.now().timetuple()) %2)==0

dag = DAG(
    dag_id='BAD_dynamic_dag',
    default_args=args,
    schedule_interval="@once")

run_this_first = DummyOperator(task_id='run_this_first', dag=dag)

if foo_flag:
    options = ['branch_a', 'branch_b', 'branch_c', 'branch_d'] 
else:
    options = ['cranch']

branching = BranchPythonOperator(
    task_id='branching',
    python_callable=lambda: random.choice(options),
    dag=dag)
branching.set_upstream(run_this_first)

join = DummyOperator(
    task_id='join',
    trigger_rule='one_success',
    dag=dag
)

for option in options:
    t = DummyOperator(task_id=option, dag=dag)
    t.set_upstream(branching)
    dummy_follow = DummyOperator(task_id='follow_' + option, dag=dag)
    t.set_downstream(dummy_follow)
    dummy_follow.set_downstream(join)