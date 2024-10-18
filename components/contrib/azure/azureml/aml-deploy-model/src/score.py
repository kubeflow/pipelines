from azureml.core.model import Model
import numpy as np
def init():
    pass

def run(data):
    try:
        # You can return any data type, as long as it is JSON serializable.
        return 'AML Model inference result, the data is {0}'.format(data)
    except Exception as e:
        error = str(e)
        return error