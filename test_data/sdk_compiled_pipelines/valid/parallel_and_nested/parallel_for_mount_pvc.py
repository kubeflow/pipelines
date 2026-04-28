"""Pipeline: Single component inside ParallelFor using mount_pvc.

Regression coverage for pipeline-parameter propagation through ParallelFor
when the parameter is only referenced from Kubernetes platform config.
"""
from kfp import compiler
from kfp import dsl
from kfp import kubernetes


@dsl.component(base_image="python:3.11")
def example_component(item: str) -> str:
    print(f"Item: {item}")
    return item


@dsl.pipeline(name="pipeline-parallel-for-mount-pvc")
def pipeline_parallel_for_mount_pvc(my_pvc_name_suffix: str = "test-pvc"):
    """Pipeline with 1 component inside ParallelFor using mount_pvc."""
    pvc = kubernetes.CreatePVC(
        pvc_name_suffix='-my-pvc',
        access_modes=['ReadWriteOnce'],
        size='5Mi',
        storage_class_name='standard',
    )

    with dsl.ParallelFor(items=["a"]) as item:
        task = example_component(item=item)
        kubernetes.mount_pvc(
            task,
            pvc_name=pvc.outputs["name"],
            mount_path="/data",
        )


if __name__ == "__main__":
    compiler.Compiler().compile(
        pipeline_func=pipeline_parallel_for_mount_pvc,
        package_path="parallel_for_mount_pvc.yaml",
    )
    print("Compiled parallel_for_mount_pvc.yaml")
