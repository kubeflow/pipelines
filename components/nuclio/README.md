# Nuclio (Serverless) Components

[Nuclio](https://nuclio.io/) is a native and high performance serverless platform over Kubernetes
which automate the process of build, deployment, monitoring, and auto-scaling of micro-services.
Nuclio support variety of data and data-science related features (e.g. stream processing, 
GPUs, volume/DB mounts, high concurrency, etc.)

To install Nuclio over Kubernetes follow the [instruction in Github](https://github.com/nuclio/nuclio), 
or this [interactive tutorial](https://www.katacoda.com/javajon/courses/kubernetes-serverless/nuclio).

Nuclio functions can be used in the following ML pipline tasks:
* Data collectors, ETL, stream processing
* Data preparation and analysis
* Hyper parameter model training
* Real-time model serving
* Feature vector assembly (real-time data preparation)
 
Read more on the use of Nuclio in [data-science here](https://towardsdatascience.com/serverless-can-it-simplify-data-science-projects-8821369bf3bd).
Nuclio functions can be generated automatically from 8 code languages, from Jupyter Notebooks, Zip, Git, Docker, etc.
The [nuclio-jupyter repo](https://github.com/nuclio/nuclio-jupyter) provide guidance and many examples.
 
## Components 
 
There are currently 3 components in this package:
* [deploy](deploy/component.yaml) - Automatically build and deploy/re-deploy functions
  from code/zip/notebooks/git/.. and/or override various deployment configurations such as
  setting cpu/mem/gpu resources, scaling, environment variables, triggers, etc.
* [delete](delete/component.yaml) - Delete a function
* [invoker](invoker/component.yaml) - invoke a function and return the results/logs
 
Additional components and examples will be added soon for parallel batch/stream processing    

## Examples

**Deploy a function (from Github)**

```python
nuclio_dep = kfp.components.load_component_from_file('deploy/component.yaml')

def my_pipeline():
    new_func = nuclio_dep(url='git://github.com/nuclio/nuclio#master:/hack/examples/python/helloworld', name='myfunc', project='myproj', tag='0.11') 
    
    ...
```
