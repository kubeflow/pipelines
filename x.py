from kubernetes import client, config

config.load_kube_config()
v1 = client.CoreV1Api()

namespaces_list = v1.list_namespace()
namespaces = [item.metadata.name for item in namespaces_list.items]
print(namespaces)
pod_list = v1.list_namespaced_pod(namespace='default')
pods = [item.metadata.name for item in pod_list.items]
print(pod_list)
containers = []
container1 = client.V1Container(name='my-nginx-container', image='nginx')
containers.append(container1)

pod_spec = client.V1PodSpec(containers=containers)
pod_metadata = client.V1ObjectMeta(name='my-pod', namespace='default')

pod_body = client.V1Pod(api_version='v1',
                        kind='Pod',
                        metadata=pod_metadata,
                        spec=pod_spec)
# v1.create_namespaced_pod(namespace='default', body=pod_body)

# pod_logs = v1.read_namespaced_pod_log(name='my-pod', namespace='default')

# v1.delete_namespaced_pod(namespace='default', name='my-pod')