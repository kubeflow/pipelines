FROM python:3.11-slim
COPY dist/*.whl /wheels/
RUN pip install --no-cache-dir --no-deps /wheels/*.whl
COPY samples/argo-native/components.py /demo.py
# Runtime imports only the functions, just like a lightweight component image.
RUN python - <<'PY'
import ast
from pathlib import Path
path = Path('/demo.py')
module = ast.parse(path.read_text())
for node in module.body:
    if isinstance(node, ast.FunctionDef):
        node.decorator_list = []
path.write_text(ast.unparse(module))
PY
ENV _KFP_RUNTIME=true
