# Python SDK

The Kubeflow Pipelines SDK (the [`kfp`](https://pypi.org/project/kfp/) Python
package) is the primary way to author pipelines and components. You define your
workflow as Python code, compile it to a platform-neutral
[IR YAML](concepts/ir-yaml.md), and submit it to run on any KFP-conformant
backend.

This section covers installing the SDK, a quickstart, and the auto-generated API
and command line reference.

## Python version support

Current KFP packages require **Python 3.11 or newer**. Python 3.9 and 3.10 are no
longer supported. This applies to `kfp`, `kfp-kubernetes`, `kfp-pipeline-spec`, and
`kfp-server-api`, as well as the repository's development tools.

Python 3.9 reached upstream end of life in October 2025, and Python 3.10 reaches
end of life in October 2026. The [support decision](https://github.com/kubeflow/pipelines/issues/14534)
replaces the previously announced Python 3.10 successor with Python 3.11, avoiding
another immediate migration. See the [Python lifecycle](https://devguide.python.org/versions/).

Before upgrading to a KFP release with this requirement:

1. Upgrade the interpreter in your authoring environment, notebooks, and CI to
   Python 3.11 or newer, and recreate virtual environments.
2. Update custom component images that install the new KFP packages, rebuild
   them, and refresh their dependency locks. The default lightweight component
   image already uses Python 3.11.
3. Recompile and test representative pipelines with the updated SDK and images.

The Python interpreter used to author a pipeline and the interpreter inside
its component images are separate. Updating one does not update the other.
Generic container components can still use other languages or runtimes when
those containers do not install the new KFP packages.

Previously published KFP releases retain their declared interpreter support;
use a compatible older release while migrating an older environment. This
change does not rewrite existing compiled pipelines or promise continued fixes
for end-of-life Python interpreters.

```{toctree}
:maxdepth: 1

Quickstart <sdk/source/quickstart>
Install the SDK <sdk/source/installation>
GenAI <sdk/source/genai>
API Reference <sdk/source/kfp>
Command Line Interface <sdk/source/cli>
```
