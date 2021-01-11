import kfp 
import yaml 
import mysql.connector 
import click
import sys
import logging
from typing import Tuple
from datetime import datetime


def get_job(job_id: str) -> Tuple[bool, str, str]:
    """Function to fetch the job info from the databse

    Args:
        job_id (str): The UUID for the pipeline job of interest

    Raises:
        ConnectionError: Fail to connect to database

    Returns:
        Tuple[bool, str, str]: job enabled, name, workflow  
    """
    print("HERE HERE TWO")
    print("Try to get the job")
    conn = mysql.connector.connect(host='mysql.kubeflow.svc.cluster.local', database='mlpipeline', user='root') 
    if  not conn.is_connected:
        raise ConnectionError("Failed to connect to database")
    cursor = conn.cursor(buffered=True) 
    print("Query the data from mysql database")
    cursor.execute("SELECT Enabled, DisplayName, WorkflowSpecManifest FROM jobs WHERE UUID ='{}'".format(job_id)) 
    res = cursor.fetchall()
    if res[0][0] ==  1:
        print("Job is enabled")
        return True, res[0][1], res[0][2]
    print("The job is not enabled")
    return False, "", ""


@click.command()
@click.option('--job-id', type=str)
@click.option('--experiment-id', type=str)
def main(job_id: str, experiment_id: str):
    print("ONE ONE ONE ")
    client = kfp.Client(host="http://ml-pipeline-ui.kubeflow.svc.cluster.local:80")
    print("TWO TWO TWO")
    run, name, workflow = get_job(job_id=job_id)
    print("THREE THREE THREE")
    now = datetime.now()
    with open('/tmp/workflow.yaml', 'w') as file:
        _ = yaml.dump(yaml.safe_load(workflow) , file)

    if run:
        client.run_pipeline(experiment_id=experiment_id, 
        job_name=name.replace(" ", "_")+now.strftime("%m_%d_%Y_%H_%M_%S"),
        pipeline_package_path="/tmp/workflow.yaml")


if __name__ == "__main__":
    main()