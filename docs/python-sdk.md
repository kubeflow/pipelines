# Python SDK

The Kubeflow Pipelines SDK (the [`kfp`](https://pypi.org/project/kfp/) Python
package) is the primary way to author pipelines and components. You define your
workflow as Python code, compile it to a platform-neutral
[IR YAML](concepts/ir-yaml.md), and submit it to run on any KFP-conformant
backend.

This section covers installing the SDK, a quickstart, and the auto-generated API
and command line reference.

```{toctree}
:maxdepth: 1

Quickstart <sdk/source/quickstart>
Install the SDK <sdk/source/installation>
GenAI <sdk/source/genai>
API Reference <sdk/source/kfp>
Command Line Interface <sdk/source/cli>
```
