# Model import


```python
from typing import List, NamedTuple
import kfp
from kfp import dsl
from kfp.components import func_to_container_op
from kubernetes.client.models impoort V1ContainerPort
from kfp.components import InputPath, OutputPath
```

# Adjust the number of Clients



```python
NUM_OF_CLIENTS = 2 #Custom number of clients
```

# Import necessary libraries and modules:

Flask: Used for building web server applications.
Other Python standard libraries and third-party libraries, such as NumPy, Pandas, and TensorFlow, are used for data processing, machine learning, etc.
threading: used to establish multiple threads to achieve synchronous and asynchronous operations.

# Define some global variables and locks:

global_value: A dictionary that stores some shared global variables, such as execution status, data status, global count, average weight, etc.
Different locks are used for different purposes, such as initialization locks, client local count locks, scaling local weight list locks, calculating average weight locks, etc.

# Define Flask’s routing function:

/data: used to receive data from the client, calculate the global weight, and return the global weight.
/shutdown: Used to shut down the server.


```python
def server(NUM_OF_CLIENTS:int):
    import json
    import pandas as pd
    import numpy as np
    import pickle
    import threading
    import time
    import tensorflow as tf
    from flask import Flask, jsonify,request
    import os
    
    app = Flask(__name__)
    clients_local_count = []
    scaled_local_weight_list = []
    global_value = { #Share variable
                    'last_run_state' : False, #last run finish or not
                    'data_state' : None,      #global_count finish or not
                    'global_count' : None,
                    'scale_state' : None,
                    'weight_state' : None,
                    'average_weights' : None,
                    'shutdown' : 0}
    
    
    
    init_lock = threading.Lock()
    clients_local_count_lock = threading.Lock()
    scaled_local_weight_list_lock = threading.Lock()
    cal_weight_lock = threading.Lock()
    shutdown_lock = threading.Lock()
    
    @app.before_request
    def before_request():
        print('get request')
        
        
    @app.route('/data', methods=['POST'])
    def flask_server():
        with init_lock:  #check last run is finish and init varible
            
            while True:
                
                if(len(clients_local_count)==0 and global_value['last_run_state'] == False):#init the variable by first client enter
                    global_value['last_run_state'] = True
                    global_value['data_state'] = False
                    global_value['scale_state'] = False
                    global_value['weight_state'] = False
                    break
                
                elif(global_value['last_run_state'] == True):
                    break
                time.sleep(3)
        
        local_count = int(request.form.get('local_count'))          #get data
        bs = int(request.form.get('bs'))
        local_weight = json.loads(request.form.get('local_weight'))
        local_weight = [np.array(lst) for lst in local_weight]
        

        
        
        def scale_model_weights(weight, scalar):
            weight_final = []
            steps = len(weight)
            for i in range(steps):
                weight_final.append(scalar * weight[i])
            return weight_final
        def sum_scaled_weights(scaled_weight_list):
            
            avg_grad = list()
            #get the average grad accross all client gradients
            for grad_list_tuple in zip(*scaled_weight_list):
                layer_mean = tf.math.reduce_sum(grad_list_tuple, axis=0)
                avg_grad.append(layer_mean)

            return avg_grad
        
        with clients_local_count_lock:
            clients_local_count.append(int(local_count))
            
        with scaled_local_weight_list_lock:
            while True:
                
                if (len(clients_local_count) == NUM_OF_CLIENTS and global_value['data_state'] != True):
                    global_value['last_run_state'] = False
                    sum_of_local_count=sum(clients_local_count)
                    
                    
                    global_value['global_count'] = sum_of_local_count     
                    
                    scaling_factor=local_count/global_value['global_count']
                    scaled_weights = scale_model_weights(local_weight, scaling_factor)
                    scaled_local_weight_list.append(scaled_weights)
                    
                    global_value['scale_state'] = True 
                    global_value['data_state'] = True
                    break
                elif (global_value['data_state'] == True and global_value['scale_state'] == True):
                    scaling_factor=local_count/global_value['global_count']
                    scaled_weights =scale_model_weights(local_weight, scaling_factor)
                    scaled_local_weight_list.append(scaled_weights)
        
                    break
                time.sleep(1)
               
        with cal_weight_lock:
            
            while True:
                if(len(scaled_local_weight_list) == NUM_OF_CLIENTS and global_value['weight_state'] != True):
                    
                    global_value['average_weights'] = sum_scaled_weights(scaled_local_weight_list)
                    global_value['weight_state'] = True
                    global_value['average_weights'] = json.dumps([np.array(w).tolist() for w in global_value['average_weights']])
                    
                    break
                    
                elif(global_value['weight_state'] == True):
                    
                    break
                
                time.sleep(1)
                
                
        clients_local_count.clear()
        scaled_local_weight_list.clear()
        
        return jsonify({'result': (global_value['average_weights'])})
        
    
    @app.route('/shutdown', methods=['GET'])
    def shutdown_server():
        global_value['shutdown'] +=1 
        with shutdown_lock:
            while True:
                if(global_value['shutdown'] == NUM_OF_CLIENTS):
                    os._exit(0)
                    return 'Server shutting down...'
                time.sleep(1)
    
    
    app.run(host="0.0.0.0", port=8080)
```

