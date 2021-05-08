

#### build image
* 方法一
```
docker build -t docker.io/chongchuanbing/kubeflow-launcher-component:v1.2.1 .
docker push docker.io/chongchuanbing/kubeflow-launcher-component:v1.2.1
```

#### Versions
| 版本 | 说明 |
| ---    | --- |
| v1     | for循环查询接口信息进行条件判断 |
| v1.1   | k8s watch监听<br>crd子pod失败判断<br>基础镜像修改 |
| v1.1.1 | asyncio + k8s client-python优化 |
| v1.2.1 | 集成mpi |