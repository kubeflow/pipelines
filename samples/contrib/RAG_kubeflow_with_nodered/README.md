# RAG-on-kubeflow-with-nodered
A simple example of RAG on kubeflow with node-red

## Implementation

```
git clone https://github.com/sefgsefg/RAG-pipeline-with-nodered.git
```

```
cd RAG_with_node-red/example
```

```
./run.sh main
```

Problem solve: -bash: ./run.sh: Permission denied
```
chmod +x run.sh
```

```
cd scripts
```

```
chmod +x entrypoint.sh
```

```
cd ..
```
Run ./run.sh main again
```
./run.sh main
```
1.Insstall dependency and build the RAG flow on node-red.

![](https://github.com/sefgsefg/RAG-pipeline-with-nodered/blob/main/node-red_1.png)

2.Double click the RAG node and edit the infos.

![](https://github.com/sefgsefg/RAG-pipeline-with-nodered/blob/main/node-red_2.png)

3.Deploy and run the flow. It will run on the kubeflow pipeline.

![](https://github.com/sefgsefg/RAG-pipeline-with-nodered/blob/main/pipeline.png)
