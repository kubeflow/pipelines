from typing import Callable

import kfp.dsl as dsl

def add_common_labels(param):

    def _add_common_labels(op: dsl.ContainerOp) -> dsl.ContainerOp:
        return op.add_pod_label('param', param)

    return _add_common_labels

@dsl.pipeline(
    name="Parameters in Op transformation functions",
    description="Test that parameters used in Op transformation functions as pod labels "
                "would be correcly identified and set as arguments in he generated yaml"
)
def param_substitutions(param):
    dsl.get_pipeline_conf().op_transformers.append(add_common_labels(param))

    op = dsl.ContainerOp(
        name="cop",
        image="image",
    )


if __name__ == '__main__':
    import kfp.compiler as compiler
    compiler.Compiler().compile(param_substitutions, __file__ + '.yaml')