# Change the server function to container function


```python
server_op=func_to_container_op(server,base_image='tensorflow/tensorflow',packages_to_install=['flask','pandas'])
```

# Download data
In this example, we use data with two labels(normal and abnormal) and download them from cloud. You can use your own data or the hemodialysis information set we provide

Then generate labels for normal and abnormal data respectively, with normal labels being [1, 0] and abnormal labels being [0, 1], and merge the data and labels into one dataset.


```python
def download_data(log_folder:str,num_of_clients:int)-> NamedTuple('Outputs', [('data', str), ('label', str)]):
    from typing import NamedTuple
    import pandas as pd
    import numpy as np
    import json
    import os
    
    
    normal_url='<change yourself>' 
    abnormal_url='<change yourself>'

    normal_data = pd.read_csv(normal_url)
    abnormal_data = pd.read_csv(abnormal_url)

    num_features = len(normal_data.columns)
    print(num_features)

    normal_label = np.array([[1, 0]] * len(normal_data))
    abnormal_label = np.array([[0, 1]] * len(abnormal_data))

    data = np.vstack((normal_data, abnormal_data))
    data_label = np.vstack((normal_label, abnormal_label))

    shuffler = np.random.permutation(len(data))
    data = data[shuffler]
    data_label = data_label[shuffler]

    data = data.reshape(len(data), num_features, 1)
    label = data_label.reshape(len(data_label), 2)

    np.save(os.path.join(log_folder, 'data.npy'), data)
    np.save(os.path.join(log_folder, 'label.npy'), label)
    result = NamedTuple('Outputs', [('data', str), ('label', str)])
    
    return result(os.path.join(log_folder, 'data.npy'),
                    os.path.join(log_folder, 'label.npy'),)
```

# Change the download function to container function


```python
download_data_op = func_to_container_op(download_data,packages_to_install=['pandas','numpy'])
```

# This function is mainly used on the client to train the model and update local weights to the server.

We use "request" for sending an HTTP request to the server.
Actually it's not a correct federated learning code because each client should have its own data. This code is for practice only.

If necessary, you can adjust the model in the "class SimpleMLP()".

Every time the client completes training n times (you can change n in epochs), it will upload the weights to the server and wait for the adjusted weights to be sent back for the next training.


```python
def client(log_folder:str,data_path: str, 
                label_path: str, batch:int,num_of_clients:int) -> NamedTuple('Outputs', [("last_accuracy",float)]):
    import json
    import requests
    import time
    import pandas as pd
    import numpy as np
    import tensorflow as tf
    from tensorflow.keras.models import Sequential
    from tensorflow.keras.layers import Conv1D
    from tensorflow.keras.layers import MaxPooling1D
    from tensorflow.keras.layers import Activation
    from tensorflow.keras.layers import Flatten
    from tensorflow.keras.layers import Dense
    from tensorflow.keras.optimizers import SGD
    from tensorflow.keras import backend as K
    
    train_data=np.load(data_path)
    train_label=np.load(label_path)
    print(train_data.shape)
    print(train_label.shape)
    def split_and_get_batch(data, labels, x, batch_index):
        
        batch_size = len(data) // x

        
        data_batches = np.array_split(data, x)
        label_batches = np.array_split(labels, x)

        
        selected_data_batch = data_batches[batch_index]
        selected_label_batch = label_batches[batch_index]

        return selected_data_batch, selected_label_batch
    
    data,label = split_and_get_batch(train_data,train_label,num_of_clients,batch-1)
    print(data.shape)
    print(label.shape)
    full_data = list(zip(data,label))
    class SimpleMLP:
        @staticmethod
        def build(shape, classes):
            model = Sequential()
            model.add(Conv1D(filters=4, kernel_size=3, input_shape=(17,1)))
            model.add(MaxPooling1D(3))
            model.add(Flatten())
            model.add(Dense(8, activation="relu"))
            model.add(Dense(2, activation = 'softmax'))

            return model
    
    
    
    print('data len= ',len(full_data))
    def batch_data(data_shard, bs=32):
    
        #seperate shard into data and labels lists
        data, label = zip(*data_shard)
        dataset = tf.data.Dataset.from_tensor_slices((list(data), list(label)))
        return dataset.shuffle(len(label)).batch(bs)
    
    dataset=batch_data(full_data)
    #print(dataset)
    
    bs = next(iter(dataset))[0].shape[0]
    local_count = tf.data.experimental.cardinality(dataset).numpy()*bs
    
    
    loss='categorical_crossentropy'
    metrics = ['accuracy']
    optimizer = 'adam'
    
    
    smlp_model = SimpleMLP()
    
    server_url="http://http-service:5000/data"
    for comm_round in range(5):
        print('The ',comm_round+1, 'round')
        client_model = smlp_model.build(17, 1)
        client_model.compile(loss=loss, 
                      optimizer=optimizer, 
                      metrics=metrics)
        
        if(comm_round == 0):
            history = client_model.fit(dataset, epochs=1, verbose=1)
        else:
            client_model.set_weights(avg_weight)
            history = client_model.fit(dataset, epochs=1, verbose=1)
        
        local_weight = client_model.get_weights()
        local_weight = [np.array(w).tolist() for w in local_weight]
        
        client_data = {"local_count": local_count,'bs': bs, 'local_weight': json.dumps(local_weight)}
        
        while True:
            try:
                weight = (requests.post(server_url,data=client_data))
                
                if weight.status_code == 200:
                    print(f"exist")

                    break
                else:
                    print(f"server error")

            except requests.exceptions.RequestException:

                print(f"not exist")
                
            time.sleep(5)
            
        data = weight.json()
        avg_weight = data.get('result')
        avg_weight = json.loads(avg_weight)
        avg_weight = [np.array(lst) for lst in avg_weight]
        
    shutdown_url="http://http-service:5000/shutdown"    
    try:
        response = requests.get(shutdown_url)
    except requests.exceptions.ConnectionError:
        print('already shutdown')
    last_accuracy = history.history['accuracy'][-1]
    print(last_accuracy)
    return([last_accuracy])
```

