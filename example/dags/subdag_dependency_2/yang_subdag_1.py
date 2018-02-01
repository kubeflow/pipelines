import airflow

from airflow.models import DAG
from airflow.operators.dummy_operator import DummyOperator
from airflow.operators.subdag_operator import SubDagOperator
from deps.subdags import *

DAG_NAME = 'yang_subdag_1'

args = {
    'owner': 'airflow',
    'start_date': airflow.utils.dates.days_ago(2),
}

with open("/root/airflow/subdag_1_outside_dag.txt", "a") as fh:
    fh.write("sys.path :" + str(sys.path)+" \n")
    fh.write(some_string()+" \n")

dag = DAG(
    dag_id=DAG_NAME,
    default_args=args,
    schedule_interval="@once",
)

start = DummyOperator(
    task_id='start',
    default_args=args,
    dag=dag,
)

section_1 = SubDagOperator(
    task_id='section-1',
    subdag=subdag(DAG_NAME, 'section-1', args),
    default_args=args,
    dag=dag,
)

some_other_task = DummyOperator(
    task_id='some-other-task',
    default_args=args,
    dag=dag,
)

section_2 = SubDagOperator(
    task_id='section-2',
    subdag=subdag(DAG_NAME, 'section-2', args),
    default_args=args,
    dag=dag,
)

end = DummyOperator(
    task_id='end',
    default_args=args,
    dag=dag,
)

start.set_downstream(section_1)
section_1.set_downstream(some_other_task)
some_other_task.set_downstream(section_2)
section_2.set_downstream(end)