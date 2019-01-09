import logging
import time

import googleapiclient.discovery as discovery

def create_ml_client():
    return discovery.build('ml', 'v1')

def wait_for_operation_done(ml_client, operation_name, wait_interval):
    while True:
        operation = ml_client.projects().operations().get(
            name = operation_name
        ).execute()
        done = operation.get('done', False)
        if done:
            return operation
        logging.info('Operation {} is not done. Wait for {}s.'.format(operation_name, wait_interval))
        time.sleep(wait_interval)