# Change the client function to container function


```python
client_op=func_to_container_op(client,base_image='tensorflow/tensorflow',packages_to_install=['requests','pandas'])
```

# Receive and convert the test accuracy results into string format for subsequent processing or display.

Convert the test accuracy results into string format and separate each value with commas.
Remove the last comma to avoid extra delimiters.

Then change the show_result function to container function


```python
def show_results(test_acc: List[float]) -> NamedTuple('Outputs', [('test_accuracy', str)]):
    
    result_str = ', '.join(map(str, test_acc))
    print(result_str)
    
    result_str = result_str[:-1]
    
    return (result_str,)

show_results_op = func_to_container_op(show_results)
```

# Generate custom number of clients

This function will generate the containers(task) and connect them together(download_data_task, client_task and show_results_task)


```python
def generate_clients(log_folder,n):# generate the num of clients
    global vop
    client_tasks = []
    download_data_task = download_data_op(log_folder,n)
    download_data_task.set_cpu_request('0.1').set_cpu_limit('0.1').add_pvolumes({
        log_folder:vop.volume,
    })
    for i in range(n):
        client_task = globals()['client_task_' + str(i + 1)] = client_op(log_folder, download_data_task.outputs['data'],
                                        download_data_task.outputs['label'],i + 1,n)
        client_task.set_cpu_request('0.1').set_cpu_limit('0.1').add_pvolumes({
        log_folder:vop.volume,
    })
        client_tasks.append(client_task)
    
    globals()['show_results_task'] = show_results_op([ct.outputs['last_accuracy'] for ct in client_tasks]).set_cpu_request('0.1').set_cpu_limit('0.1')
```

# Define the pipeline

The first half part we define the path(log_folder) for saving the data and create a volume.
Then the server part we use "Service" to let client containers can connect to the server container. After the server shutdown, this service will be deleted.

The second half we use generate_clients() to generate the other container, include download_data, clients and show result


```python
@dsl.pipeline(
    name='FL test'
    )
def fl_pipeline(namespace='kubeflow-user-thu01'):
    global NUM_OF_CLIENTS
    log_folder = '/data'
    global vop
    pvc_name = "mypvc"
    vop = dsl.VolumeOp(
        name=pvc_name,
        resource_name="newpvc",
        size="1Gi",
        modes=dsl.VOLUME_MODE_RWO
    )
    
    
    service = dsl.ResourceOp(
        name='http-service',
        k8s_resource={
            'apiVersion': 'v1',
            'kind': 'Service',
            'metadata': {
                'name': 'http-service'
            },
            'spec': {
                'selector': {
                    'app': 'http-service'
                },
                'ports': [
                    {
                        'protocol': 'TCP',
                        'port': 5000,
                        'targetPort': 8080
                    }
                ]
            }
        }
    )#service_end
    server_task=server_op(NUM_OF_CLIENTS)
    server_task.add_pod_label('app', 'http-service')
    server_task.add_port(V1ContainerPort(name='my-port', container_port=8080))
    server_task.set_cpu_request('0.1').set_cpu_limit('0.1')
    server_task.after(service)
    
    delete_service = dsl.ResourceOp(
        name='delete-service',
        k8s_resource={
            'apiVersion': 'v1',
            'kind': 'Service',
            'metadata': {
                'name': 'http-service'
            },
            'spec': {
                'selector': {
                    'app': 'http-service'
                },
                'ports': [
                    {
                        'protocol': 'TCP',
                        'port': 80,
                        'targetPort': 8080
                    }
                ],
                'type': 'NodePort'  
            }
        },
        action="delete" #刪除
    ).after(server_task)
    
    generate_clients(log_folder,NUM_OF_CLIENTS)

```

# Convert the pipeline into yaml file


```python
if __name__ == '__main__':
    kfp.compiler.Compiler().compile(fl_pipeline, 'fl_pipeline.yaml')
```
