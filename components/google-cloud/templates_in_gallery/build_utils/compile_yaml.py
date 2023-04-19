"""Compiles python DSL code into YAML file."""

from collections.abc import Sequence
import importlib
import re

from absl import app
from absl import flags
from absl import logging
from kfp import compiler


_DSL = flags.DEFINE_string(
    "dsl", None, "Path to the dsl file in GCPC.", required=True
)

_FUNC = flags.DEFINE_string(
    "func", None, "The name of the pipeline function to compile.", required=True
)

_OUTPUT = flags.DEFINE_string(
    "output", None, "Where to write output files.", required=True
)

GCPC_PATTERN = re.compile(
    r"third_party/py/(google_cloud_pipeline_components/google_cloud_pipeline_components/.+)\.py$"
)

MODULE_IMPORT_HELPER = """
This compilation binary requires your python code being importable when GCPC is specified in the deps on the build rule.
For example, please test that following code is compilable

```python
# test_import.py

### UPDATE THIS ###
YOUR_DSL_FILE_PATH = "third_party/py/google_cloud_pipeline_components/google_cloud_pipeline_components/path/to/your/component.py"
###################

import importlib
import re
GCPC_PATTERN = re.compile(r"third_party/py/(google_cloud_pipeline_components/google_cloud_pipeline_components/.+)\.py$")
dsl_module_path = GCPC_PATTERN.search(YOUR_DSL_FILE_PATH).group(1).replace("/", ".")
importlib.import_module(dsl_module_path)
```

using this build rule
```
# BUILD
...

py_binary(
    name = "test_import",
    srcs = ["test_import.py"],
    python_version = "PY3",
    deps = ["//third_party/py/google_cloud_pipeline_components/google_cloud_pipeline_components"],
)
```
"""


def main(argv: Sequence[str]) -> None:
  del argv  # Unused.
  logging.info(
      "Prepare to compile %s from %s to %s.",
      _FUNC.value,
      _DSL.value,
      _OUTPUT.value,
  )
  match = GCPC_PATTERN.search(_DSL.value)
  assert (
      match
  ), f"Expect dsl file from {GCPC_PATTERN} but actually from {_DSL.value}"
  dsl_module_path = match.group(1).replace("/", ".")
  print(dsl_module_path)
  try:
    tmpl = importlib.import_module(dsl_module_path)
  except ModuleNotFoundError as e:
    logging.error(e)
    logging.error(MODULE_IMPORT_HELPER)
    return
  compiler.Compiler().compile(
      pipeline_func=getattr(tmpl, _FUNC.value),
      package_path=_OUTPUT.value,
  )


if __name__ == "__main__":
  app.run(main)
