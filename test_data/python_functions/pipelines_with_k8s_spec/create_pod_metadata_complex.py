from kfp import dsl, compiler
from kfp.kubernetes import add_pod_annotation
from kfp.kubernetes import add_pod_label
from kfp.kubernetes import use_field_path_as_env


@dsl.component
def validate_pod_metadata(annotation_path_1: str = None,
                          annotation_exp_val_1: str = None,
                          annotation_path_2: str = None,
                          annotation_exp_val_2: str = None,
                          label_path_1: str = None,
                          label_exp_val_1: str = None,
                          label_path_2: str = None,
                          label_exp_val_2: str = None) -> bool:
    import os

    validated_metadata_count = 0
    if annotation_path_1 is not None:
        annotation_val_1 = os.getenv(annotation_path_1)
        if annotation_val_1 is not None and annotation_val_1 != annotation_exp_val_1:
            raise ValueError(
                f"Pod annotation is {annotation_val_1} but is supposed to be {annotation_exp_val_1}."
            )
        validated_metadata_count+=1
    if annotation_path_2 is not None:
        annotation_val_2 = os.getenv(annotation_path_2)
        if annotation_val_2 is not None and annotation_val_2 != annotation_exp_val_2:
            raise ValueError(
                f"Pod annotation is {annotation_val_2} but is supposed to be {annotation_exp_val_2}."
            )
        validated_metadata_count+=1
    if label_path_1 is not None:
        label_val_1 = os.getenv(label_path_1)
        if label_val_1 is not None and label_val_1 != label_exp_val_1:
            raise ValueError(
                f"Pod label is {label_val_1} but is supposed to be {label_exp_val_1}."
            )
        validated_metadata_count+=1
    if label_path_2 is not None:
        label_val_2 = os.getenv(label_path_2)
        if label_val_2 is not None and label_val_2 != label_exp_val_2:
            raise ValueError(
                f"Pod label is {label_val_2} but is supposed to be {label_exp_val_2}."
            )
        validated_metadata_count+=1
    if validated_metadata_count <1:
        raise RuntimeError(f"No pod metadata found to validate.")
    return True

@dsl.component
def validate_no_pod_metadata(annotation_path: str, label_path: str) -> bool:
    import os
    annotation = os.getenv(annotation_path)
    if annotation != "":
        raise ValueError(
            f"Pod annotation is {annotation} but is supposed to be None."
        )
    label = os.getenv(label_path)
    if label != "":
        raise ValueError(
            f"Pod label is {label} but is supposed to be None."
        )
    return True


@dsl.pipeline
def pipeline_with_pod_metadata():
    # task_a and task_b are set with different metadata and the separate pods representing each component should have
    # separate, different metadata.
    task_a = validate_pod_metadata(
        annotation_path_1="POD_TASK_ANNOTATION",
        annotation_exp_val_1='annotation',
        label_path_1="POD_TASK_LABEL_1",
        label_exp_val_1='label-1',
        label_path_2="POD_TASK_LABEL_2",
        label_exp_val_2="label-2").set_caching_options(False)
    add_pod_label(task_a, 'task-label-1', 'label-1')
    add_pod_label(task_a, 'task-label-2', 'label-2')
    add_pod_annotation(task_a, 'task-annotation', 'annotation')
    # expose pod metadata annotation and label in container
    use_field_path_as_env(task_a, 'POD_TASK_ANNOTATION',
                          "metadata.annotations['task-annotation']")
    use_field_path_as_env(task_a, 'POD_TASK_LABEL_1', "metadata.labels['task-label-1']")
    use_field_path_as_env(task_a, "POD_TASK_LABEL_2", "metadata.labels['task-label-2']")

    task_b = validate_pod_metadata(
        annotation_path_1="POD_TASK_ANNOTATION_1",
        annotation_exp_val_1='annotation-1',
        annotation_path_2="POD_TASK_ANNOTATION_2",
        annotation_exp_val_2="annotation-2").set_caching_options(False)
    add_pod_annotation(task_b, 'task-annotation-1', 'annotation-1')
    add_pod_annotation(task_b, 'task-annotation-2', 'annotation-2')

    use_field_path_as_env(task_b, 'POD_TASK_ANNOTATION_1', "metadata.annotations['task-annotation-1']")
    use_field_path_as_env(task_b, "POD_TASK_ANNOTATION_2", "metadata.annotations['task-annotation-2']")

    # task c is set with no metadata
    task_c = validate_no_pod_metadata(annotation_path="POD_TASK_ANNOTATION", label_path="POD_TASK_LABEL").set_caching_options(False)

if __name__ == '__main__':
    compiler.Compiler().compile(
        pipeline_func=pipeline_with_pod_metadata,
        package_path=__file__.replace('.py', '.yaml'))

