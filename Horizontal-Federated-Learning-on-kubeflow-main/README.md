# Horizontal-Federated-Learning-on-kubeflow
A sample code of the federated learning with kubeflow pipeline

## Description
The architecture of the simulated federated learning is as follows. The clients will train on data in parallel in their own container and send model parameters information to the server. The federated learning works by a round table procedure where each client submits its training results to the server in each unit epoch.  The server sums up the separate model parameters to form a new model.  The new model is then passed on to each client residing on different local container.  When they receive a response from server, clients will set the new model weights and proceed to the next federated learning round.

Architecture
---
![](https://github.com/sefgsefg/FL-on-kubeflow/blob/main/kubeflow_Architecture_v2.png)


Federated-Learning process
---
![](https://github.com/sefgsefg/FL-on-kubeflow/blob/main/FL_flow_chart.png)

Installation
---
```
pip install kfp
```
Change number of clients
---
```
NUM_OF_CLIENTS = 2 #Custom number of clients
```
Server
---
We use Flask to create a server. Initially, we create a list to store upload data from clients and make a dictionary for sharing variables between client contexts.

```
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
    
```
Create locks to ensure that the server calculation is correct
```
init_lock = threading.Lock()
clients_local_count_lock = threading.Lock()
scaled_local_weight_list_lock = threading.Lock()
cal_weight_lock = threading.Lock()
shutdown_lock = threading.Lock()
```
In the beginning, the first entering client will lock and initialize the global variable; subsequent clients do not need to do it again. Then the server will start receiving clients' data.
```
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
```
Here is an example of the locking process. 
The first client entering will detect if the length of 'clients_local_count' equals NUM_OF_CLIENTS and set 'global_value['data_status']' to True 

The subsequent clients will enter "elif" part to prevent doing the same action.
```
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
```
After finishing the weight calculation, the server has to clear the data to ensure the next FL round is correct. Then return the weights to clients.
```
clients_local_count.clear()
scaled_local_weight_list.clear()

return jsonify({'result': (global_value['average_weights'])})
```
After all FL rounds have finished, the clients will send a signal to the server. When the number of signals equals NUM_OF_CLIENTS, the server will shut down.
```
@app.route('/shutdown', methods=['GET'])
def shutdown_server():
    global_value['shutdown'] +=1 
    with shutdown_lock:
        while True:
            if(global_value['shutdown'] == NUM_OF_CLIENTS):
                os._exit(0)
                return 'Server shutting down...'
            time.sleep(1)
```


Client
---
This part initializes the dataset that you use. In this example, we use the hemodialysis information.
```
normal_url='<change yourself>' 
abnormal_url='<change yourself>'

normal_data = pd.read_csv(normal_url)		#init data, you should change method by your own data
abnormal_data = pd.read_csv(abnormal_url)
num_features = len(normal_data.columns)

normal_label = np.array([[1, 0]] * len(normal_data))
abnormal_label = np.array([[0, 1]] * len(abnormal_data))


data = np.vstack((normal_data, abnormal_data))
data_label = np.vstack((normal_label, abnormal_label))


shuffler = np.random.permutation(len(data))
data = data[shuffler]
data_label = data_label[shuffler]


data = data.reshape(len(data), num_features, 1)
data_label = data_label.reshape(len(data_label), 2)


full_data = list(zip(data, data_label))
data_length=len(full_data)
```
If necessary, you can modify the method used by the model clients here.
```
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
```
This part is not correct federated learning code because each client should have its own data. Instead, it's a way to obtain a dataset from a subset of the full dataset.
```
if(batch==1):
    full_data=full_data[0:int(data_length/2)] #batch data
else:
    full_data=full_data[int(data_length/2):data_length] #The client should have its own data, not like this. It's a lazy method.
```

It's a URL that clients connect to the server in Kubeflow (K8s). It's defined in Kubernetes service.

In k8s, service's url is http://&lt;service-name&gt;:port
```
server_url="http://http-service:5000/data"
```
The Federated Learning training begins and the number of rounds in this example is 5. In the first round, the model weights remain unchanged, but subsequent rounds will utilize the 'avg_weight' returned by the server. After completing the fitting process, the client will pack its information, such as the local weights, into JSON format for sending to the server.
```
for comm_round in range(5):
    print('The ',comm_round+1, 'round')
    client_model = smlp_model.build(17, 1)
    client_model.compile(loss=loss, 
                  optimizer=optimizer, 
                  metrics=metrics)
    
    if(comm_round == 0):
        history = client_model.fit(dataset, epochs=50, verbose=1)
    else:
        client_model.set_weights(avg_weight)
        history = client_model.fit(dataset, epochs=50, verbose=1)
    
    local_weight = client_model.get_weights()
    local_weight = [np.array(w).tolist() for w in local_weight]
    
    client_data = {"local_count": local_count,'bs': bs, 'local_weight': json.dumps(local_weight)}
```

Then the clients will use a 'request' to send data to the server. But the clients and servers are established at the same time, so use a 'while True' loop to continuously check if the server has been established. After receiving a response, the client will obtain the average weight for the next model fitting.
```
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
```

When all FL rounds are finished, clients will send a shutdown request to the server.
```
shutdown_url="http://http-service:5000/shutdown"    
try:
    response = requests.get(shutdown_url)
except requests.exceptions.ConnectionError:
    print('already shutdown')
```

Pipeline
---

Transfer the client and server functions to Kubeflow functions using 'func_to_container_op'.
```
server_op=func_to_container_op(server,base_image='tensorflow/tensorflow',packages_to_install=['flask','pandas'])
client_op=func_to_container_op(client,base_image='tensorflow/tensorflow',packages_to_install=['requests','pandas'])
```
Create a Kubernetes service to allow clients to connect to the server. Ensure that the 'selector' (label) and 'targetPort' match with the server.

```
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
)
```
Make the server task and add the label. Ensure that the 'label' and the 'container_port' match with the service.
```
server_task=server_op()
server_task.add_pod_label('app', 'http-service')
server_task.add_port(V1ContainerPort(name='my-port', container_port=8080))
server_task.set_cpu_request('0.2').set_cpu_limit('0.2')
server_task.after(service)
```
Delete the service after server shutdown. Use action="delete".
```
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
        action="delete" #delete
    ).after(server_task)
```
Create clients. In this example is 2 clients.


```
client_task_1=client_op(1)
client_task_1.set_cpu_request('0.2').set_cpu_limit('0.2')
clienttask_2=client_op(2)
clienttask_2.set_cpu_request('0.2').set_cpu_limit('0.2')
```

The federated learning pipeline.


![](https://github.com/sefgsefg/Federated-Learning-on-kubeflow/blob/main/kubeflow_pipeline_v2.png)

Here is the client's log in example output.
![](https://github.com/sefgsefg/Federated-Learning-on-kubeflow/blob/main/client_log.png)


Reference
---
https://github.com/stijani/tutorial?fbclid=IwAR2AvmE3DzXzuF6MxHuVUaP7_KLyOVIZK679d548jR2Gx4PlXKjZOU_DzuM

