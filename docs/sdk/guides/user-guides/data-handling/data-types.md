# Data Types

KFP components and pipelines can accept inputs and create outputs. To do so, they must declare typed interfaces through their function signatures and annotations.

There are two groups of types in KFP: parameters and artifacts. Parameters are useful for passing small amounts of data between components. Artifacts types are the mechanism by which KFP provides first-class support for ML artifact outputs, such as datasets, models, metrics, etc.

So far [Hello World pipeline][hello-world] and the examples in [Components][components] have demonstrated how to use input and output parameters.

KFP automatically tracks the way parameters and artifacts are passed between components and stores the this data passing history in [ML Metadata][ml-metadata]. This enables out-of-the-box ML artifact lineage tracking and easily reproducible pipeline executions. Furthermore, KFP's strongly-typed components provide a data contract between tasks in a pipeline.

[hello-world]: ../../getting-started.md
[components]: ../components/index.md
[ml-metadata]: https://github.com/google/ml-metadata
