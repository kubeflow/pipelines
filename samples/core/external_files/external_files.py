import os
import kfp
from kfp import dsl


def json_op(json: str):
    op = dsl.ContainerOp(
        name='json',
        image='library/bash:4.4.23',
        command=['sh', '-c'],
    )
    op.command += [fr"""
cat << 'EOF' > hello_world.json
{json}
EOF
cat hello_world.json
"""]
    return op


def py_op(py: str):
    op = dsl.ContainerOp(
        name='py',
        image='python:3.8',
        command=['sh', '-c'],
    )
    op.command += [fr"""
cat << 'EOF' > hello_world.py
{py}
EOF
python hello_world.py
"""]
    return op


def yaml_op(yaml: str):
    op = dsl.ContainerOp(
        name='yaml',
        image='library/bash:4.4.23',
        command=['sh', '-c'],
    )
    op.command += [fr"""
cat << 'EOF' > hello_world.yaml
{yaml}
EOF

cat hello_world.yaml
"""]
    return op


def sh_op(sh: str):
    op = dsl.ContainerOp(
        name='sh',
        image='library/bash:4.4.23',
        command=['sh', '-c'],
    )
    op.command += [fr"""
{sh}
"""]
    return op


@dsl.pipeline(
    name='external-files',
    description='Use external files'
)
def pipeline():
    dirname = os.path.dirname(__file__)
    with open(f"{dirname}/hello_world.json") as f:
        json = f.read()
    with open(f"{dirname}/hello_world.py") as f:
        py = f.read()
    with open(f"{dirname}/hello_world.sh") as f:
        sh = f.read()
    with open(f"{dirname}/hello_world.yaml") as f:
        yaml = f.read()

    json_op(json)
    py_op(py)
    sh_op(sh)
    yaml_op(yaml)


if __name__ == '__main__':
    kfp.compiler.Compiler().compile(pipeline, __file__ + '.yaml')
